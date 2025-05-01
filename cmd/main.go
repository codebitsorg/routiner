package main

import (
	"fmt"
	"time"

	routiner "github.com/codebitsorg/routiner"
)

type inputObject struct {
	id int
}

func main() {
	BasicUsage()
}

func BasicUsage() {
	r := routiner.New()

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= 10; i++ {
			r.Send(i)
		}
	}

	worker := func(r *routiner.Routiner, m any) {
		time.Sleep(500 * time.Millisecond)
		r.Info(fmt.Sprintf("Worker %d", m.(int)))
	}

	r.Run(manager, worker, "default", 3)
}
