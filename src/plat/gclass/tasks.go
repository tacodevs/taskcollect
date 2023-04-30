package gclass

import (
	"sync"
	"time"

	"codeberg.org/kvo/std/errors"
	"google.golang.org/api/classroom/v1"

	"main/plat"
)

// Retrieve Google Classroom task information using a workload submission point ID.
func getTask(
	studSub *classroom.StudentSubmission, svc *classroom.Service, class string,
	task *plat.Task, e *errors.Error, taskWG *sync.WaitGroup,
) {
	defer taskWG.Done()

	gcTask, er := svc.Courses.CourseWork.Get(
		studSub.CourseId, studSub.CourseWorkId,
	).Fields(
		"alternateLink",
		"creationTime",
		"dueTime",
		"dueDate",
		"maxPoints",
		"title",
	).Do()
	if er != nil {
		err := errors.New(er.Error(), nil)
		*e = errors.New("failed to get coursework", err)
		return
	}

	var hours, minutes, seconds, nanoseconds int
	task.Id = studSub.CourseId + "-" + studSub.CourseWorkId + "-" + studSub.Id

	posted, er := time.Parse(time.RFC3339Nano, gcTask.CreationTime)
	if er != nil {
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
		task.Graded = true
		task.Score = studSub.AssignedGrade / gcTask.MaxPoints * 100
	}

	task.Name = gcTask.Title
	task.Class = class
	task.Link = gcTask.AlternateLink
	task.Platform = "gclass"
}

// Get a list of work submission points for a Google Classroom class.
func getSubmissions(c *classroom.Course, svc *classroom.Service, tasks *[]plat.Task, e *[]errors.Error, swg *sync.WaitGroup) {
	defer swg.Done()
	resp, er := svc.Courses.CourseWork.StudentSubmissions.List(
		c.Id, "-",
	).Fields(
		"studentSubmissions/id",
		"studentSubmissions/state",
		"studentSubmissions/courseId",
		"studentSubmissions/courseWorkId",
		"studentSubmissions/assignedGrade",
	).Do()
	if er != nil {
		err := errors.New(er.Error(), nil)
		*e = []errors.Error{errors.New("failed to get student submissions", err)}
		return
	}

	submissions := make([]plat.Task, len(resp.StudentSubmissions))
	errs := make([]errors.Error, len(resp.StudentSubmissions))
	var taskWG sync.WaitGroup

	for i, studSub := range resp.StudentSubmissions {
		taskWG.Add(1)
		go getTask(studSub, svc, c.Name, &submissions[i], &errs[i], &taskWG)
	}

	taskWG.Wait()
	*tasks = submissions
	*e = errs
}

// Retrieve a list of tasks from Google Classroom for a user.
func ListTasks(creds User, t chan map[string][]plat.Task, e chan [][]errors.Error) {
	svc, err := Auth(creds)
	if err != nil {
		e <- [][]errors.Error{{errors.New("Google auth failed", err)}}
		return
	}

	resp, er := svc.Courses.List().CourseStates("ACTIVE").Fields(
		"courses/name",
		"courses/id",
	).Do()
	if er != nil {
		err = errors.New(er.Error(), nil)
		t <- nil
		e <- [][]errors.Error{{errors.New("failed to get response", err)}}
		return
	}

	if len(resp.Courses) == 0 {
		t <- nil
		e <- nil
		return
	}

	tasks := make([][]plat.Task, len(resp.Courses))
	errs := make([][]errors.Error, len(resp.Courses))
	var swg sync.WaitGroup

	for i, c := range resp.Courses {
		swg.Add(1)
		go getSubmissions(c, svc, &tasks[i], &errs[i], &swg)
	}

	swg.Wait()

	for _, classErrs := range errs {
		if errors.Join(classErrs...) != nil {
			t <- nil
			e <- errs
			return
		}
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
			if tasks[x][y].Graded {
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
