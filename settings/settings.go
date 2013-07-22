package settings

import (
	conf "code.google.com/p/goconf/conf"
	logger "code.google.com/p/goconvert/logger"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

const SETTINGS_FILE_NAME = "goconvert.conf"

const SECTION_DEPLOY = "deploy"
const SECTION_FTP = "ftp"
const SECTION_CONVERT = "convert"

const OPTION_DEPLOY_PUBLISHDIR = "publishdir"
const OPTION_DEPLOY_HOMEDIR = "homedir"
const OPTION_DEPLOY_PIWIGOGALLERYDIR = "piwigogallerydir"
const OPTION_DEPLOY_PIWIGOGALLERYHIGHDIRNAME = "piwigogalleryhighdirname"

const OPTION_CONVERT_WIDTH = "width"
const OPTION_CONVERT_HEIGHT = "height"
const OPTION_CONVERT_NOSIMULTANEOUSRESIZE = "nosimultaneousresize"
const OPTION_CONVERT_MOVEORIGINAL = "moveoriginal"

const OPTION_FTP_ADDRESS = "address"
const OPTION_FTP_USERNAME = "username"

var argv0 = os.Args[0]
var Debug = false

type missingSettingsFile string

func (s missingSettingsFile) String() string { return string(s) }

type ConversionSettings struct {
	Width                int  `json:"width"`
	Height               int  `json:"height"`
	MoveOriginal         bool `json:"moveOriginal"`
	NoSimultaneousResize int  `json:"noSimultaneousResize"`
}

type FtpSettings struct {
	Address  string `json:"address"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (sets *ConversionSettings) AreaInPixed() int {
	return sets.Height * sets.Width
}

// settings
type Settings struct {
	SaveConfig               bool                  `json:"saveConfig"`
	CollName                 string                `json:"collName"`
	SourceDir                string                `json:"sourceDir"`
	PublishDir               string                `json:"publishDir"`
	HomeDir                  string                `json:"homeDir"`
	PiwigoGalleryDir         string                `json:"piwigoGalleryDir"`
	PiwigoGalleryHighDirName string                `json:"piwigoGalleryHighDirName"`
	ConversionSettings       *ConversionSettings   `json:"conversionSettings"`
	FtpSettings              *FtpSettings          `json:"ftpSettings"`
	TimeoutMsec              int                   `json:"timeout_msec"`
	Logger                   logger.SemanticLogger `json:"-"`
}

func newSettings() *Settings {
	s := new(Settings)
	s.ConversionSettings = new(ConversionSettings)
	s.FtpSettings = new(FtpSettings)
	s.SourceDir = "."
	s.TimeoutMsec = 10000
	s.Logger = logger.NewConsoleSemanticLogger("goconvert", os.Stdout, logger.INFO)
	return s
}

func NewDefaultSettings(collectionName string, sourceDir string) *Settings {

	s := newSettings()

	// mandatory settings
	s.CollName = collectionName
	s.SourceDir = sourceDir

	homeDir := GetHomeDir()
	s.HomeDir = homeDir
	s.PublishDir = filepath.Join(homeDir, "Pictures")
	s.PiwigoGalleryDir = filepath.Join(homeDir, "piwigo", "galleries")
	s.PiwigoGalleryHighDirName = "pwg_high"
	s.ConversionSettings.Width = 1024
	s.ConversionSettings.Height = 768
	s.ConversionSettings.NoSimultaneousResize = 1
	s.ConversionSettings.MoveOriginal = false
	s.FtpSettings.Address = ""
	s.FtpSettings.Username = ""
	s.SaveConfig = true

	return s
}

type Param interface {
	Set(string) bool
	String() string
}

type boolParam bool

func (b *boolParam) Set(s string) bool {
	//v, err := strconv.Atob(s)
	var v bool
	if s == "y" {
		v = true
	}
	*b = boolParam(v)
	return true
}

func newBoolParam(val bool, b *bool) *boolParam {
	*b = val
	return (*boolParam)(b)
}

func (b *boolParam) String() string {
	if bool(*b) {
		return "y"
	}
	return "n"
}

type stringParam string

func (sp *stringParam) Set(s string) bool {
	*sp = stringParam(s)
	return true
}

func newStringParam(val string, p *string) *stringParam {
	*p = val
	return (*stringParam)(p)
}

func (sp *stringParam) String() string {
	return string(*sp)
}

type intParam int

func (i *intParam) Set(s string) bool {
	v, err := strconv.ParseInt(s, 0, 64)
	*i = intParam(v)
	return err == nil
}

func newIntParam(val int, i *int) *intParam {
	*i = val
	return (*intParam)(i)
}

func (i *intParam) String() string {
	return strconv.Itoa(int(*i))
}

type Question struct {
	title    string
	param    Param
	question string
}

func LoadSettingsFromFile(c *conf.ConfigFile) (s *Settings, err error) {
	s = newSettings()

	//s.CollName =
	s.PublishDir, _ = c.GetString(SECTION_DEPLOY, OPTION_DEPLOY_PUBLISHDIR)
	s.HomeDir, _ = c.GetString(SECTION_DEPLOY, OPTION_DEPLOY_HOMEDIR)
	s.PiwigoGalleryDir, _ = c.GetString(SECTION_DEPLOY, OPTION_DEPLOY_PIWIGOGALLERYDIR)
	s.PiwigoGalleryHighDirName, _ = c.GetString(SECTION_DEPLOY, OPTION_DEPLOY_PIWIGOGALLERYHIGHDIRNAME)

	s.ConversionSettings.Height, _ = c.GetInt(SECTION_CONVERT, OPTION_CONVERT_HEIGHT)
	s.ConversionSettings.Width, _ = c.GetInt(SECTION_CONVERT, OPTION_CONVERT_WIDTH)
	s.ConversionSettings.NoSimultaneousResize, _ = c.GetInt(SECTION_CONVERT, OPTION_CONVERT_NOSIMULTANEOUSRESIZE)
	s.ConversionSettings.MoveOriginal, _ = c.GetBool(SECTION_CONVERT, OPTION_CONVERT_MOVEORIGINAL)

	s.FtpSettings.Address, _ = c.GetString(SECTION_FTP, OPTION_FTP_ADDRESS)
	s.FtpSettings.Username, _ = c.GetString(SECTION_FTP, OPTION_FTP_USERNAME)

	return
}

func SaveSettingsToFile(s *Settings) (err error) {

	d, _ := filepath.Split(argv0)
	fn := filepath.Join(d, SETTINGS_FILE_NAME)

	c := conf.NewConfigFile()

	c.AddSection(SECTION_DEPLOY)
	c.AddOption(SECTION_DEPLOY, OPTION_DEPLOY_PUBLISHDIR, s.PublishDir)
	c.AddOption(SECTION_DEPLOY, OPTION_DEPLOY_HOMEDIR, s.HomeDir)
	c.AddOption(SECTION_DEPLOY, OPTION_DEPLOY_PIWIGOGALLERYDIR, s.PiwigoGalleryDir)
	c.AddOption(SECTION_DEPLOY, OPTION_DEPLOY_PIWIGOGALLERYHIGHDIRNAME, s.PiwigoGalleryHighDirName)

	c.AddSection(SECTION_CONVERT)
	c.AddOption(SECTION_CONVERT, OPTION_CONVERT_WIDTH, strconv.Itoa(s.ConversionSettings.Width))
	c.AddOption(SECTION_CONVERT, OPTION_CONVERT_HEIGHT, strconv.Itoa(s.ConversionSettings.Height))
	c.AddOption(SECTION_CONVERT, OPTION_CONVERT_NOSIMULTANEOUSRESIZE, strconv.Itoa(s.ConversionSettings.NoSimultaneousResize))
	c.AddOption(SECTION_CONVERT, OPTION_CONVERT_MOVEORIGINAL, strconv.FormatBool(s.ConversionSettings.MoveOriginal))

	c.AddSection(SECTION_FTP)
	c.AddOption(SECTION_FTP, OPTION_FTP_ADDRESS, s.FtpSettings.Address)
	c.AddOption(SECTION_FTP, OPTION_FTP_USERNAME, s.FtpSettings.Username)

	err = c.WriteConfigFile(fn, 0666, "goconvert configuration settings")
	return
}

var questionOrder = []string{
	"homedir",
	"publishdir",
	"piwigogallerydir",
	"piwigogalleryhighdirname",
	"width",
	"height",
	"nosimultaneousresize",
	"moveoriginal",
	"address",
	"username",
	"password",
	"saveconfig",
}

func (s *Settings) GetConfigQuestions(mandatoryOnly bool) (l map[string]*Question) {

	homeDir := GetHomeDir()

	var getOptionalQuesions = func() map[string]*Question {
		o := map[string]*Question{
			"homedir":                  &Question{"Home Directory", newStringParam(homeDir, &s.HomeDir), "The home folder for the current user"},
			"publishdir":               &Question{"Publish Directory", newStringParam(filepath.Join(homeDir, "Pictures"), &s.PublishDir), "The picture folder where to back up the images"},
			"piwigogallerydir":         &Question{"Piwigo Gallery Directory", newStringParam(filepath.Join(homeDir, "piwigo", "galleries"), &s.PiwigoGalleryDir), "The folder where the Piwigo galleries are stored"},
			"piwigogalleryhighdirname": &Question{"High resolution subfolder name", newStringParam("pwg_high", &s.PiwigoGalleryHighDirName), "The name of the subfolder where to archive the original high resolution images"},
			"width":                    &Question{"Resize: width", newIntParam(1024, &s.ConversionSettings.Width), "The width in pixel to convert an image to when resizing"},
			"height":                   &Question{"Resize: height", newIntParam(768, &s.ConversionSettings.Height), "The height in pixel to convert an image to when resizing"},
			"nosimultaneousresize":     &Question{"Resize: simulaneous processes", newIntParam(1, &s.ConversionSettings.NoSimultaneousResize), "Number of simultaneous resize processes, if you don't know what this means return"},
			"moveoriginal":             &Question{"Remove images after processing", newBoolParam(false, &s.ConversionSettings.MoveOriginal), "Whether to remove the images from the working folder after processing and archiving"},
			"address":                  &Question{"FTP address", newStringParam("", &s.FtpSettings.Address), "The address of the FTP server, leave blank to skip the upload"},
			"username":                 &Question{"FTP username", newStringParam("", &s.FtpSettings.Username), "The username to log onto the FTP server"},
			"saveconfig":               &Question{"Save the new settings to a file", newBoolParam(true, &s.SaveConfig), "Whether to save the settings for next time (passwords will not be saved!)"},
		}
		return o
	}

	mandQ := &Question{"FTP password", newStringParam("", &s.FtpSettings.Password), "The password to log onto the FTP server"}

	if mandatoryOnly {
		return map[string]*Question{"password": mandQ}
	}

	// add mandatory question to default ones
	l = getOptionalQuesions()
	l["password"] = mandQ

	return
}

func AskForSettings(collName string, srcfolder string) (s *Settings, err error) {

	var newSettingsFile bool

	var c *conf.ConfigFile
	d, _ := filepath.Split(argv0)
	fn := filepath.Join(d, SETTINGS_FILE_NAME)
	fmt.Printf("Config file path:%s\n", fn)

	if _, err = os.Stat(fn); err != nil {
		newSettingsFile = true
	} else if c, err = conf.ReadConfigFile(fn); err != nil {
		return nil, errors.New(fmt.Sprintf("Error reading the file %s", fn))
	}

	var useFile bool
	var askQuestion = func(q *Question) {
		ans, e := askParameter(fmt.Sprintf("%s? [%s]: ", q.question, q.param.String()))
		if e != nil {
			return
		}

		a := strings.TrimSpace(ans)
		if len(a) == 0 {
			a = q.param.String()
		}
		q.param.Set(a) // set value
	}

	if newSettingsFile {
		fmt.Println("No settings file have been found. Please answer the following questions to proceed.\nThe default value is in square brackets, just return to use it.")
	} else {
		q := &Question{"Whether to use the settings file.", newBoolParam(true, &useFile), "A settings file has been found. Would you like to use these settings, y = yes, n = no"}
		askQuestion(q)
	}

	if useFile {
		s, err = LoadSettingsFromFile(c)
		if err != nil {
			log.Fatalf("A fatal error has occurred: %s", err)
		}
	} else {
		s = newSettings()
	}

	fmt.Printf("Use file is:%s\n", useFile)
	qs := s.GetConfigQuestions(useFile)

	ftpkeys := []string{"address", "password", "username"} // sorted!
	sort.Strings(ftpkeys)

	var skipFtp = useFile && len(s.FtpSettings.Address) == 0

	for _, qkey := range questionOrder {

		idx := sort.SearchStrings(ftpkeys, qkey)

		if (idx < len(ftpkeys)) && ftpkeys[idx] == qkey && skipFtp {
			continue // do not ask about ftp settings
		}

		q, ok := qs[qkey]
		if ok {
			askQuestion(q)
			if qkey == "address" {
				skipFtp = len(q.param.String()) == 0
			}
		}

	}

	if s.SaveConfig {
		fmt.Printf("Saving configuration file\n")
		err = SaveSettingsToFile(s)
		if err != nil {
			log.Fatalf("A fatal error has occurred: %s", err)
		}
	}

	s.CollName = collName
	s.SourceDir = srcfolder

	if skipFtp {
		fmt.Println("\nTHE FTP UPLOAD WILL BE SKIPPED! If you want to use it restart the conversion without using any saved settings.\n")
	}

	return

}

func GetHomeDir() string {
	var homeDir string = os.ExpandEnv("$HOME")
	if len(homeDir) == 0 {
		switch runtime.GOOS {
		case "windows":
			homeDir = filepath.Join(os.ExpandEnv("$HOMEDRIVE"), os.ExpandEnv("$HOMEPATH"))
		default:
			homeDir = os.ExpandEnv("$HOME")
		}
	}
	return homeDir
}

func askParameter(question string) (inputValue string, err error) {
	fmt.Print(question)
	//originalStdout := os.Stdout
	//os.Stdout, _ = os.OpenFile(os.DevNull, os.O_RDONLY, 0)
	//defer func(){os.Stdout = originalStdout}()
	const NBUF = 512
	var buf [NBUF]byte
	switch nr, er := os.Stdin.Read(buf[:]); true {
	case nr < 0:
		fmt.Print(os.Stderr, "Error reading parameter. Error: ", er)
		os.Exit(1)
	case nr == 0: //EOF
		//os.NewError("Invalid parameter")
	case nr > 0:
		inputValue, err = string(buf[0:nr]), nil
	}
	//fmt.Println("The input value is:", inputValue, " with length: ", len(inputValue))
	return
}
