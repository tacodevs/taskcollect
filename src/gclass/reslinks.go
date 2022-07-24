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

func getres(c *classroom.Course, svc *classroom.Service, links *map[string][][2]string, rwg *sync.WaitGroup, gerrchan chan error) {
	defer rwg.Done()

	resources, err := svc.Courses.CourseWorkMaterials.List(c.Id).Fields(
		"courseWorkMaterial/title",
		"courseWorkMaterial/alternateLink",
	).Do()

	if err != nil {
		panic(err)
		gerrchan <- err
		return
	}

	for _, res := range resources.CourseWorkMaterial {
		(*links)[c.Name] = append(
			(*links)[c.Name],
			[2]string{res.AlternateLink, res.Title},
		)
	}
}

func Reslinks(creds User, gcid []byte, r chan map[string][][2]string, e chan error) {
        ctx := context.Background()

        gauthcnf, err := google.ConfigFromJSON(
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
        oatok := &oauth2.Token{}
        err = json.NewDecoder(reader).Decode(oatok)

	if err != nil {
		r <- nil
		e <- err
		return
	}

        client := gauthcnf.Client(context.Background(), oatok)

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

	reslinks := map[string][][2]string{}
	gerrchan := make(chan error)
	var rwg sync.WaitGroup
	i := 0

	for _, c := range resp.Courses {
		reslinks[c.Name] = [][2]string{}
		rwg.Add(1)
		go getres(c, svc, &reslinks, &rwg, gerrchan)
		i++
	}

	rwg.Wait()

	select {
	case gcerr := <-gerrchan:
		r <- nil
		e <- gcerr
		return
	default:
		break
	}

	r <- reslinks
	e <- nil
}
