package main

import (
	"fmt"

	routiner "github.com/codebitsorg/routiner"
)

func main() {
	r := routiner.Init(routiner.WithWorkers(4))

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= 4; i++ {
			r.Work(i)
		}
	}

	worker := func(r *routiner.Routiner, i interface{}) {
		iteration := i.(int)
		r.Info(fmt.Sprintf("Worker %d", iteration))
	}

	r.Run(manager, worker)
}
