package gclass

import (
	"codeberg.org/kvo/std/errors"

	"main/plat"
)

// Retrieve a list of graded tasks from Google Classroom for a user.
func Graded(creds plat.User, c chan []plat.Task, ok chan errors.Error, done *int) {
	var tasks []plat.Task
	var err errors.Error

	defer plat.Deliver(c, &tasks, done)
	defer plat.Deliver(ok, &err, done)
	defer plat.Done(done)

	gcCreds := User{
		ClientID: plat.GAuthID,
		Timezone: creds.Timezone,
		Token:    creds.SiteTokens["gclass"],
	}
	ch := make(chan map[string][]plat.Task)
	errs := make(chan [][]errors.Error)
	finished := -1
	go ListTasks(gcCreds, ch, errs, &finished)
	for e := range errs {
		for _, i := range e {
			for _, j := range i {
				errors.Join(err, j)
			}
		}
	}
	gcTasks := <-ch
	tasks = gcTasks["graded"]
}
