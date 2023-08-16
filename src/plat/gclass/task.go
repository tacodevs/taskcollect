package gclass

import (
	"mime/multipart"

	"codeberg.org/kvo/std/errors"

	"main/plat"
)

// Get a task from Google Classroom for a user.
func GetTask(creds User, id string) (plat.Task, errors.Error) {
	return plat.Task{}, nil
}

// Submit a Google Classroom task on behalf of a user.
func SubmitTask(creds User, id string) errors.Error {
	return nil
}

// Upload a file as a user's work for a Google Classroom task.
func UploadWork(creds User, id string, files *multipart.Reader) errors.Error {
	return nil
}

// Remove a file (a user's work) from a Google Classroom task.
func RemoveWork(creds User, id string, filenames []string) errors.Error {
	// Remove file submission.
	return nil
}
