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
