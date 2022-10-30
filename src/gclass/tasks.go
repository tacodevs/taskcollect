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

func getTask(studSub *classroom.StudentSubmission, svc *classroom.Service, class string, task *Task, taskWG *sync.WaitGroup, gErrChan chan error) {
	defer taskWG.Done()

	gcTask, err := svc.Courses.CourseWork.Get(
		studSub.CourseId, studSub.CourseWorkId,
	).Fields("creationTime", "dueTime", "dueDate", "title", "alternateLink").Do()

	if err != nil {
		gErrChan <- err
		return
	}

	var hours, minutes, seconds, nanoseconds int
	task.Id = studSub.CourseId + "-" + studSub.CourseWorkId + "-" + studSub.Id

	posted, err := time.Parse(time.RFC3339Nano, gcTask.CreationTime)

	if err != nil {
		task.Posted = time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)
	} else {
		task.Posted = posted
	}

	if gcTask.DueTime == nil {
		hours, minutes, seconds, nanoseconds = 0, 0, 0, 0
	} else {
		hours = int(gcTask.DueTime.Hours)
		minutes = int(gcTask.DueTime.Minutes)
		seconds = int(gcTask.DueTime.Seconds)
		nanoseconds = int(gcTask.DueTime.Nanos)
	}

	if gcTask.DueDate != nil {
		task.Due = time.Date(
			int(gcTask.DueDate.Year),
			time.Month(gcTask.DueDate.Month),
			int(gcTask.DueDate.Day),
			hours,
			minutes,
			seconds,
			nanoseconds,
			time.UTC,
		)
	}

	if studSub.State == "TURNED_IN" || studSub.State == "RETURNED" {
		task.Submitted = true
	}

	task.Name = gcTask.Title
	task.Class = class
	task.Link = gcTask.AlternateLink
	task.Platform = "gclass"
}

func getSubmissions(c *classroom.Course, svc *classroom.Service, tasks *[]Task, swg *sync.WaitGroup, gErrChan chan error) {
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
	var taskWG sync.WaitGroup
	i := 0

	for _, studSub := range resp.StudentSubmissions {
		taskWG.Add(1)
		go getTask(studSub, svc, c.Name, &submissions[i], &taskWG, gErrChan)
		i++
	}

	taskWG.Wait()
	*tasks = submissions
}

func ListTasks(creds User, t chan map[string][]Task, e chan error) {
	ctx := context.Background()

	gAuthConfig, err := google.ConfigFromJSON(
		creds.ClientID,
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

	client := gAuthConfig.Client(context.Background(), oauthTok)

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
	tasks := make([][]Task, len(resp.Courses))
	var swg sync.WaitGroup
	i := 0

	for _, c := range resp.Courses {
		swg.Add(1)
		go getSubmissions(c, svc, &tasks[i], &swg, gErrChan)
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

	gcTasks := map[string][]Task{
		"active":     {},
		"notDue":    {},
		"overdue":   {},
		"submitted": {},
	}

	for x := 0; x < len(tasks); x++ {
		for y := 0; y < len(tasks[x]); y++ {
			if tasks[x][y].Submitted {
				gcTasks["submitted"] = append(
					gcTasks["submitted"],
					tasks[x][y],
				)
			} else if (tasks[x][y].Due == time.Time{}) {
				gcTasks["notDue"] = append(
					gcTasks["notDue"],
					tasks[x][y],
				)
			} else if tasks[x][y].Due.Before(time.Now()) {
				gcTasks["overdue"] = append(
					gcTasks["overdue"],
					tasks[x][y],
				)
			} else {
				gcTasks["active"] = append(
					gcTasks["active"],
					tasks[x][y],
				)
			}
		}
	}

	t <- gcTasks
	e <- nil
}
