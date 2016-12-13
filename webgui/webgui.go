package webgui

import (
	"errors"
	"os"
	//"go/build"
	"encoding/json"
	"fmt"
	"html/template"
	"io"

	"github.com/mezzato/goconvert/imageconvert"
	lg "github.com/mezzato/goconvert/logger"
	settings "github.com/mezzato/goconvert/settings"
	//"io/ioutil"
	//"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Page struct {
	Title          string
	WebPort        int
	SettingsAsJson string
}

type appendSliceWriter struct {
	Buffer *[]string
	Eof    bool
}

func (s *appendSliceWriter) Reset() error {

	b := make([]string, 0, 100)
	s.Buffer = &b
	s.Eof = false

	return nil
}

func (s *appendSliceWriter) Write(p []byte) (int, error) {
	*s.Buffer = append(*s.Buffer, string(p))
	//imageconvert.WriteInfof("The applendSliceWriter.Write slice length is: %d", len(w))
	return len(p), nil
}

func (s *appendSliceWriter) ReadAll() (lines []string) {
	n := len(*s.Buffer)
	//imageconvert.WriteInfof("The applendSliceWriter.ReadAll() return slice length is: %d", n)
	if n == 0 {
		return *s.Buffer
	}

	r := (*s.Buffer)[0:n]
	// trim the buffer
	*s.Buffer = (*s.Buffer)[n:]
	return r
}

type Response struct {
	Messages []string `json:"messages"`
	Errors   []string `json:"compile_errors"`
	Eof      bool     `json:"eof"`
}

type requestProcessor func(r *http.Request) (msgs []string, err error, eof bool)

var (
	templates       = make(map[string]*template.Template)
	logger          *appendSliceWriter
	slogger         lg.SemanticLogger  = lg.NewConsoleSemanticLogger("goconvert", os.Stdout, lg.DEBUG)
	homeImgDir                         = filepath.Join(settings.GetHomeDir(), "Pictures", "ToResize")
	defaultSettings *settings.Settings = settings.NewDefaultSettings("", homeImgDir)
	compressing     bool
	quitChannel     chan bool
)

var webresources = make(map[string]string)

var webroot string = "website"

/*
// Echo the data received on the Web Socket.
func echoServer(ws *websocket.Conn) {
	WriteInfo("Message received from websocket")
	io.Copy(ws, ws);
}
*/

func StartWebgui() (browserCmd *exec.Cmd, server *Server, err error) {

	setVariables()
	// start up a local web server

	b := make([]string, 0, 100)
	logger = &appendSliceWriter{Buffer: &b}

	slogger.Info(fmt.Sprintf("Starting up web server on port %d, click or copy this link to open up the page: %s", WEBLOG_PORT, hosturl))

	// find and serve the goconvert files
	//t, _, err := build.FindTree(basePkg)

	if err != nil {
		slogger.Info(fmt.Sprintf("Couldn't find goconvert files: %v", err))
	} else {
		//root := webroot //filepath.Join(t.SrcDir(), basePkg, webroot)

		for k, v := range webresources {
			//fp := filepath.Join(root, tmpl)
			//s, e := os.Stat(fp)

			// write out to let it be served later as a static file
			fp := filepath.Join(webroot, k)
			slogger.Info(fmt.Sprintf("Deploying resource to file system:%s", fp))
			err = createFileAndWriteText(fp, v)
			if err != nil {
				return
			}

		}

		slogger.Info(fmt.Sprintf("Serving content from %s", webroot))

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			//WriteInfo("Handler for / called. URL.Path = " + r.URL.Path)
			if r.URL.Path == "/favicon.ico" { //|| r.URL.Path == "/" {
				fn := filepath.Join(webroot, r.URL.Path[1:])
				http.ServeFile(w, r, fn)
				return
			} else {
				fkey := r.URL.Path[1:]
				if len(fkey) == 0 {
					fkey = "index.html"
				}
				fp := webroot + "/" + fkey

				//fn := filepath.Join(root, r.URL.Path[1:])
				//http.ServeFile(w, r, fn)
				//_, ok := templates[fkey]
				if !strings.HasSuffix(fkey, "html") {
					//fp := filepath.Join(webroot, r.URL.Path[1:])
					slogger.Info(fmt.Sprintf("Serving static resource:%s", fp))
					http.ServeFile(w, r, fp)
					return
				}

				settingsAsJson, err := json.Marshal(defaultSettings)
				if err != nil {
					return
				}

				p := &Page{WebPort: WEBLOG_PORT, SettingsAsJson: string(settingsAsJson)}
				renderTemplate(w, fkey, p)
				return
			}
			http.Error(w, "not found", 404)
		})
		http.Handle("/"+webroot+"/", http.FileServer(http.Dir(webroot)))

		// web socket
		http.Handle("/socket", websocketHandler)
		http.HandleFunc("/compress", wrapHandler(compress))
		http.HandleFunc("/compress/status", wrapHandler(compressStatus))
		http.HandleFunc("/cancel", wrapHandler(stopCompressing))

		// websocket
		//http.Handle("/echo", websocket.Handler(echoServer))
		//http.Handle("/echo", websocket.Draft75Handler(echoServer))
		server = NewServer(nil)
		serverAddr := server.Listener.Addr().String()
		slogger.Info(fmt.Sprintf("Test WebSocket server listening on %s", serverAddr))

		slogger.Info(fmt.Sprintf("Serving at http://%s/", serverAddr))
		// go http.ListenAndServe(*httpListen, nil)
		browserCmd, _ = runBrowser(".", serverAddr)

	}
	// go http.ListenAndServe(":" + strconv.Itoa(WEBLOG_PORT), nil)

	return
}

func createFileAndWriteText(fp string, text string) (err error) {
	dir, _ := filepath.Split(fp)
	_, e := os.Stat(dir)
	if e != nil {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return
		}
	}

	var f *os.File
	/*
		_, e = os.Stat(fp)
		if e != nil {
			WriteInfo("Creating file:", fp)
			f, err = os.Create(fp)
		} else {
			f, err = os.Open(fp)
		}
	*/

	slogger.Info(fmt.Sprintf("Creating file:%s", fp))
	f, err = os.Create(fp)

	if err != nil {
		return
	}

	defer f.Close()
	_, err = io.WriteString(f, text)
	return

}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {

	ok := false
	var t *template.Template

	t, ok = templates[tmpl]

	if !ok {
		fp := filepath.Join(webroot, tmpl)
		s, e := os.Stat(fp)
		if e == nil && !s.IsDir() {
			t = template.Must(template.New(tmpl).ParseFiles(fp))
			slogger.Info(fmt.Sprintf("Loaded template \"%s\" from file system.", fp))
		} else {
			http.Error(w, fmt.Sprintf("template %s does not exist", tmpl), http.StatusInternalServerError)
			return
		}

		// cache the template
		if !settings.Debug {
			//t := template.Must(template.ParseFile(fp))
			templates[tmpl] = t
		}
	}

	ctype := "text/html; charset=utf-8"
	w.Header().Set("Content-Type", ctype)
	err := t.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// run is a simple wrapper for exec.Run/Close
func runBrowser(dir string, url string) (cmd *exec.Cmd, err error) {

	browsers := []string{"google-chrome", "firefox"}

	switch runtime.GOOS {
	case "windows":
		browsers = []string{
			os.ExpandEnv("${LOCALAPPDATA}\\Google\\Chrome\\Application\\chrome.exe"),
			os.ExpandEnv("${PROGRAMFILES}\\Mozilla Firefox\\firefox"),
			os.ExpandEnv("${PROGRAMFILES}\\Internet Explorer\\iexplore"),
		}
	default:
		//
	}

	for _, b := range browsers {
		cmd = exec.Command(b, url)
		cmd.Dir = dir
		//cmd.Env = envv
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err == nil {
			return
		}
	}
	return nil, errors.New("No known browser could be started. Do it manually!")
}

func launchConversionFromWeb(settings *settings.Settings, logger *appendSliceWriter, compressionStatus *bool) (responseChannel chan *imageconvert.ConvertResponse, quitChannel chan bool, err error) {
	startNanosecs := time.Now()

	quitChannel = make(chan bool)

	io.WriteString(logger, "Analysing folder and starting up compression.")
	responseChannel, quit, fileno, collPublishFolder, err := imageconvert.Convert(
		settings.CollName,
		settings.SourceDir,
		settings.PublishDir,
		settings.PiwigoGalleryHighDirName,
		settings.ConversionSettings)

	if err != nil {
		return
	}

	go func() {

		defer func() {
			quit <- true // stopping the server
			*compressionStatus = false
			logger.Eof = true
			io.WriteString(logger, fmt.Sprintf("The conversion took %.3f seconds", float32(time.Now().Sub(startNanosecs))/1e9))
			io.WriteString(logger, "Images successfully resized to folder: "+collPublishFolder)
		}()

		*compressionStatus = true
		// collect responses
		slogger.Info(fmt.Sprintf("Collecting results. Number of images: %d", fileno))

		for i := 0; i < fileno; i++ {

			select {
			case <-quitChannel:
				io.WriteString(logger, "Compression cancelled by the user.")
				io.WriteString(logger, fmt.Sprintf("The conversion took %.3f seconds", float32(time.Now().Sub(startNanosecs))/1e9))
				return

			case r := <-responseChannel:
				fname := filepath.Base(r.ImgF.Path)
				var msg string
				if r.Error == nil {
					msg = fmt.Sprintf("Success, file %s resized and archived", fname)
				} else {
					msg = fmt.Sprintf("Error, file %s, the error was %s", fname, r.Error)
				}
				slogger.Info(msg)
				io.WriteString(logger, msg)
			}
		}

	}()

	return responseChannel, quitChannel, err
}
