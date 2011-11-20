package main

import (
	"os"
	"go/build"
	"path/filepath"
	"http"
	"log"
	"template"
)

var webresources = make(map[string] string)

/*
// Echo the data received on the Web Socket.
func echoServer(ws *websocket.Conn) {
	writeInfo("Message received from websocket")
	io.Copy(ws, ws);
}
*/

func StartWebgui() os.Error {

	setVariables()
	// start up a local web server

	writeInfof("Starting up web server on port %d, click or copy this link to open up the page: %s\n", WEBLOG_PORT, hosturl)

	// find and serve the goconvert files
	t, _, err := build.FindTree(basePkg)

	if err != nil {
		log.Printf("Couldn't find goconvert files: %v\n", err)
	} else {
		root := filepath.Join(t.SrcDir(), basePkg, webroot)

		for _, tmpl := range []string{"index.html"} {
			fp := filepath.Join(root, tmpl)
			s, e := os.Stat(fp)
			writeInfo("File", fp, "exists", e == nil && !s.IsDirectory())
			t := template.Must(template.ParseFile(fp))
			templates[tmpl] = t
		}

		writeInfo("Serving content from", root)

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			//writeInfo("Handler for / called. URL.Path = " + r.URL.Path)
			if r.URL.Path == "/favicon.ico" { //|| r.URL.Path == "/" {
				fn := filepath.Join(root, r.URL.Path[1:])
				http.ServeFile(w, r, fn)
				return
			} else {
				tmpl := r.URL.Path[1:]
				if len(tmpl) == 0 {
					tmpl = "index.html"
				}
				//fn := filepath.Join(root, r.URL.Path[1:])
				//http.ServeFile(w, r, fn)
				_, ok := templates[tmpl]
				if !ok {
					fp := filepath.Join(root, r.URL.Path[1:])
					writeInfo(fp)
					http.ServeFile(w, r, fp)
					return
				}
				p := &Page{WebPort: WEBLOG_PORT}
				renderTemplate(w, tmpl, p)
				return
			}
			http.Error(w, "not found", 404)
		})
		http.Handle("/"+webroot+"/", http.FileServer(http.Dir(root)))

		// websocket
		//http.Handle("/echo", websocket.Handler(echoServer))
		//http.Handle("/echo", websocket.Draft75Handler(echoServer))
		server := NewServer(nil)
		serverAddr := server.Listener.Addr().String()
		log.Print("Test WebSocket server listening on ", serverAddr)

		writeInfof("Serving at http://%s/\n", serverAddr)
		// go http.ListenAndServe(*httpListen, nil)


	}
	// go http.ListenAndServe(":" + strconv.Itoa(WEBLOG_PORT), nil)

	return nil
}
