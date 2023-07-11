package gclass

import (
	"codeberg.org/kvo/std/errors"

	"main/plat"
)

// Retrieve a list of tasks from Google Classroom for a user.
func ListTasks(creds User, c chan map[string][]plat.Task, ok chan [][]errors.Error, done *int) {
	gcTasks := map[string][]plat.Task{
		"active":    {},
		"notDue":    {},
		"overdue":   {},
		"submitted": {},
		"graded":    {},
	}
	var err errors.Error

	defer plat.Deliver(c, &gcTasks, done)
	defer plat.Deliver(ok, &[][]errors.Error{{err}}, done)
	defer plat.Done(done)
}
