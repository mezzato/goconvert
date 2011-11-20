package main

import (
	"path/filepath"
	"fmt"
	"os"
	ftp4go "ftp4go.googlecode.com/hg/ftp4go"
	"flag"
	//"log"
	//"strconv"
	//"strings"
	"http"
	//"io"
	//"websocket"

	"template"
	//"http/httptest"
)

type LogLevel int

type Page struct {
	Title   string
	WebPort int
}

const (
	Info LogLevel = 1 << iota
	Verbose
	basePkg = "goconvert.googlecode.com/hg"
)

var (
	LogLevelForRun LogLevel = Info
	argv0                   = os.Args[0]
	htmlOutput              = flag.Bool("html", false, "render program output as HTML")
	templates               = make(map[string]*template.Template)
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

	writeInfo(`
goconvert is a command line tool to convert, archive and upload images to an ftp server.

Type -help for help with the command line arguments.
Some examples:

linux 	-> ./goconvert -f a/b/myImageFolder -c mycollectionname
windows -> goconvert.exe -f "c:\myfolder with space\myimages" -c mycollectionname

Have fun!

	`)

	if err := StartWebgui(); err != nil {
		writeInfo("The local web server could not be started, using the console instead")
	}

	settings, err := StartConsolegui()
	if err != nil {
		writeInfo("Error while collecting the settings:", err)
		//return nil, os.NewError(fmt.Sprintf("The folder '%s' is not a valid directory.", srcfolder))
		os.Exit(1)
	}

	collPublishFolder, err := Convert(
		settings.CollName,
		settings.SourceDir,
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
