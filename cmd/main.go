package main

import (
	"fmt"
	"time"

	routiner "github.com/codebitsorg/routiner"
)

func main() {
	// Basic()
	MultipleInputs()
}

func Basic() {
	r := routiner.New()

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= 10; i++ {
			r.Send(i)
		}
	}

	worker := func(r *routiner.Routiner, m any) {
		time.Sleep(500 * time.Millisecond)
		r.Info(fmt.Sprintf("Number %d", m.(int)))
	}

	err := r.Run(manager, worker, "default", 3)
	if err != nil {
		fmt.Println(err)
	}
}

func MultipleInputs() {
	r := routiner.New()

	manager := func(r *routiner.Routiner) {
		for i := 1; i <= 10; i++ {
			r.Send(i)
		}
	}

	workerForAllNumbers := func(r *routiner.Routiner, m any) {
		n := m.(int)
		r.Info(fmt.Sprintf("All nubmers: #%d", n))
		if n%2 == 0 {
			r.SendTo("even", m)
		}
	}

	workerForEvenNumbers := func(r *routiner.Routiner, m any) {
		r.Info(fmt.Sprintf("Even number: #%d", m.(int)))
	}

	r.AddManager(manager).
		AddWorker(workerForAllNumbers, "default", 3).
		AddWorker(workerForEvenNumbers, "even", 2).
		Start()
}
