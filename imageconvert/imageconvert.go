package imageconvert

import (
	exif4go "code.google.com/p/exif4go"
	"code.google.com/p/goconvert/settings"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

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
	ts, sk, err1 := getFileExifInfo(fpath)

	if err1 != nil {
		//return
		WriteInfof("Error getting image EXIF info: %s\n", err1)
	}

	err = nil
	return &imgFile{ts, sk, fpath}, err
}

type ConversionFileSystem struct {
	collName                string
	sourceDir               string
	extensions              []string
	imgFiles                []*imgFile
	CollectionPublishFolder string
	CollectionArchiveFolder string
	timeoutMsec             int
	conversionSettings      *settings.ConversionSettings
}

func extractConversionFileSystem(sets *settings.Settings) (f *ConversionFileSystem, err error) {
	f = new(ConversionFileSystem)

	f.conversionSettings = sets.ConversionSettings

	f.timeoutMsec = sets.TimeoutMsec
	f.collName = sets.CollName
	f.sourceDir = sets.SourceDir
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
		decoratedCollectionName := strings.Join([]string{ff.sortkey, lf.sortkey, sets.CollName}, "_")
		f.CollectionPublishFolder = filepath.Join(sets.PublishDir, decoratedCollectionName)
		f.CollectionArchiveFolder = filepath.Join(f.CollectionPublishFolder, sets.PiwigoGalleryHighDirName)
	}

	return
}

func (f *ConversionFileSystem) getImgFiles() (imgFiles []*imgFile, err error) {

	var fi os.FileInfo
	fi, err = os.Stat(f.sourceDir)
	if err != nil {
		return
	}

	imgFiles = make([]*imgFile, 0, 50)
	var files []string
	if fi.IsDir() {
		WriteInfo("Gettings image files in folder", f.sourceDir)
		gs := filepath.Join(f.sourceDir, "*")
		files, _ = filepath.Glob(gs) // find all files in folder
		//WriteVerbose(fmt.Sprintf("Number of files via glob search %s found in folder: %d", gs,len(files)))
		sort.Strings(files)

	} else {
		files = []string{f.sourceDir} // it is a file not a folder
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

// TODO: remove below

type ConvertRequest struct {
	imgF     *imgFile
	response chan *ConvertResponse
}

type ConvertResponse struct {
	ImgF  *imgFile
	Error error
}

func Convert(collname string,
	srcfolder string,
	publishfolder string,
	archivesubfoldername string,
	convSettings *settings.ConversionSettings) (responseChannel chan *ConvertResponse, quitChannel chan bool, fileno int, collPublishFolder string, err error) {
	WriteInfo("Starting the image conversion")

	//var imgFolder *imgFolder

	if len(collname) == 0 {
		err = errors.New("The collection name can not be empty.")
		return
	}

	// imgFolder, err = newImgFolder(collname, srcfolder, publishfolder, archivesubfoldername)

	if err != nil {
		return
	}

	// REMOVE THIS METHOD
	return

}
