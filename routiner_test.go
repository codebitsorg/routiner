package routiner_test

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/codebitsorg/routiner"
)

func TestRun(t *testing.T) {
	r := routiner.Init(routiner.WithWorkers(10))

	workerOutput := make([]string, r.Workers())

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= r.Workers(); i++ {
			r.Work(i)
		}
	}

	worker := func(r *routiner.Routiner, o any) {
		id := o.(int)
		r.CallSafe(func() {
			workerOutput[id-1] = fmt.Sprintf("Worker %d", id)
		})
	}

	r.Run(manager, worker)

	for i := 0; i < r.Workers(); i++ {
		r.CallSafe(func() {
			if workerOutput[i] != fmt.Sprintf("Worker %d", i+1) {
				t.Errorf("Worker %d did not run", i+1)
			}
		})
	}
}

func TestJobCanBeQuitAtAnyMoment(t *testing.T) {
	var workerOutput []string

	r := routiner.Init(routiner.WithWorkers(3))

	// Create a WaitGroup to hold all the workers
	// until the quit signal is received.
	wgForQuit := new(sync.WaitGroup)
	wgForQuit.Add(1)

	worker := func(r *routiner.Routiner, o any) {
		id := o.(int)

		if id == 2 {
			workerOutput = append(workerOutput, fmt.Sprintf("Worker %d", id))

			r.Quit()
		} else {
			wgForQuit.Wait()
			workerOutput = append(workerOutput, fmt.Sprintf("Worker %d", id))
		}
	}

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= r.Workers(); i++ {
			r.Work(i)
		}
	}

	r.Run(manager, worker)

	if r.ActiveWorkers() != 0 {
		t.Errorf("Active Workers should be set to 0, but got %d", r.ActiveWorkers())
	}

	if len(workerOutput) != 1 || workerOutput[0] != "Worker 2" {
		t.Errorf("Worker 2 was not found: %s", workerOutput)
	}
}

func TestRoutinerCanTrackActiveWorkers(t *testing.T) {
	r := routiner.Init(routiner.WithWorkers(10))

	workerChannels := make([]chan *sync.WaitGroup, r.Workers())
	waitGroups := make([]*sync.WaitGroup, r.Workers())

	worker := func(r *routiner.Routiner, o any) {
		ch := o.(chan *sync.WaitGroup)
		wg := <-ch
		wg.Done()
	}

	if r.ActiveWorkers() != 0 {
		t.Errorf("Active workers should be 0, but got %d", r.ActiveWorkers())
	}

	// The testing process should only start once all workers are in the active state.
	// We can achieve that by passing a WaitGroup to the manager clouser and call
	// Wait after the Run method.
	//
	// *The manager process in the Run method is started only after all
	// workers has been set to an active state.
	wgReadyToTest := new(sync.WaitGroup)
	wgReadyToTest.Add(1)
	go r.Run(func(r *routiner.Routiner) {
		wgReadyToTest.Done()
	}, worker)
	wgReadyToTest.Wait()

	for i := 0; i < r.Workers(); i++ {
		waitGroups[i] = new(sync.WaitGroup)
		waitGroups[i].Add(1)

		workerChannels[i] = make(chan *sync.WaitGroup)
		r.Work(workerChannels[i])
	}

	// time.Sleep(time.Millisecond * 100)
	if r.ActiveWorkers() != r.Workers() {
		t.Errorf("Active workers should be %d, but got %d", r.Workers(), r.ActiveWorkers())
	}

	for i := 0; i < r.Workers()/2; i++ {
		workerChannels[i] <- waitGroups[i]
		waitGroups[i].Wait()
	}

	if r.ActiveWorkers() != r.Workers()/2 {
		t.Errorf("Active workers should be %d, but got %d", r.Workers()/2, r.ActiveWorkers())
	}

	for i := r.Workers() / 2; i < r.Workers()-2; i++ {
		workerChannels[i] <- waitGroups[i]
		waitGroups[i].Wait()
	}

	if r.ActiveWorkers() != 2 {
		t.Errorf("Active workers should be 2, but got %d", r.ActiveWorkers())
	}

	for i := r.Workers() - 2; i < r.Workers(); i++ {
		workerChannels[i] <- waitGroups[i]
		waitGroups[i].Wait()
	}

	if r.ActiveWorkers() != 0 {
		t.Errorf("Active workers should be 0, but got %d", r.ActiveWorkers())
	}
}

func TestCallSafe(t *testing.T) {
	safeSum := 0

	r := routiner.Init(routiner.WithWorkers(100))

	worker := func(r *routiner.Routiner, o any) {
		for i := 0; i < 1000; i++ {
			r.CallSafe(func() {
				safeSum++
			})
		}
	}

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= r.Workers(); i++ {
			r.Work(i)
		}
	}

	r.Run(manager, worker)

	if safeSum != 100000 {
		t.Errorf("Safe sum should be 100000, but got %d", safeSum)
	}
}

func TestWork(t *testing.T) {
	r := routiner.Init(routiner.WithBufferedInputChannel(1))

	testMessage := "Hello World!"
	r.Work(testMessage)

	select {
	case message := <-r.Input():
		if message != testMessage {
			t.Errorf("Expected message %q, got %q", testMessage, message)
		}
	case <-time.After(time.Millisecond * 100):
	}
}

func TestInfo_LogOutputType(t *testing.T) {
	// Redirecting log output to a buffer, so we can test it.
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(nil)
	}()

	expectedOutput := "Worker has finished!"
	routiner.Init().RunWorkers(func(r *routiner.Routiner, o any) {
		r.Info(expectedOutput)
	})

	logOutput := buf.String()

	if !strings.Contains(buf.String(), "Worker has finished!") {
		t.Errorf("Expected output %q, got %q", expectedOutput, logOutput)
	}
}
