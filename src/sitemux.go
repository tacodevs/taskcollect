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
	Name string
	Class string
	Link string
	Desc string
	Due time.Time
	Reslinks [][2]string
	Upload bool
	Worklinks [][2]string
	Submitted bool
	Grade string
	Comment string
	Platform string
	Id string
}

type lesson struct {
	Start time.Time
	End time.Time
	Class string
	Room string
	Teacher string
	Notice string
}

func reslistContains(reslist [][2]string, reslink [2]string) bool {
	for i := 0; i < len(reslist); i++ {
		if reslist[i] == reslink {
			return true
		}
	}

	return false
}

func getLessons(creds user) ([][]lesson, error) {
	lessons := [][]lesson{}

	dmcreds := daymap.User{
		Timezone: creds.Timezone,
		Token: creds.SiteTokens["daymap"],
	}

	dmlessons, err := daymap.GetLessons(dmcreds)

	for i := 0; i < len(dmlessons); i++ {
		day := []lesson{}

		for j := 0; j < len(dmlessons[i]); j++ {
			day = append(day, lesson(dmlessons[i][j]))
		}

		lessons = append(lessons, day)
	}

	return lessons, err
}

func getTasks(creds user, gcid []byte) (map[string][]task, error) {
	gcchan := make(chan map[string][]gclass.Task)
	gcerr := make(chan error)

	gccreds := gclass.User{
		Timezone: creds.Timezone,
		Token: creds.SiteTokens["gclass"],
	}

	go gclass.ListTasks(gccreds, gcid, gcchan, gcerr)

	dmchan := make(chan map[string][]daymap.Task)
	dmerr := make(chan error)

	dmcreds := daymap.User{
		Timezone: creds.Timezone,
		Token: creds.SiteTokens["daymap"],
	}

	go daymap.ListTasks(dmcreds, dmchan, dmerr)

	t := map[string][]task{}
	tasks := map[string][]task{}
	gctasks, err := <-gcchan, <-gcerr

	if err != nil {
		log.Println(err)
	}

	dmtasks, err := <-dmchan, <-dmerr

	if err != nil {
		log.Println(err)
	}

	for c, tasklist := range gctasks {
		for i := 0; i < len(tasklist); i++ {
			t[c] = append(t[c], task(tasklist[i]))
		}
	}

	for c, tasklist := range dmtasks {
		for i := 0; i < len(tasklist); i++ {
			t[c] = append(t[c], task(tasklist[i]))
		}
	}

	for c, tasklist := range t {
		if c == "tasks" {
			times := []int{}
			tidx := map[int]int{}

			for i := 0; i < len(tasklist); i++ {
				time := int(tasklist[i].Due.Unix())
				times = append(times, time)
				tidx[time] = i
			}

			sort.Ints(times)

			for i := 0; i < len(times); i++ {
				x := tidx[times[i]]
				tasks[c] = append(tasks[c], tasklist[x])
			}
		} else {
			names := []string{}
			tidx := map[string]int{}

			for i := 0; i < len(tasklist); i++ {
				names = append(names, tasklist[i].Name)
				tidx[tasklist[i].Name] = i
			}

			sort.Strings(names)

			for i := 0; i < len(names); i++ {
				x := tidx[names[i]]
				tasks[c] = append(tasks[c], tasklist[x])
			}
		}
	}

	return tasks, err
}

func getReslinks(creds user, gcid []byte) ([]string, map[string][][2]string, error) {
	grchan := make(chan map[string][][2]string)
	gechan := make(chan error)

	gccreds := gclass.User{
		Timezone: creds.Timezone,
		Token: creds.SiteTokens["gclass"],
	}

	go gclass.Reslinks(gccreds, gcid, grchan, gechan)

	dmrchan := make(chan map[string][][2]string)
	dmechan := make(chan error)

	dmcreds := daymap.User{
		Timezone: creds.Timezone,
		Token: creds.SiteTokens["daymap"],
	}

	go daymap.Reslinks(dmcreds, dmrchan, dmechan)

	r := map[string][][2]string{}
	gcreslinks, err := <-grchan, <-gechan

	if err != nil {
		log.Println(err)
	}

	dmreslinks, err := <-dmrchan, <-dmechan

	if err != nil {
		log.Println(err)
	}

	for c, reslist := range gcreslinks {
		for i := 0; i < len(reslist); i++ {
			if !reslistContains(r[c], reslist[i]) {
				r[c] = append(r[c], reslist[i])
			}
		}
	}

	for c, reslist := range dmreslinks {
		for i := 0; i < len(reslist); i++ {
			if !reslistContains(r[c], reslist[i]) {
				r[c] = append(r[c], reslist[i])
			}
		}
	}

	reslinks := map[string][][2]string{}
	classes := []string{}

	for c := range r {
		classes = append(classes, c)
	}

	sort.Strings(classes)

	for c, rls := range r {
		res := []string{}
		residx := map[string]int{}

		for i := 0; i < len(rls); i++ {
			res = append(res, rls[i][1])
			residx[rls[i][1]] = i
		}

		sort.Strings(res)

		for i := 0; i < len(res); i++ {
			linkIdx := residx[res[i]]

			reslinks[c] = append(
				reslinks[c],
				[2]string{rls[linkIdx][0], res[i]},
			)
		}
	}

	return classes, reslinks, err
}

func getTask(platform, taskId string, creds user, gcid []byte) (task, error) {
	assignment := task{}
	err := errNoPlatform

	switch platform {
	case "gclass":
		gccreds := gclass.User{
			Timezone: creds.Timezone,
			Token: creds.SiteTokens["gclass"],
		}

		gctask, gcerr := gclass.GetTask(gccreds, gcid, taskId)
		assignment = task(gctask)
		err = gcerr
	case "daymap":
		dmcreds := daymap.User{
			Timezone: creds.Timezone,
			Token: creds.SiteTokens["daymap"],
		}

		dmtask, dmerr := daymap.GetTask(dmcreds, taskId)
		assignment = task(dmtask)
		err = dmerr
	}

	return assignment, err
}

func submitTask(creds user, platform, taskId string, gcid []byte) error {
	err := errNoPlatform

	switch platform {
	case "gclass":
		gccreds := gclass.User{
			Timezone: creds.Timezone,
			Token: creds.SiteTokens["gclass"],
		}

		err = gclass.SubmitTask(gccreds, gcid, taskId)
	}

	return err
}

func uploadWork(creds user, platform, id, filename string, f *io.Reader, gcid []byte) error {
	err := errNoPlatform

	switch platform {
	case "gclass":
		gccreds := gclass.User{
			Timezone: creds.Timezone,
			Token: creds.SiteTokens["gclass"],
		}

		err = gclass.UploadWork(gccreds, gcid, id, filename, f)
	case "daymap":
		dmcreds := daymap.User{
			Timezone: creds.Timezone,
			Token: creds.SiteTokens["daymap"],
		}

		err = daymap.UploadWork(dmcreds, id, filename, f)
	}

	return err
}

func removeWork(creds user, platform, taskId string, filenames []string, gcid []byte) error {
	err := errNoPlatform

	switch platform {
	case "gclass":
		gccreds := gclass.User{
			Timezone: creds.Timezone,
			Token: creds.SiteTokens["gclass"],
		}

		err = gclass.RemoveWork(gccreds, gcid, taskId, filenames)
	case "daymap":
		dmcreds := daymap.User{
			Timezone: creds.Timezone,
			Token: creds.SiteTokens["daymap"],
		}

		err = daymap.RemoveWork(dmcreds, taskId, filenames)
	}

	return err
}
