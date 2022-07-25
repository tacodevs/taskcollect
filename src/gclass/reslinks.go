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

func getResp(c *classroom.Course, svc *classroom.Service, links *map[string][][2]string, rwg *sync.WaitGroup, gErrChan chan error) {
	defer rwg.Done()

	resources, err := svc.Courses.CourseWorkMaterials.List(c.Id).Fields(
		"courseWorkMaterial/title",
		"courseWorkMaterial/alternateLink",
	).Do()

	if err != nil {
		panic(err)
		gErrChan <- err
		return
	}

	for _, res := range resources.CourseWorkMaterial {
		(*links)[c.Name] = append(
			(*links)[c.Name],
			[2]string{res.AlternateLink, res.Title},
		)
	}
}

func ResLinks(creds User, gcid []byte, r chan map[string][][2]string, e chan error) {
	ctx := context.Background()

	gauthConfig, err := google.ConfigFromJSON(
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

	client := gauthConfig.Client(context.Background(), oauthTok)

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
	var rwg sync.WaitGroup // TODO: Rename variable
	i := 0

	for _, c := range resp.Courses {
		resLinks[c.Name] = [][2]string{}
		rwg.Add(1)
		go getResp(c, svc, &resLinks, &rwg, gErrChan)
		i++
	}

	rwg.Wait()

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
