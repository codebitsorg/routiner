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
	f      func(r *Routiner, o any)
	input  string
	active bool
	//quantity int
}

// InputCollection is a struct that represents
// a collection of input channels.
type InputCollection struct {
	inputs []*Input
}

func newInputCollection() *InputCollection {
	return &InputCollection{
		inputs: make([]*Input, 0),
	}
}

func (ic *InputCollection) Count() int {
	return len(ic.inputs)
}

func (ic *InputCollection) Add(input *Input) {
	ic.inputs = append(ic.inputs, input)
}

func (ic *InputCollection) Get(name string) *Input {
	for _, in := range ic.inputs {
		if in.name == name {
			return in
		}
	}

	return nil
}

func (ic *InputCollection) Done() {
	for _, in := range ic.inputs {
		in.Done()
	}
}

// Input is a struct that represents an input channel.
type Input struct {
	name   string
	ch     chan any
	quit   chan int
	closed bool
	cond   *sync.Cond
}

func (i *Input) Done() {
	i.cond.L.Lock()
	defer i.cond.L.Unlock()
	if !i.closed {
		i.closed = true
		close(i.ch)
		i.cond.Signal()
	}
}

// Routiner is a struct that represents a routiner.
type Routiner struct {
	manager        func(r *Routiner)
	workers        []*Worker
	inputs         *InputCollection
	output         chan string
	quitJob        chan int
	activeWorkers  int
	wg             *sync.WaitGroup // Wait group to track the number of active workers.
	mu             *sync.RWMutex
	closeInputCond *sync.Cond
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

// New is an alias for NewRoutiner.
func New(opts ...option) *Routiner {
	return NewRoutiner(opts...)
}

// NewRoutiner creates a new Routiner.
func NewRoutiner(opts ...option) *Routiner {
	r := &Routiner{
		output:  make(chan string),
		inputs:  newInputCollection(),
		quitJob: make(chan int),
		wg:      new(sync.WaitGroup),
		mu:      new(sync.RWMutex),
	}

	r.closeInputCond = sync.NewCond(&sync.Mutex{})

	r.With(opts...)

	return r
}

// With is a helper function to set options
func (r *Routiner) With(opts ...option) {
	for _, opt := range opts {
		opt.apply(r)
	}
}

func WithOutputChannel(n int) option {
	return newOption("WithOutputChannel", func(r *Routiner) {
		// Set the output channel.
	})
}

// RegisterManager registers a manager function
// in the Routiner
func (r *Routiner) RegisterManager(f func(r *Routiner)) {
	r.manager = f
}

// AddManager is an alias for RegisterManager.
func (r *Routiner) AddManager(f func(r *Routiner)) *Routiner {
	r.RegisterManager(f)

	return r
}

// RegisterWorkers registers a worker function with the
// provided input channel. It creates an input channel
// with the provided name if it doesn't exist.
func (r *Routiner) RegisterWorkers(f func(r *Routiner, m any), input string, quantity int) {
	for i := 0; i < quantity; i++ {
		r.workers = append(r.workers, &Worker{
			f:     f,
			input: input,
		})
	}

	// Create an input channel if it doesn't exist.
	if in := r.inputs.Get(input); in == nil {
		r.inputs.Add(&Input{
			name: input,
			ch:   make(chan any),
			cond: r.closeInputCond,
		})
	}
}

// AddWorker is an alias for RegisterWorkers.
func (r *Routiner) AddWorker(f func(r *Routiner, m any), input string, quantity int) *Routiner {
	r.RegisterWorkers(f, input, quantity)

	return r
}

func (r *Routiner) Start() {
	defer close(r.output)

	if r.inputs.Count() == 0 {
		r.inputs.Add(&Input{
			name: "default",
			ch:   make(chan any),
			cond: r.closeInputCond,
		})
	}

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

func (r *Routiner) Input(name string) *Input {
	return r.inputs.Get(name)
}

func (r *Routiner) Inputs() *InputCollection {
	return r.inputs
}

// Send sends the message to the default channel.
func (r *Routiner) Send(message any) bool {
	return r.SendTo("default", message)
}

// SendTo sends the message to the specified input channel.
func (r *Routiner) SendTo(channel string, message any) bool {
	var in *Input

	if in = r.Input(channel); in == nil || in.closed {
		return false
	}

	in.ch <- message

	return true
}

func (r *Routiner) TotalInputWorkers(input string) int {
	count := 0

	for _, w := range r.workers {
		if w.input == input {
			count += 1
		}
	}

	return count
}

func (r *Routiner) TotalWorkers() int {
	return len(r.workers)
}

// CountInputChannels returns the number of input channels.
func (r *Routiner) CountInputChannels() int {
	return r.inputs.Count()
}

// ActiveWorkers return the number of active workers.
func (r *Routiner) ActiveWorkers() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.activeWorkers
}

// Quit terminates the job and quits immediately.
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
		go func(w *Worker) {
			defer r.recover()()

			// Should be the method of the worker
			r.activateWorker(&wgActiveWorkers)

		inputProcessing:
			for {
				select {
				case message, ok := <-r.inputs.Get(w.input).ch:
					if !ok {
						break inputProcessing
					}

					w.active = true
					w.f(r, message)
					w.active = false
					r.closeInputCond.Signal()
				case <-r.inputs.Get(w.input).quit:
					break inputProcessing
				}
			}

			// *It should probably be a Worker's method
			r.deactivateWorker()
		}(w)
	}
}

func (r *Routiner) startManager() {
	defer r.recover()()

	r.manager(r)

	// Once the manager is finished, it needs to wait for all
	// workers to complete their job.
	r.closeInputCond.L.Lock()
	for {
		ready := r.readyToFinish()
		if ready {
			break
		}

		r.closeInputCond.Wait()
	}
	r.closeInputCond.L.Unlock()

	r.Inputs().Done()

	r.wg.Wait()
	r.quitJob <- 0
}

func (r *Routiner) readyToFinish() bool {
	if r.countOpenInputChannels() == 0 {
		return true
	}

	// TODO: Update the activeWorkers method to count Workers' active property
	activeWorkersCount := 0
	r.mu.RLock()
	for _, w := range r.workers {
		if w.active {
			activeWorkersCount++
		}
	}
	r.mu.RUnlock()

	return activeWorkersCount == 0
}

func (r *Routiner) countOpenInputChannels() int {
	var count int

	for _, i := range r.inputs.inputs {
		if !i.closed {
			count++
		}
	}

	return count
}

// Activate a worker.
func (r *Routiner) activateWorker(wgActiveWorkers *sync.WaitGroup) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.wg.Add(1)
	r.activeWorkers++
	wgActiveWorkers.Done()
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
