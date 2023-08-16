Conventions
===========

Returning values
----------------

### Non-concurrent execution conventions

Go conventions for functions executing in non-concurrent environments are clear. If the current function is called sequentially, it should return one or more values. If there is a possibility of an error occuring, it should also return an `error`.

```go
func f(v ...variable) ([]variable, error) {
	vars, err := g(v)
	if err != nil {
		return nil, err
	}
	return vars, nil
}
```

By Go convention, `f` (the caller function) stores any error from `g` (the function being called) in a variable named `err`. If `err != nil`, the function returns nil values for all the regular return values, and returns the error as-is to its own caller. If no error occurs throughout the entire function, the function returns the regular return values and a `nil` error value.

In our experience, the builtin `error` type is too simplistic and not useful for debugging purposes. As a result, we have created our own `errors` package, and our own `errors.Error` type. In TaskCollect, functions should return error values as `errors.Error` rather than the overly simplistic `error`. Additionally, any function returning an error to its own caller should briefly summarise what the function was attempting to do when it received the error. Thus, the previous example would be written like so in TaskCollect:

```go
import "codeberg.org/kvo/std/errors"

func f(v ...variable) ([]variable, errors.Error) {
	vars, err := g(v)
	if err != nil {
		return nil, errors.New("error weeding out nil values", err)
	}
	return vars, nil
}
```

### Concurrent execution conventions

Go style guides for writing concurrent code are not specific, so, from our own development experience, we have decided to outline the following conventions for writing concurrent code in Go.

If we take the previous example for non-concurrent execution conventions in TaskCollect, but we need to execute a number of functions concurrently, and receive both regular and error values from each, we would do it like so:

```go
func f(v ...variable) ([]variable, errors.Error) {
	var vars []variable
	ch := make(chan variable)
	errs := make(chan errors.Error)
	var finished int
	for i := 0; i < len(v), i++ {
		finished--
		go g(v[i], ch, errs, &finished)
	}
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}
	for v := range ch {
		vars = append(vars, v)
	}
	return vars, nil
}
```

In the beginning, we create two channels. These correspond to what would be the return values of a function in a non-concurrent environment.

One is called `ch`, through which regular values will be received by the function `f`. If the function being called only "returns" one regular value, then the channel for regular values should be named `ch`. If there are multiple "return values" being sent, then the channels should be named `cx`, where `x` is a letter that is descriptive of the channel's function. For example, a channel for resource information would be called `cr`, and a channel for task information would be called `ct`.

The other channel is the error channel, over which an error value is sent from instances of `g` to `f`. By our convention, it is called `errs` in the calling function `f`.

Different goroutines will finish at different times. However, to finish any looping over values sent through a channel, the channel must be closed. The cleanest method for this to be done, according to our investigations, is to provide a shared integer variable between different goroutines. The integer is decremented for every goroutine launched, and when a goroutine exits, it increments the shared integer variable. When all goroutines exit, the integer value becomes 0. It is each goroutine's job to check, upon exiting, if it is the last running goroutine in its series. That is, if the shared integer value becomes 0 upon exiting, it should close all the channels it's using.

This brings us to the example for the conventions of the function being called:

```go
func g(v variable, c chan variable, ok chan errors.Error, done *int) {
	var x variable
	var err errors.Error

	defer plat.Deliver(c, &x, done)
	defer plat.Deliver(ok, &err, done)
	defer plat.Done(done)

	x, err = h(v)
	if err != nil {
		return
	}
}
```

Note the use of `defer`, the functions used, and the order in which they are called. `defer` statements execute in first-in-last-out order, and input parameters are evaluated when the `defer` statement is evaluated. At the top of the `g` function, the variables which will be sent to the channels must be declared. The variables are then provided to `plat.Deliver` by reference, not by value, meaning that `plat.Deliver` will execute using the final values of the provided variables.

When function `g` returns (this can be upon encountering an error, or successful completion), the deferred statements are run. Firstly, `plat.Done(done)` is run, incrementing the shared integer value which indicates how many goroutines in the family of `g` are active. By running `plat.Done(done)`, `g` indicates that it has finished executing. Next, it runs `plat.Deliver`, which does two things. Firstly, it "delivers" (sends) the variable provided by reference to the specified channel. Having delivered the value over the channel, it checks whether the caller is the last goroutine to deliver values over the channel. If it is not, it exits. If it isn't, it closes the channel.

Imagine it like this: a household dispatches a number of deliverers to each retrieve a parcel. The front gate to the house is opened. As each deliverer leaves the house, they add one to a tally on the gate. Once they have all been dispatched, the household waits for each deliverer to come back. Each deliverer comes back at different times. Before each deliverer passes through the gate, they remove one from the tally on the gate. The last deliverer will notice that, having removed one from the tally, the tally amounts to 0. Upon noticing this, they will pass through, and shut the gate, in the same way as the last goroutine closes the channel.

Back to the function `g` we were talking about earlier...

Note that the parameters to `g` are referred to by different names to `f`. This is done to prevent name collisions if `g` itself spawns other goroutines. For instance, `errs` will always refer to the error channel which the current function *receives* from, and `ok` will always refer to the error channel which the current function *sends* to. Likewise, `done` is always an input parameter, and `finished` is always a variable created by the current function. For channels, there is no clear convention if there are multiple channels for regular return values (e.g. `cr`, `cs`, `ct`). For such channels, it is up to the programmer to pick suitable input parameter names in the `cx` format. However, if there is only one such channel (i.e. `ch`), it should be referred to as `c` in the function that is being spawned as a goroutine.
