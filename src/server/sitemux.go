package server

import (
	"mime/multipart"
	"net/http"
	"sort"

	"git.sr.ht/~kvo/libgo/errors"

	"main/logger"
	"main/plat"
	"main/plat/daymap"
	"main/plat/gclass"
)

// Map of school names to the associated platform multiplexers.
var schools = map[string]*plat.Mux{}

// configMux configures the school platform multiplexers.
func configMux() {
	schools["gihs"] = plat.NewMux()
	//schools["gihs"].AddAuth(saml.Auth)
	//schools["gihs"].AddAuth(daymap.Auth)
	//schools["gihs"].AddAuth(gclass.Auth)
	//schools["gihs"].AddClasses(daymap.Classes)
	//schools["gihs"].AddClasses(gclass.Classes)
	//schools["gihs"].AddDueTasks(daymap.DueTasks)
	//schools["gihs"].AddDueTasks(gclass.DueTasks)
	//schools["gihs"].AddEvents(outlook.Events)
	schools["gihs"].AddGraded(daymap.Graded)
	schools["gihs"].AddGraded(gclass.Graded)
	//schools["gihs"].AddItems(daymap.Items)
	//schools["gihs"].AddItems(gclass.Items)
	//schools["gihs"].SetLessons(daymap.Lessons)
	//schools["gihs"].AddMessages(daymap.Messages)
	//schools["gihs"].SetReports(learnprof.Reports)
}

func getLessons(creds plat.User) ([][]plat.Lesson, errors.Error) {
	lessons := [][]plat.Lesson{}

	dmCreds := daymap.User{
		Timezone: creds.Timezone,
		Token:    creds.SiteTokens["daymap"],
	}

	dmLessons, err := daymap.GetLessons(dmCreds)

	for i := 0; i < len(dmLessons); i++ {
		day := []plat.Lesson{}
		for j := 0; j < len(dmLessons[i]); j++ {
			day = append(day, plat.Lesson(dmLessons[i][j]))
		}
		sort.SliceStable(day, func(i, j int) bool {
			return day[i].Start.In(creds.Timezone).Unix() < day[j].Start.In(creds.Timezone).Unix()
		})
		lessons = append(lessons, day)
	}

	return lessons, err
}

func getTasks(creds plat.User) map[string][]plat.Task {
	gcChan := make(chan map[string][]plat.Task)
	gcErrChan := make(chan [][]errors.Error)

	gcCreds := gclass.User{
		Timezone: creds.Timezone,
		Token:    creds.SiteTokens["gclass"],
	}

	finished := -1
	go gclass.ListTasks(gcCreds, gcChan, gcErrChan, &finished)

	dmChan := make(chan map[string][]plat.Task)
	dmErrChan := make(chan [][]errors.Error)
	go daymap.ListTasks(creds, dmChan, dmErrChan)

	t := map[string][]plat.Task{}
	tasks := map[string][]plat.Task{}

	for errs := range gcErrChan {
		for _, i := range errs {
			for _, j := range i {
				if j != nil {
					logger.Error(errors.New("failed to get task list from gclass", j))
				}
			}
		}
	}
	gcTasks := <-gcChan

	dmTasks, dmErrs := <-dmChan, <-dmErrChan
	for _, classErrs := range dmErrs {
		for _, err := range classErrs {
			if err != nil {
				logger.Debug(errors.New("failed to get task list from daymap", err))
			}
		}
	}

	for c, taskList := range gcTasks {
		if c == "graded" {
			continue
		}
		for i := 0; i < len(taskList); i++ {
			t[c] = append(t[c], plat.Task(taskList[i]))
		}
	}

	for c, taskList := range dmTasks {
		if c == "graded" {
			continue
		}
		for i := 0; i < len(taskList); i++ {
			t[c] = append(t[c], plat.Task(taskList[i]))
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

func getResources(creds plat.User) ([]string, map[string][]plat.Resource) {
	gResChan := make(chan []plat.Resource)
	gErrChan := make(chan []errors.Error)

	gcCreds := gclass.User{
		Timezone: creds.Timezone,
		Token:    creds.SiteTokens["gclass"],
	}

	go gclass.ListRes(gcCreds, gResChan, gErrChan)

	dmResChan := make(chan []plat.Resource)
	dmErrChan := make(chan []errors.Error)

	dmCreds := daymap.User{
		Timezone: creds.Timezone,
		Token:    creds.SiteTokens["daymap"],
	}

	go daymap.ListRes(dmCreds, dmResChan, dmErrChan)

	unordered := map[string][]plat.Resource{}

	gcResLinks, errs := <-gResChan, <-gErrChan
	for _, err := range errs {
		if err != nil {
			logger.Debug(errors.New("failed to get list of resources from gclass", err))
		}
	}

	dmResLinks, errs := <-dmResChan, <-dmErrChan
	for _, err := range errs {
		if err != nil {
			logger.Debug(errors.New("failed to get list of resources from daymap", err))
		}
	}

	for _, r := range gcResLinks {
		unordered[r.Class] = append(unordered[r.Class], plat.Resource(r))
	}

	for _, r := range dmResLinks {
		unordered[r.Class] = append(unordered[r.Class], plat.Resource(r))
	}

	resources := map[string][]plat.Resource{}
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
func getTask(platform, taskId string, creds plat.User) (plat.Task, errors.Error) {
	assignment := plat.Task{}
	err := plat.ErrNoPlatform.Here()

	switch platform {
	case "gclass":
		gcCreds := gclass.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["gclass"],
		}
		gcTask, gcErr := gclass.GetTask(gcCreds, taskId)
		assignment = plat.Task(gcTask)
		err = gcErr
	case "daymap":
		dmCreds := daymap.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["daymap"],
		}
		dmTask, dmErr := daymap.GetTask(dmCreds, taskId)
		assignment = plat.Task(dmTask)
		err = dmErr
	}

	return assignment, err
}

// Get a resource from the given platform.
func getResource(platform, resId string, creds plat.User) (plat.Resource, errors.Error) {
	res := plat.Resource{}
	err := plat.ErrNoPlatform.Here()

	switch platform {
	case "gclass":
		gcCreds := gclass.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["gclass"],
		}
		gcRes, gcErr := gclass.GetResource(gcCreds, resId)
		res = plat.Resource(gcRes)
		err = gcErr
	case "daymap":
		dmCreds := daymap.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["daymap"],
		}
		dmRes, dmErr := daymap.GetResource(dmCreds, resId)
		res = plat.Resource(dmRes)
		err = dmErr
	}

	return res, err
}

// Submit task to a given platform.
func submitTask(creds plat.User, platform, taskId string) errors.Error {
	err := plat.ErrNoPlatform.Here()

	switch platform {
	case "gclass":
		gcCreds := gclass.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["gclass"],
		}
		err = gclass.SubmitTask(gcCreds, taskId)
	}

	return err
}

// Return an appropriate reader for a multipart MIME file upload request.
func reqFiles(r *http.Request) (*multipart.Reader, errors.Error) {
	reader, e := r.MultipartReader()
	if e != nil {
		err := errors.New(e.Error(), nil)
		return reader, err
	}
	return reader, nil
}

// Upload work to a given platform.
func uploadWork(creds plat.User, platform string, id string, r *http.Request) errors.Error {
	files, err := reqFiles(r)
	if err != nil {
		return err
	}

	err = plat.ErrNoPlatform.Here()
	switch platform {
	case "gclass":
		gcCreds := gclass.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["gclass"],
		}
		err = gclass.UploadWork(gcCreds, id, files)
	case "daymap":
		dmCreds := daymap.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["daymap"],
		}
		err = daymap.UploadWork(dmCreds, id, files)
	}

	return err
}

// Remove work from a given platform.
func removeWork(creds plat.User, platform, taskId string, filenames []string) errors.Error {
	err := plat.ErrNoPlatform.Here()

	switch platform {
	case "gclass":
		gcCreds := gclass.User{
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
