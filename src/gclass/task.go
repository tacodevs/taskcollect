package gclass

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/classroom/v1"
	"google.golang.org/api/option"
)

type Task struct {
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

func getclass(svc *classroom.Service, courseId string, cchan chan string, echan chan error) {
	course, err := svc.Courses.Get(courseId).Fields("name").Do()

	if err != nil {
		cchan <- ""
		echan <- err
		return
	}

	cchan <- course.Name
	echan <- nil
	return
}

func getGtask(svc *classroom.Service, courseId, workId string, gchan chan classroom.CourseWork, errchan chan error) {
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
		gchan <- classroom.CourseWork{}
		errchan <- err
		return
	}

	gchan <- *task
	errchan <- nil
	return
}

/*
ISSUE: The Google Classroom API does not seem to have any mechanism to request
teacher comments for a task, so task.Comment is always empty.
*/

func GetTask(creds User, gcid []byte, id string) (Task, error) {
	cid := strings.SplitN(id, "-", 3)

	if len(cid) != 3 {
		return Task{}, errors.New("gclass: invalid task ID")
	}

	ctx := context.Background()

	gauthcnf, err := google.ConfigFromJSON(
		gcid,
		classroom.ClassroomCoursesReadonlyScope,
		classroom.ClassroomStudentSubmissionsMeReadonlyScope,
		classroom.ClassroomCourseworkMeScope,
		classroom.ClassroomCourseworkmaterialsReadonlyScope,
	)

	if err != nil {
		return Task{}, err
	}

	r := strings.NewReader(creds.Token)
	oatok := &oauth2.Token{}
	err = json.NewDecoder(r).Decode(oatok)

	if err != nil {
		return Task{}, err
	}

	client := gauthcnf.Client(context.Background(), oatok)

	svc, err := classroom.NewService(
		ctx,
		option.WithHTTPClient(client),
	)

	if err != nil {
		return Task{}, err
	}

	cchan := make(chan string)
	echan := make(chan error)
	go getclass(svc, cid[0], cchan, echan)

	gchan := make(chan classroom.CourseWork)
	errchan := make(chan error)
	go getGtask(svc, cid[0], cid[1], gchan, errchan)

	s, err := svc.Courses.CourseWork.StudentSubmissions.Get(
		cid[0], cid[1], cid[2],
	).Fields("state", "assignedGrade", "assignmentSubmission").Do()

	if err != nil {
		return Task{}, err
	}

	c, err := <-gchan, <-errchan

	if err != nil {
		return Task{}, err
	}

	class, err := <-cchan, <-echan

	if err != nil {
		return Task{}, err
	}

	task := Task{
		Name: c.Title,
		Class: class,
		Link: c.AlternateLink,
		Desc: c.Description,
		Platform: "gclass",
		Id: id,
	}

	if c.WorkType == "ASSIGNMENT" {
		task.Upload = true
	}

	if s.AssignmentSubmission != nil {
		for _, w := range s.AssignmentSubmission.Attachments {
			var link, name string

			if w.DriveFile != nil {
				link = w.DriveFile.AlternateLink
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
			task.Worklinks = append(task.Worklinks, worklink)
		}
	}

	for _, m := range c.Materials {
		var link, name string

		if m.DriveFile != nil {
			link = m.DriveFile.DriveFile.AlternateLink
			name = m.DriveFile.DriveFile.Title
		} else if m.Form != nil {
			link = m.Form.FormUrl
			name = m.Form.Title
		} else if m.YoutubeVideo != nil {
			link = m.YoutubeVideo.AlternateLink
			name = m.YoutubeVideo.Title
		} else if m.Link != nil {
			link = m.Link.Url
			name = m.Link.Title
		} else {
			continue
		}

		if name == "" {
			name = link
		}

		reslink := [2]string{link, name}
		task.Reslinks = append(task.Reslinks, reslink)
	}

	if s.AssignedGrade != 0 && c.MaxPoints != 0 {
		percent := s.AssignedGrade/c.MaxPoints*100
		task.Grade = fmt.Sprintf("%.f%%", percent)
	}

	if s.State == "TURNED_IN" || s.State == "RETURNED" {
		task.Submitted = true
	}

	var hours, minutes, seconds, nanoseconds int

	if c.DueTime == nil {
		hours, minutes, seconds, nanoseconds = 0, 0, 0, 0
	} else {
		hours = int(c.DueTime.Hours)
		minutes = int(c.DueTime.Minutes)
		seconds = int(c.DueTime.Seconds)
		nanoseconds = int(c.DueTime.Nanos)
	}

	if c.DueDate != nil {
		task.Due = time.Date(
			int(c.DueDate.Year),
			time.Month(c.DueDate.Month),
			int(c.DueDate.Day),
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
ISSUE: All functions relating to submitting, uploading, and removing work do
*not* work!

Google had this strange idea that "the only service that can submit assignments
is the one that made it". Which translates to "the only service that can submit
assignments is Google Classroom" because the only service that teachers use to
interface with Google Classroom is Google Classroom itself.

So, in theory, this function works, but in practice, the Google Classroom API
returns "Error 403: @ProjectPermissionDenied".

At the moment I've commented out the SubmitTask function, after the writing of
which I encountered the error. This means that the user will have no indication
of any error and will return back to the task's page immediately.

How this issue should be managed is open to discussion.
*/

func SubmitTask(creds User, gcid []byte, id string) error {
	/*
	cid := strings.SplitN(id, "-", 3)

	if len(cid) != 3 {
		return errors.New("gclass: invalid task ID")
	}

	ctx := context.Background()

	gauthcnf, err := google.ConfigFromJSON(
		gcid,
		classroom.ClassroomCoursesReadonlyScope,
		classroom.ClassroomStudentSubmissionsMeReadonlyScope,
		classroom.ClassroomCourseworkMeScope,
		classroom.ClassroomCourseworkmaterialsReadonlyScope,
	)

	if err != nil {
		return err
	}

	r := strings.NewReader(creds.Token)
	oatok := &oauth2.Token{}
	err = json.NewDecoder(r).Decode(oatok)

	if err != nil {
		return err
	}

	client := gauthcnf.Client(context.Background(), oatok)

	svc, err := classroom.NewService(
		ctx,
		option.WithHTTPClient(client),
	)

	if err != nil {
		return err
	}

	_, err = svc.Courses.CourseWork.StudentSubmissions.TurnIn(
		cid[0], cid[1], cid[2],
		&classroom.TurnInStudentSubmissionRequest{},
	).Do()

	if err != nil {
		return err
	}
	*/

	return nil
}

func UploadWork(creds User, gcid []byte, id string, filename string, f *io.Reader) error {
	// Upload a file as a submission.
	_, err := io.Copy(os.Stdout, *f)
	return err
}

func RemoveWork(creds User, gcid []byte, id string, filenames []string) error {
	// Remove file submission.
	return nil
}
