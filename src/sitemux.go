package main

import (
	"io"
	"log"
	"main/daymap"
	"main/gclass"
	"sort"
	"time"
)

type task struct {
	Name      string
	Class     string
	Link      string
	Desc      string
	Due       time.Time
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

func resListContains(resList [][2]string, resLink [2]string) bool {
	for i := 0; i < len(resList); i++ {
		if resList[i] == resLink {
			return true
		}
	}

	return false
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

func getTasks(creds tcUser, gcid []byte) (map[string][]task, error) {
	gcChan := make(chan map[string][]gclass.Task)
	gcErr := make(chan error)

	gcCreds := gclass.User{
		Timezone: creds.Timezone,
		Token:    creds.SiteTokens["gclass"],
	}

	go gclass.ListTasks(gcCreds, gcid, gcChan, gcErr)

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
		log.Println(err)
	}

	dmTasks, err := <-dmChan, <-dmErr

	if err != nil {
		log.Println(err)
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
		if c == "tasks" {
			times := []int{}
			taskIdx := map[int]int{}

			for i := 0; i < len(taskList); i++ {
				time := int(taskList[i].Due.Unix())
				times = append(times, time)
				taskIdx[time] = i
			}

			sort.Ints(times)

			for i := 0; i < len(times); i++ {
				x := taskIdx[times[i]]
				tasks[c] = append(tasks[c], taskList[x])
			}
		} else {
			names := []string{}
			taskIdx := map[string]int{}

			for i := 0; i < len(taskList); i++ {
				names = append(names, taskList[i].Name)
				taskIdx[taskList[i].Name] = i
			}

			sort.Strings(names)

			for i := 0; i < len(names); i++ {
				x := taskIdx[names[i]]
				tasks[c] = append(tasks[c], taskList[x])
			}
		}
	}

	return tasks, err
}

func getResLinks(creds tcUser, gcid []byte) ([]string, map[string][][2]string, error) {
	gResChan := make(chan map[string][][2]string)
	gErrChan := make(chan error)

	gcCreds := gclass.User{
		Timezone: creds.Timezone,
		Token:    creds.SiteTokens["gclass"],
	}

	go gclass.ResLinks(gcCreds, gcid, gResChan, gErrChan)

	dmResChan := make(chan map[string][][2]string)
	dmErrChan := make(chan error)

	dmCreds := daymap.User{
		Timezone: creds.Timezone,
		Token:    creds.SiteTokens["daymap"],
	}

	go daymap.ResLinks(dmCreds, dmResChan, dmErrChan)

	r := map[string][][2]string{}
	gcResLinks, err := <-gResChan, <-gErrChan

	if err != nil {
		log.Println(err)
	}

	dmResLinks, err := <-dmResChan, <-dmErrChan

	if err != nil {
		log.Println(err)
	}

	for c, resList := range gcResLinks {
		for i := 0; i < len(resList); i++ {
			if !resListContains(r[c], resList[i]) {
				r[c] = append(r[c], resList[i])
			}
		}
	}

	for c, resList := range dmResLinks {
		for i := 0; i < len(resList); i++ {
			if !resListContains(r[c], resList[i]) {
				r[c] = append(r[c], resList[i])
			}
		}
	}

	resLinks := map[string][][2]string{}
	classes := []string{}

	for c := range r {
		classes = append(classes, c)
	}

	sort.Strings(classes)

	for c, rls := range r {
		res := []string{}
		resIdx := map[string]int{}

		for i := 0; i < len(rls); i++ {
			res = append(res, rls[i][1])
			resIdx[rls[i][1]] = i
		}

		sort.Strings(res)

		for i := 0; i < len(res); i++ {
			linkIdx := resIdx[res[i]]

			resLinks[c] = append(
				resLinks[c],
				[2]string{rls[linkIdx][0], res[i]},
			)
		}
	}

	return classes, resLinks, err
}

func getTask(platform, taskId string, creds tcUser, gcid []byte) (task, error) {
	assignment := task{}
	err := errNoPlatform

	switch platform {
	case "gclass":
		gcCreds := gclass.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["gclass"],
		}
		gcTask, gcErr := gclass.GetTask(gcCreds, gcid, taskId)
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

func submitTask(creds tcUser, platform, taskId string, gcid []byte) error {
	err := errNoPlatform

	switch platform {
	case "gclass":
		gcCreds := gclass.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["gclass"],
		}
		err = gclass.SubmitTask(gcCreds, gcid, taskId)
	}

	return err
}

func uploadWork(creds tcUser, platform, id, filename string, f *io.Reader, gcid []byte) error {
	err := errNoPlatform

	switch platform {
	case "gclass":
		gcCreds := gclass.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["gclass"],
		}
		err = gclass.UploadWork(gcCreds, gcid, id, filename, f)
	case "daymap":
		dmCreds := daymap.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["daymap"],
		}
		err = daymap.UploadWork(dmCreds, id, filename, f)
	}

	return err
}

func removeWork(creds tcUser, platform, taskId string, filenames []string, gcid []byte) error {
	err := errNoPlatform

	switch platform {
	case "gclass":
		gcCreds := gclass.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["gclass"],
		}
		err = gclass.RemoveWork(gcCreds, gcid, taskId, filenames)
	case "daymap":
		dmCreds := daymap.User{
			Timezone: creds.Timezone,
			Token:    creds.SiteTokens["daymap"],
		}
		err = daymap.RemoveWork(dmCreds, taskId, filenames)
	}

	return err
}
