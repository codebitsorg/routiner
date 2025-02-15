package main

import (
	"fmt"

	routiner "github.com/codebitsorg/routiner"
)

type inputObject struct {
	id int
}

func main() {
	//Run() method: wxample with InputObject
	RunUsingInputObject()

	// Run() method: example with manager and workers
	// RunWithManager()

	// Run() method: example with only workers
	// RunWorkers()
}

func RunUsingInputObject() {
	r := routiner.Init(routiner.WithWorkers(4))

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= r.Workers(); i++ {
			r.Work(inputObject{id: i})
		}
	}

	worker := func(r *routiner.Routiner, o interface{}) {
		obj := o.(inputObject)
		r.Info(fmt.Sprintf("Worker %d", obj.id))
	}

	r.Run(manager, worker)

	fmt.Println("All done!")
}

func RunWorkers() {
	r := routiner.Init(routiner.WithWorkers(4))

	worker := func(r *routiner.Routiner, o interface{}) {
		number := o.(int)
		r.Info(fmt.Sprintf("Worker %d", number))
	}

	r.RunWorkers(worker)

	fmt.Println("All done!")
}

func RunWithManager() {
	r := routiner.Init(routiner.WithWorkers(4))

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= r.Workers(); i++ {
			// Do some work before starting the worker
			r.Work(i)
		}
	}

	worker := func(r *routiner.Routiner, i interface{}) {
		number := i.(int)
		r.Info(fmt.Sprintf("Worker %d", number))
	}

	r.Run(manager, worker)

	fmt.Println("All done!")
}
