TaskCollect - a uniform interface to educational platforms
==========================================================

Motivation
----------

There are a great number of schools around the world that provide educational
resources, homework, methods for communication, etc. through networked
technologies, in particular web-based educational platforms. However, due to
many factors, schools may employ multiple platforms in their education programs,
typically with noticeable overlaps in functionality. In such cases, students
are disadvantaged. They have no choice but to manage the large number of
platforms in use by their school to access their homework and interact with
educational resources. This inconvenience becomes much more problematic with
each new platform incorporated into school use.

The obvious solution to this problem is to provide an intuitive, multiplexed
interface to the functionality provided by educational web platforms. This
document explores the design and processes of our implementation of this
interface, which we have named TaskCollect.

Implementation
--------------

Initially, we considered implementing a graphical user interface (GUI) using
existing graphics modules, but above all we wanted to make sure TaskCollect was
portable and its codebase accessible to all potential contributors. We found
major graphics libraries (Qt, GTK, etc.) to be counter-intuitive to this goal,
because statically linking these libraries with TaskCollect was infeasible, and
their programmer interface/syntax to be unnecessarily complex. We decided to
take an alternative approach and implement TaskCollect as a web interface,
seeing as web standards are well met by major web browsers, and a number of
networking modules provide simple, extensible, and expressive support for the
implementation of web servers.

The Go programming language was a natural choice for this problem domain where
both simplicity of implementation and reliable, portable networking support was
available.

The three main components we needed in TaskCollect are:
  - The user interface
  - The platform multiplexer
  - The platform support layer

We decided to implement these in separate folders of a single TaskCollect
repository so that we could build a single TaskCollect program (for easier
deployment). In the future, we could separate these layers into individual
components that communicate with each other via a lightweight protocol such as
9P. Both the platform support layer and the platform multiplexer could be
implemented as Plan 9 style file servers for a more elegant implementation. A
move in this direction, however, would necessitate integration with a
cross-platform package manager, of which there are no implementations which we
are truly satisfied with.

Our TaskCollect repository uses custom Python scripts such as `build`, `run`,
and `test` for source code management. We originally used the venerable `make`
for these purposes, but found it too restrictive for our production and
deployment needs. While Python is not the most elegant solution, it reasonably
meets our needs. A more expressive scripting language that meets our needs is
yet to be found.

Conventions
-----------

### Returning values

#### Non-concurrent execution conventions

Go conventions for functions executing in non-concurrent environments are clear.
If the current function is called sequentially, it should return one or more
values. If there is a possibility of an error occuring, it should also return an
`error`.

```go
func f(v ...variable) ([]variable, error) {
	vars, err := g(v)
	if err != nil {
		return nil, err
	}
	return vars, nil
}
```

By Go convention, `f` (the caller function) stores any error from `g` (the
function being called) in a variable named `err`. If `err != nil`, the function
returns nil values for all the regular return values, and returns the error
as-is to its own caller. If no error occurs throughout the entire function, the
function returns the regular return values and a `nil` error value.

In our experience, the builtin `error` type is too simplistic and not useful for
debugging purposes. As a result, we have created our own `errors` package, and
our own `errors.Error` type. In TaskCollect, functions should return error
values as `errors.Error` rather than the overly simplistic `error`.
Additionally, any function returning an error to its own caller should briefly
summarise what the function was attempting to do when it received the error.
Thus, the previous example would be written like so in TaskCollect:

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

#### Concurrent execution conventions

Go style guides for writing concurrent code are not specific, so, from our own
development experience, we have decided to outline the following conventions for
writing concurrent code in Go.

If we take the previous example for non-concurrent execution conventions in
TaskCollect, but we need to execute a number of functions concurrently, and
receive both regular and error values from each, we would do it like so:

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

In the beginning, we create two channels. These correspond to what would be the
return values of a function in a non-concurrent environment.

One is called `ch`, through which regular values will be received by the
function `f`. If the function being called only "returns" one regular value,
then the channel for regular values should be named `ch`. If there are multiple
"return values" being sent, then the channels should be named `cx`, where `x` is
a letter that is descriptive of the channel's function. For example, a channel
for resource information would be called `cr`, and a channel for task
information would be called `ct`.

The other channel is the error channel, over which an error value is sent from
instances of `g` to `f`. By our convention, it is called `errs` in the calling
function `f`.

Different goroutines will finish at different times. However, to finish any
looping over values sent through a channel, the channel must be closed. The
cleanest method for this to be done, according to our investigations, is to
provide a shared integer variable between different goroutines. The integer is
decremented for every goroutine launched, and when a goroutine exits, it
increments the shared integer variable. When all goroutines exit, the integer
value becomes 0. It is each goroutine's job to check, upon exiting, if it is the
last running goroutine in its series. That is, if the shared integer value
becomes 0 upon exiting, it should close all the channels it's using.

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

Note the use of `defer`, the functions used, and the order in which they are
called. `defer` statements execute in first-in-last-out order, and input
parameters are evaluated when the `defer` statement is evaluated. At the top of
the `g` function, the variables which will be sent to the channels must be
declared. The variables are then provided to `plat.Deliver` by reference, not by
value, meaning that `plat.Deliver` will execute using the final values of the
provided variables.

When function `g` returns (this can be upon encountering an error, or successful
completion), the deferred statements are run. Firstly, `plat.Done(done)` is run,
incrementing the shared integer value which indicates how many goroutines in the
family of `g` are active. By running `plat.Done(done)`, `g` indicates that it
has finished executing. Next, it runs `plat.Deliver`, which does two things.
Firstly, it "delivers" (sends) the variable provided by reference to the
specified channel. Having delivered the value over the channel, it checks
whether the caller is the last goroutine to deliver values over the channel. If
it is not, it exits. If it isn't, it closes the channel.

Imagine it like this: a household dispatches a number of deliverers to each
retrieve a parcel. The front gate to the house is opened. As each deliverer
leaves the house, they add one to a tally on the gate. Once they have all been
dispatched, the household waits for each deliverer to come back. Each deliverer
comes back at different times. Before each deliverer passes through the gate,
they remove one from the tally on the gate. The last deliverer will notice that,
having removed one from the tally, the tally amounts to 0. Upon noticing this,
they will pass through, and shut the gate, in the same way as the last goroutine
closes the channel.

Back to the function `g` we were talking about earlier...

Note that the parameters to `g` are referred to by different names to `f`. This
is done to prevent name collisions if `g` itself spawns other goroutines. For
instance, `errs` will always refer to the error channel which the current
function *receives* from, and `ok` will always refer to the error channel which
the current function *sends* to. Likewise, `done` is always an input parameter,
and `finished` is always a variable created by the current function. For
channels, there is no clear convention if there are multiple channels for
regular return values (e.g. `cr`, `cs`, `ct`). For such channels, it is up to
the programmer to pick suitable input parameter names in the `cx` format.
However, if there is only one such channel (i.e. `ch`), it should be referred to
as `c` in the function that is being spawned as a goroutine.

Implementation details
----------------------

This document outlines how features are implemented in TaskCollect.

### Public interface

TaskCollect's public interface is stable as of v1.0.0 and is defined below, in
this section of this document.

TaskCollect is a web server that presents a web interface with a navigation bar
and a body for each web page. Users must authenticate to TaskCollect to use this
interface through a TaskCollect login page.

The navigation bar allows switching between different tabs which provide access
to different types of resources necessary for students.

The **Timetable** tab provides the user an HTML timetable showing their lessons
for the school week.

The **Tasks** tab provides the user a list of their tasks separated into at
least the following categories:

  - Active tasks
  - Tasks with no due date
  - Overdue tasks
  - Submitted tasks

Links are provided on the Tasks tab to **individual task pages** which display
information and provide any other necessary functionality for interacting with
individual tasks.

The **Resources** tab provides the user a list of educational resources
available to them, organised by the class the educational resources were
assigned for.

Links are provided on the Resources tab to **individual resource pages** which
display information and provide any other necessary functionality for
interacting with individual resources.

The **Grades** tab provides the user a list of all their graded tasks. Links are
provided on the Grades tab which link to the corresponding individual task
pages.

The navigation bar also includes a logout link which shall log the user out of
an active TaskCollect session.

### Database

TaskCollect uses Redis 7 as its primary user credentials database. TaskCollect
will not accept any other version of Redis.

User credentials are stored using Redis hashes, using the following key
reference:

```
school:<school>:studentID:<ID>
```

where `<school>` is the school name and `<ID>` is the unique student ID that the
school provides. This allows TaskCollect to support multiple schools if needed. 

Currently, the following information on students are stored:
  - `token`: TaskCollect session token
  - `school`: student's school codename
  - `username`: student's username
  - `password`: student's password
  - `daymap`: DayMap session token
  - `gclass`: Google Classroom authentication token

Furthermore, TaskCollect also creates an index of current session tokens, using
the following key reference, where `<token>` is the current session token.

```
studentToken:<token>
```

`studentToken:<token>` also stores a Redis hash and contains the following
information:

  - `studentID`: student's username
  - `school`: student's school name

Currently, TaskCollect's session tokens last for 3 days and by extension, this
data is only stored for 3 days as well.

By using this index rather than `school:<school>:studentID:<ID>`, it allows for
faster look-ups using the TaskCollect token as a reference, rather than the
alternative of looping through each student ID in an attempt to fetch the
session token.

A list of students per school is also stored via `school:<school>:studentList`,
which is a Redis set containing the unique student IDs for each school.
Currently, it is not used for anything, although it may be useful in the future.

## Logging

Due to the limitations of Go's standard library logger, a custom solution has
been implemented. One of the limitations was that the datetime formatting could
not even be done in ISO 8601 format. Hence, in the `logger` package,
`logger/log.go` is a partial reimplementation of Go's built-in logging library.
From this, loggers in `logger/logger.go` are able to be created with different
logging levels.

Logging levels:
  - **FATAL**: A problem is unable to be resolved and as such, the application cannot continue. Results in the termination of the application.
  - **ERROR**: A problem has occurred which prevents normal program execution, although the application may be allowed to continue running.
  - **WARN**: A potential problem has occurred or has been noticed which is worth noting. However, the application is able to continue running.
  - **INFO**: Provides useful information on what the application is doing.
  - **DEBUG**: Used by developers to log diagnostic information which is often more verbose than other logging levels. End-users should not be exposed to debug statements.

Logs are always printed to standard output. However, logging to a file can be
optionally enabled in the `config.json` file by setting `useLogFile` to `true`.
To account for the possibility of opening or writing to the log file resulting
in an error, a fail-safe that stops logging to the file after a certain number
of errors has been put in place. It is referred to as the `logFileFailLimit`
and this is currently set to 20.

Performance
-----------

TaskCollect is a piece of software for which performance is important, but may,
for various reasons, not perform well. Unfortunately, the platforms that
TaskCollect multiplexes have poor performance. TaskCollect, which retrieves data
from these platforms, must unfortunately pass this issue onto the user. It
cannot, unfortunately, magically fix the performance issue, which is on the
source platforms' end.

This document collects statistics regarding various aspects of performance in
TaskCollect.

### Daymap

#### Authentication

Daymap authentication for TaskCollect users of GIHS takes way too long. This is
a result of EdPass, which is a single sign-on platform that handles
authentication on Daymap's behalf. Unfortunately, EdPass has a shockingly
under-performant implementation which:

  - Makes at least ten different HTTP redirects during the authentication flow
  - Requires sending at least seven HTTP requests before any credentials can be sent
  - Requires sending another four HTTP requests after credentials are actually sent
  - Returns several hundred kilobytes of data to the user per authentication attempt (in the browser, this figure is 10-30 times larger)
  - Is ridiculously overcomplicated for an authentication mechanism

The following table shows how slow Daymap authentication actually takes:

| Auth stage no. |   Trial 1 |   Trial 2 |   Trial 3 |   Trial 4 |   Trial 5 |   *Average* |
| :------------: | --------: | --------: | --------: | --------: | --------: | ----------: |
|    Stage 1     |    1124ms |     994ms |    1133ms |     945ms |     930ms |    *1025ms* |
|    Stage 2     |     112ms |     113ms |      67ms |      65ms |      99ms |      *91ms* |
|    Stage 3     |     203ms |     205ms |     528ms |     240ms |     297ms |     *295ms* |
|    Stage 4     |     272ms |     483ms |     427ms |     512ms |     259ms |     *391ms* |
|    Stage 5     |     264ms |     268ms |     308ms |     249ms |     570ms |     *332ms* |
|    Stage 6     |      62ms |      59ms |      61ms |      64ms |     109ms |      *71ms* |
|    Stage 7     |     617ms |     823ms |     652ms |     808ms |     629ms |     *706ms* |
|    Stage 8     |     558ms |     490ms |     516ms |     515ms |     592ms |     *534ms* |
|    Stage 9     |     980ms |     923ms |    1121ms |     974ms |    1021ms |    *1004ms* |
|    Stage 10    |     425ms |     409ms |     425ms |     560ms |     416ms |     *447ms* |
|    Stage 11    |     811ms |     921ms |     911ms |     820ms |     817ms |     *856ms* |
|    **TOTAL**   | **5.43s** | **5.69s** | **6.15s** | **5.76s** | **5.74s** | ***5.75s*** |

###### *(Results collected on 29 April 2023)*
