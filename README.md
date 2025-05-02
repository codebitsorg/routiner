## Basic usage

- First, initialize a new Routiner:

```golang
import "github.com/codebitsorg/routiner"

func main() {
	r := routiner.New()
}
```

- Then define the worker & manager clousers that should do the job.

```golang
manager := func(r *routiner.Routiner) {
    for i := 1; i <= 10; i++ {
        r.Send(i)
    }
}

worker := func(r *routiner.Routiner, m any) {
    time.Sleep(500 * time.Millisecond)
    r.Info(fmt.Sprintf("Worker %d", m.(int)))
}
```

- Run the job.

```golang
r.Run(manager, worker, "default", 3)
```