package main

import (
	"fmt"
	"os"
	"flag"
	"log"
	"strconv"
	"strings"
)

func StartConsolegui() (settings *Settings, err os.Error) {

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
		//writeInfo(fmt.Sprintf("The folder '%s' is not a valid directory.", srcfolder))
		return nil, os.NewError(fmt.Sprintf("The folder '%s' is not a valid directory.", srcfolder))
		//os.Exit(1)
	}

	collName := *collFlag

	if collName == "collectionnamewithoutspaces" {
		writeInfo("No collection name was specified, this is required to store your images\n")
		flag.Usage()
		return nil, os.NewError("No collection name was specified, this is required to store your images\n")
		//os.Exit(1)
	}

	settings, err = AskForSettings(collName, srcfolder)
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
	writeInfof(padS("Image folder"), settings.SourceDir)
	writeInfof(padS("Collection name"), settings.CollName)
	writeInfof(padS("Home folder"), settings.HomeDir)
	writeInfof(padS("Publish folder"), settings.PublishDir)
	writeInfof(padS("Piwigo gallery"), settings.PiwigoGalleryDir)
	writeInfof(padS("Number of resize processes"), strconv.Itoa(settings.ConversionSettings.NoSimultaneousResize))
	writeInfof(padS("ftp server"), settings.FtpSettings.Address)
	writeInfof(padS("ftp user"), settings.FtpSettings.Username)
	writeInfof(strings.Repeat("-", pad*2) + "\n")

	return settings, nil

}
