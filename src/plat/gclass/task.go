package gclass

import (
	"mime/multipart"

	"main/plat"
)

func GetTask(creds User, id string) (plat.Task, error) {
	return plat.Task{}, nil
}

func SubmitTask(creds User, id string) error {
	return nil
}

func UploadWork(creds User, id string, files *multipart.Reader) error {
	return nil
}

func RemoveWork(creds User, id string, filenames []string) error {
	return nil
}
