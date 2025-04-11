package routiner_test

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/codebitsorg/routiner"
)

func TestInit_InitializingRoutinerAddsOneWorkerByDefault(t *testing.T) {
	t.Parallel()

	r := routiner.Init()
	if r.Workers() != 1 {
		t.Errorf("Workers should be 1, but got %d", r.Workers())
	}
}

func TestInit_CanOptionalySetWorkers(t *testing.T) {
	t.Parallel()

	r := routiner.Init(routiner.WithWorkers(4))
	if r.Workers() != 4 {
		t.Errorf("Workers should be 4, but got %d", r.Workers())
	}
}

func TestRun(t *testing.T) {
	t.Parallel()

	r := routiner.Init(routiner.WithWorkers(10))

	workerOutput := make([]string, r.Workers())

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= r.Workers(); i++ {
			r.Send(i)
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
	t.Parallel()

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
			r.Send(i)
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
	t.Parallel()

	r := routiner.Init(routiner.WithWorkers(10))

	// We need to create slices of worker channels and wait groups
	// to keep track of the workers and their states. Because
	// wait groups will be passed through the channels, we
	// need to use pointers for proper synchronization.
	workerChannels := make([]chan *sync.WaitGroup, r.Workers())
	waitGroups := make([]*sync.WaitGroup, r.Workers())

	// Each worker will receive a channel through which it
	// will receive a wait group. Once wg is received,
	// the worker will call Done on it.
	//
	// We will be manually sending wait groups to the channels. That
	// way we can control the order in which the workers finish.
	worker := func(r *routiner.Routiner, o any) {
		ch := o.(chan *sync.WaitGroup)
		wg := <-ch
		wg.Done()
	}

	// The testing process should only start once all workers are in the
	// active state. We can achieve that by passing a WaitGroup to the
	// manager clouser and call Wait after the Run method.
	//
	// *The manager process in the Run method is started only after all
	// workers has been set to an active state.
	wgReadyToTest := new(sync.WaitGroup)
	wgReadyToTest.Add(1)

	manager := func(r *routiner.Routiner) {
		for i := 0; i < r.Workers(); i++ {
			// Creating a new wait group for each worker and adding
			// a pointer to it to the waitGroups slice.
			waitGroups[i] = new(sync.WaitGroup)
			// Incrementing the wait group counter for each worker.
			waitGroups[i].Add(1)

			// Creating a channel for each worker and adding
			// it to the workerChannels slice.
			workerChannels[i] = make(chan *sync.WaitGroup)
			// Send the channel to the worker. These channels will
			// be used to send the wait groups to the workers.
			r.Send(workerChannels[i])
		}

		// Once all workers have been initialized
		// we can start the testing process.
		wgReadyToTest.Done()
	}

	// Check that there are no active workers
	// before starting the Run method.
	if r.ActiveWorkers() != 0 {
		t.Fatalf("Active workers should be 0, but got %d", r.ActiveWorkers())
	}

	go r.Run(manager, worker)

	// Waiting for all workers to be in the active state
	wgReadyToTest.Wait()

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
	t.Parallel()

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
			r.Send(i)
		}
	}

	r.Run(manager, worker)

	if safeSum != 100000 {
		t.Errorf("Safe sum should be 100000, but got %d", safeSum)
	}
}

func TestInfo_LogOutputType(t *testing.T) {
	t.Parallel()

	// Redirecting log output to a buffer, so we can test it.
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(nil)
	}()

	expectedOutput := "Worker has finished!"
	manager := func(r *routiner.Routiner) {
		r.Send(expectedOutput)
	}
	worker := func(r *routiner.Routiner, o any) {
		r.Info(o.(string))
	}
	routiner.Init().Run(manager, worker)

	logOutput := buf.String()

	if !strings.Contains(buf.String(), "Worker has finished!") {
		t.Errorf("Expected output %q, got %q", expectedOutput, logOutput)
	}
}
