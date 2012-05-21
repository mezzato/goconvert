package imageconvert

import (
	"fmt"
)

type LogLevel int

const (
	Info LogLevel = 1 << iota
	Verbose
	basePkg = "goconvert.googlecode.com/hg"
)

var (
	LogLevelForRun LogLevel = Info
)

func WriteLog(ll LogLevel, msgs ...interface{}) (n int, err error) {
	if ll <= LogLevelForRun {
		return fmt.Println(msgs...)
	}
	return
}

func WriteLogf(ll LogLevel, format string, msgs ...interface{}) (n int, err error) {
	if ll <= LogLevelForRun {
		return fmt.Printf(format, msgs...)
	}
	return
}

func WriteInfo(msgs ...interface{}) (n int, err error) {
	return WriteLog(Info, msgs...)
}

func WriteInfof(format string, msgs ...interface{}) (n int, err error) {
	return WriteLogf(Info, format, msgs...)
}

func WriteVerbose(msgs ...interface{}) (n int, err error) {
	return WriteLog(Verbose, msgs...)
}
