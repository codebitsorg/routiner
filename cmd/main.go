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
	RunSimple()
	// RunUsingInputObject()
}

func RunSimple() {
	r := routiner.Init(routiner.WithWorkers(3))

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= 8; i++ {
			r.Send(fmt.Sprintf("%d.png", i))
		}
	}

	worker := func(r *routiner.Routiner, o any) {
		fmt.Printf("Upload image %s\n", o.(string))
		time.Sleep(1 * time.Second)
	}

	r.Run(manager, worker)
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
