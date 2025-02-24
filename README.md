## Basic usage

- First, initialize a new Routiner:

```golang
import "github.com/codebitsorg/routiner"

func main() {
	r := routiner.Init(routiner.WithWorkers(3))
}
```

- Then define the worker & manager clousers that should do the job.

```golang
manager := func(r *routiner.Routiner) {
    for i := 1; i <= r.Workers(); i++ {
        r.Send(i)
    }
}

worker := func(r *routiner.Routiner, o any) {
    // The second parameter in the worker clouser can be anything.
    // You need to type assert it
    r.Info(fmt.Sprintf("Worker %d", o.(int)))
    time.Sleep(1 * time.Second)
}
```

- Run the job.

```golang
r.Run(manager, worker)
```