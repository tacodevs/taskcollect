package gclass

import (
	"image/color"
	"net/url"
	"strings"
	"time"

	"google.golang.org/api/classroom/v1"

	"main/errors"
	"main/plat"
)

// Create a direct download link to a Google Drive file from a web view link.
func getDirectDriveLink(inputUrl string) (string, error) {
	urlResult, err := url.Parse(inputUrl)
	if err != nil {
		newErr := errors.NewError("gclass.getDirectDriveLink", "URL parse error", err)
		return "", newErr
	}

	// NOTE: urlResult.Path contains a leading "/": "/file/d/1234567890/view"
	// so the split list will have an extra element at the start hence splitUrl[3] and not splitUrl[2]

	splitUrl := strings.Split(urlResult.Path, "/")
	if len(splitUrl) < 4 {
		newErr := errors.NewError("gclass.getDirectDriveLink", "split URL does not contain enough elements", nil)
		return "", newErr
	}

	finalUrl := "https://drive.google.com/uc?export=download&confirm=t&id=" + splitUrl[3]
	return finalUrl, nil
}

// Fetch the name of the class a task belongs to from Google Classroom.
func getClass(svc *classroom.Service, courseId string, classChan chan string, classErrChan chan error) {
	course, err := svc.Courses.Get(courseId).Fields("name").Do()
	if err != nil {
		newErr := errors.NewError("gclass.getClass", "failed to get class", err)
		classChan <- ""
		classErrChan <- newErr
		return
	}

	classChan <- course.Name
	classErrChan <- nil
}

// Fetch task information (excluding the class name) from Google Classroom for a user.
func getGCTask(svc *classroom.Service, courseId, workId string, taskChan chan classroom.CourseWork, taskErrChan chan error) {
	task, err := svc.Courses.CourseWork.Get(courseId, workId).Fields(
		"title",
		"alternateLink",
		"description",
		"materials",
		"maxPoints",
		"dueDate",
		"dueTime",
		"workType",
	).Do()

	if err != nil {
		newErr := errors.NewError("gclass.getGCTask", "failed to get task information", err)
		taskChan <- classroom.CourseWork{}
		taskErrChan <- newErr
		return
	}

	taskChan <- *task
	taskErrChan <- nil
}

// Get a task from Google Classroom for a user.
func GetTask(creds User, id string) (plat.Task, error) {
	cid := strings.SplitN(id, "-", 3)

	if len(cid) != 3 {
		return plat.Task{}, errInvalidTaskID
	}

	svc, err := Auth(creds)
	if err != nil {
		newErr := errors.NewError("gclass.GetTask", "Google auth failed", err)
		return plat.Task{}, newErr
	}

	classChan := make(chan string)
	classErrChan := make(chan error)
	go getClass(svc, cid[0], classChan, classErrChan)

	taskChan := make(chan classroom.CourseWork)
	taskErrChan := make(chan error)
	go getGCTask(svc, cid[0], cid[1], taskChan, taskErrChan)

	studSub, err := svc.Courses.CourseWork.StudentSubmissions.Get(
		cid[0], cid[1], cid[2],
	).Fields("state", "assignedGrade", "assignmentSubmission").Do()
	if err != nil {
		newErr := errors.NewError("gclass.GetTask", "failed to get student submission", err)
		return plat.Task{}, newErr
	}

	gc, err := <-taskChan, <-taskErrChan
	if err != nil {
		newErr := errors.NewError("gclass.GetTask", "from taskErrChan", err)
		return plat.Task{}, newErr
	}

	class, err := <-classChan, <-classErrChan
	if err != nil {
		newErr := errors.NewError("gclass.GetTask", "from classErrChan", err)
		return plat.Task{}, newErr
	}

	task := plat.Task{
		Name:     gc.Title,
		Class:    class,
		Link:     gc.AlternateLink,
		Desc:     gc.Description,
		Comment:  "The Google Classroom API does not support retrieving teacher comments.",
		Platform: "gclass",
		Id:       id,
	}

	if gc.WorkType == "ASSIGNMENT" {
		task.Upload = true
	}

	if studSub.AssignmentSubmission != nil {
		for _, w := range studSub.AssignmentSubmission.Attachments {
			var link, name string

			if w.DriveFile != nil {
				link = w.DriveFile.AlternateLink
				if strings.Contains(link, "://drive.google.com/") {
					link, err = getDirectDriveLink(w.DriveFile.AlternateLink)
					if err != nil {
						newErr := errors.NewError("gclass.GetTask", "failed to get direct drive link", err)
						return plat.Task{}, newErr
					}
				}
				name = w.DriveFile.Title
			} else if w.Form != nil {
				link = w.Form.FormUrl
				name = w.Form.Title
			} else if w.YouTubeVideo != nil {
				link = w.YouTubeVideo.AlternateLink
				name = w.YouTubeVideo.Title
			} else if w.Link != nil {
				link = w.Link.Url
				name = w.Link.Title
			} else {
				continue
			}

			if name == "" {
				name = link
			}

			worklink := [2]string{link, name}
			task.WorkLinks = append(task.WorkLinks, worklink)
		}
	}

	task.ResLinks, err = resFromMaterials(gc.Materials)
	if err != nil {
		newErr := errors.NewError("gclass.GetTask", "failed getting resource links from task", err)
		return plat.Task{}, newErr
	}

	if studSub.AssignedGrade != 0 && gc.MaxPoints != 0 {
		percent := studSub.AssignedGrade / gc.MaxPoints * 100
		task.Result.Mark = percent
		task.Result.Color = color.RGBA{}
	}

	if studSub.State == "TURNED_IN" || studSub.State == "RETURNED" {
		task.Submitted = true
	}

	var hours, minutes, seconds, nanoseconds int

	if gc.DueTime == nil {
		hours, minutes, seconds, nanoseconds = 0, 0, 0, 0
	} else {
		hours = int(gc.DueTime.Hours)
		minutes = int(gc.DueTime.Minutes)
		seconds = int(gc.DueTime.Seconds)
		nanoseconds = int(gc.DueTime.Nanos)
	}

	if gc.DueDate != nil {
		task.Due = time.Date(
			int(gc.DueDate.Year),
			time.Month(gc.DueDate.Month),
			int(gc.DueDate.Day),
			hours,
			minutes,
			seconds,
			nanoseconds,
			time.UTC,
		)
	}

	return task, nil
}

/*
ISSUE(#3): All functions relating to submitting, uploading, and removing work do
*not* work!

Google has a requirement that "the only service that can submit assignments
is the one that made it". Which translates to "the only service that can submit
assignments is Google Classroom" because the only service that teachers use to
interface with Google Classroom is Google Classroom itself.

So, in theory, this function works, but in practice, the Google Classroom API
returns "Error 403: @ProjectPermissionDenied".

At the moment I've commented out the SubmitTask function, after the writing of
which I encountered the error. This means that the user will have no indication
of any error and will return back to the task's page immediately.

How this issue should be managed is open to discussion:
https://codeberg.org/kvo/taskcollect/issues/3
*/

// Submit a Google Classroom task on behalf of a user.
func SubmitTask(creds User, id string) error {
	/*svc, err := Auth(creds)
	if err != nil {
		newErr := errors.NewError("gclass.SubmitTask", "Google auth failed", err)
		e <- newErr
		return
	}

	_, err = svc.Courses.CourseWork.StudentSubmissions.TurnIn(
		cid[0], cid[1], cid[2],
		&classroom.TurnInStudentSubmissionRequest{},
	).Do()

	if err != nil {
		newErr := errors.NewError("gclass.SubmitTask", "error turning in task", err)
		return newErr
	}*/

	return nil
}

// Upload a file as a user's work for a Google Classroom task.
func UploadWork(creds User, id string, files []plat.File) error {
	// Upload a file as a submission.
	return nil
}

// Remove a file (a user's work) from a Google Classroom task.
func RemoveWork(creds User, id string, filenames []string) error {
	// Remove file submission.
	return nil
}
