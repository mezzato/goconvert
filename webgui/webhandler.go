package webgui

import (
	settings "code.google.com/p/goconvert/settings"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

func wrapHandler(processor requestProcessor) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := new(Response)
		out, err, eof := processor(r)
		resp.Eof = eof
		if err != nil {
			resp.Errors = []string{err.Error()}
		} else {
			resp.Messages = out
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Println(err)
		}

	}
}

func compress(r *http.Request) (msg []string, err error, eof bool) {
	var reader io.Reader = r.Body
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		eof = true
		return
	}
	var jsonSettings *settings.Settings
	fmt.Println("request body: " + string(b))
	err = json.Unmarshal(b, &jsonSettings)
	if err != nil {
		fmt.Println(err)
		eof = true
		return
	}
	if compressing {
		return nil, errors.New("Still Compressing\n another folder.\nPlese wait until it has finished."), !compressing
	}

	compressing = true
	// launch asynchronously to avoid delays
	go func() {
		_, quitChannel, err = launchConversionFromWeb(jsonSettings, logger, &compressing)
	}()

	return []string{fmt.Sprintf("Compressing\nfolder: %s\nCollection name: %s", jsonSettings.SourceDir, jsonSettings.CollName)}, err, !compressing
}

func compressStatus(r *http.Request) (msgs []string, err error, eof bool) {
	newLines := logger.ReadAll()
	if len(newLines) > 0 {
		msgs = newLines
	}
	return msgs, nil, !compressing
}

func stopCompressing(r *http.Request) (msgs []string, err error, eof bool) {
	if !compressing || quitChannel == nil {
		return nil, errors.New("No compression has been launched yet."), false
	}

	quitChannel <- true
	return []string{"Stopping the compression"}, nil, !compressing
}
