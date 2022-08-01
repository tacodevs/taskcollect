package gclass

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/classroom/v1"
	"google.golang.org/api/option"
)

type Task struct {
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

func getDirectGoogleDriveLink(inputUrl string) (string, error){
	urlResult, err := url.Parse(inputUrl)
	if err != nil {
		return "", err
	}

	// urlResult.Path contains a leading "/": "/file/d/1234567890/view"
	// so the split list will have an extra element at the start hence splitUrl[3] and not splitUrl[2]
	
	splitUrl := strings.Split(urlResult.Path, "/")
	finalUrl := "https://drive.google.com/uc?export=download&id=" + splitUrl[3]
	return finalUrl, nil
}

func getClass(svc *classroom.Service, courseId string, classChan chan string, classErrChan chan error) {
	course, err := svc.Courses.Get(courseId).Fields("name").Do()

	if err != nil {
		classChan <- ""
		classErrChan <- err
		return
	}

	classChan <- course.Name
	classErrChan <- nil
	return
}

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
		taskChan <- classroom.CourseWork{}
		taskErrChan <- err
		return
	}

	taskChan <- *task
	taskErrChan <- nil
	return
}

/*
ISSUE(#6): The Google Classroom API does not seem to have any mechanism to request
teacher comments for a task, so task.Comment is always empty.
*/

func GetTask(creds User, gcid []byte, id string) (Task, error) {
	cid := strings.SplitN(id, "-", 3)

	if len(cid) != 3 {
		return Task{}, errors.New("gclass: invalid task ID")
	}

	ctx := context.Background()

	gAuthConfig, err := google.ConfigFromJSON(
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
	oauthTok := &oauth2.Token{}
	err = json.NewDecoder(r).Decode(oauthTok)

	if err != nil {
		return Task{}, err
	}

	client := gAuthConfig.Client(context.Background(), oauthTok)

	svc, err := classroom.NewService(
		ctx,
		option.WithHTTPClient(client),
	)

	if err != nil {
		return Task{}, err
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
		return Task{}, err
	}

	gc, err := <-taskChan, <-taskErrChan
	if err != nil {
		return Task{}, err
	}
	class, err := <-classChan, <-classErrChan

	if err != nil {
		return Task{}, err
	}

	task := Task{
		Name:     gc.Title,
		Class:    class,
		Link:     gc.AlternateLink,
		Desc:     gc.Description,
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
					link, err = getDirectGoogleDriveLink(w.DriveFile.AlternateLink)
					if err != nil {
						return Task{}, err
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

	for _, m := range gc.Materials {
		var link, name string

		if m.DriveFile != nil {
			link = m.DriveFile.DriveFile.AlternateLink
			if strings.Contains(link, "://drive.google.com/") {
				link, err = getDirectGoogleDriveLink(m.DriveFile.DriveFile.AlternateLink)
				if err != nil {
					return Task{}, err
				}
			}
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

		resLink := [2]string{link, name}
		task.ResLinks = append(task.ResLinks, resLink)
	}

	if studSub.AssignedGrade != 0 && gc.MaxPoints != 0 {
		percent := studSub.AssignedGrade / gc.MaxPoints * 100
		task.Grade = fmt.Sprintf("%.f%%", percent)
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

Google had this strange idea that "the only service that can submit assignments
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

func SubmitTask(creds User, gcid []byte, id string) error {
	/*
		cid := strings.SplitN(id, "-", 3)

		if len(cid) != 3 {
			return errors.New("gclass: invalid task ID")
		}

		ctx := context.Background()

		gAuthConfig, err := google.ConfigFromJSON(
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
		oauthTok := &oauth2.Token{}
		err = json.NewDecoder(r).Decode(oauthTok)

		if err != nil {
			return err
		}

		client := gAuthConfig.Client(context.Background(), oauthTok)

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
