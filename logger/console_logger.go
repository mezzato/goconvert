package logger

import (
	"fmt"
	"io"
	"log"
)

// ConsoleSemanticLogger logs to the console. True story.
type ConsoleSemanticLogger struct {
	log      *log.Logger
	logLevel LogLevel
}

// NewConsoleSemanticLogger returns a *ConsoleSemanticLogger with the
// given name that logs to the given io.Writer (usually os.Stdin or
// os.Stderr).
func NewConsoleSemanticLogger(name string, w io.Writer, logLevel LogLevel) *ConsoleSemanticLogger {
	cl := ConsoleSemanticLogger{
		// TODO: Set this format to match Clarity's Ruby SemanticLogger
		log:      log.New(w, fmt.Sprintf("%s: ", name), log.LstdFlags),
		logLevel: logLevel,
	}
	return &cl
}

func (cl *ConsoleSemanticLogger) LogLevel() LogLevel {
	return cl.logLevel
}

func (cl *ConsoleSemanticLogger) logIfSevere(level LogLevel, msg string) {
	if level.LessSevereThan(cl.logLevel) {
		return // skip
	}
	cl.log.Printf("%v: %s\n", level, msg)
}

// logIfSevere uses select parts of the given payload and logs it to the
// console.
func (cl *ConsoleSemanticLogger) Log(payload *LogPayload) {
	// TODO: Consider using more payload fields
	cl.logIfSevere(payload.Level, payload.Message)
}

func (cl *ConsoleSemanticLogger) Trace(msg string) {
	// TODO: Consider using more payload fields
	cl.logIfSevere(TRACE, msg)
}

func (cl *ConsoleSemanticLogger) Debug(msg string) {
	// TODO: Consider using more payload fields
	cl.logIfSevere(DEBUG, msg)
}

func (cl *ConsoleSemanticLogger) Info(msg string) {
	// TODO: Consider using more payload fields
	cl.logIfSevere(INFO, msg)
}

func (cl *ConsoleSemanticLogger) Warn(msg string) {
	// TODO: Consider using more payload fields
	cl.logIfSevere(WARN, msg)
}

func (cl *ConsoleSemanticLogger) Error(msg string) {
	// TODO: Consider using more payload fields
	cl.logIfSevere(ERROR, msg)
}

// Fatal logs the given payload to the console, then panics.
func (cl *ConsoleSemanticLogger) Fatal(msg string) {
	payload := NewLogPayload(FATAL, msg)
	payload.SetException()
	cl.Log(payload)
	panic(payload)
}

// BenchmarkInfo currently does nothing but should measure the time
// it takes to execute `f` based on the log level.
func (cl *ConsoleSemanticLogger) BenchmarkInfo(level LogLevel, msg string,
	f func(logger SemanticLogger)) {
	// TODO: Implement
}
