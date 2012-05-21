package main

import (
	ftp4go "code.google.com/p/ftp4go"
	"code.google.com/p/goconvert/imageconvert"
	"code.google.com/p/goconvert/settings"
	webgui "code.google.com/p/goconvert/webgui"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hi there!")
}

func ParseCommandLine() (usewebgui bool, sourcefolder string, collectionname string) {

	debug := flag.Bool("d", false, "debug mode")
	webgui := flag.Bool("w", false, "whether to use a web browser instead of the command line")
	srcfolder := flag.String("f", ".", "the image folder")
	collname := flag.String("c", "collectionnamewithoutspaces", "the collection name")
	LogLevelForRunFlag := flag.Int("l", int(imageconvert.Info), "The log level")
	flag.Parse()

	imageconvert.LogLevelForRun = imageconvert.LogLevel(*LogLevelForRunFlag)
	settings.Debug = *debug

	return *webgui, *srcfolder, *collname
}

func GetSettings(srcfolder, collName string) (s *settings.Settings, err error) {

	if srcfolder == "." {
		imageconvert.WriteInfo("No source folder specified, using the current: '.'")
	}

	// check existence
	if fi, err := os.Stat(srcfolder); err != nil || !fi.IsDir() {
		//imageconvert.WriteInfo(fmt.Sprintf("The folder '%s' is not a valid directory.", srcfolder))
		return nil, errors.New(fmt.Sprintf("The folder '%s' is not a valid directory.", srcfolder))
		//os.Exit(1)
	}

	if collName == "collectionnamewithoutspaces" {
		imageconvert.WriteInfo("No collection name was specified, this is required to store your images\n")
		flag.Usage()
		return nil, errors.New("No collection name was specified, this is required to store your images\n")
		//os.Exit(1)
	}

	s, err = settings.AskForSettings(collName, srcfolder)
	if err != nil {
		log.Fatalf("A fatal error has occurred: %s", err)
	}

	var pad int = 30
	var padString string = strconv.Itoa(pad)

	var padS = func(s string) string {
		return fmt.Sprintf("%-"+padString+"s", s) + ": %s\n"
	}

	imageconvert.WriteInfof(strings.Repeat("-", pad*2) + "\n")
	imageconvert.WriteInfof("%"+padString+"s\n", "Settings")
	imageconvert.WriteInfof(padS("Image folder"), s.SourceDir)
	imageconvert.WriteInfof(padS("Collection name"), s.CollName)
	imageconvert.WriteInfof(padS("Home folder"), s.HomeDir)
	imageconvert.WriteInfof(padS("Publish folder"), s.PublishDir)
	imageconvert.WriteInfof(padS("Piwigo gallery"), s.PiwigoGalleryDir)
	imageconvert.WriteInfof(padS("Number of resize processes"), strconv.Itoa(s.ConversionSettings.NoSimultaneousResize))
	imageconvert.WriteInfof(padS("ftp server"), s.FtpSettings.Address)
	imageconvert.WriteInfof(padS("ftp user"), s.FtpSettings.Username)
	imageconvert.WriteInfof(strings.Repeat("-", pad*2) + "\n")

	return s, nil

}

func main() {

	imageconvert.WriteInfo(`
goconvert is a command line tool to convert, archive and upload images to an ftp server.

Type -help for help with the command line arguments.
Some examples:

linux 	-> ./goconvert -f a/b/myImageFolder -c mycollectionname
windows -> goconvert.exe -f "c:\myfolder with space\myimages" -c mycollectionname

Have fun!

	`)

	usewebgui, srcfolder, collectionname := ParseCommandLine()

	// remove this once tested
	usewebgui = true
	settings.Debug = true

	if usewebgui {
		browserCmd, server, err := webgui.StartWebgui()
		if err != nil {
			imageconvert.WriteInfo("The local web server could not be started, using the console instead.")
		} else {
			if browserCmd != nil {
				imageconvert.WriteInfo("Close the browser to shut down the process when you are finished.")
				err = browserCmd.Wait()
				<-server.Quit
			} else {
				imageconvert.WriteInfo("Open a browser manually and go the link specified. Press then Ctrl+C to shut down the process.")
				<-server.Quit
			}
			return
		}
	}

	s, err := GetSettings(srcfolder, collectionname)

	if err != nil {
		imageconvert.WriteInfo("Error while collecting the settings:", err)
		//return nil, os.NewError(fmt.Sprintf("The folder '%s' is not a valid directory.", srcfolder))
		os.Exit(1)
	}

	// convert the images and collect the results
	collPublishFolder := LaunchConversion(s)

	//ftpAddress := "mezzsplace.dyndns.org" //, _ := askParameter("The name of the ftp server:[mezzsplace.dyndns.org]", "mezzsplace.dyndns.org")
	//username := "enrico"                  //, _ := askParameter("The name of the user:[enrico]", "enrico")

	// ftpAddress := "mezzsplace.dyndns.org" //, _ := askParameter("The name of the ftp server:[mezzsplace.dyndns.org]", "mezzsplace.dyndns.org")
	// username := "enrico"                  //, _ := askParameter("The name of the user:[enrico]", "enrico")
	// password, _ := askParameter(fmt.Sprintf("The password for username %s:", username), "")

	if len(s.FtpSettings.Address) == 0 {
		imageconvert.WriteInfo("The ftp address was not specified and the upload will be skipped.")
		os.Exit(1)
	}

	ftpClient := ftp4go.NewFTP(0) // 1 for debugging
	ftpClient.SetPassive(true)

	/*
		var (
			resp *ftp4go.Response
		)
	*/

	imageconvert.WriteInfo("Connecting to host", s.FtpSettings.Address)
	// connect
	_, err = ftpClient.Connect(s.FtpSettings.Address, ftp4go.DefaultFtpPort)
	if err != nil {
		imageconvert.WriteInfo("The FTP connection could not be established, error: ", err.Error())
		os.Exit(1)
	}

	defer ftpClient.Quit()

	_, err = ftpClient.Login(s.FtpSettings.Username, s.FtpSettings.Password, "")
	if err != nil {
		imageconvert.WriteInfo("The FTP login was invalid, error: ", err.Error())
		os.Exit(1)
	}

	err = PublishCollToFtp(
		ftpClient,
		collPublishFolder,
		"./piwigo/galleries",
		EXCLUDED_DIRS)

	if err != nil {
		imageconvert.WriteInfo("Error upload to FTP, error: ", err)
		return
	} else {
		imageconvert.WriteInfo("Files successfully uploaded")
	}

}

var EXCLUDED_DIRS []string = []string{"pwg_high"}

func LaunchConversion(s *settings.Settings) (collPublishFolder string) {
	startNanosecs := time.Now()
	responseChannel, quitChannel, fileno, collPublishFolder, err := imageconvert.Convert(
		s.CollName,
		s.SourceDir,
		s.PublishDir,
		s.PiwigoGalleryHighDirName,
		s.ConversionSettings)

	if err != nil {
		panic(err)
	}

	// collect responses
	imageconvert.WriteInfo(fmt.Sprintf("Collecting results"))

	for i := 0; i < fileno; i++ {

		r := <-responseChannel
		fname := filepath.Base(r.ImgF.Path)
		if r.Error == nil {
			imageconvert.WriteInfof("Success, file %s resized and archived\n", fname)
		} else {
			imageconvert.WriteInfo(fmt.Sprintf("Error, file %s, the error was %s", fname, r.Error))
		}
	}

	quitChannel <- true // stopping the server
	imageconvert.WriteInfo(fmt.Sprintf("The conversion took %.3f seconds", float32(time.Now().Sub(startNanosecs))/1e9))
	imageconvert.WriteInfo("Images successfully resized")

	return collPublishFolder
}

func PublishCollToFtp(fc *ftp4go.FTP, localDir string, remoteRoorDir string, excludedDirs []string) (err error) {
	imageconvert.WriteInfo(fmt.Sprintf("Publishing to FTP root folder: %s, from local directory: %s.\nExcluded folders:%s", remoteRoorDir, localDir, excludedDirs))

	collName := filepath.Base(localDir)
	// careful!!!
	if len(collName) > 0 {
		/*
			remoteDir := filepath.Join(remoteRoorDir, collName)
			imageconvert.WriteInfo("Removing old ftp folder tree if present at:", remoteDir)
			err = fc.RemoveRemoteDirTree(remoteDir)
		*/
	} else {
		return errors.New("The collection name can not be empty")
	}

	imageconvert.WriteInfo("Uploading folder tree:", filepath.Base(localDir))
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
						imageconvert.WriteInfof("Successfully uploaded file: %s\n", st.Resourcename)
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
