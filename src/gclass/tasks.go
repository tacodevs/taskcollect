package gclass

import (
	"image/color"
	"sync"
	"time"

	"google.golang.org/api/classroom/v1"

	"main/errors"
	"main/plat"
)

// Retrieve Google Classroom task information using a workload submission point ID.
func getTask(
	studSub *classroom.StudentSubmission, svc *classroom.Service, class string,
	task *plat.Task, taskWG *sync.WaitGroup, gErrChan chan error,
) {
	defer taskWG.Done()

	gcTask, err := svc.Courses.CourseWork.Get(
		studSub.CourseId, studSub.CourseWorkId,
	).Fields(
		"alternateLink",
		"creationTime",
		"dueTime",
		"dueDate",
		"maxPoints",
		"title",
	).Do()

	if err != nil {
		newErr := errors.NewError("gclass.getTask", "failed to get coursework", err)
		gErrChan <- newErr
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

	if studSub.AssignedGrade != 0 && gcTask.MaxPoints != 0 {
		percent := studSub.AssignedGrade / gcTask.MaxPoints * 100
		task.Result.Grade = "-"
		task.Result.Mark = percent
		if percent < 50 {
			task.Result.Color = color.RGBA{0xc9, 0x16, 0x14, 0xff} //RED
		} else if (50 <= percent) && (percent < 70) {
			task.Result.Color = color.RGBA{0xd9, 0x6b, 0x0a, 0xff} //AMBER/ORANGE
		} else if (70 <= percent) && (percent < 85) {
			task.Result.Color = color.RGBA{0xf6, 0xde, 0x0a, 0xff} //YELLOW
		} else if percent >= 85 {
			task.Result.Color = color.RGBA{0x03, 0x6e, 0x05, 0xff} //GREEN
		}
	} else {
		task.Result.Grade = "-"
		task.Result.Mark = 0.0
		task.Result.Color = color.RGBA{0xff, 0xff, 0xff, 0xff}
	}

	task.Name = gcTask.Title
	task.Class = class
	task.Link = gcTask.AlternateLink
	task.Platform = "gclass"
}

// Get a list of work submission points for a Google Classroom class.
func getSubmissions(c *classroom.Course, svc *classroom.Service, tasks *[]plat.Task, swg *sync.WaitGroup, gErrChan chan error) {
	defer swg.Done()
	resp, err := svc.Courses.CourseWork.StudentSubmissions.List(
		c.Id, "-",
	).Fields(
		"studentSubmissions/id",
		"studentSubmissions/state",
		"studentSubmissions/courseId",
		"studentSubmissions/courseWorkId",
		"studentSubmissions/assignedGrade",
	).Do()

	if err != nil {
		newErr := errors.NewError("gclass.getSubmissions", "failed to get student submissions", err)
		gErrChan <- newErr
		return
	}

	submissions := make([]plat.Task, len(resp.StudentSubmissions))
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

// Retrieve a list of tasks from Google Classroom for a user.
func ListTasks(creds User, t chan map[string][]plat.Task, e chan error) {
	svc, err := Auth(creds)
	if err != nil {
		newErr := errors.NewError("gclass.ListTasks", "Google auth failed", err)
		e <- newErr
		return
	}

	resp, err := svc.Courses.List().CourseStates("ACTIVE").Fields(
		"courses/name",
		"courses/id",
	).Do()

	if err != nil {
		newErr := errors.NewError("gclass.ListTasks", "failed to get response", err)
		t <- nil
		e <- newErr
		return
	}

	if len(resp.Courses) == 0 {
		t <- nil
		e <- nil
		return
	}

	gErrChan := make(chan error)
	tasks := make([][]plat.Task, len(resp.Courses))
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

	gcTasks := map[string][]plat.Task{
		"active":    {},
		"notDue":    {},
		"overdue":   {},
		"submitted": {},
		"graded":    {},
	}

	for x := 0; x < len(tasks); x++ {
		for y := 0; y < len(tasks[x]); y++ {
			if tasks[x][y].Result.Grade != "-" {
				gcTasks["graded"] = append(
					gcTasks["graded"],
					tasks[x][y],
				)
			} else if tasks[x][y].Submitted {
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
