package gclass

import (
	"sync"
	"time"

	"google.golang.org/api/classroom/v1"

	"main/errors"
)

type Resource struct {
	Name     string
	Class    string
	Link     string
	Desc     string
	Posted   time.Time
	ResLinks [][2]string
	Platform string
	Id       string
}

// Get a list of announcements for a Google Classroom class.
func classAnnouncements(
	course *classroom.Course, svc *classroom.Service, annChan chan []Resource,
	errChan chan error,
) {
	announcements := []Resource{}

	resp, err := svc.Courses.Announcements.List(course.Id).Fields(
		"announcements/text",
		"announcements/alternateLink",
		"announcements/creationTime",
		"announcements/id",
	).Do()

	if err != nil {
		newErr := errors.NewError("gclass: classAnnouncements", "failed to get course announcements", err)
		errChan <- newErr
		return
	}

	for _, r := range resp.Announcements {
		resource := Resource{}

		resource.Id = course.Id + "-a" + r.Id
		posted, err := time.Parse(time.RFC3339Nano, r.CreationTime)

		if err != nil {
			resource.Posted = time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)
		} else {
			resource.Posted = posted
		}

		resName := []rune(r.Text)

		if len(resName) >= 50 {
			resName = resName[:50]
			resName = append(resName, '…')
		}

		resource.Name = string(resName)
		resource.Class = course.Name
		resource.Link = r.AlternateLink
		resource.Platform = "gclass"

		announcements = append(announcements, resource)
	}

	annChan <- announcements
	errChan <- err
}

// Get a list of resources for a Google Classroom class.
func classResources(
	course *classroom.Course, svc *classroom.Service, res *[]Resource,
	resWG *sync.WaitGroup, gErrChan chan error,
) {
	defer resWG.Done()
	annChan := make(chan []Resource)
	annErrors := make(chan error)
	go classAnnouncements(course, svc, annChan, annErrors)

	resources, err := svc.Courses.CourseWorkMaterials.List(course.Id).Fields(
		"courseWorkMaterial/title",
		"courseWorkMaterial/alternateLink",
		"courseWorkMaterial/creationTime",
		"courseWorkMaterial/id",
	).Do()

	if err != nil {
		newErr := errors.NewError("gclass: classResources", "failed to get coursework materials", err)
		gErrChan <- newErr
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

	announcements, err := <-annChan, <-annErrors

	if err != nil {
		gErrChan <- err
		return
	}

	for _, r := range announcements {
		*res = append(*res, r)
	}
}

// Get a list of resources from Google Classroom for a user.
func ListRes(creds User, r chan []Resource, e chan error) {
	svc, err := Auth(creds)
	if err != nil {
		newErr := errors.NewError("gclass: ListRes", "Google auth failed", err)
		r <- nil
		e <- newErr
		return
	}

	resp, err := svc.Courses.List().CourseStates("ACTIVE").Fields(
		"courses/name",
		"courses/id",
	).Do()

	if err != nil {
		newErr := errors.NewError("gclass: ListRes", "failed to get response", err)
		r <- nil
		e <- newErr
		return
	}

	if len(resp.Courses) == 0 {
		r <- nil
		e <- nil
		return
	}

	unordered := make([][]Resource, len(resp.Courses))
	gErrChan := make(chan error)
	var resWG sync.WaitGroup
	i := 0

	for _, course := range resp.Courses {
		resWG.Add(1)
		go classResources(course, svc, &unordered[i], &resWG, gErrChan)
		i++
	}

	resWG.Wait()

	// See issue #45
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
		resources = append(resources, resList...)
	}

	r <- resources
	e <- err
}
