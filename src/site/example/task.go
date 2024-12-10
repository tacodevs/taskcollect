package example

import (
	"fmt"
	"io"
	"main/site"
	"mime/multipart"
	"sort"
	"strconv"
	"strings"

	"git.sr.ht/~kvo/go-std/errors"
)

func Task(user site.User, id string) (site.Task, error) {
	tasks := map[string]site.Task{
		bio[0].Id:     bio[0],
		bio[1].Id:     bio[1],
		chem[0].Id:    chem[0],
		chem[1].Id:    chem[1],
		english[0].Id: english[0],
		history[0].Id: history[0],
		history[1].Id: history[1],
		maths[0].Id:   maths[0],
		maths[1].Id:   maths[1],
	}
	task, exists := tasks[id]
	if !exists {
		return task, errors.New(fmt.Sprintf("no task with ID %s exists", id), nil)
	}
	return task, nil
}

func Submit(user site.User, id string) error {
	tasks := map[string]*site.Task{
		bio[0].Id:     &(bio[0]),
		bio[1].Id:     &(bio[1]),
		chem[0].Id:    &(chem[0]),
		chem[1].Id:    &(chem[1]),
		english[0].Id: &(english[0]),
		history[0].Id: &(history[0]),
		history[1].Id: &(history[1]),
		maths[0].Id:   &(maths[0]),
		maths[1].Id:   &(maths[1]),
	}
	_, exists := tasks[id]
	if !exists {
		return errors.New(fmt.Sprintf("no task with ID %s exists", id), nil)
	}
	tasks[id].Submitted = true
	return nil
}

func UploadWork(user site.User, id string, files *multipart.Reader) error {
	tasks := map[string]*site.Task{
		"783663248": &(bio[0]),
		"873468673": &(bio[1]),
		"725987605": &(chem[0]),
		"576252975": &(chem[1]),
		"756438139": &(english[0]),
		"723671061": &(history[0]),
		"547394651": &(history[1]),
		"125726502": &(maths[0]),
		"196728422": &(maths[1]),
	}
	task, exists := tasks[id]
	if !exists {
		return errors.New(fmt.Sprintf("no task with ID %s exists", id), nil)
	}
	sort.SliceStable(task.WorkLinks, func(i, j int) bool {
		id1, _ := strconv.Atoi(strings.TrimPrefix(task.WorkLinks[i][0], "https://example.com/"))
		id2, _ := strconv.Atoi(strings.TrimPrefix(task.WorkLinks[j][0], "https://example.com/"))
		return id1 < id2
	})
	index := len(task.WorkLinks) - 1
	i := 0
	if index != -1 {
		last := task.WorkLinks[index]
		i, _ = strconv.Atoi(strings.TrimPrefix(last[0], "https://example.com/"))
	}
	file, mimeErr := files.NextPart()
	for mimeErr == nil {
		filename := file.FileName()
		link := "https://example.com/" + strconv.Itoa(i+1)
		// NOTE: file implements io.Reader
		tasks[id].WorkLinks = append(tasks[id].WorkLinks, [2]string{link, filename})
		file, mimeErr = files.NextPart()
	}
	err := errors.New(mimeErr.Error(), nil)
	if mimeErr == io.EOF {
		return nil
	} else {
		return errors.New("cannot parse multipart MIME", err)
	}
	return nil
}

func RemoveWork(user site.User, id string, filenames []string) error {
	tasks := map[string]*site.Task{
		"783663248": &(bio[0]),
		"873468673": &(bio[1]),
		"725987605": &(chem[0]),
		"576252975": &(chem[1]),
		"756438139": &(english[0]),
		"723671061": &(history[0]),
		"547394651": &(history[1]),
		"125726502": &(maths[0]),
		"196728422": &(maths[1]),
	}
	task, exists := tasks[id]
	if !exists {
		return errors.New(fmt.Sprintf("no task with ID %s exists", id), nil)
	}
	var cleaned [][2]string
	for _, worklink := range task.WorkLinks {
		matched := false
		for _, filename := range filenames {
			if worklink[1] == filename {
				matched = true
				break
			}
		}
		if !matched {
			cleaned = append(cleaned, worklink)
		}
	}
	tasks[id].WorkLinks = cleaned
	return nil
}
