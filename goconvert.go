package main

import (
	"path/filepath"
	"fmt"
	"os"
	ftp4go "ftp4go.googlecode.com/hg/ftp4go"
	"flag"
	"log"
	"strconv"
	"strings"
	"http"
	"io"
	"websocket"
	"go/build"
	"template"
	"http/httptest"
)

type LogLevel int

type Page struct {
	Title   string
	WebPort int
}

const (
	Info LogLevel = 1 << iota
	Verbose
	WEBLOG_PORT = 4999
	basePkg     = "goconvert.googlecode.com/hg"
)

var (
	LogLevelForRun LogLevel = Info
	argv0                   = os.Args[0]

	hosturl    = fmt.Sprintf("127.0.0.1:%d", WEBLOG_PORT)
	httpListen = flag.String("http", hosturl, "host:port to listen on")
	htmlOutput = flag.Bool("html", false, "render program output as HTML")
	templates  = make(map[string]*template.Template)
)

func writeLog(ll LogLevel, msgs ...interface{}) (n int, err os.Error) {
	if ll <= LogLevelForRun {
		return fmt.Println(msgs...)
	}
	return
}

func writeLogf(ll LogLevel, format string, msgs ...interface{}) (n int, err os.Error) {
	if ll <= LogLevelForRun {
		return fmt.Printf(format, msgs...)
	}
	return
}

func writeInfo(msgs ...interface{}) (n int, err os.Error) {
	return writeLog(Info, msgs...)
}

func writeInfof(format string, msgs ...interface{}) (n int, err os.Error) {
	return writeLogf(Info, format, msgs...)
}

func writeVerbose(msgs ...interface{}) (n int, err os.Error) {
	return writeLog(Verbose, msgs...)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hi there!")
}

// Echo the data received on the Web Socket.
func echoServer(ws *websocket.Conn) {
	writeInfo("Message received from websocket")
	io.Copy(ws, ws);
}

var webroot string = "web"

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	t, ok := templates[tmpl]
	if !ok {
		http.Error(w, fmt.Sprintf("template %s does not exist", tmpl), http.StatusInternalServerError)
		return
	}
	ctype := "text/html; charset=utf-8"
	w.Header().Set("Content-Type", ctype)
	err := t.Execute(w, data)
	if err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
}

func main() {

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

		
		writeInfof("Serving at http://%s/\n", *httpListen)
		go http.ListenAndServe(*httpListen, nil)
		
		// websocket
		http.Handle("/echo", websocket.Handler(echoServer))
    	server := httptest.NewServer(nil)
    	serverAddr := server.Listener.Addr().String()
    	log.Print("Test WebSocket server listening on ", serverAddr)
		
	}
	// go http.ListenAndServe(":" + strconv.Itoa(WEBLOG_PORT), nil)

	writeInfo(`
goconvert is a command line tool to convert, archive and upload images to an ftp server.

Type -help for help with the command line arguments.
Some examples:

linux 	-> ./goconvert -f a/b/myImageFolder -c mycollectionname
windows -> goconvert.exe -f "c:\myfolder with space\myimages" -c mycollectionname

Have fun!

	`)

	srcfolderFlag := flag.String("f", ".", "the image folder")
	collFlag := flag.String("c", "collectionnamewithoutspaces", "the collection name")
	LogLevelForRunFlag := flag.Int("l", int(Info), "The log level")
	flag.Parse()
	srcfolder := *srcfolderFlag
	if srcfolder == "." {
		writeInfo("No source folder specified, using the current: '.'")
	}

	LogLevelForRun = LogLevel(*LogLevelForRunFlag)

	// check existence
	if fi, err := os.Stat(srcfolder); err != nil || !fi.IsDirectory() {
		writeInfo(fmt.Sprintf("The folder '%s' is not a valid directory.", srcfolder))
		os.Exit(1)
	}

	collName := *collFlag

	if collName == "collectionnamewithoutspaces" {
		writeInfo("No collection name was specified, this is required to store your images\n")
		flag.Usage()
		os.Exit(1)
	}

	settings, err := AskForSettings(collName)
	if err != nil {
		log.Fatalf("A fatal error has occurred: %s", err)
	}

	var pad int = 30
	var padString string = strconv.Itoa(pad)

	var padS = func(s string) string {
		return fmt.Sprintf("%-"+padString+"s", s) + ": %s\n"
	}

	writeInfof(strings.Repeat("-", pad*2) + "\n")
	writeInfof("%"+padString+"s\n", "Settings")
	writeInfof(padS("Image folder"), srcfolder)
	writeInfof(padS("Collection name"), settings.CollName)
	writeInfof(padS("Home folder"), settings.HomeDir)
	writeInfof(padS("Publish folder"), settings.PublishDir)
	writeInfof(padS("Piwigo gallery"), settings.PiwigoGalleryDir)
	writeInfof(padS("Number of resize processes"), strconv.Itoa(settings.ConversionSettings.NoSimultaneousResize))
	writeInfof(padS("ftp server"), settings.FtpSettings.Address)
	writeInfof(padS("ftp user"), settings.FtpSettings.Username)
	writeInfof(strings.Repeat("-", pad*2) + "\n")

	collPublishFolder, err := Convert(
		settings.CollName,
		srcfolder,
		settings.PublishDir,
		settings.PiwigoGalleryHighDirName,
		settings.ConversionSettings)

	if err != nil {
		panic(err)
	}
	writeInfo("Images successfully resized")

	//ftpAddress := "mezzsplace.dyndns.org" //, _ := askParameter("The name of the ftp server:[mezzsplace.dyndns.org]", "mezzsplace.dyndns.org")
	//username := "enrico"                  //, _ := askParameter("The name of the user:[enrico]", "enrico")

	// ftpAddress := "mezzsplace.dyndns.org" //, _ := askParameter("The name of the ftp server:[mezzsplace.dyndns.org]", "mezzsplace.dyndns.org")
	// username := "enrico"                  //, _ := askParameter("The name of the user:[enrico]", "enrico")
	// password, _ := askParameter(fmt.Sprintf("The password for username %s:", username), "")

	if len(settings.FtpSettings.Address) == 0 {
		writeInfo("The ftp address was not specified and the upload will be skipped.")
		os.Exit(1)
	}

	ftpClient := ftp4go.NewFTP(0) // 1 for debugging
	ftpClient.SetPassive(true)

	/*
		var (
			resp *ftp4go.Response
		)
	*/

	writeInfo("Connecting to host", settings.FtpSettings.Address)
	// connect
	_, err = ftpClient.Connect(settings.FtpSettings.Address, ftp4go.DefaultFtpPort)
	if err != nil {
		writeInfo("The FTP connection could not be established, error: ", err.String())
		os.Exit(1)
	}

	defer ftpClient.Quit()

	_, err = ftpClient.Login(settings.FtpSettings.Username, settings.FtpSettings.Password, "")
	if err != nil {
		writeInfo("The FTP login was invalid, error: ", err.String())
		os.Exit(1)
	}

	err = PublishCollToFtp(
		ftpClient,
		collPublishFolder,
		"./piwigo/galleries",
		EXCLUDED_DIRS)

	if err != nil {
		writeInfo("Error upload to FTP, error: ", err)
		return
	} else {
		writeInfo("Files successfully uploaded")
	}

}

var EXCLUDED_DIRS []string = []string{"pwg_high"}

func PublishCollToFtp(fc *ftp4go.FTP, localDir string, remoteRoorDir string, excludedDirs []string) (err os.Error) {
	writeInfo(fmt.Sprintf("Publishing to FTP root folder: %s, from local directory: %s.\nExcluded folders:%s", remoteRoorDir, localDir, excludedDirs))

	collName := filepath.Base(localDir)
	// careful!!!
	if len(collName) > 0 {
		/*
			remoteDir := filepath.Join(remoteRoorDir, collName)
			writeInfo("Removing old ftp folder tree if present at:", remoteDir)
			err = fc.RemoveRemoteDirTree(remoteDir)
		*/
	} else {
		return os.NewError("The collection name can not be empty")
	}

	writeInfo("Uploading folder tree:", filepath.Base(localDir))
	maxSimultaneousConns := 4

	stats, fileUploaded, quit := startStats()
	// define the callback as an asychronous stats collector to pass information to the stats go routine
	var collector = func(info *ftp4go.CallbackInfo) {
		stats <- info // pipe in for stats
	}

	var n int
	n, err = fc.UploadDirTree(localDir, remoteRoorDir, maxSimultaneousConns, excludedDirs, collector)

	// wait for all stats to finish
	for k := 0; k < n; k++ {
		<-fileUploaded
	}
	quit <- true

	return
}

func startStats() (stats chan *ftp4go.CallbackInfo, fileUploaded chan bool, quit chan bool) {
	stats = make(chan *ftp4go.CallbackInfo) // no buffer, this never blocks the go routine
	quit = make(chan bool)
	fileUploaded = make(chan bool, 50)

	go func() {
		for {
			select {
			case st := <-stats:
				// do not wait here, the buffered request channel is the barrier
				go func() {
					if st.Eof {
						writeInfof("Successfully uploaded file: %s\n", st.Resourcename)
						fileUploaded <- true // done here
					}
				}()
			case <-quit:
				fmt.Println("Stopping collecting upload statistics")
				return // get out
			}
		}
	}()
	return
}
