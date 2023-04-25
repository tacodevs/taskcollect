package gclass

import (
	"sync"
	"time"

	"google.golang.org/api/classroom/v1"

	"main/errors"
	"main/plat"
)

// Get a list of announcements for a Google Classroom class.
func classAnnouncements(
	course *classroom.Course, svc *classroom.Service, annChan chan []plat.Resource,
	errChan chan error,
) {
	announcements := []plat.Resource{}

	resp, err := svc.Courses.Announcements.List(course.Id).Fields(
		"announcements/text",
		"announcements/alternateLink",
		"announcements/creationTime",
		"announcements/id",
	).Do()

	if err != nil {
		errChan <- errors.NewError("gclass.classAnnouncements", "failed to get course announcements", err)
		return
	}

	for _, r := range resp.Announcements {
		resource := plat.Resource{}

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
	course *classroom.Course, svc *classroom.Service, res *[]plat.Resource,
	e *error, resWG *sync.WaitGroup,
) {
	defer resWG.Done()
	annChan := make(chan []plat.Resource)
	annErrors := make(chan error)
	go classAnnouncements(course, svc, annChan, annErrors)

	resources, err := svc.Courses.CourseWorkMaterials.List(course.Id).Fields(
		"courseWorkMaterial/title",
		"courseWorkMaterial/alternateLink",
		"courseWorkMaterial/creationTime",
		"courseWorkMaterial/id",
	).Do()

	if err != nil {
		*e = errors.NewError("gclass.classResources", "failed to get coursework materials", err)
		return
	}

	for _, r := range resources.CourseWorkMaterial {
		resource := plat.Resource{}

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
		*e = err
		return
	}

	for _, r := range announcements {
		*res = append(*res, r)
	}
}

// Get a list of resources from Google Classroom for a user.
func ListRes(creds User, r chan []plat.Resource, e chan []error) {
	svc, err := Auth(creds)
	if err != nil {
		r <- nil
		e <- []error{errors.NewError("gclass.ListRes", "Google auth failed", err)}
		return
	}

	resp, err := svc.Courses.List().CourseStates("ACTIVE").Fields(
		"courses/name",
		"courses/id",
	).Do()

	if err != nil {
		r <- nil
		e <- []error{errors.NewError("gclass.ListRes", "failed to get response", err)}
		return
	}

	if len(resp.Courses) == 0 {
		r <- nil
		e <- nil
		return
	}

	unordered := make([][]plat.Resource, len(resp.Courses))
	errs := make([]error, len(resp.Courses))
	var resWG sync.WaitGroup

	for i, course := range resp.Courses {
		resWG.Add(1)
		go classResources(course, svc, &unordered[i], &errs[i], &resWG)
	}

	resWG.Wait()

	if !errors.HasOnly(errs, nil) {
		r <- nil
		e <- errs
		return
	}

	resources := []plat.Resource{}

	for _, resList := range unordered {
		resources = append(resources, resList...)
	}

	r <- resources
	e <- errs
}
