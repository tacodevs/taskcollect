package gclass

import (
	"main/plat"
)

// Retrieve a list of graded tasks from Google Classroom for a user.
func Graded(creds plat.User, c chan []plat.Task, ok chan error, done *int) {
	var tasks []plat.Task
	var err error

	defer plat.Deliver(c, &tasks, done)
	defer plat.Deliver(ok, &err, done)
	defer plat.Done(done)
}
