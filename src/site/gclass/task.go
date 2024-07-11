package gclass

import (
	"mime/multipart"

	"main/site"
)

func GetTask(creds User, id string) (site.Task, error) {
	return site.Task{}, nil
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
