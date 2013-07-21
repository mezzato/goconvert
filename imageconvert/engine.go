package imageconvert

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
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
		[]string{"-resize", strconv.Itoa(convSettings.AreaInPixed()) + "@", "-mattecolor", "gray4", "-frame", "4x4+2+2", "-font", "helvetica", "-fill", "black"},
		"",
		"",
	}

	thumbnailPars := &imgParams{
		[]string{"-resize", "128x128", "-mattecolor", "gray4", "-font", "helvetica"},
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

				if err != nil {
					msg := fmt.Sprintf("Step: %s for image file %s failed to process due to error %v\n", cmd.StepName, tr.Path, err)
					outCh <- &Message{
						Id: id, Kind: "stderr",
						Body: msg,
					}
					//wp.activeRequests.Done()
					break // get out here
				}

				outCh <- &Message{
					Id: id, Kind: "stdout",
					Body: fmt.Sprintf("Step: %s for image file %s correctly executed\n", cmd.StepName, tr.Path),
				}

				//fmt.Printf("worker %s processed without errors\n", cmd.Step)

				//wp.activeRequests.Done()
				out <- tr // pass on
			case <-quit: // via broadcast
				//fmt.Printf("worker %s quitting\n", cmd.Step)
				quit = nil // HL
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
					panic(e)
				case error:
					ec <- e.(error)
				default:
					panic(e)
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

		WriteVerbose(fmt.Sprintf("Resizing img: %s.", filepath.Base(img.Path)))

		for _, set := range convSets {
			if len(set.cmdArgs) == 0 {
				return errors.New("Command arguments must be specified")
			}

			newImgName := set.prefix + filepath.Base(img.Path)
			subFolderPath := filepath.Join(collPublishFolder, set.subFolderRelPath)
			var fi os.FileInfo
			if fi, err = os.Stat(subFolderPath); err != nil || !fi.IsDir() {
				// create dirs
				WriteVerbose("Creating folder:", subFolderPath)
				if err = os.MkdirAll(subFolderPath, 0777); err != nil {
					return err
				}
			}

			newImgPath := filepath.Join(subFolderPath, newImgName)

			fullCmd := []string{"convert", img.Path}
			fullCmd = append(fullCmd, set.cmdArgs...)
			fullCmd = append(fullCmd, newImgPath)

			c := p.cmd("", fullCmd...) //&Cmd{Args: fullCmd}
			WriteVerbose("Running cmd:", c)
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
			WriteVerbose("Creating folder:", collArchiveFolder)
			if err = os.MkdirAll(collArchiveFolder, 0777); err != nil {
				return err
			}
		}
		movePath := filepath.Join(collArchiveFolder, filepath.Base(img.Path))

		if moveOriginal {
			WriteVerbose("Archiving original file to:", movePath)
			os.Rename(img.Path, movePath)
		} else {
			WriteVerbose("Copying original file to:", movePath)
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

// Cmd

/*
// A Cmd describes an individual command.
type Cmd struct {
	Args   []string // command-line
	Stdout string   // write standard output to this file, "" is passthrough
	Dir    string   // working directory
	Env    []string // environment
}

func (c *Cmd) String() string {
	return strings.Join(c.Args, " ")
}

// Run executes the Cmd.
func (c *Cmd) Run() error {
	out := new(bytes.Buffer)
	cmd := exec.Command(c.Args[0], c.Args[1:]...)
	cmd.Dir = c.Dir
	cmd.Env = c.Env
	cmd.Stdout = out
	cmd.Stderr = out

	// remember to release the process
	defer func() {
		if cmd != nil {
			if err := cmd.Process.Release(); err != nil {
				//panic(err)
				//WriteInfof("Error while waiting releasing the process %s to finish\n",cmd.Args[0])
			}
		}
	}()

	if c.Stdout != "" {
		f, err := os.Create(c.Stdout)
		if err != nil {
			return err
		}
		defer f.Close()
		cmd.Stdout = f
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command %q: %v\n%v", c, err, out)
	}

	return nil
}
*/
