## Basic usage

- First define a Routiner object.

```golang
import "github.com/codebitsorg/routiner"

func main() {
	r := routiner.Init()
    r.Workers = 4
}
```

- Then define manager & worker clousers that should do the job.

```golang
manager := func(r *routiner.Routiner) {
    for i := 1; i <= 4; i++ {
        r.Work(inputObject{ID: i})
    }
}

worker := func(r *routiner.Routiner, o interface{}) {
    obj := o.(inputObject)
    time.Sleep(time.Second)
    r.Info(fmt.Sprintf("Worker %d", obj.ID))
}
```

- Because the second parameter in the worker clouser is designed to accept an object, don't forget to create it also.

```golang
type inputObject struct {
	ID int
}
```

- Run the job.

```golang
r.Run(manager, worker)

fmt.Println("The task has been finished.")
```

- Also note that it's possible to pass simple types: **int**, **string** etc. Then, in your worker clouser you just need to assert that type:

```golang
worker := func(r *routiner.Routiner, o interface{}) {
    obj := o.(string) // o.(int)
    ...
```