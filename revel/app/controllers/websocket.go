package controllers

import (
	"code.google.com/p/go.net/websocket"
	"code.google.com/p/goconvert/imageconvert"
	"encoding/json"
	"github.com/robfig/revel"
	"io"
	"log"
)

const msgLimit = 1000 // max number of messages to send per session

type WebSocket struct {
	*revel.Controller
}

func (ws WebSocket) ConvertSocket(c *websocket.Conn) revel.Result {
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
			log.Println("Feeding in message with id: " + m.Id)
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
				log.Println("try to kill process with id: " + m.Id)
				proc[m.Id].Kill()
				lOut := limiter(in, out)
				p, _, e := imageconvert.CreateAndStartProcess(m.Id, m.Body, lOut, m.Options)
				if e != nil {
					log.Println(e)
					break
				}
				log.Println("running process with id: " + m.Id)
				proc[m.Id] = p
			case "kill":
				proc[m.Id].Kill()
				log.Println("killed process with id: " + m.Id)
			}
		case err := <-errc:
			if err != io.EOF {
				// A encode or decode has failed; bail.
				log.Println(err)
			}
			log.Println("shutting down")
			// Shut down any running processes.
			for _, p := range proc {
				p.Kill()
			}
			return nil
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
