package logger

import (
	"fmt"
	"os"
)

// FileSemanticLogger logs logging data to files... semantically!
type FileSemanticLogger struct {
	*ConsoleSemanticLogger
}

// NewFileSemanticLogger creates a new logger with the given name that
// logs to the given filename
func NewFileSemanticLogger(name, filename string, logLevel LogLevel) (*FileSemanticLogger, error) {
	// Open file with append permissions
	flags := os.O_APPEND | os.O_WRONLY | os.O_CREATE
	file, err := os.OpenFile(filename, flags, 0666)
	if err != nil {
		return nil, fmt.Errorf("Error opening '%v': %v", filename, err)
	}
	// Oddity: `file.Close()` never gets called. Everything seems to work.
	fl := FileSemanticLogger{
		NewConsoleSemanticLogger(name, file, logLevel),
	}
	return &fl, nil
}
