package semaphore_test

import (
	"testing"

	"github.com/codebitsorg/routiner/semaphore"
)

func TestSemaphore(t *testing.T) {
	sem := semaphore.NewSemaphore(0)

	if sem.Permits() != 0 {
		t.Errorf("Expected 0 permits, got %d", sem.Permits())
	}

	sem.Release()
	if sem.Permits() != 1 {
		t.Errorf("Expected 1 permit, got %d", sem.Permits())
	}

	sem.Acquire()
	if sem.Permits() != 0 {
		t.Errorf("Expected 0 permits, got %d", sem.Permits())
	}
}
