package semaphore

import "sync"

type Semaphore struct {
	// Permits remaining on the semaphore
	permits int
	// Condition variable to restrict access to the
	// semaphore when there are not enough permits
	cond *sync.Cond
}

func NewSemaphore(permits int) *Semaphore {
	return &Semaphore{
		permits: permits,
		cond:    sync.NewCond(&sync.Mutex{}),
	}
}

// Acquire takes one permit from the semaphore
// and decrements the permit count
func (s *Semaphore) Acquire() {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()

	for s.permits <= 0 {
		s.cond.Wait()
	}
	s.permits--
}

// Release releases one permit back to the semaphore
// and increments the permit count
func (s *Semaphore) Release() {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()

	s.permits++
	s.cond.Signal()
}

func (s *Semaphore) Permits() int {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()

	return s.permits
}
