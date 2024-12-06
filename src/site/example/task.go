package example

import (
	"fmt"
	"main/site"

	"git.sr.ht/~kvo/go-std/errors"
)

func Task(user site.User, id string) (site.Task, error) {
	tasks := map[string]site.Task{
		"783663248": bio[0],
		"873468673": bio[1],
		"725987605": chem[0],
		"576252975": chem[1],
		"756438139": english[0],
		"723671061": history[0],
		"547394651": history[1],
		"125726502": maths[0],
		"196728422": maths[1],
	}
	task, exists := tasks[id]
	if !exists {
		return task, errors.New(fmt.Sprintf("no task with ID %s exists", id), nil)
	}
	return task, nil
}
