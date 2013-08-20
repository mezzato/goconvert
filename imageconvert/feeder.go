// Copyright 2012 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !appengine

// Package socket implements an WebSocket-based playground backend.
// Clients connect to a websocket handler and send run/kill commands, and
// the server sends the output and exit status of the running processes.
// Multiple clients running multiple processes may be served concurrently.
// The wire format is JSON and is described by the Message type.
//
// This will not run on App Engine as WebSockets are not supported there.
package imageconvert

import (
	"bytes"
	"code.google.com/p/goconvert/logger"
	"code.google.com/p/goconvert/settings"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"unicode/utf8"
)

// Environ provides an environment when a binary, such as the go tool, is
// invoked.
var Environ func() []string = os.Environ

// Message is the wire format for the websocket connection to the browser.
// It is used for both sending output messages and receiving commands, as
// distinguished by the Kind field.
type Message struct {
	Id      string // client-provided unique id for the process
	Kind    string // in: "run", "kill" out: "stdout", "stderr", "end"
	Body    string
	Options *Options `json:",omitempty"`
}

// Options specify additional message options.
type Options struct {
	Settings *settings.Settings `json:"settings"`
}

// process represents a running process.
type Process struct {
	id     string
	out    chan<- *Message
	done   chan struct{} // closed when wait completes
	run    *exec.Cmd
	killCh chan struct{}
	waitCh chan error
	Logger logger.SemanticLogger
}

// startProcess builds and runs the given program, sending its output
// and end event as Messages on the provided channel.
func newProcess(id string, out chan<- *Message, logLevel logger.LogLevel) *Process {
	p := &Process{
		id:     id,
		out:    out,
		done:   make(chan struct{}),
		killCh: make(chan struct{}),
		waitCh: make(chan error),
		Logger: logger.NewConsoleSemanticLogger("goconvert", os.Stdout, logLevel),
	}

	return p
}

func CreateAndStartProcess(id, body string, out chan<- *Message, opt *Options) (p *Process, cfs *ConversionFileSystem, err error) {
	p = newProcess(id, out, logger.ERROR)

	if cfs, err = p.tryStart(body, opt.Settings, p.createExecutors); err != nil {
		p.end(err)
		return
	}
	go p.Wait()
	return
}

// start builds and starts the given program, sending its output to p.out,
// and stores the running *exec.Cmd in the run field.
func (p *Process) tryStart(body string, settings *settings.Settings, executorCreator func(*ConversionFileSystem) []*Executor) (cfs *ConversionFileSystem, err error) {
	// We "go build" and then exec the binary so that the
	// resultant *exec.Cmd is a handle to the user's program
	// (rather than the go tool process).
	// This makes Kill work.

	cfs, err = extractConversionFileSystem(settings, p.Logger)
	if err != nil {
		return
	}

	executors := executorCreator(cfs)

	if len(cfs.collName) == 0 {
		err = errors.New("The collection name can not be empty.")
		return
	}

	// check imgmagick
	args := []string{"convert", "-version"}
	c := exec.Command(args[0], args[1:]...)
	p.Logger.Info("Testing ImageMagick installation")
	err = c.Run()
	if err != nil {
		err = fmt.Errorf("Error running ImageMagick, check that it is correctly installed. Error: %s", err.Error())
	}

	if err != nil {
		return
	}

	if len(cfs.imgFiles) == 0 {
		err = errors.New("No image files in folder: " + cfs.sourceDir)
		return
	}

	var out, inChan, outChan, in chan *imgFile

	// wrap the first input channel with the priority queue and expose it as an executor
	inChan = make(chan *imgFile)
	outChan = make(chan *imgFile)

	// executors in chain order, timewise first to last
	executorno := len(executors)

	workerNumber := cfs.conversionSettings.NoSimultaneousResize

	for j := 0; j < workerNumber; j++ {
		out = inChan
		for i := 0; i < executorno; i++ {
			in = out

			// the last executor fans in the requests into the same output channel
			if i == executorno-1 {
				out = outChan
			} else {
				out = make(chan *imgFile)
			}

			w := createWorker(cfs.timeoutMsec, executors[i], p.id, p.out, p.killCh)
			go w(out, in)
		}
	}

	// start feeding
	// start collecting in a go routine
	go func() {
		for _, f := range cfs.imgFiles {
			inChan <- f
		}
	}()

	// consume all images
	go func() {
		for j := 0; j < len(cfs.imgFiles); j++ {
			<-outChan
		}
		p.waitCh <- nil
	}()

	return cfs, nil
}

// wait waits for the running process to complete
// and sends its error state to the client.
func (p *Process) Wait() (err error) {
	err = <-p.waitCh // wait for signal by wait channel
	p.end(err)
	close(p.done) // unblock waiting Kill calls
	return err
}

// end sends an "end" message to the client, containing the process id and the
// given error value.
func (p *Process) end(err error) {
	m := &Message{Id: p.id, Kind: "end"}
	if err != nil {
		m.Body = err.Error()
	}
	p.out <- m
}

// Kill stops the process if it is running and waits for it to exit.
func (p *Process) Kill() {
	if p == nil || p.killCh == nil {
		return
	}
	//p.run.Process.Kill()
	//p.killCh <- struct{}{}
	// send a broadcast message
	close(p.killCh)
	p.killCh = nil
	<-p.done // block until process exits
}

// cmd builds an *exec.Cmd that writes its standard output and error to the
// process' output channel.
func (p *Process) cmd(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = Environ()
	cmd.Stdout = &messageWriter{p.id, "stdout", p.out}
	cmd.Stderr = &messageWriter{p.id, "stderr", p.out}
	return cmd
}

// messageWriter is an io.Writer that converts all writes to Message sends on
// the out channel with the specified id and kind.
type messageWriter struct {
	id, kind string
	out      chan<- *Message
}

func (w *messageWriter) Write(b []byte) (n int, err error) {
	w.out <- &Message{Id: w.id, Kind: w.kind, Body: safeString(b)}
	return len(b), nil
}

// safeString returns b as a valid UTF-8 string.
func safeString(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	var buf bytes.Buffer
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		b = b[size:]
		buf.WriteRune(r)
	}
	return buf.String()
}

var tmpdir string

func init() {
	// find real path to temporary directory
	var err error
	tmpdir, err = filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		log.Fatal(err)
	}
}

var uniq = make(chan int) // a source of numbers for naming temporary files

func init() {
	go func() {
		for i := 0; ; i++ {
			uniq <- i
		}
	}()
}
