package imageconvert

import (
	"code.google.com/p/goconvert/logger"
	"code.google.com/p/goconvert/settings"
	//"fmt"
	"os"
	"path/filepath"
	"testing"
)

func createTestExecutors(c *ConversionFileSystem) (pipe []*Executor) {
	ntasks := 2
	pipe = make([]*Executor, ntasks)
	var do func(*imgFile) error

	do = func(t *imgFile) (err error) {
		//fmt.Printf("test step for file: %s\n", t.Path)
		return
	}
	pipe[0] = &Executor{Do: do, StepName: "step1"}
	pipe[1] = &Executor{Do: do, StepName: "step2"}

	// tracking callback
	//pipe[1] = &Executor{Do: CreatePostTracker(), Step: CALLBACK}

	return

}

func TestEngine(t *testing.T) {
	//id, body string, out chan<- *Message, opt *Options, executors []*Executor
	opt := new(Options)

	homeImgDir := "../test"
	settings := settings.NewDefaultSettings("testcollection", homeImgDir)
	opt.Settings = settings
	outCh := make(chan *Message)
	var p *Process
	var e error
	t.Log("test starting")
	count := 0

	go func() {
		for m := range outCh {
			t.Logf("messsage: kind %s, id: %s, message: %s", m.Kind, m.Id, m.Body)
			if m.Kind != "end" {
				count++
			}

		}
	}()

	t.Logf("Process starting\n")
	p = newProcess("test", outCh, logger.ERROR)

	_, e = p.tryStart("body", settings, createTestExecutors)
	t.Log("Process started")
	if e != nil {
		t.Fatalf("error in process creation: %v", e)
	}

	e = p.Wait()
	if e != nil {
		t.Fatalf("error %q", e)
	}
	expectedCount := 2 * 15
	if count != expectedCount {
		t.Fatalf("The number of messages is %d, expected %d", count, expectedCount)
	}
	//c.Wait()

}

func TestConversion(t *testing.T) {
	//id, body string, out chan<- *Message, opt *Options, executors []*Executor
	var p *Process
	var e error
	var cfs *ConversionFileSystem
	srcdir := "../test"

	m, e := filepath.Glob(srcdir + "/*.jpg")
	srccount := len(m)

	sets := settings.NewDefaultSettings("testcollection", srcdir)
	sets.PublishDir = filepath.Join(os.TempDir(), "imageconverttest")

	//os.RemoveAll(sets.PublishDir)

	opt := &Options{Settings: sets}
	outCh := make(chan *Message)
	count := 0

	go func() {
		for m := range outCh {
			t.Logf("messsage: kind %s, id: %s, message: %s", m.Kind, m.Id, m.Body)
			if m.Kind != "end" {
				count++
			}

		}
	}()

	t.Logf("process starting\n")
	t.Logf("Testing to folder %s\n", sets.PublishDir)

	p, cfs, e = CreateAndStartProcess("test", "body", outCh, opt)
	t.Log("process started")
	if e != nil {
		t.Fatalf("error in process creation: %v", e)
	}

	t.Logf("Converting images into folder %s\n", cfs.CollectionPublishFolder)
	defer os.RemoveAll(cfs.CollectionPublishFolder)

	e = p.Wait()
	if e != nil {
		t.Fatalf("error %q", e)
	}

	m, e = filepath.Glob(cfs.CollectionPublishFolder + "/*.jpg")
	destcount := len(m)

	if srccount != destcount {
		t.Fatalf("The number of messages is %d, expected %d", destcount, srccount)
	}
	//c.Wait()

}
