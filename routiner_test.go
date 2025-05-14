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

func TestTotalInputWorkers(t *testing.T) {
	t.Parallel()

	expectedHighInputWorkers := 2
	expectedLowInputWorkers := 4

	r := routiner.New()

	r.AddWorker(
		func(r *routiner.Routiner, m any) {},
		"high",
		expectedHighInputWorkers,
	)

	r.AddWorker(
		func(r *routiner.Routiner, m any) {},
		"low",
		expectedLowInputWorkers,
	)

	if r.TotalInputWorkers("high") != expectedHighInputWorkers {
		t.Errorf("Expected %d high input workers, but got %d", expectedHighInputWorkers, r.TotalInputWorkers("high"))
	}

	if r.TotalInputWorkers("low") != expectedLowInputWorkers {
		t.Errorf("Expected %d low input workers, but got %d", expectedLowInputWorkers, r.TotalInputWorkers("low"))
	}
}

func TestTotalWorkers(t *testing.T) {
	t.Parallel()

	r := routiner.New()
	r.AddWorker(
		func(r *routiner.Routiner, m any) {},
		"default",
		10,
	)

	if r.TotalWorkers() != 10 {
		t.Errorf("Expected 10 workers, but got %d", r.TotalWorkers())
	}
}

func TestSingleInputChannel(t *testing.T) {
	t.Parallel()

	r := routiner.New()

	totalWorkers := 10
	workerOutput := make([]string, totalWorkers)

	manager := func(r *routiner.Routiner) {
		for i := 0; i < r.TotalWorkers(); i++ {
			r.Send(i)
		}
	}

	worker := func(r *routiner.Routiner, m any) {
		id := m.(int)
		r.CallSafe(func() {
			workerOutput[id] = fmt.Sprintf("Worker %d", id)
		})

		if id == totalWorkers-1 {
			r.Input("default").Done()
		}
	}

	r.AddManager(manager).AddWorker(worker, "default", totalWorkers)
	r.Start()

	for i := 0; i < r.TotalWorkers(); i++ {
		r.CallSafe(func() {
			if workerOutput[i] != fmt.Sprintf("Worker %d", i) {
				t.Errorf("Worker %d did not run", i)
			}
		})
	}
}

func TestMultipleInputChannels_WorkersForEachInputWorkOnDifferentTasks(t *testing.T) {
	t.Parallel()

	r := routiner.New()

	highPriorityIterations := 5
	highPriorityOutput := make([]string, 0, highPriorityIterations)
	lowPriorityIterations := 10
	lowPriorityOutput := make([]string, 0, lowPriorityIterations)

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= highPriorityIterations; i++ {
			r.SendTo("high", i)
		}

		r.Input("high").Done()

		for i := 1; i <= lowPriorityIterations; i++ {
			r.SendTo("low", i)
		}

		r.Input("low").Done()
	}

	workersWithHighPriority := func(r *routiner.Routiner, m any) {
		id := m.(int)
		r.CallSafe(func() {
			highPriorityOutput = append(
				highPriorityOutput,
				fmt.Sprintf("High: %d", id),
			)
		})
	}

	workersWithLowPriority := func(r *routiner.Routiner, m any) {
		id := m.(int)
		r.CallSafe(func() {
			lowPriorityOutput = append(
				lowPriorityOutput,
				fmt.Sprintf("Low: %d", id),
			)
		})
	}

	r.AddManager(manager).
		AddWorker(workersWithHighPriority, "high", 2).
		AddWorker(workersWithLowPriority, "low", 4)

	r.Start()

	lowPriorityOutputString := strings.Join(lowPriorityOutput, ", ")
	highPriorityOutputString := strings.Join(highPriorityOutput, ", ")

	r.CallSafe(func() {
		if len(highPriorityOutput) != highPriorityIterations {
			t.Errorf("High priority workers did not run %d times", highPriorityIterations)
		}

		if len(lowPriorityOutput) != lowPriorityIterations {
			t.Errorf("Low priority workers did not run %d times", lowPriorityIterations)
		}

		for i := 1; i <= highPriorityIterations; i++ {
			if !strings.Contains(highPriorityOutputString, fmt.Sprintf("High: %d", i)) {
				t.Errorf("High priority worker %d did not run", i)
			}
		}

		for i := 1; i <= lowPriorityIterations; i++ {
			if !strings.Contains(lowPriorityOutputString, fmt.Sprintf("Low: %d", i)) {
				t.Errorf("Low priority worker %d did not run", i)
			}
		}
	})
}

func TestMultipleInputChannels_WorkersFromOneInputProcessMessagesFromAnotherInput(t *testing.T) {
	t.Parallel()

	r := routiner.New()

	var output []string

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= 10; i++ {
			r.Send(i)
		}

		//r.Input("default").Done()
	}

	workerForAllNumbers := func(r *routiner.Routiner, m any) {
		n := m.(int)

		r.CallSafe(func() {
			output = append(output, fmt.Sprintf("All numbers: #%d", n))
		})

		//if n%2 == 0 {
		//	r.SendTo("even", m)
		//}
	}

	workerForEvenNumbers := func(r *routiner.Routiner, m any) {
		time.Sleep(250 * time.Millisecond)
		n := m.(int)

		r.CallSafe(func() {
			output = append(output, fmt.Sprintf("Even number: #%d", n))
		})

		//if n == 10 {
		//	r.Input("even").Done()
		//}
	}

	r.AddManager(manager).
		AddWorker(workerForAllNumbers, "default", 3).
		AddWorker(workerForEvenNumbers, "even", 2).
		Start()

	outputString := strings.Join(output, ", ")

	r.CallSafe(func() {
		if len(output) != 15 {
			t.Errorf("Expected 15 outputs, but got %d", len(output))
		}

		for i := 1; i <= 10; i++ {
			if !strings.Contains(outputString, fmt.Sprintf("All numbers: #%d", i)) {
				t.Errorf("All numbers worker %d did not run", i)
			}

			if i%2 == 0 {
				if !strings.Contains(outputString, fmt.Sprintf("Even number: #%d", i)) {
					t.Errorf("Even numbers worker %d did not run", i)
				}
			}
		}
	})
}

func TestMultipleInputChannels_OrphanedChannelsCloseThemselves(t *testing.T) {
	t.Parallel()

	r := routiner.New()

	var output []string

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= 3; i++ {
			r.Send(i)
		}
	}

	worker := func(r *routiner.Routiner, m any) {
		n := m.(int)

		r.CallSafe(func() {
			output = append(output, fmt.Sprintf("#%d", n))
		})

		if n == 3 {
			r.Inputs().Done()
		}
	}

	r.AddManager(manager).
		AddWorker(worker, "default", 3).
		AddWorker(func(r *routiner.Routiner, m any) {}, "orphan1", 2).
		AddWorker(func(r *routiner.Routiner, m any) {}, "orphan2", 2).
		Start()

	outputString := strings.Join(output, ", ")

	r.CallSafe(func() {
		if len(output) != 3 {
			t.Errorf("Expected 10 outputs, but got %d", len(output))
		}

		for i := 1; i <= 3; i++ {
			if !strings.Contains(outputString, fmt.Sprintf("#%d", i)) {
				t.Errorf("All numbers worker %d did not run", i)
			}
		}
	})
}

func TestSendTo(t *testing.T) {
	t.Parallel()

	var output string
	expected := "Hello World!"

	manager := func(r *routiner.Routiner) {
		r.SendTo("default", expected)
		r.Input("default").Done()
	}

	worker := func(r *routiner.Routiner, m any) {
		output = m.(string)
	}

	routiner.New().
		AddManager(manager).
		AddWorker(worker, "default", 1).
		Start()

	if output != expected {
		t.Errorf("Expected output %s, but got %s", expected, output)
	}
}

func TestSendTo_MessageCantBeSentToAClosedChannel(t *testing.T) {
	t.Parallel()

	var output []string

	manager := func(r *routiner.Routiner) {
		r.Input("default").Done()
		go func() {
			for i := 0; i < 3; i++ {
				ok := r.SendTo("default", i)
				if ok {
					t.Errorf("Message %d shouldn't have been sent to a closed channel", i)
				}
			}
		}()
	}

	worker := func(r *routiner.Routiner, m any) {
		id := m.(int)
		r.CallSafe(func() {
			output = append(output, fmt.Sprintf("Worker %d", id))
		})
	}

	routiner.New().
		AddManager(manager).
		AddWorker(worker, "default", 1).
		Start()

	if len(output) != 0 {
		t.Errorf("Worker output should be empty, but got %v", output)
	}
}

func TestJobCanBeQuitAtAnyMoment(t *testing.T) {
	t.Parallel()

	var output []string

	// Create a WaitGroup to hold all the workers
	// until the quit signal is received.
	wgForQuit := new(sync.WaitGroup)
	wgForQuit.Add(1)

	worker := func(r *routiner.Routiner, m any) {
		id := m.(int)

		if id == 2 {
			output = append(output, fmt.Sprintf("Worker %d", id))

			r.Quit()
		} else {
			wgForQuit.Wait()
			output = append(output, fmt.Sprintf("Worker %d", id))
		}
	}

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= r.TotalWorkers(); i++ {
			r.Send(i)
		}
		r.Input("default").Done()
	}

	r := routiner.New()
	r.AddManager(manager).
		AddWorker(worker, "default", 3).
		Start()

	if r.ActiveWorkers() != 0 {
		t.Errorf("Active Workers should be set to 0, but got %d", r.ActiveWorkers())
	}

	if len(output) != 1 || output[0] != "Worker 2" {
		t.Errorf("Worker 2 was not found: %s", output)
	}
}

func TestRoutinerCanTrackActiveWorkers(t *testing.T) {
	t.Parallel()

	r := routiner.New()

	// Each worker will receive a channel through which it
	// will receive a wait group. Once wg is received,
	// the worker will call Done on it.
	//
	// We will be manually sending wait groups to the channels. That
	// way we can control the order in which the workers finish.
	worker := func(r *routiner.Routiner, m any) {
		ch := m.(chan *sync.WaitGroup)
		wg := <-ch
		wg.Done()
	}

	r.AddWorker(worker, "default", 10)

	// We need to create slices of worker channels and wait groups
	// to keep track of the workers and their states. Because
	// wait groups will be passed through the channels, we
	// need to use pointers for proper synchronization.
	workerChannels := make([]chan *sync.WaitGroup, r.TotalWorkers())
	waitGroups := make([]*sync.WaitGroup, r.TotalWorkers())

	// The testing process should only start once all workers are in the
	// active state. We can achieve that by passing a WaitGroup to the
	// manager closure and calling Wait after the Run method.
	//
	// *The manager process in the Run method is started only after all
	// workers have been set to an active state.
	wgReadyToTest := new(sync.WaitGroup)
	wgReadyToTest.Add(1)

	manager := func(r *routiner.Routiner) {
		for i := 0; i < r.TotalWorkers(); i++ {
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

		r.Input("default").Done()

		// Once all workers have been initialized
		// we can start the testing process.
		wgReadyToTest.Done()
	}

	// Check that there are no active workers
	// before starting the Run method.
	if r.ActiveWorkers() != 0 {
		t.Fatalf("Active workers should be 0, but got %d", r.ActiveWorkers())
	}

	r.AddManager(manager)

	go r.Start()

	// Waiting for all workers to be in the active state
	wgReadyToTest.Wait()

	if r.ActiveWorkers() != r.TotalWorkers() {
		t.Errorf("Active workers should be %d, but got %d", r.TotalWorkers(), r.ActiveWorkers())
	}

	for i := 0; i < r.TotalWorkers()/2; i++ {
		workerChannels[i] <- waitGroups[i]
		waitGroups[i].Wait()
	}

	if r.ActiveWorkers() != r.TotalWorkers()/2 {
		t.Errorf("Active workers should be %d, but got %d", r.TotalWorkers()/2, r.ActiveWorkers())
	}

	for i := r.TotalWorkers() / 2; i < r.TotalWorkers()-2; i++ {
		workerChannels[i] <- waitGroups[i]
		waitGroups[i].Wait()
	}

	if r.ActiveWorkers() != 2 {
		t.Errorf("Active workers should be 2, but got %d", r.ActiveWorkers())
	}

	for i := r.TotalWorkers() - 2; i < r.TotalWorkers(); i++ {
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

	worker := func(r *routiner.Routiner, m any) {
		for i := 0; i < 1000; i++ {
			r.CallSafe(func() {
				safeSum++
			})
		}
	}

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= r.TotalWorkers(); i++ {
			r.Send(i)
		}
		r.Input("default").Done()
	}

	routiner.New().
		AddManager(manager).
		AddWorker(worker, "default", 100).
		Start()

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
		r.Input("default").Done()
	}
	worker := func(r *routiner.Routiner, m any) {
		r.Info(m.(string))
	}
	routiner.New().Run(manager, worker)

	logOutput := buf.String()

	if !strings.Contains(buf.String(), "Worker has finished!") {
		t.Errorf("Expected output %q, got %q", expectedOutput, logOutput)
	}
}

func TestRun(t *testing.T) {
	t.Parallel()

	r := routiner.New()

	totalWorkers := 2
	output := make([]string, totalWorkers)

	manager := func(r *routiner.Routiner) {
		for i := 0; i < r.TotalWorkers(); i++ {
			r.Send(i)
		}
		r.Input("default").Done()
	}

	worker := func(r *routiner.Routiner, m any) {
		id := m.(int)
		r.CallSafe(func() {
			output[id] = fmt.Sprintf("Worker %d", id)
		})
	}

	_ = r.Run(manager, worker, "default", totalWorkers)

	if r.TotalWorkers() != totalWorkers {
		t.Errorf("Expected %d workers, but got %d", totalWorkers, r.TotalWorkers())
	}

	for i := 0; i < totalWorkers; i++ {
		io := fmt.Sprintf("Worker %d", i)
		if output[i] != io {
			t.Errorf("Expected output %s, but got %s", io, output[i])
		}
	}
}
