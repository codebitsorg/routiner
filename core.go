package routiner

import (
	"fmt"
	"log"
	"runtime/debug"
	"sync"
)

// startManager starts the manager process and waits
// for all the workers to finish.
func (r *Routiner) startManager(manager func(r *Routiner)) {
	defer r.recover()()

	r.inputThrough = make(chan any)

	for range r.Workers() {
		go func() {
			defer r.recover()()

			r.work(r.inputThrough)
		}()
	}

	manager(r)

	// The inputThrough channel must be closed before
	// the main input channel to avoid deadlocks.
	close(r.inputThrough)

	r.waitToFinish()
}

// startWorkers Run the worker's handler. This function increases the
// number of active workers and holds the worker until the data is
// available in the input channel.
//
// The function will finish only when all workers have been
// set to the active state.
//
// Once the handler is done, the worker is deactivated:
// the number of active workers and the activeWorkers
// wait group are decremented.
func (r *Routiner) startWorkers(worker func(r *Routiner, input any)) {
	// Wait for all workers to be active.
	wgAcitveWorkers := &sync.WaitGroup{}
	wgAcitveWorkers.Add(r.Workers())
	defer wgAcitveWorkers.Wait()

	// Start the workers.
	for range r.Workers() {
		go func() {
			defer r.recover()()

			r.activateWorker(wgAcitveWorkers)

			for input := range r.input {
				for message := range input.(chan any) {
					worker(r, message)
				}
				r.deactivateWorker()
			}
		}()
	}
}

// work initiate the worker process by sending 
// the data to the input channel.
func (r *Routiner) work(obj any) {
	r.input <- obj
}

// waitToFinish waits for all workers to finish.
func (r *Routiner) waitToFinish() {
	r.wg.Wait()
	r.quitJob <- 0
}

// Activate a worker.
func (r *Routiner) activateWorker(wgAcitveWorkers *sync.WaitGroup) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.wg.Add(1)
	r.activeWorkers++
	wgAcitveWorkers.Done()
}

// Deactivate a worker.
func (r *Routiner) deactivateWorker() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.activeWorkers > 0 {
		r.activeWorkers--
		r.wg.Done()
	}
}

// Reset Active Workers.
func (r *Routiner) resetActiveWorkers() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := 0; i < r.activeWorkers; i++ {
		r.wg.Done()
	}

	r.activeWorkers = 0
}

// Recover the application in case of a panic.
func (r *Routiner) recover() func() {
	f := func() {
		if err := recover(); err != nil {
			log.Print(fmt.Errorf("panic: %s\n%s", err, debug.Stack()))
		}
	}

	return f
}
