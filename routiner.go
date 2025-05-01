package routiner

import (
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
)

// Worker is a struct that represents a worker.
type Worker struct {
	f        func(r *Routiner, o any)
	input    string
	quantity int
}

// Input is a struct that represents an input channel.
type Input struct {
	name   string
	ch     chan any
	closed bool
}

// Routiner is a struct that represents a routiner.
type Routiner struct {
	manager       func(r *Routiner)
	workers       []Worker
	input         map[string]*Input
	output        chan string
	quitJob       chan int
	activeWorkers int
	wg            *sync.WaitGroup // Wait group to track the number of active workers.
	mu            sync.RWMutex
}

// ***** Options *****
type option struct {
	name   string
	action func(*Routiner)
}

func newOption(name string, action func(*Routiner)) option {
	return option{
		name:   name,
		action: action,
	}
}

func (o option) apply(r *Routiner) {
	o.action(r)
}

// ***** End Options *****

// New is a helper function to create a new Routiner.
func New(opts ...option) *Routiner {
	return NewRoutiner(opts...)
}

// NewRoutiner creates a new Routiner.
func NewRoutiner(opts ...option) *Routiner {
	r := &Routiner{
		output:  make(chan string),
		wg:      new(sync.WaitGroup),
		quitJob: make(chan int),
	}

	r.With(opts...)

	// If WithInputChannels option is not set the input
	// field will be nil. In this case we need to
	// initialize it with a default channel.
	if r.input == nil {
		r.input = make(map[string]*Input, 1)
		r.input["default"] = &Input{
			name: "default",
			ch:   make(chan any),
		}
	}

	return r
}

// With is a helper function to set options
func (r *Routiner) With(opts ...option) {
	// Some other options might rely on amount of workers
	// set. So we need to apply the WithWorkers option
	// first if it is present in the options slice.
	for i, opt := range opts {
		if opt.name == "WithWorkers" {
			opt.apply(r)
			// Remove WithWorkers from the slice.
			opts = append(opts[:i], opts[i+1:]...)
			break
		}
	}

	// Apply the remaining options.
	for _, opt := range opts {
		opt.apply(r)
	}
}

func WithOutputChannel(n int) option {
	return newOption("WithOutputChannel", func(r *Routiner) {
		// Set the output channel.
	})
}

func (r *Routiner) AddManager(f func(r *Routiner)) *Routiner {
	r.RegisterManager(f)

	return r
}

func (r *Routiner) RegisterManager(f func(r *Routiner)) {
	r.manager = f
}

func (r *Routiner) AddWorker(f func(r *Routiner, m any), input string, quantity int) *Routiner {
	r.RegisterWorkers(f, input, quantity)

	return r
}
func (r *Routiner) RegisterWorkers(f func(r *Routiner, m any), input string, quantity int) {
	r.workers = append(r.workers, Worker{f, input, quantity})

	if _, ok := r.input[input]; !ok {
		r.input[input] = &Input{
			name: input,
			ch:   make(chan any),
		}
	}
}

func (r *Routiner) Start() {
	defer close(r.output)

	r.startWorkers()

	go r.startManager()

	for {
		select {
		case message := <-r.output:
			log.Println(message)
		case <-r.quitJob:
			return
		}
	}
}

// Run starts the job processes. It first initializes all
// the workers and set them in the active state.Then
// launches the manager process.
//
// To keep track of any notifications that might occur
// in the worker processes, it listens to the input
// channel. Current only log output is supported.
//
// The job will finish when a quit signal
// is received.
func (r *Routiner) Run(
	manager func(r *Routiner),
	worker ...any,
	// worker func(r *Routiner, o any),
) error {
	defer close(r.output)

	r.RegisterManager(manager)

	if len(worker) < 1 {
		return errors.New("no worker provided")
	}

	f, ok := worker[0].(func(r *Routiner, o any))
	if !ok {
		return errors.New("worker must be a function")
	}

	input := "default"
	if len(worker) > 1 && worker[1] != nil {
		input, ok = worker[1].(string)
		if !ok {
			return errors.New("input must be a string")
		}
	}

	quantity := 1
	if len(worker) > 2 && worker[2] != nil {
		quantity, ok = worker[2].(int)
		if !ok {
			return errors.New("quantity must be an int")
		}
	}

	r.RegisterWorkers(f, input, quantity)

	r.startWorkers()

	go r.startManager()

	for {
		select {
		case message := <-r.output:
			log.Println(message)
		case <-r.quitJob:
			return nil
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

// Send sends the message to the default channel.
func (r *Routiner) Send(obj any) bool {
	if r.input["default"].closed {
		return false
	}

	r.input["default"].ch <- obj

	return true
}

// SendTo sends the message to the specified input channel.
func (r *Routiner) SendTo(channel string, obj any) bool {
	if r.input[channel].closed {
		return false
	}

	r.input[channel].ch <- obj

	return true
}

func (r *Routiner) TotalInputWorkers(input string) int {
	for _, w := range r.workers {
		if w.input == input {
			return w.quantity
		}
	}

	return 0
}

func (r *Routiner) TotalWorkers() int {
	var workers int

	for _, w := range r.workers {
		if w.f != nil {
			workers += w.quantity
		}
	}

	return workers
}

func (r *Routiner) CountInputChannels() int {
	return len(r.input)
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

func (r *Routiner) startWorkers() {
	// Wait for all workers to be active.
	wgActiveWorkers := sync.WaitGroup{}
	wgActiveWorkers.Add(r.TotalWorkers())
	defer wgActiveWorkers.Wait()

	// Start the workers.
	for _, w := range r.workers {
		for i := 0; i < w.quantity; i++ {
			go func(w Worker, i int) {
				defer r.recover()()

				// Should be the method of the worker
				r.activateWorker(&wgActiveWorkers)

				for message := range r.input[w.input].ch {
					w.f(r, message)
				}

				// Should be the method of the worker
				r.deactivateWorker()
			}(w, i)
		}
	}
}

func (r *Routiner) startManager() {
	defer r.recover()()

	r.manager(r)

	for _, i := range r.input {
		close(i.ch)
		i.closed = true
	}

	r.waitToFinish()
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
