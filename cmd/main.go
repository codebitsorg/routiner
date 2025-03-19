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
	// Run() method: example with InputObject
	// RunUsingInputObject()

	// Run() method: example with manager and workers
	// RunWithManager()

	// Run() method: example with only workers
	// RunWorkers()

	RunThroughChannel()
}

func RunThroughChannel() {
	r := routiner.Init(routiner.WithWorkers(4))

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= 8; i++ {
			r.Send(fmt.Sprintf("%d.png", i))
		}
	}

	worker := func(r *routiner.Routiner, in chan any) {
		for image := range in {
			fmt.Printf("Upload image %s\n", image.(string))
			time.Sleep(1 * time.Second)
		}
	}

	r.RunThroughChannel(manager, worker)
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
