package gclass

import (
	"strings"
	"time"

	"codeberg.org/kvo/std"
	"google.golang.org/api/classroom/v1"

	"main/errors"
	"main/plat"
)

// Return resource links from a classroom.Material slice.
func resFromMaterials(materials []*classroom.Material) ([][2]string, error) {
	if materials == nil {
		return nil, nil
	}

	resLinks := [][2]string{}

	for _, m := range materials {
		var link, name string

		if m.DriveFile != nil {
			link = m.DriveFile.DriveFile.AlternateLink
			if strings.Contains(link, "://drive.google.com/file/d/") {
				link = directDriveLink(m.DriveFile.DriveFile.AlternateLink)
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
		resLinks = append(resLinks, resLink)
	}

	return resLinks, nil
}

// Get a resource from Google Classroom for a user.
func GetResource(creds User, id string) (plat.Resource, error) {
	resource := plat.Resource{}
	resource.Id = id
	resource.Platform = "gclass"
	isAnn := false
	idSlice := strings.SplitN(id, "-", 2)
	var err error
	courseId, err := std.Access(idSlice, 0)
	if err != nil {
		return plat.Resource{}, errors.NewError("gclass.GetResource", "invalid resource ID", err)
	}
	resId, err := std.Access(idSlice, 1)
	if err != nil {
		return plat.Resource{}, errors.NewError("gclass.GetResource", "invalid resource ID", err)
	}
	if strings.HasPrefix(resId, "a") {
		isAnn = true
		resId = resId[1:]
	}

	svc, err := Auth(creds)
	if err != nil {
		return plat.Resource{}, errors.NewError("gclass.GetResource", "Google auth failed", err)
	}

	classChan := make(chan string)
	classErrChan := make(chan error)
	go getClass(svc, courseId, classChan, classErrChan)

	if isAnn {
		r, err := svc.Courses.Announcements.Get(courseId, resId).Do()
		if err != nil {
			return plat.Resource{}, errors.NewError("gclass.classAnnouncements", "failed to get course announcements", err)
		}

		posted, err := time.Parse(time.RFC3339Nano, r.CreationTime)

		if err != nil {
			resource.Posted = time.Time{}
		} else {
			resource.Posted = posted
		}

		resName := []rune(r.Text)

		if len(resName) >= 50 {
			resName = resName[:50]
			resName = append(resName, '…')
		}

		resource.Name = string(resName)
		resource.Link = r.AlternateLink
		resource.Desc = r.Text
		resource.ResLinks, err = resFromMaterials(r.Materials)

		if err != nil {
			return plat.Resource{}, errors.NewError("gclass.GetResource", "failed getting resource links from gclass announcement", err)
		}
	} else {
		r, err := svc.Courses.CourseWorkMaterials.Get(
			courseId, resId,
		).Fields("title", "alternateLink", "creationTime", "description", "materials").Do()

		if err != nil {
			return plat.Resource{}, errors.NewError("gclass.classResources", "failed to get coursework materials", err)
		}

		posted, err := time.Parse(time.RFC3339Nano, r.CreationTime)

		if err != nil {
			resource.Posted = time.Time{}
		} else {
			resource.Posted = posted
		}

		resource.Name = r.Title
		resource.Link = r.AlternateLink
		resource.Desc = r.Description
		resource.ResLinks, err = resFromMaterials(r.Materials)

		if err != nil {
			return plat.Resource{}, errors.NewError("gclass.GetResource", "failed getting resource links from gclass resource", err)
		}
	}

	resource.Class, err = <-classChan, <-classErrChan
	if err != nil {
		return plat.Resource{}, errors.NewError("gclass.GetResource", "failed to get class name from ID", err)
	}

	return resource, nil
}
