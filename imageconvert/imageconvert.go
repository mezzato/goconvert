package imageconvert

import (
	"bytes"
	exif4go "code.google.com/p/exif4go"
	"code.google.com/p/goconvert/settings"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type request struct {
	imgF     *imgFile
	response chan *Response
}

type Response struct {
	ImgF  *imgFile
	Error error
}

type imgParams struct {
	cmdArgs          []string
	subFolderRelPath string
	prefix           string
}

type imgFile struct {
	timestamp string
	sortkey   string
	Path      string
}

func getFileExifInfo(fp string) (timestamp string, sortkey string, err error) {
	f, err := os.Open(fp)
	if err != nil {
		return
	}
	defer f.Close()

	//fmt.Println("Exif for file:", fp)
	tags, err := exif4go.Process(f, false)
	if err != nil {
		return
	}
	if tags == nil {
		err = errors.New("No EXIF information for file: " + fp)
		return
	}

	var t time.Time
	if tag, ok := tags["Image DateTime"]; ok {
		// t, err = time.Parse("%Y:%m:%d %H:%M:%S", tag.Values[0]) //[0:6]
		t, err = time.Parse("2006:01:02 15:04:05", tag.Values[0])
		if err != nil {
			return
		}
	} else {
		// Ensure date is present
		t = time.Now()
	}

	timestamp = strconv.FormatInt(t.Unix(), 10)
	sortkey = t.Format("20060102")

	return
}

func newImgFile(fpath string) (i *imgFile, err error) {
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

func newImgFolder(collName string, iFolder string, publishFolder string, archiveSubfolderName string) (f *imgFolder, err error) {
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

func (f *imgFolder) getImgFiles() (imgFiles []*imgFile, err error) {

	var fi os.FileInfo
	fi, err = os.Stat(f.ImgFolder)
	if err != nil {
		return
	}

	imgFiles = make([]*imgFile, 0, 50)
	var files []string
	if fi.IsDir() {
		WriteInfo("Gettings image files in folder", f.ImgFolder)
		gs := filepath.Join(f.ImgFolder, "*")
		files, _ = filepath.Glob(gs) // find all files in folder
		//WriteVerbose(fmt.Sprintf("Number of files via glob search %s found in folder: %d", gs,len(files)))
		sort.Strings(files)

	} else {
		files = []string{f.ImgFolder} // it is a file not a folder
	}

	//fmt.Println("The sorted extensions are:", f.extensions)

	for _, fp := range files {
		ext := strings.ToLower(filepath.Ext(fp))
		//WriteVerbose(fmt.Sprintf("Number of extensions = %d",len(f.extensions)))
		idx := sort.SearchStrings(f.extensions, ext)
		WriteVerbose("The value of idx in the extension slice is:", idx)
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
	convSettings *settings.ConversionSettings) (responseChannel chan *Response, quitChannel chan bool, fileno int, collPublishFolder string, err error) {
	WriteInfo("Starting the image conversion")

	var imgFolder *imgFolder

	if len(collname) == 0 {
		err = errors.New("The collection name can not be empty.")
		return
	}

	imgFolder, err = newImgFolder(collname, srcfolder, publishfolder, archivesubfoldername)

	if err != nil {
		return
	}

	if len(imgFolder.imgFiles) == 0 {
		err = errors.New("No image files in folder: " + srcfolder)
		return
	}

	// check imgmagick
	cmd := []string{"convert", "-version"}
	c := &Cmd{Args: cmd}
	WriteVerbose("Testing ImageMagick installation")
	err = c.Run()
	if err != nil {
		err = fmt.Errorf("Error running ImageMagick, check that it is correctly installed. Error: %s", err.Error())
	}

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
	WriteInfo("Processing images and saving to folder:", publishfolder)
	WriteVerbose("Number of conversion sets:", len(convSets))

	res := createResizeHandler(imgFolder.collectionPublishFolder, convSets)
	arch := createArchiveHandler(imgFolder.collectionArchiveFolder, convSettings.MoveOriginal)
	comp := createCompositeHandler([]filehandler{res, arch})
	reqs, quit := startWorkers(comp, noOfWorkers)

	//resps := make(map[*imgFile]*request)

	respChannel := make(chan *Response, len(imgFolder.imgFiles)) // put a buffer not to lock the feedback calls

	WriteVerbose(fmt.Sprintf("Number of images in folder: %d", len(imgFolder.imgFiles)))

	// start feeding asynchronously
	go func() {
		for _, imgf := range imgFolder.imgFiles {
			WriteVerbose(fmt.Sprintf("Processing image file at: %s, date timestamp %s, sortkey %s", imgf.Path, imgf.timestamp, imgf.sortkey))

			req := &request{imgf, respChannel}
			reqs <- req
		}
	}()

	return respChannel, quit, len(imgFolder.imgFiles), imgFolder.collectionPublishFolder, nil

}

func startWorkers(h filehandler, noOfWorkers int) (reqs chan *request, quit chan bool) {
	reqs = make(chan *request)
	quit = make(chan bool)
	sem := make(chan int, noOfWorkers)

	WriteVerbose(fmt.Sprintf("Starting %d workers and waiting for requests", noOfWorkers))
	go func() {
		for {
			select {

			case <-quit:
				WriteInfo("Stopping workers")
				return // get out

			case req := <-reqs:
				// do not wait here, the buffered request channel is the barrier
				//go func() {
				sem <- 1 // Wait for active queue to drain.
				// feed the response channel

				// WriteInfof("Processing: %s\n", req.imgF.Path)
				e := h(req.imgF)
				req.response <- &Response{req.imgF, e}
				<-sem // Done; enable next request to run.
				//}()

			}
		}
	}()

	return
}

type filehandler func(f *imgFile) (err error)

func createCompositeHandler(handlerChain []filehandler) (handler filehandler) {
	var chainHandler = func(img *imgFile) (err error) {
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
	var resizeHandler = func(img *imgFile) (err error) {

		WriteVerbose(fmt.Sprintf("Resizing img: %s.", filepath.Base(img.Path)))
		var c *Cmd

		for _, set := range convSets {
			if len(set.cmdArgs) == 0 {
				return errors.New("Command arguments must be specified")
			}

			newImgName := set.prefix + filepath.Base(img.Path)
			subFolderPath := filepath.Join(collPublishFolder, set.subFolderRelPath)
			var fi os.FileInfo
			if fi, err = os.Stat(subFolderPath); err != nil || !fi.IsDir() {
				// create dirs
				WriteVerbose("Creating folder:", subFolderPath)
				if err = os.MkdirAll(subFolderPath, 0777); err != nil {
					return err
				}
			}

			newImgPath := filepath.Join(subFolderPath, newImgName)

			fullCmd := []string{"convert", img.Path}
			fullCmd = append(fullCmd, set.cmdArgs...)
			fullCmd = append(fullCmd, newImgPath)

			c = &Cmd{Args: fullCmd}
			WriteVerbose("Running cmd:", c)
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
	var archiveHandler = func(img *imgFile) (err error) {

		var fi os.FileInfo
		if fi, err = os.Stat(collArchiveFolder); err != nil || !fi.IsDir() {
			// create dirs
			WriteVerbose("Creating folder:", collArchiveFolder)
			if err = os.MkdirAll(collArchiveFolder, 0777); err != nil {
				return err
			}
		}
		movePath := filepath.Join(collArchiveFolder, filepath.Base(img.Path))

		if moveOriginal {
			WriteVerbose("Archiving original file to:", movePath)
			os.Rename(img.Path, movePath)
		} else {
			WriteVerbose("Copying original file to:", movePath)
			// take a copy			
			f, _ := os.Open(img.Path)
			defer f.Close()
			f1, _ := os.Create(movePath)
			defer f1.Close()
			if _, err = io.Copy(f1, f); err != nil {
				return
			}
			fi, _ = f.Stat()
			// copy stats			
			if err = os.Chtimes(movePath, fi.ModTime(), fi.ModTime()); err != nil {
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
func (c *Cmd) Run() error {
	out := new(bytes.Buffer)
	cmd := exec.Command(c.Args[0], c.Args[1:]...)
	cmd.Dir = c.Dir
	cmd.Env = c.Env
	cmd.Stdout = out
	cmd.Stderr = out

	// remember to release the process
	defer func() {
		if cmd != nil {
			if err := cmd.Process.Release(); err != nil {
				//panic(err)
				//WriteInfof("Error while waiting releasing the process %s to finish\n",cmd.Args[0])
			}
		}
	}()

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
