package gclass

import (
	"main/plat"
)

// Retrieve a list of tasks from Google Classroom for a user.
func ListTasks(creds User, c chan map[string][]plat.Task, ok chan [][]error, done *int) {
	gcTasks := map[string][]plat.Task{
		"active":    {},
		"notDue":    {},
		"overdue":   {},
		"submitted": {},
		"graded":    {},
	}
	var err error

	defer plat.Deliver(c, &gcTasks, done)
	defer plat.Deliver(ok, &[][]error{{err}}, done)
	defer plat.Done(done)
}
