package main

import (
	"errors"
	"os"
	//"go/build"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type appendSliceWriter []string

func (w appendSliceWriter) Write(p []byte) (int, error) {
	w = append(w, string(p))
	return len(p), nil
}

func (w appendSliceWriter) ReadAll() (lines []string) {
	n := len(w)
	r := w[0 : n-1]
	// trim the buffer
	w = w[n:]
	return r
}

type Response struct {
	Output string `json:"output"`
	Errors string `json:"compile_errors"`
}

type JsonRequest struct {
	FolderPath     string `json:"folder"`
	CollectionName string `json:"collection"`
}

type requestProcessor func(r *http.Request) (msg string, err error)

var logger appendSliceWriter

func compress(r *http.Request) (msg string, err error) {
	var reader io.Reader = r.Body
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	var jsonR JsonRequest
	fmt.Println("request body: " + string(b))
	err = json.Unmarshal(b, &jsonR)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	s := NewDefaultSettings(jsonR.CollectionName, jsonR.FolderPath)

	launchConversionFromWeb(s, logger)

	return fmt.Sprintf("Compressing\nfolder: %s\nCollection name: %s", jsonR.FolderPath, jsonR.CollectionName), err
}

func compressStatus(r *http.Request) (msg string, err error) {
	newLines := logger.ReadAll()
	msg = ""
	if len(newLines) > 0 {
		msg = strings.Join(newLines, "\n")
	}
	return msg, nil
}

func wrapHandler(processor requestProcessor) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := new(Response)
		out, err := processor(r)
		if err != nil {
			if len(out) > 0 {
				resp.Errors = string(out)
			} else {
				resp.Errors = err.Error()
			}
		} else {
			resp.Output = string(out)
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
	writeInfo("Message received from websocket")
	io.Copy(ws, ws);
}
*/

func StartWebgui() (browserCmd *exec.Cmd, server *Server, err error) {

	setVariables()
	// start up a local web server

	logger = make(appendSliceWriter, 100)

	writeInfof("Starting up web server on port %d, click or copy this link to open up the page: %s\n", WEBLOG_PORT, hosturl)

	// find and serve the goconvert files
	//t, _, err := build.FindTree(basePkg)

	if err != nil {
		log.Printf("Couldn't find goconvert files: %v\n", err)
	} else {
		//root := webroot //filepath.Join(t.SrcDir(), basePkg, webroot)

		for k, v := range webresources {
			//fp := filepath.Join(root, tmpl)
			//s, e := os.Stat(fp)
			if !Debug && strings.HasSuffix(k, "html") {
				//writeInfo("File", fp, "exists", e == nil && !s.IsDirectory())
				writeInfo("File", k, "is a template")
				t := template.Must(template.New(k).Parse(v))
				//t := template.Must(template.ParseFile(fp))
				templates[k] = t
			} else {
				// write out to let it be served later as a static file
				fp := filepath.Join(webroot, k)
				writeInfo("Deploying resource to file system:", fp)
				err = createFileAndWriteText(fp, v)
				if err != nil {
					return
				}
			}
		}

		writeInfo("Serving content from", webroot)

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			//writeInfo("Handler for / called. URL.Path = " + r.URL.Path)
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
				_, ok := templates[fkey]
				if !ok {
					//fp := filepath.Join(webroot, r.URL.Path[1:])
					writeInfo("Serving static resource:", fp)
					http.ServeFile(w, r, fp)
					return
				}
				p := &Page{WebPort: WEBLOG_PORT}
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

		writeInfof("Serving at http://%s/\n", serverAddr)
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
			writeInfo("Creating file:", fp)
			f, err = os.Create(fp)
		} else {
			f, err = os.Open(fp)
		}
	*/

	writeInfo("Creating file:", fp)
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

	if Debug && strings.HasSuffix(tmpl, "html") {
		fp := filepath.Join(webroot, tmpl)
		s, e := os.Stat(fp)
		if e == nil && !s.IsDir() {
			t, e = template.ParseFiles(fp)
			ok = e == nil
		}
	} else {
		t, ok = templates[tmpl]
	}
	if !ok {
		http.Error(w, fmt.Sprintf("template %s does not exist", tmpl), http.StatusInternalServerError)
		return
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

	browsers := []string{"google-chrome", "chrome", "firefox", "iexplore"}
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
	return nil, errors.New("No browser could be started. Do it manually!")
}

func launchConversionFromWeb(settings *Settings, logger io.Writer) (responseChannel chan *response, quitChannel chan bool) {
	startNanosecs := time.Now()
	responseChannel, quitChannel, fileno, collPublishFolder, err := Convert(
		settings.CollName,
		settings.SourceDir,
		settings.PublishDir,
		settings.PiwigoGalleryHighDirName,
		settings.ConversionSettings)

	if err != nil {
		panic(err)
	}

	go func() {

		// collect responses
		writeInfo(fmt.Sprintf("Collecting results"))

		for i := 0; i < fileno; i++ {

			r := <-responseChannel
			fname := filepath.Base(r.imgF.path)
			var msg string
			if r.error == nil {
				msg = fmt.Sprintf("Success, file %s resized and archived\n", fname)
			} else {
				msg = fmt.Sprintf("Error, file %s, the error was %s", fname, r.error)
			}
			io.WriteString(logger, msg)
		}

		quitChannel <- true // stopping the server
		io.WriteString(logger, fmt.Sprintf("The conversion took %.3f seconds", float32(time.Now().Sub(startNanosecs))/1e9))
		io.WriteString(logger, "Images successfully resized to folder: "+collPublishFolder)
	}()

	return responseChannel, quitChannel
}
