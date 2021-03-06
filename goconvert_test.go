// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

/*
import (
	//"bufio"
	"bytes"
	//"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	//"net"
	"sync"
	"testing"
	//"url"
	"strings"
	"websocket"
)

var serverAddr string
var once sync.Once

func echoServerTest(ws *websocket.Conn) { io.Copy(ws, ws) }

func startServer() {
	http.Handle("/echo", websocket.Handler(echoServerTest))
	//http.Handle("/echoDrat75", Draft75Handler(echoServer))
	server := httptest.NewServer(nil)
	serverAddr = server.Listener.Addr().String()
	log.Print("Test WebSocket server listening on ", serverAddr)
}

func TestEcho(t *testing.T) {
	once.Do(startServer)

	// websocket.Dial()
	//ws, err := websocket.Dial("ws://localhost/ws", "", "http://localhost/");
	for {
		client, err := websocket.Dial("ws://"+serverAddr+"/echo", "tcp", "http://localhost/")
		if err != nil {
			t.Fatal("dialing", err)
		}

		msg := []byte("hello, world\n")
		if _, err := client.Write(msg); err != nil {
			t.Errorf("Write: %v", err)
		}
		var actual_msg = make([]byte, 512)
		n, err := client.Read(actual_msg)
		if err != nil {
			t.Errorf("Read: %v", err)
		}
		actual_msg = actual_msg[0:n]
		if !bytes.Equal(msg, actual_msg) {
			t.Errorf("Echo: expected %q got %q", msg, actual_msg)
		}
		client.Close()
		r, e := askParameter("Press return to stop the server, type \"y\" to rerun the test: ")
		r = strings.TrimSpace(r)

		if e != nil || r != "y" {
			break
		}
	}

}
*/
