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
package webgui

import (
	"code.google.com/p/go.net/websocket"
	"code.google.com/p/goconvert/imageconvert"
	"encoding/json"
	"io"
	"log"
	"os"
)

const msgLimit = 1000 // max number of messages to send per session

// Handler implements a WebSocket handler for a client connection.
var websocketHandler = websocket.Handler(socketHandler)

// Environ provides an environment when a binary, such as the go tool, is
// invoked.
var Environ func() []string = os.Environ

// socketHandler handles the websocket connection for a given present session.
// It handles transcoding Messages to and from JSON format, and starting
// and killing processes.
func socketHandler(c *websocket.Conn) {
	in, out := make(chan *imageconvert.Message), make(chan *imageconvert.Message)
	errc := make(chan error, 1)

	// Decode messages from client and send to the in channel.
	go func() {
		dec := json.NewDecoder(c)
		for {
			var m imageconvert.Message
			if err := dec.Decode(&m); err != nil {
				errc <- err
				return
			}
			in <- &m
		}
	}()

	// Receive messages from the out channel and encode to the client.
	go func() {
		enc := json.NewEncoder(c)
		for m := range out {
			if err := enc.Encode(m); err != nil {
				errc <- err
				return
			}
		}
	}()

	in, out = make(chan *imageconvert.Message), make(chan *imageconvert.Message)
	errc = make(chan error, 1)

	// Start and kill processes and handle errors.
	proc := make(map[string]*imageconvert.Process)
	for {
		select {
		case m := <-in:
			switch m.Kind {
			case "run":
				proc[m.Id].Kill()
				lOut := limiter(in, out)
				p, _, e := imageconvert.CreateAndStartProcess(m.Id, m.Body, lOut, m.Options)
				if e != nil {
					break
				}
				proc[m.Id] = p

				go p.Wait()

			case "kill":
				proc[m.Id].Kill()
			}
		case err := <-errc:
			if err != io.EOF {
				// A encode or decode has failed; bail.
				log.Println(err)
			}
			// Shut down any running processes.
			for _, p := range proc {
				p.Kill()
			}
			return
		}
	}
}

// limiter returns a channel that wraps dest. Messages sent to the channel are
// sent to dest. After msgLimit Messages have been passed on, a "kill" imageconvert.Message
// is sent to the kill channel, and only "end" messages are passed.
func limiter(kill chan<- *imageconvert.Message, dest chan<- *imageconvert.Message) chan<- *imageconvert.Message {
	ch := make(chan *imageconvert.Message)
	go func() {
		n := 0
		for m := range ch {
			switch {
			case n < msgLimit || m.Kind == "end":
				dest <- m
				if m.Kind == "end" {
					return
				}
			case n == msgLimit:
				// process produced too much output. Kill it.
				kill <- &imageconvert.Message{Id: m.Id, Kind: "kill"}
			}
			n++
		}
	}()
	return ch
}
