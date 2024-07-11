package server

import (
	"mime/multipart"
	"net/http"
	"sort"

	"git.sr.ht/~kvo/go-std/errors"

	"main/logger"
	"main/site"
	"main/site/daymap"
)

func getLessons(user site.User) ([][]site.Lesson, error) {
	lessons := [][]site.Lesson{}

	dmCreds := daymap.User{
		Timezone: user.Timezone,
		Token:    user.SiteTokens["daymap"],
	}

	dmLessons, err := daymap.GetLessons(dmCreds)

	for i := 0; i < len(dmLessons); i++ {
		day := []site.Lesson{}
		for j := 0; j < len(dmLessons[i]); j++ {
			day = append(day, site.Lesson(dmLessons[i][j]))
		}
		sort.SliceStable(day, func(i, j int) bool {
			return day[i].Start.In(user.Timezone).Unix() < day[j].Start.In(user.Timezone).Unix()
		})
		lessons = append(lessons, day)
	}

	return lessons, err
}

func getTasks(user site.User) map[string][]site.Task {
	dmChan := make(chan map[string][]site.Task)
	dmErrChan := make(chan [][]error)
	go daymap.ListTasks(user, dmChan, dmErrChan)

	t := map[string][]site.Task{}
	tasks := map[string][]site.Task{}

	dmTasks, dmErrs := <-dmChan, <-dmErrChan
	for _, classErrs := range dmErrs {
		for _, err := range classErrs {
			if err != nil {
				logger.Debug(errors.New("failed to get task list from daymap", err))
			}
		}
	}

	for c, taskList := range dmTasks {
		if c == "graded" {
			continue
		}
		for i := 0; i < len(taskList); i++ {
			t[c] = append(t[c], site.Task(taskList[i]))
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

	return tasks
}

func getResources(user site.User) ([]string, map[string][]site.Resource) {
	dmResChan := make(chan []site.Resource)
	dmErrChan := make(chan []error)

	dmCreds := daymap.User{
		Timezone: user.Timezone,
		Token:    user.SiteTokens["daymap"],
	}

	go daymap.ListRes(dmCreds, dmResChan, dmErrChan)

	unordered := map[string][]site.Resource{}

	dmResLinks, errs := <-dmResChan, <-dmErrChan
	for _, err := range errs {
		if err != nil {
			logger.Debug(errors.New("failed to get list of resources from daymap", err))
		}
	}

	for _, r := range dmResLinks {
		unordered[r.Class] = append(unordered[r.Class], site.Resource(r))
	}

	resources := map[string][]site.Resource{}
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

	return classes, resources
}

// Get a task from the given platform.
func getTask(platform, taskId string, user site.User) (site.Task, error) {
	assignment := site.Task{}
	err := errors.Raise(site.ErrNoPlatform)

	switch platform {
	case "daymap":
		dmCreds := daymap.User{
			Timezone: user.Timezone,
			Token:    user.SiteTokens["daymap"],
		}
		dmTask, dmErr := daymap.GetTask(dmCreds, taskId)
		assignment = site.Task(dmTask)
		err = dmErr
	}

	return assignment, err
}

// Get a resource from the given platform.
func getResource(platform, resId string, user site.User) (site.Resource, error) {
	res := site.Resource{}
	err := errors.Raise(site.ErrNoPlatform)

	switch platform {
	case "daymap":
		dmCreds := daymap.User{
			Timezone: user.Timezone,
			Token:    user.SiteTokens["daymap"],
		}
		dmRes, dmErr := daymap.GetResource(dmCreds, resId)
		res = site.Resource(dmRes)
		err = dmErr
	}

	return res, err
}

func submitTask(user site.User, platform, taskId string) error {
	return errors.Raise(site.ErrNoPlatform)
}

// Return an appropriate reader for a multipart MIME file upload request.
func reqFiles(r *http.Request) (*multipart.Reader, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return reader, err
	}
	return reader, nil
}

// Upload work to a given platform.
func uploadWork(user site.User, platform string, id string, r *http.Request) error {
	files, err := reqFiles(r)
	if err != nil {
		return err
	}

	err = errors.Raise(site.ErrNoPlatform)
	switch platform {
	case "daymap":
		dmCreds := daymap.User{
			Timezone: user.Timezone,
			Token:    user.SiteTokens["daymap"],
		}
		err = daymap.UploadWork(dmCreds, id, files)
	}

	return err
}

// Remove work from a given platform.
func removeWork(user site.User, platform, taskId string, filenames []string) error {
	err := errors.Raise(site.ErrNoPlatform)

	switch platform {
	case "daymap":
		dmCreds := daymap.User{
			Timezone: user.Timezone,
			Token:    user.SiteTokens["daymap"],
		}
		err = daymap.RemoveWork(dmCreds, taskId, filenames)
	}

	return err
}
