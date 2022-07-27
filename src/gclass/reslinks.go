package gclass

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/classroom/v1"
	"google.golang.org/api/option"
)

func getResp(course *classroom.Course, svc *classroom.Service, links *map[string][][2]string, resWG *sync.WaitGroup, gErrChan chan error) {
	defer resWG.Done()

	resources, err := svc.Courses.CourseWorkMaterials.List(course.Id).Fields(
		"courseWorkMaterial/title",
		"courseWorkMaterial/alternateLink",
	).Do()

	if err != nil {
		panic(err)
		gErrChan <- err
		return
	}

	for _, res := range resources.CourseWorkMaterial {
		(*links)[course.Name] = append(
			(*links)[course.Name],
			[2]string{res.AlternateLink, res.Title},
		)
	}
}

func ResLinks(creds User, gcid []byte, r chan map[string][][2]string, e chan error) {
	ctx := context.Background()

	gAuthConfig, err := google.ConfigFromJSON(
		gcid,
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

	resLinks := map[string][][2]string{}
	gErrChan := make(chan error)
	var resWG sync.WaitGroup // TODO: Rename variable
	i := 0

	for _, course := range resp.Courses {
		resLinks[course.Name] = [][2]string{}
		resWG.Add(1)
		go getResp(course, svc, &resLinks, &resWG, gErrChan)
		i++
	}

	resWG.Wait()

	select {
	case gcErr := <-gErrChan:
		r <- nil
		e <- gcErr
		return
	default:
		break
	}

	r <- resLinks
	e <- nil
}
