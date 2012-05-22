package webgui

import (
	"errors"
	"os"
	//"go/build"
	"code.google.com/p/goconvert/imageconvert"
	settings "code.google.com/p/goconvert/settings"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Page struct {
	Title           string
	WebPort         int
	SettingsAsJson	string
}

type appendSliceWriter struct {
	Buffer []string
	Eof    bool
}

func (s *appendSliceWriter) Write(p []byte) (int, error) {
	s.Buffer = append(s.Buffer, string(p))
	//imageconvert.WriteInfof("The applendSliceWriter.Write slice length is: %d", len(w))
	return len(p), nil
}

func (s *appendSliceWriter) ReadAll() (lines []string) {
	n := len(s.Buffer)
	//imageconvert.WriteInfof("The applendSliceWriter.ReadAll() return slice length is: %d", n)
	if n == 0 {
		return s.Buffer
	}

	r := s.Buffer[0 : n-1]
	// trim the buffer
	s.Buffer = s.Buffer[n:]
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
	homeImgDir                         = filepath.Join(settings.GetHomeDir(), "Pictures", "ToResize")
	defaultSettings *settings.Settings = settings.NewDefaultSettings("", homeImgDir)
)

func compress(r *http.Request) (msg []string, err error, eof bool) {
	var reader io.Reader = r.Body
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		eof = true
		return
	}
	var jsonSettings *settings.Settings
	fmt.Println("request body: " + string(b))
	err = json.Unmarshal(b, &jsonSettings)
	if err != nil {
		fmt.Println(err)
		eof = true
		return
	}
	_, _, err = launchConversionFromWeb(jsonSettings, logger)

	return []string{fmt.Sprintf("Compressing\nfolder: %s\nCollection name: %s", jsonSettings.SourceDir, jsonSettings.CollName)}, err, err != nil
}

func compressStatus(r *http.Request) (msgs []string, err error, eof bool) {
	newLines := logger.ReadAll()
	if len(newLines) > 0 {
		msgs = newLines
	}
	return msgs, nil, logger.Eof
}

func wrapHandler(processor requestProcessor) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := new(Response)
		out, err, eof := processor(r)
		if err != nil {
			resp.Errors = []string{err.Error()}
			resp.Eof = true
		} else {
			resp.Messages = out
			resp.Eof = eof
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Println(err)
		}

	}
}

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

	logger = &appendSliceWriter{Buffer: make([]string, 0, 100)}

	imageconvert.WriteInfof("Starting up web server on port %d, click or copy this link to open up the page: %s\n", WEBLOG_PORT, hosturl)

	// find and serve the goconvert files
	//t, _, err := build.FindTree(basePkg)

	if err != nil {
		log.Printf("Couldn't find goconvert files: %v\n", err)
	} else {
		//root := webroot //filepath.Join(t.SrcDir(), basePkg, webroot)

		for k, v := range webresources {
			//fp := filepath.Join(root, tmpl)
			//s, e := os.Stat(fp)

			// write out to let it be served later as a static file
			fp := filepath.Join(webroot, k)
			imageconvert.WriteInfo("Deploying resource to file system:", fp)
			err = createFileAndWriteText(fp, v)
			if err != nil {
				return
			}

		}

		imageconvert.WriteInfo("Serving content from", webroot)

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
					imageconvert.WriteInfo("Serving static resource:", fp)
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

		http.HandleFunc("/compress", wrapHandler(compress))
		http.HandleFunc("/compress/status", wrapHandler(compressStatus))

		// websocket
		//http.Handle("/echo", websocket.Handler(echoServer))
		//http.Handle("/echo", websocket.Draft75Handler(echoServer))
		server = NewServer(nil)
		serverAddr := server.Listener.Addr().String()
		log.Print("Test WebSocket server listening on ", serverAddr)

		imageconvert.WriteInfof("Serving at http://%s/\n", serverAddr)
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

	imageconvert.WriteInfo("Creating file:", fp)
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
			imageconvert.WriteInfof("Loaded template \"%s\" from file system.\n", fp)
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

func launchConversionFromWeb(settings *settings.Settings, logger *appendSliceWriter) (responseChannel chan *imageconvert.Response, quitChannel chan bool, err error) {
	startNanosecs := time.Now()
	responseChannel, quitChannel, fileno, collPublishFolder, err := imageconvert.Convert(
		settings.CollName,
		settings.SourceDir,
		settings.PublishDir,
		settings.PiwigoGalleryHighDirName,
		settings.ConversionSettings)

	if err != nil {
		return
	}

	go func() {

		// collect responses
		imageconvert.WriteInfo(fmt.Sprintf("Collecting results. Number of images: %d", fileno))

		for i := 0; i < fileno; i++ {

			r := <-responseChannel
			fname := filepath.Base(r.ImgF.Path)
			var msg string
			if r.Error == nil {
				msg = fmt.Sprintf("Success, file %s resized and archived", fname)
			} else {
				msg = fmt.Sprintf("Error, file %s, the error was %s", fname, r.Error)
			}
			imageconvert.WriteInfo(msg)
			io.WriteString(logger, msg)
		}

		io.WriteString(logger, fmt.Sprintf("The conversion took %.3f seconds", float32(time.Now().Sub(startNanosecs))/1e9))
		io.WriteString(logger, "Images successfully resized to folder: "+collPublishFolder)
		logger.Eof = true
		quitChannel <- true // stopping the server
	}()

	return responseChannel, quitChannel, err
}
