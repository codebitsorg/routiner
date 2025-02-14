package routiner

import (
	"sync"
)

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

func (r *Routiner) With(opts ...option) {
	for _, opt := range opts {
		opt(r)
	}
}

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
