package gclass

import (
	"mime/multipart"
	"net/url"
	"strings"
	"time"

	"codeberg.org/kvo/std"
	"codeberg.org/kvo/std/errors"
	"google.golang.org/api/classroom/v1"

	"main/plat"
)

// Create a direct download link to a Google Drive file from a web view link. If
// an error occurs, return the original specified link.
func directDriveLink(link string) string {
	urlResult, err := url.Parse(link)
	if err != nil {
		return link
	}

	// NOTE: urlResult.Path contains a leading "/": "/file/d/1234567890/view"
	// so the split list will have an extra element at the start hence splitUrl[3] and not splitUrl[2]

	splitUrl := strings.Split(urlResult.Path, "/")
	fileId, err := std.Access(splitUrl, 3)
	if err != nil {
		return link
	}
	finalUrl := "https://drive.google.com/uc?export=download&confirm=t&id=" + fileId
	return finalUrl
}

// Fetch the name of the class a task belongs to from Google Classroom.
func getClass(svc *classroom.Service, courseId string, classChan chan string, classErrChan chan errors.Error) {
	course, e := svc.Courses.Get(courseId).Fields("name").Do()
	if e != nil {
		err := errors.New(e.Error(), nil)
		classChan <- ""
		classErrChan <- errors.New("failed to get class", err)
		return
	}
	classChan <- course.Name
	classErrChan <- nil
}

// Fetch task information (excluding the class name) from Google Classroom for a user.
func getGCTask(svc *classroom.Service, courseId, workId string, taskChan chan classroom.CourseWork, taskErrChan chan errors.Error) {
	task, e := svc.Courses.CourseWork.Get(courseId, workId).Fields(
		"title",
		"alternateLink",
		"description",
		"materials",
		"maxPoints",
		"dueDate",
		"dueTime",
		"workType",
	).Do()
	if e != nil {
		err := errors.New(e.Error(), nil)
		taskChan <- classroom.CourseWork{}
		taskErrChan <- errors.New("failed to get task information", err)
		return
	}

	taskChan <- *task
	taskErrChan <- nil
}

// Get a task from Google Classroom for a user.
func GetTask(creds User, id string) (plat.Task, errors.Error) {
	cid := strings.SplitN(id, "-", 3)

	if len(cid) != 3 {
		return plat.Task{}, plat.ErrInvalidTaskID.Here()
	}

	svc, err := Auth(creds)
	if err != nil {
		return plat.Task{}, errors.New("Google auth failed", err)
	}

	classChan := make(chan string)
	classErrChan := make(chan errors.Error)
	go getClass(svc, cid[0], classChan, classErrChan)

	taskChan := make(chan classroom.CourseWork)
	taskErrChan := make(chan errors.Error)
	go getGCTask(svc, cid[0], cid[1], taskChan, taskErrChan)

	studSub, e := svc.Courses.CourseWork.StudentSubmissions.Get(
		cid[0], cid[1], cid[2],
	).Fields("state", "assignedGrade", "assignmentSubmission").Do()
	if e != nil {
		err = errors.New(e.Error(), nil)
		return plat.Task{}, errors.New("failed to get student submission", err)
	}

	gc, e := <-taskChan, <-taskErrChan
	if e != nil {
		err = errors.New(e.Error(), nil)
		return plat.Task{}, errors.New("from taskErrChan", err)
	}

	class, e := <-classChan, <-classErrChan
	if e != nil {
		err = errors.New(e.Error(), nil)
		return plat.Task{}, errors.New("from classErrChan", err)
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

	// task.Upload will always be false until the Google Classroom API
	// permits the upload and removal of work submissions.
	/*if gc.WorkType == "ASSIGNMENT" {
		task.Upload = true
	}*/

	if studSub.AssignmentSubmission != nil {
		for _, w := range studSub.AssignmentSubmission.Attachments {
			var link, name string

			if w.DriveFile != nil {
				link = w.DriveFile.AlternateLink
				if strings.Contains(link, "://drive.google.com/file/d/") {
					link = directDriveLink(w.DriveFile.AlternateLink)
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

	task.ResLinks, e = resFromMaterials(gc.Materials)
	if e != nil {
		err = errors.New(e.Error(), nil)
		return plat.Task{}, errors.New("failed getting resource links from task", err)
	}

	if studSub.AssignedGrade != 0 && gc.MaxPoints != 0 {
		task.Graded = true
		task.Score = studSub.AssignedGrade / gc.MaxPoints * 100
	}

	// task.Submitted will always be true until the Google Classroom
	// API supports task submissions.
	task.Submitted = true
	/*if studSub.State == "TURNED_IN" || studSub.State == "RETURNED" {
		task.Submitted = true
	}*/

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
ISSUE #3: All functions relating to submitting, uploading, and removing work do
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
func SubmitTask(creds User, id string) errors.Error {
	/*svc, err := Auth(creds)
	if err != nil {
		e <- errors.New("Google auth failed", err)
		return
	}

	_, err = svc.Courses.CourseWork.StudentSubmissions.TurnIn(
		cid[0], cid[1], cid[2],
		&classroom.TurnInStudentSubmissionRequest{},
	).Do()

	if err != nil {
		return errors.New("error turning in task", err)
	}*/
	return plat.ErrGclassApiRestriction
}

// Upload a file as a user's work for a Google Classroom task.
func UploadWork(creds User, id string, files *multipart.Reader) errors.Error {
	// Upload a file as a submission.
	return plat.ErrGclassApiRestriction
}

// Remove a file (a user's work) from a Google Classroom task.
func RemoveWork(creds User, id string, filenames []string) errors.Error {
	// Remove file submission.
	return plat.ErrGclassApiRestriction
}
