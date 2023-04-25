package gclass

import "main/plat"

// Retrieve a list of graded tasks from Google Classroom for a user.
func GradedTasks(creds User, t chan []plat.Task, e chan [][]error) {
	tasksChan := make(chan map[string][]plat.Task)
	go ListTasks(creds, tasksChan, e)
	tasks := <-tasksChan
	t <- tasks["graded"]
}
