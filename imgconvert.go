package main

import (
	"fmt"
	"os"
	"time"
	"strconv"
	"exif"
	"path/filepath"
	"sort"
	"strings"
	"exec"
	"bytes"
	"io"
)

func getFileExifInfo(fp string) (timestamp string, sortkey string, err os.Error) {
	f, err := os.Open(fp)
	if err != nil {
		return
	}
	defer f.Close()

	//fmt.Println("Exif for file:", fp)
	tags, err := exif.Process(f, false)
	if err != nil {
		return
	}
	if tags == nil {
		err = os.NewError("No EXIF information for file: " + fp)
		return
	}

	var t *time.Time
	if tag, ok := tags["Image DateTime"]; ok {
		// t, err = time.Parse("%Y:%m:%d %H:%M:%S", tag.Values[0]) //[0:6]
		t, err = time.Parse("2006:01:02 15:04:05", tag.Values[0])
		if err != nil {
			return
		}
	} else {
		// Ensure date is present
		t = time.LocalTime()
	}

	timestamp = strconv.Itoa64(t.Seconds())
	sortkey = t.Format("20060102")

	return
}

type imgParams struct {
	cmdArgs          []string
	subFolderRelPath string
	prefix           string
}

type imgFile struct {
	timestamp string
	sortkey   string
	path      string
}

func newImgFile(fpath string) (i *imgFile, err os.Error) {
	var ts, sk string
	ts, sk, err = getFileExifInfo(fpath)
	if err != nil {
		return
	}
	return &imgFile{ts, sk, fpath}, err
}

type imgFolder struct {
	CollName                string
	ImgFolder               string
	PublishFolder           string
	ArchiveSubfolderName    string
	extensions              []string
	imgFiles                []*imgFile
	collectionPublishFolder string
	collectionArchiveFolder string
}

func newImgFolder(collName string, iFolder string, publishFolder string, archiveSubfolderName string) (f *imgFolder, err os.Error) {
	f = new(imgFolder)

	f.CollName = collName
	f.ImgFolder = iFolder
	f.PublishFolder = publishFolder
	f.ArchiveSubfolderName = archiveSubfolderName
	f.extensions = []string{".bmp", ".jpeg", ".jpg", ".gif", ".png"}
	// NOTE: sort extensions to look through them
	sort.Strings(f.extensions)

	// find files
	f.imgFiles, err = f.getImgFiles()
	if err != nil {
		return
	}

	// resolve collection names 
	if len(f.imgFiles) > 0 {
		ff := f.imgFiles[0]
		lf := f.imgFiles[len(f.imgFiles)-1]
		decoratedCollectionName := strings.Join([]string{ff.sortkey, lf.sortkey, collName}, "_")
		f.collectionPublishFolder = filepath.Join(publishFolder, decoratedCollectionName)
		f.collectionArchiveFolder = filepath.Join(f.collectionPublishFolder, archiveSubfolderName)
	}

	return
}

func (f *imgFolder) getImgFiles() (imgFiles []*imgFile, err os.Error) {

	var fi *os.FileInfo
	fi, err = os.Stat(f.ImgFolder)
	if err != nil {
		return
	}

	imgFiles = make([]*imgFile, 0, 50)
	var files []string
	if fi.IsDirectory() {
		writeInfo("Gettings image files in folder", f.ImgFolder)
		gs := filepath.Join(f.ImgFolder, "*")
		files, _ = filepath.Glob(gs) // find all files in folder
		//writeVerbose(fmt.Sprintf("Number of files via glob search %s found in folder: %d", gs,len(files)))
		sort.Strings(files)

	} else {
		files = []string{f.ImgFolder} // it is a file not a folder
	}

	//fmt.Println("The sorted extensions are:", f.extensions)

	for _, fp := range files {
		ext := strings.ToLower(filepath.Ext(fp))
		//writeVerbose(fmt.Sprintf("Number of extensions = %d",len(f.extensions)))
		idx := sort.SearchStrings(f.extensions, ext)
		writeVerbose("The value of idx in the extension slice is:", idx)
		if idx < len(f.extensions) && f.extensions[idx] == ext {
			var ifile *imgFile
			ifile, err = newImgFile(fp)
			if err != nil {
				return
			}
			imgFiles = append(imgFiles, ifile)
		}
	}

	return
}

func Convert(collname string,
srcfolder string,
publishfolder string,
archivesubfoldername string,
convSettings *ConversionSettings) (collPublishFolder string, err os.Error) {
	writeInfo("Starting the image conversion")
	startNanosecs := time.Nanoseconds()

	noOfWorkers := convSettings.NoSimultaneousResize

	smallPars := &imgParams{
		[]string{"-resize", strconv.Itoa(convSettings.AreaInPixed()) + "@", "-mattecolor", "gray4", "-frame", "4x4+2+2", "-font", "helvetica", "-fill", "black"},
		"",
		"",
	}

	thumbnailPars := &imgParams{
		[]string{"-resize", "128x128", "-mattecolor", "gray4", "-font", "helvetica"},
		"thumbnail",
		"TN-",
	}

	convSets := []*imgParams{smallPars, thumbnailPars}
	writeInfo("Processing images and saving to folder:", publishfolder)
	writeVerbose("Number of conversion sets:", len(convSets))

	var imgFolder *imgFolder
	imgFolder, err = newImgFolder(collname, srcfolder, publishfolder, archivesubfoldername)

	if err != nil {
		return
	}

	res := createResizeHandler(imgFolder.collectionPublishFolder, convSets)
	arch := createArchiveHandler(imgFolder.collectionArchiveFolder, convSettings.MoveOriginal)
	comp := createCompositeHandler([]filehandler{res, arch})
	reqs, quit := startWorkers(comp, noOfWorkers)

	//resps := make(map[*imgFile]*request)

	respChannel := make(chan *response, len(imgFolder.imgFiles)) // put a buffer not to lock the feedback calls

	writeVerbose(fmt.Sprintf("Number of images in folder: %d", len(imgFolder.imgFiles)))
	for _, imgf := range imgFolder.imgFiles {
		writeVerbose(fmt.Sprintf("Processing image file at: %s, date timestamp %s, sortkey %s", imgf.path, imgf.timestamp, imgf.sortkey))

		req := &request{imgf, respChannel}
		reqs <- req
	}

	// collect responses
	writeInfo(fmt.Sprintf("Collecting results"))
	//for e:= range respChannel{
	for i := 0; i < len(imgFolder.imgFiles); i++ {

		r := <-respChannel
		fname := filepath.Base(r.imgF.path)
		if r.error == nil {
			writeInfof("Success, file %s resized and archived\n", fname)
		} else {
			writeInfo(fmt.Sprintf("Error, file %s, the error was %s", fname, r.error))
		}
	}

	quit <- true // stopping the server

	writeInfo(fmt.Sprintf("The conversion took %.3f seconds", float32(time.Nanoseconds()-startNanosecs)/1e9))

	return imgFolder.collectionPublishFolder, nil
}

func startWorkers(h filehandler, noOfWorkers int) (reqs chan *request, quit chan bool) {
	reqs = make(chan *request)
	quit = make(chan bool)
	sem := make(chan int, noOfWorkers)

	writeVerbose(fmt.Sprintf("Starting %d workers and waiting for requests", noOfWorkers))
	go func() {
		for {
			select {
			case req := <-reqs:
				// do not wait here, the buffered request channel is the barrier
				go func() {
					sem <- 1 // Wait for active queue to drain.
					// feed the response channel
					e := h(req.imgF)
					req.response <- &response{req.imgF, e}
					<-sem // Done; enable next request to run.
				}()
			case <-quit:
				writeVerbose("Stopping workers")
				return // get out
			}
		}
	}()

	return
}

type request struct {
	imgF     *imgFile
	response chan *response
}

type response struct {
	imgF  *imgFile
	error os.Error
}

type filehandler func(f *imgFile) (err os.Error)

func createCompositeHandler(handlerChain []filehandler) (handler filehandler) {
	var chainHandler = func(img *imgFile) (err os.Error) {
		for _, h := range handlerChain {
			if err = h(img); err != nil {
				return err
			}
		}
		return

	}
	return chainHandler
}

func createResizeHandler(collPublishFolder string, convSets []*imgParams) (handler filehandler) {
	var resizeHandler = func(img *imgFile) (err os.Error) {

		writeVerbose(fmt.Sprintf("Resizing img: %s.", filepath.Base(img.path)))

		for _, set := range convSets {
			if len(set.cmdArgs) == 0 {
				return os.NewError("Command arguments must be specified")
			}

			newImgName := set.prefix + filepath.Base(img.path)
			subFolderPath := filepath.Join(collPublishFolder, set.subFolderRelPath)
			var fi *os.FileInfo
			if fi, err = os.Stat(subFolderPath); err != nil || !fi.IsDirectory() {
				// create dirs
				writeVerbose("Creating folder:", subFolderPath)
				if err = os.MkdirAll(subFolderPath, 0777); err != nil {
					return err
				}
			}

			newImgPath := filepath.Join(subFolderPath, newImgName)

			fullCmd := []string{"convert", img.path}
			fullCmd = append(fullCmd, set.cmdArgs...)
			fullCmd = append(fullCmd, newImgPath)

			c := &Cmd{Args: fullCmd}
			writeVerbose("Running cmd:", c)
			err = c.Run()
			if err != nil {
				return
			}
		}

		return
	}
	return resizeHandler
}

func createArchiveHandler(collArchiveFolder string, moveOriginal bool) (handler filehandler) {
	var archiveHandler = func(img *imgFile) (err os.Error) {

		var fi *os.FileInfo
		if fi, err = os.Stat(collArchiveFolder); err != nil || !fi.IsDirectory() {
			// create dirs
			writeVerbose("Creating folder:", collArchiveFolder)
			if err = os.MkdirAll(collArchiveFolder, 0777); err != nil {
				return err
			}
		}
		movePath := filepath.Join(collArchiveFolder, filepath.Base(img.path))

		if moveOriginal {
			writeVerbose("Archiving original file to:", movePath)
			os.Rename(img.path, movePath)
		} else {
			writeVerbose("Copying original file to:", movePath)
			// take a copy			
			f, _ := os.Open(img.path)
			defer f.Close()
			f1, _ := os.Create(movePath)
			defer f1.Close()
			if _, err = io.Copy(f1, f); err != nil {
				return
			}
			fi, _ = f.Stat()
			// copy stats			
			if err = os.Chtimes(movePath, fi.Atime_ns, fi.Mtime_ns); err != nil {
				return
			}
		}
		return
	}
	return archiveHandler
}

// A Cmd describes an individual command.
type Cmd struct {
	Args   []string // command-line
	Stdout string   // write standard output to this file, "" is passthrough
	Dir    string   // working directory
	Env    []string // environment
}

func (c *Cmd) String() string {
	return strings.Join(c.Args, " ")
}

// Run executes the Cmd.
func (c *Cmd) Run() os.Error {
	out := new(bytes.Buffer)
	cmd := exec.Command(c.Args[0], c.Args[1:]...)
	cmd.Dir = c.Dir
	cmd.Env = c.Env
	cmd.Stdout = out
	cmd.Stderr = out
	if c.Stdout != "" {
		f, err := os.Create(c.Stdout)
		if err != nil {
			return err
		}
		defer f.Close()
		cmd.Stdout = f
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command %q: %v\n%v", c, err, out)
	}
	return nil
}
