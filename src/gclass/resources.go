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

type Resource struct {
	Name	string
	Class	string
	Link	string
	Desc	string
	Posted	time.Time
	ResLinks	[][2]string
	Platform	string
	Id	string
}

func getClassRes(course *classroom.Course, svc *classroom.Service, res *[]Resource, resWG *sync.WaitGroup, gErrChan chan error) {
	defer resWG.Done()

	resources, err := svc.Courses.CourseWorkMaterials.List(course.Id).Fields(
		"courseWorkMaterial/title",
		"courseWorkMaterial/alternateLink",
		"courseWorkMaterial/creationTime",
		"courseWorkMaterial/id",
	).Do()

	if err != nil {
		gErrChan <- err
		return
	}

	for _, r := range resources.CourseWorkMaterial {
		resource := Resource{}

		resource.Id = course.Id + "-" + r.Id
		posted, err := time.Parse(time.RFC3339Nano, r.CreationTime)

		if err != nil {
			resource.Posted = time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)
		} else {
			resource.Posted = posted
		}

		resource.Name = r.Title
		resource.Class = course.Name
		resource.Link = r.AlternateLink
		resource.Platform = "gclass"

		*res = append(*res, resource)
	}
}

func ListRes(creds User, r chan []Resource, e chan error) {
	ctx := context.Background()

	gAuthConfig, err := google.ConfigFromJSON(
		creds.ClientID,
		classroom.ClassroomCoursesReadonlyScope,
		classroom.ClassroomStudentSubmissionsMeReadonlyScope,
		classroom.ClassroomCourseworkMeScope,
		classroom.ClassroomCourseworkmaterialsReadonlyScope,
	)

	if err != nil {
		r <- nil
		e <- err
		return
	}

	reader := strings.NewReader(creds.Token)
	oauthTok := &oauth2.Token{}
	err = json.NewDecoder(reader).Decode(oauthTok)

	if err != nil {
		r <- nil
		e <- err
		return
	}

	client := gAuthConfig.Client(context.Background(), oauthTok)

	svc, err := classroom.NewService(
		ctx,
		option.WithHTTPClient(client),
	)

	if err != nil {
		r <- nil
		e <- err
		return
	}

	resp, err := svc.Courses.List().CourseStates("ACTIVE").Fields(
		"courses/name",
		"courses/id",
	).Do()

	if err != nil {
		r <- nil
		e <- err
		return
	}

	if len(resp.Courses) == 0 {
		r <- nil
		e <- nil
		return
	}

	unordered := make([][]Resource, len(resp.Courses))
	gErrChan := make(chan error)
	var resWG sync.WaitGroup // TODO: Rename variable
	i := 0

	for _, course := range resp.Courses {
		resWG.Add(1)
		go getClassRes(course, svc, &unordered[i], &resWG, gErrChan)
		i++
	}

	resWG.Wait()

	select {
	case err = <-gErrChan:
		r <- nil
		e <- err
		return
	default:
		break
	}

	resources := []Resource{}

	for _, resList := range unordered {
		for _, r := range resList {
			resources = append(resources, r)
		}
	}

	r <- resources
	e <- err
}