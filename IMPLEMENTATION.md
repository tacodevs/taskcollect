# Implementation details
This document outlines how features are implemented in TaskCollect.

## Public Interface

TaskCollect's public interface is stable as of v1.0.0 and is defined below, in this section of this document.

TaskCollect is a web server that presents a web interface with a navigation bar and a body for each web page. Users must authenticate to TaskCollect to use this interface through a TaskCollect login page.

The navigation bar allows switching between different tabs which provide access to different types of resources necessary for students.

The **Timetable** tab provides the user an HTML timetable showing their lessons for the school week.

The **Tasks** tab provides the user a list of their tasks separated into at least the following categories:
* Active tasks
* Tasks with no due date
* Overdue tasks
* Submitted tasks

Links are provided on the Tasks tab to **individual task pages** which display information and provide any other necessary functionality for interacting with individual tasks.

The **Resources** tab provides the user a list of educational resources available to them, organised by the class the educational resources were assigned for.

Links are provided on the Resources tab to **individual resource pages** which display information and provide any other necessary functionality for interacting with individual resources.

The **Grades** tab provides the user a list of all their graded tasks. Links are provided on the Grades tab which link to the corresponding individual task pages.

The navigation bar also includes a logout link which shall log the user out of an active TaskCollect session.

## Database

TaskCollect uses Redis 7 as its primary user credentials database. TaskCollect will not accept any other version of Redis.

User credentials are stored using Redis hashes, using the following key reference:

```
school:<school>:studentID:<ID>
```

where `<school>` is the school name and `<ID>` is the unique student ID that the school provides. This allows TaskCollect to support multiple schools if needed. 

Currently, the following information on students are stored:
- `token`: TaskCollect session token
- `school`: student's school codename
- `username`: student's username
- `password`: student's password
- `daymap`: DayMap session token
- `gclass`: Google Classroom authentication token

Furthermore, TaskCollect also creates an index of current session tokens, using the following key reference, where `<token>` is the current session token.

```
studentToken:<token>
```

`studentToken:<token>` also stores a Redis hash and contains the following information:
- `studentID`: student's username
- `school`: student's school name

Currently, TaskCollect's session tokens last for 3 days and by extension, this data is only stored for 3 days as well. 

By using this index rather than `school:<school>:studentID:<ID>`, it allows for faster look-ups using the TaskCollect token as a reference, rather than the alternative of looping through each student ID in an attempt to fetch the session token.

A list of students per school is also stored via `school:<school>:studentList`, which is a Redis set containing the unique student IDs for each school. Currently, it is not used for anything, although it may be useful in the future.

## Logging

Due to the limitations of Go's standard library logger, a custom solution has been implemented. One of the limitations was that the datetime formatting could not even be done in ISO 8601 format. Hence, in the `logger` package, `logger/log.go` is a partial reimplementation of Go's built-in logging library. From this, loggers in `logger/logger.go` are able to be created with different logging levels.

Logging levels:
- `INFO`
- `DEBUG`
- `WARN`
- `ERROR`
- `FATAL`

## Error Handling

Error logging is another important aspect in ensuring the TaskCollect server runs smoothly and if an error were to occur, that it will be reported. The `errors` package adds new functionality to suit the needs of the projects such as a custom error wrapper that has the ability to provide more context about where the error originated, what the error type is, and allow for better management of tracing errors.

To prevent the need for two error library imports, Go's standard error library has been implemented right into our own library. 
