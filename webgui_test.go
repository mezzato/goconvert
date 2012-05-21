package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"
)

func TestRunBrowser(t *testing.T) {
	c, e := runB(".", "http://www.google.de")
	if e != nil {
		t.Fatalf("error %q", e)
	}
	//c.Wait()
	t.Log(c.Args)
}

// run is a simple wrapper for exec.Run/Close
func runB(dir string, url string) (cmd *exec.Cmd, err error) {
	browsers := []string{"google-chrome", "firefox"}
	switch runtime.GOOS {
	case "windows":
		browsers = []string{
			os.ExpandEnv("${PROGRAMFILES}\\craphere"),
			os.ExpandEnv("${LOCALAPPDATA}\\Google\\Chrome\\Application\\chrome.exe"),
			os.ExpandEnv("${PROGRAMFILES}\\Mozilla Firefox\\firefox"),
			os.ExpandEnv("${PROGRAMFILES}\\Internet Explorer\\iexplore"),
		}
	default:
		//
	}

	for _, b := range browsers {
		fmt.Printf("Trying: %s\n", b)
		cmd = exec.Command(b, url)
		cmd.Dir = dir
		//cmd.Env = envv
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err == nil {
			return
		}
	}
	return nil, errors.New("No known browser could be started. Do it manually!")
}
