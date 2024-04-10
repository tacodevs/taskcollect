package gclass

import (
	"mime/multipart"

	"main/plat"
)

// Get a task from Google Classroom for a user.
func GetTask(creds User, id string) (plat.Task, error) {
	return plat.Task{}, nil
}

// Submit a Google Classroom task on behalf of a user.
func SubmitTask(creds User, id string) error {
	return nil
}

// Upload a file as a user's work for a Google Classroom task.
func UploadWork(creds User, id string, files *multipart.Reader) error {
	return nil
}

// Remove a file (a user's work) from a Google Classroom task.
func RemoveWork(creds User, id string, filenames []string) error {
	// Remove file submission.
	return nil
}
