package gclass

// Retrieve a list of graded tasks from Google Classroom for a user.
func GradedTasks(creds User, t chan []Task, e chan error) {
	tasksChan := make(chan map[string][]Task)
	errChan := make(chan error)
	go ListTasks(creds, tasksChan, errChan)
	tasks, err := <-tasksChan, <-errChan

	if err != nil {
		t <- nil
		e <- err
	} else {
		t <- tasks["graded"]
		e <- nil
	}
}