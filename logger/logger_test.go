package logger

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestSeverityLogging(t *testing.T) {
	var b *bytes.Buffer = new(bytes.Buffer)
	for _, ll := range LogLevels {

		cl := NewConsoleSemanticLogger("testlogger", b, ll)
		for _, llwrite := range LogLevels {
			b.Reset()
			m := fmt.Sprintf(" |test level: %s| ", llwrite)
			cl.logIfSevere(llwrite, m)
			if llwrite.LessSevereThan(ll) && b.Len() > 0 {
				t.Fatalf("message for log level %s has been logged even if the logger has a log threshold of %s", llwrite, ll)
			}
			if !llwrite.LessSevereThan(ll) {
				if strings.Index(b.String(), m) < 0 {
					t.Logf("Log to write less severe: %v, %d < %d? %s < %s", llwrite.LessSevereThan(ll), llwrite, ll, llwrite, ll)
					t.Fatalf("Logger with threshold %s:%d, message level %s:%d, wrong log message, should contain %s but was %s", cl.logLevel, cl.logLevel, llwrite, llwrite, m, b.String())
				}
			}
		}
	}

}
