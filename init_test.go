package routiner_test

import (
	"testing"

	"github.com/codebitsorg/routiner"
)

func TestInit_InitializingRoutinerAddsOneWorkerByDefault(t *testing.T) {
	r := routiner.Init()
	if r.Workers() != 1 {
		t.Errorf("Workers should be 1, but got %d", r.Workers())
	}
}

func TestInit_CanOptionalySetWorkers(t *testing.T) {
	r := routiner.Init(routiner.WithWorkers(4))
	if r.Workers() != 4 {
		t.Errorf("Workers should be 4, but got %d", r.Workers())
	}
}
