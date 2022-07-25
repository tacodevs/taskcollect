package gclass

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/classroom/v1"
	"google.golang.org/api/option"
)

func getTask(s *classroom.StudentSubmission, svc *classroom.Service, class string, utask *Task, twg *sync.WaitGroup, gErrChan chan error) {
	defer twg.Done()

	task, err := svc.Courses.CourseWork.Get(
		s.CourseId, s.CourseWorkId,
	).Fields("dueTime", "dueDate", "title", "alternateLink").Do()

	if err != nil {
		gErrChan <- err
		return
	}

	var hours, minutes, seconds, nanoseconds int
	utask.Id = s.CourseId + "-" + s.CourseWorkId + "-" + s.Id

	if task.DueTime == nil {
		hours, minutes, seconds, nanoseconds = 0, 0, 0, 0
	} else {
		hours = int(task.DueTime.Hours)
		minutes = int(task.DueTime.Minutes)
		seconds = int(task.DueTime.Seconds)
		nanoseconds = int(task.DueTime.Nanos)
	}

	if task.DueDate != nil {
		utask.Due = time.Date(
			int(task.DueDate.Year),
			time.Month(task.DueDate.Month),
			int(task.DueDate.Day),
			hours,
			minutes,
			seconds,
			nanoseconds,
			time.UTC,
		)
	}

	if s.State == "TURNED_IN" || s.State == "RETURNED" {
		utask.Submitted = true
	}

	utask.Name = task.Title
	utask.Class = class
	utask.Link = task.AlternateLink
	utask.Platform = "gclass"
}

func getSubmissions(c *classroom.Course, svc *classroom.Service, utasks *[]Task, swg *sync.WaitGroup, gErrChan chan error) {
	defer swg.Done()
	resp, err := svc.Courses.CourseWork.StudentSubmissions.List(
		c.Id, "-",
	).Fields(
		"studentSubmissions/id",
		"studentSubmissions/state",
		"studentSubmissions/courseId",
		"studentSubmissions/courseWorkId",
	).Do()

	if err != nil {
		gErrChan <- err
		return
	}

	submissions := make([]Task, len(resp.StudentSubmissions))
	var twg sync.WaitGroup
	i := 0

	for _, s := range resp.StudentSubmissions {
		twg.Add(1)
		go getTask(s, svc, c.Name, &submissions[i], &twg, gErrChan)
		i++
	}

	twg.Wait()
	*utasks = submissions
}

func ListTasks(creds User, gcid []byte, t chan map[string][]Task, e chan error) {
	ctx := context.Background()

	gauthConfig, err := google.ConfigFromJSON(
		gcid,
		classroom.ClassroomCoursesReadonlyScope,
		classroom.ClassroomStudentSubmissionsMeReadonlyScope,
		classroom.ClassroomCourseworkMeScope,
		classroom.ClassroomCourseworkmaterialsReadonlyScope,
	)

	if err != nil {
		t <- nil
		e <- err
		return
	}

	r := strings.NewReader(creds.Token)
	oauthTok := &oauth2.Token{}
	err = json.NewDecoder(r).Decode(oauthTok)

	if err != nil {
		t <- nil
		e <- err
		return
	}

	client := gauthConfig.Client(context.Background(), oauthTok)

	svc, err := classroom.NewService(
		ctx,
		option.WithHTTPClient(client),
	)

	if err != nil {
		t <- nil
		e <- err
		return
	}

	resp, err := svc.Courses.List().CourseStates("ACTIVE").Fields(
		"courses/name",
		"courses/id",
	).Do()

	if err != nil {
		t <- nil
		e <- err
		return
	}

	if len(resp.Courses) == 0 {
		t <- nil
		e <- nil
		return
	}

	gErrChan := make(chan error)
	utasks := make([][]Task, len(resp.Courses))
	var swg sync.WaitGroup
	i := 0

	for _, c := range resp.Courses {
		swg.Add(1)
		go getSubmissions(c, svc, &utasks[i], &swg, gErrChan)
		i++
	}

	swg.Wait()

	select {
	case gcErr := <-gErrChan:
		t <- nil
		e <- gcErr
		return
	default:
		break
	}

	tasks := map[string][]Task{
		"tasks":     {},
		"notdue":    {},
		"overdue":   {},
		"submitted": {},
	}

	for x := 0; x < len(utasks); x++ {
		for y := 0; y < len(utasks[x]); y++ {
			if utasks[x][y].Submitted {
				tasks["submitted"] = append(
					tasks["submitted"],
					utasks[x][y],
				)
			} else if (utasks[x][y].Due == time.Time{}) {
				tasks["notdue"] = append(
					tasks["notdue"],
					utasks[x][y],
				)
			} else if utasks[x][y].Due.Before(time.Now()) {
				tasks["overdue"] = append(
					tasks["overdue"],
					utasks[x][y],
				)
			} else {
				tasks["tasks"] = append(
					tasks["tasks"],
					utasks[x][y],
				)
			}
		}
	}

	t <- tasks
	e <- nil
}
