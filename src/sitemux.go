package main

import (
	"net/http"
	"sort"
	"time"

	"main/daymap"
	"main/errors"
	"main/gclass"
	"main/logger"
)

type task struct {
	Name      string
	Class     string
	Link      string
	Desc      string
	Due       time.Time
	Posted    time.Time
	ResLinks  [][2]string
	Upload    bool
	WorkLinks [][2]string
	Submitted bool
	Grade     string
	Comment   string
	Platform  string
	Id        string
}

type lesson struct {
	Start   time.Time
	End     time.Time
	Class   string
	Room    string
	Teacher string
	Notice  string
}

type resource struct {
	Name     string
	Class    string
	Link     string
	Desc     string
	Posted   time.Time
	ResLinks [][2]string
	Platform string
	Id       string
}

type tcUser struct {
	Timezone   *time.Location
	School     string
	Username   string
	Password   string
	Token      string
	SiteTokens map[string]string
	GAuthID    []byte
}

func getLessons(creds tcUser) ([][]lesson, error) {
	lessons := [][]lesson{}

	dmCreds := daymap.User{
		Timezone: creds.Timezone,
		Token:    creds.SiteTokens["daymap"],
	}

	dmLessons, err := daymap.GetLessons(dmCreds)

	for i := 0; i < len(dmLessons); i++ {
		day := []lesson{}

		for j := 0; j < len(dmLessons[i]); j++ {
			day = append(day, lesson(dmLessons[i][j]))
		}

		lessons = append(lessons, day)
	}

	return lessons, err
}

func getTasks(creds tcUser) (map[string][]task, error) {
	gcChan := make(chan map[string][]gclass.Task)
	gcErr := make(chan error)

	gcCreds := gclass.User{
		ClientID: creds.GAuthID,
		Timezone: creds.Timezone,
		Token:    creds.SiteTokens["gclass"],
	}

	go gclass.ListTasks(gcCreds, gcChan, gcErr)

	dmChan := make(chan map[string][]daymap.Task)
	dmErr := make(chan error)

	dmCreds := daymap.User{
		Timezone: creds.Timezone,
		Token:    creds.SiteTokens["daymap"],
	}

	go daymap.ListTasks(dmCreds, dmChan, dmErr)

	t := map[string][]task{}
	tasks := map[string][]task{}

	gcTasks, err := <-gcChan, <-gcErr
	if err != nil {
		newErr := errors.NewError("main: getTasks", "failed to get list of tasks from gclass", err)
		logger.Error(newErr)
	}

	dmTasks, err := <-dmChan, <-dmErr
	if err != nil {
		newErr := errors.NewError("main: getTasks", "failed to get list of tasks from daymap", err)
		logger.Error(newErr)
	}

	for c, taskList := range gcTasks {
		for i := 0; i < len(taskList); i++ {
			t[c] = append(t[c], task(taskList[i]))
		}
	}

	for c, taskList := range dmTasks {
		for i := 0; i < len(taskList); i++ {
			t[c] = append(t[c], task(taskList[i]))
		}
	}

	for c, taskList := range t {
		times := map[int]int{}
		taskIndexes := []int{}

		for i := 0; i < len(taskList); i++ {
			var time int

			if c == "active" || c == "overdue" {
				time = int(taskList[i].Due.UTC().Unix())
			} else {
				time = int(taskList[i].Posted.UTC().Unix())
			}

			times[i] = time
			taskIndexes = append(taskIndexes, i)
		}

		if c == "active" {
			sort.SliceStable(taskIndexes, func(i, j int) bool {
				return times[taskIndexes[i]] < times[taskIndexes[j]]
			})
		} else {
			sort.SliceStable(taskIndexes, func(i, j int) bool {
				return times[taskIndexes[i]] > times[taskIndexes[j]]
			})
		}

		for _, x := range taskIndexes {
			tasks[c] = append(tasks[c], taskList[x])
		}
	}

	return tasks, err
}

func getResources(creds tcUser) ([]string, map[string][]resource, error) {
	gResChan := make(chan []gclass.Resource)
	gErrChan := make(chan error)

	gcCreds := gclass.User{
		ClientID: creds.GAuthID,
		Timezone: creds.Timezone,
		Token:    creds.SiteTokens["gclass"],
	}

	go gclass.ListRes(gcCreds, gResChan, gErrChan)

	dmResChan := make(chan []daymap.Resource)
	dmErrChan := make(chan error)

	dmCreds := daymap.User{
		Timezone: creds.Timezone,
		Token:    creds.SiteTokens["daymap"],
	}

	go daymap.ListRes(dmCreds, dmResChan, dmErrChan)

	unordered := map[string][]resource{}

	gcResLinks, err := <-gResChan, <-gErrChan
	if err != nil {
		newErr := errors.NewError("main: getResources", "failed to get list of resources from gclass", err)
		logger.Error(newErr)
	}

	dmResLinks, err := <-dmResChan, <-dmErrChan
	if err != nil {
		newErr := errors.NewError("main: getResources", "failed to get list of resources from daymap", err)
		logger.Error(newErr)
	}

	for _, r := range gcResLinks {
		unordered[r.Class] = append(unordered[r.Class], resource(r))
	}

	for _, r := range dmResLinks {
		unordered[r.Class] = append(unordered[r.Class], resource(r))
	}

	resources := map[string][]resource{}
	classes := []string{}

	for c := range unordered {
		classes = append(classes, c)
	}

	sort.Strings(classes)

	for c, resList := range unordered {
		times := map[int]int{}
		resIndexes := []int{}

		for i, r := range resList {
			posted := int(r.Posted.UTC().Unix())
			times[i] = posted
			resIndexes = append(resIndexes, i)
		}

		sort.SliceStable(resIndexes, func(i, j int) bool {
			return times[resIndexes[i]] > times[resIndexes[j]]
		})

		for _, x := range resIndexes {
			resources[c] = append(resources[c], resList[x])
		}
	}

	return classes, resources, err
}

// Get a task from the given platform.
func getTask(platform, taskId string, creds tcUser) (task, error) {
	assignment := task{}
	err := errNoPlatform.AsError()

	switch platform {
	case "gclass":
		gcCreds := gclass.User{
			ClientID: creds.GAuthID,
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["gclass"],
		}
		gcTask, gcErr := gclass.GetTask(gcCreds, taskId)
		assignment = task(gcTask)
		err = gcErr
	case "daymap":
		dmCreds := daymap.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["daymap"],
		}
		dmTask, dmErr := daymap.GetTask(dmCreds, taskId)
		assignment = task(dmTask)
		err = dmErr
	}

	return assignment, err
}

// Submit task to a given platform.
func submitTask(creds tcUser, platform, taskId string) error {
	err := errNoPlatform.AsError()

	switch platform {
	case "gclass":
		gcCreds := gclass.User{
			ClientID: creds.GAuthID,
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["gclass"],
		}
		err = gclass.SubmitTask(gcCreds, taskId)
	}

	return err
}

// Upload work to a given platform.
func uploadWork(creds tcUser, platform string, id string, r *http.Request) error {
	err := errNoPlatform.AsError()

	switch platform {
	case "gclass":
		gcCreds := gclass.User{
			ClientID: creds.GAuthID,
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["gclass"],
		}
		err = gclass.UploadWork(gcCreds, id, r)
	case "daymap":
		dmCreds := daymap.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["daymap"],
		}
		err = daymap.UploadWork(dmCreds, id, r)
	}

	return err
}

// Remove work from a given platform.
func removeWork(creds tcUser, platform, taskId string, filenames []string) error {
	err := errNoPlatform.AsError()

	switch platform {
	case "gclass":
		gcCreds := gclass.User{
			ClientID: creds.GAuthID,
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["gclass"],
		}
		err = gclass.RemoveWork(gcCreds, taskId, filenames)
	case "daymap":
		dmCreds := daymap.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["daymap"],
		}
		err = daymap.RemoveWork(dmCreds, taskId, filenames)
	}

	return err
}
