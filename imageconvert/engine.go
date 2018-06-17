package imageconvert

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"
)

var ngoroutine = 4 * runtime.GOMAXPROCS(-1)

type Executor struct {
	StepName string
	Do       func(*imgFile) error
}

func (p *Process) createExecutors(c *ConversionFileSystem) (pipe []*Executor) {

	convSettings := c.conversionSettings

	smallPars := &imgParams{
		[]string{"-resize", strconv.Itoa(convSettings.AreaInPixed()) + "@", "-frame", "4x4+2+2", "-font", "helvetica", "-fill", "black"},
		"",
		"",
	}

	thumbnailPars := &imgParams{
		[]string{"-resize", "128x128", "-font", "helvetica"},
		"thumbnail",
		"TN-",
	}

	convSets := []*imgParams{smallPars, thumbnailPars}

	ntasks := 2
	pipe = make([]*Executor, ntasks)

	pipe[0] = p.createResizeExecutor(c.CollectionPublishFolder, convSets)
	pipe[1] = p.createArchiveExecutor(c.CollectionArchiveFolder, convSettings.MoveOriginal)

	return

}

func createWorker(timeoutMsec int, cmd *Executor, id string, outCh chan<- (*Message), killCh chan (struct{})) func(o chan *imgFile, i chan *imgFile) {
	quit := killCh
	return func(out chan *imgFile, in chan *imgFile) {
		for {
			select {
			case tr := <-in:
				if quit == nil { // HL
					//w.Refuse();
					//fmt.Printf("worker %s refused\n", cmd.Step)
					break
				}

				//wp.activeRequests.Add(1)
				err := executeWithTimeout(cmd, timeoutMsec, tr)

				_, fname := path.Split(tr.Path)

				if err != nil {
					msg := fmt.Sprintf("%s for image %s failed to process due to error %v\n", cmd.StepName, fname, err)
					outCh <- &Message{
						Id: id, Kind: "stderr",
						Body: msg,
					}
					//wp.activeRequests.Done()
					break // get out here
				}

				outCh <- &Message{
					Id: id, Kind: "stdout",
					Body: fmt.Sprintf("%s for image %s correctly executed\n", cmd.StepName, fname),
				}

				//fmt.Printf("worker %s processed without errors\n", cmd.Step)

				//wp.activeRequests.Done()
				out <- tr // pass on
			case <-quit: // via broadcast
				//fmt.Printf("worker %s quitting\n", cmd.Step)
				quit = nil // HL
				// !DO NOT DO THIS: it is dangerous, gives runtime errors
				//close(out) // stop sending and close the channel to make any range loop exit
				return
			}
		}
	}

}

func executeWithTimeout(cmd *Executor, timeoutMsec int, tr *imgFile) (err error) {

	ec := make(chan error)
	go func() {

		// recover from a non-runtime error otherwise panic
		defer func() {
			e := recover()
			if e != nil {
				switch e.(type) {
				case runtime.Error:
					ec <- fmt.Errorf("runtime error: %s", string(debug.Stack()))
					//panic(e)
				case error:
					ec <- e.(error)
				default:
					ec <- fmt.Errorf("Critical error: %s", string(debug.Stack()))
					//panic(e)
				}
			}
		}()

		// use the timeout here
		tc := time.After(time.Duration(timeoutMsec) * time.Millisecond)
		select {
		case ec <- cmd.Do(tr):
		case <-tc:
			ec <- fmt.Errorf("File %s timed out after %d milliseconds", tr.Path, timeoutMsec)
		}
	}()
	err = <-ec
	return
}

// errRecover is the handler that turns panics into returns from the top
// level of Parse.
func errRecover(errp *error) {
	e := recover()
	if e != nil {
		switch err := e.(type) {
		case runtime.Error:
			panic(e)
		case error:
			*errp = err
		default:
			panic(e)
		}
	}
}

// executors

func (p *Process) createResizeExecutor(collPublishFolder string, convSets []*imgParams) (executor *Executor) {
	var resizeHandler = func(img *imgFile) (err error) {

		p.Logger.Debug(fmt.Sprintf("Resizing img: %s.", filepath.Base(img.Path)))

		for _, set := range convSets {
			if len(set.cmdArgs) == 0 {
				return errors.New("Command arguments must be specified")
			}

			newImgName := set.prefix + img.getNormalizedName(true)
			subFolderPath := filepath.Join(collPublishFolder, set.subFolderRelPath)
			var fi os.FileInfo
			if fi, err = os.Stat(subFolderPath); err != nil || !fi.IsDir() {
				// create dirs
				p.Logger.Debug(fmt.Sprintf("Creating folder:%s", subFolderPath))
				if err = os.MkdirAll(subFolderPath, 0777); err != nil {
					return err
				}
			}

			newImgPath := filepath.Join(subFolderPath, newImgName)

			fullCmd := []string{"magick", img.Path}
			fullCmd = append(fullCmd, set.cmdArgs...)
			fullCmd = append(fullCmd, newImgPath)

			c := p.cmd("", fullCmd...) //&Cmd{Args: fullCmd}
			p.Logger.Debug(fmt.Sprintf("Running cmd:%s", c))
			err = c.Run()
			if err != nil {
				return
			}
		}

		return
	}
	return &Executor{StepName: "resize", Do: resizeHandler}
}

func (p *Process) createArchiveExecutor(collArchiveFolder string, moveOriginal bool) (executor *Executor) {
	var archiveHandler = func(img *imgFile) (err error) {

		var fi os.FileInfo
		if fi, err = os.Stat(collArchiveFolder); err != nil || !fi.IsDir() {
			// create dirs
			p.Logger.Debug(fmt.Sprintf("Creating folder:", collArchiveFolder))
			if err = os.MkdirAll(collArchiveFolder, 0777); err != nil {
				return err
			}
		}
		movePath := filepath.Join(collArchiveFolder, img.getNormalizedName(false))

		if moveOriginal {
			p.Logger.Debug(fmt.Sprintf("Archiving original file to:%s", movePath))
			os.Rename(img.Path, movePath)
		} else {
			p.Logger.Debug(fmt.Sprintf("Copying original file to:%s", movePath))
			// take a copy
			f, _ := os.Open(img.Path)
			defer f.Close()
			f1, _ := os.Create(movePath)
			defer f1.Close()
			if _, err = io.Copy(f1, f); err != nil {
				return
			}
			fi, _ = f.Stat()
			// copy stats
			if err = os.Chtimes(movePath, fi.ModTime(), fi.ModTime()); err != nil {
				return
			}
		}
		return
	}
	return &Executor{StepName: "archive", Do: archiveHandler}
}
