package routiner

import (
	"log"
	"sync"
)

type Routiner struct {
	input         chan any
	inputThrough  chan any
	output        chan string
	wg            *sync.WaitGroup
	quitJob       chan int
	workers       int
	activeWorkers int
	mu            sync.RWMutex
}

// Run starts the job processes. It first initializes all
// the workers and set them in the active state.Then
// launches the manager process.
//
// To keep track of any notifications that might occur
// in the worker processes, it listens to the input
// channel. Current only log outout is supported.
//
// The job will finish when a quit signal
// is received.
func (r *Routiner) Run(
	manager func(r *Routiner),
	worker func(r *Routiner, o any),
) {
	defer close(r.output)
	defer close(r.input)

	r.startWorkers(worker)

	go r.startManager(manager)

	for {
		select {
		case message := <-r.output:
			log.Println(message)
		case <-r.quitJob:
			return
		}
	}
}

func (r *Routiner) CallSafe(f func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	f()
}

func (r *Routiner) Info(str string) {
	r.output <- str
}

func (r *Routiner) Send(obj any) {
	r.inputThrough <- obj
}

func (r *Routiner) Input() <-chan any {
	return r.input
}

func (r *Routiner) Listen() any {
	return <-r.input
}

func (r *Routiner) Workers() int {
	return r.workers
}

// ActiveWorkers return the number of active workers.
func (r *Routiner) ActiveWorkers() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.activeWorkers
}

// Quit terminates the job and quit immediately.
// It sets the number of active workers to 0.
func (r *Routiner) Quit() {
	r.resetActiveWorkers()
}
