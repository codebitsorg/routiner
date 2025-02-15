## Basic usage

- First, initialize a new Routiner:

```golang
import "github.com/codebitsorg/routiner"

func main() {
	r := routiner.Init(routiner.WithWorkers(4))
}
```

- Then define the worker & manager clousers that should do the job.

```golang
manager := func(r *routiner.Routiner) {
    for i := 1; i <= r.Workers(); i++ {
        // Do some work before starting the worker
        
        // It's possible to pass any type to the worker, e.g.:
        // Int: r.Work(1)
        // String: r.Work("Hello World")
        r.Work(inputObject{id: i}) // start the worker
    }
}

worker := func(r *routiner.Routiner, o interface{}) {
    // The second parameter in the worker clouser can be anything.
    // You need to type assert it
    obj := o.(inputObject) // o.(string) | o.(int)
    time.Sleep(time.Second)
    r.Info(fmt.Sprintf("Worker %d", obj.ID))
}
```

- In the example above the inputObject struct is used. Here is the definition:

```golang
type inputObject struct {
	ID int
}
```

- Run the job.

```golang
r.Run(manager, worker)
```