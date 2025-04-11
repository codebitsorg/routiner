package routiner

import (
	"fmt"
	"log"
	"runtime/debug"
	"sync"
)

type Routiner struct {
	input         chan any
	inputThrough  chan any
	output        chan string
	wg            *sync.WaitGroup // Wait group to track the number of active workers.
	quitJob       chan int
	workers       int
	activeWorkers int
	mu            sync.RWMutex
}

type option func(*Routiner)

// NewRoutiner creates a new Routiner.
func NewRoutiner(opts ...option) *Routiner {
	r := &Routiner{
		input:   make(chan any),
		output:  make(chan string),
		wg:      new(sync.WaitGroup),
		quitJob: make(chan int),
		workers: 1,
	}

	r.With(opts...)

	return r
}

// New is a helper function to create a new Routiner.
func New(opts ...option) *Routiner {
	return NewRoutiner(opts...)
}

// Init is a helper function to create a new Routiner.
func Init(opts ...option) *Routiner {
	return NewRoutiner(opts...)
}

// With is a helper function to set options
func (r *Routiner) With(opts ...option) {
	for _, opt := range opts {
		opt(r)
	}
}

// WithWorkers sets the number of workers to n.
func WithWorkers(n int) option {
	return func(r *Routiner) {
		if n < 1 {
			n = 1
		}

		r.workers = n
	}
}

// WithBufferedInputChannel creates a buffered input channel
// with size n.
func WithBufferedInputChannel(n int) option {
	return func(r *Routiner) {
		if n < 1 {
			n = 1
		}

		r.input = make(chan any, n)
	}
}

// WithBufferedOutputChannel creates a buffered output channel
// with size n.
func WithBufferedOutputChannel(n int) option {
	return func(r *Routiner) {
		if n < 1 {
			n = 1
		}

		r.output = make(chan string, n)
	}
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

// startManager starts the manager process and waits
// for all the workers to finish.
func (r *Routiner) startManager(manager func(r *Routiner)) {
	defer r.recover()()

	r.inputThrough = make(chan any)

	// If manager is too fast and there are too many workers to 
	// spawn out, there might be a situation (panic) where a 
	// goroutine sends closed inputThrough channel to the 
	// workers. To avoid this, we need to use WaitGroups 
	// to wait for for all workers receiving their 
	// inputThrough channel.
	wg := &sync.WaitGroup{}

	for range r.Workers() {
		wg.Add(1)
		go func() {
			defer r.recover()()
			defer wg.Done()

			r.input <- r.inputThrough
		}()
	}

	manager(r)

	// Before closing the inputThrough channel, we must be 
	// sure that it has been passed to all workers.
	wg.Wait()

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
