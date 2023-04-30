package gclass

import (
	"strings"
	"time"

	"codeberg.org/kvo/std"
	"codeberg.org/kvo/std/errors"
	"google.golang.org/api/classroom/v1"

	"main/plat"
)

// Return resource links from a classroom.Material slice.
func resFromMaterials(materials []*classroom.Material) ([][2]string, errors.Error) {
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
func GetResource(creds User, id string) (plat.Resource, errors.Error) {
	var e error
	resource := plat.Resource{}
	resource.Id = id
	resource.Platform = "gclass"
	isAnn := false
	idSlice := strings.SplitN(id, "-", 2)
	courseId, err := std.Access(idSlice, 0)
	if err != nil {
		return plat.Resource{}, errors.New("invalid resource ID", err)
	}
	resId, err := std.Access(idSlice, 1)
	if err != nil {
		return plat.Resource{}, errors.New("invalid resource ID", err)
	}
	if strings.HasPrefix(resId, "a") {
		isAnn = true
		resId = resId[1:]
	}

	svc, err := Auth(creds)
	if err != nil {
		return plat.Resource{}, errors.New("Google auth failed", err)
	}

	classChan := make(chan string)
	classErrChan := make(chan errors.Error)
	go getClass(svc, courseId, classChan, classErrChan)

	if isAnn {
		r, e := svc.Courses.Announcements.Get(courseId, resId).Do()
		if e != nil {
			err = errors.New(e.Error(), nil)
			return plat.Resource{}, errors.New("failed to get course announcements", err)
		}

		posted, e := time.Parse(time.RFC3339Nano, r.CreationTime)
		if e != nil {
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
		resource.ResLinks, e = resFromMaterials(r.Materials)
		if e != nil {
			err = errors.New(e.Error(), nil)
			return plat.Resource{}, errors.New("failed getting resource links from gclass announcement", err)
		}
	} else {
		r, e := svc.Courses.CourseWorkMaterials.Get(
			courseId, resId,
		).Fields("title", "alternateLink", "creationTime", "description", "materials").Do()
		if e != nil {
			err = errors.New(e.Error(), nil)
			return plat.Resource{}, errors.New("failed to get coursework materials", err)
		}

		posted, e := time.Parse(time.RFC3339Nano, r.CreationTime)
		if e != nil {
			resource.Posted = time.Time{}
		} else {
			resource.Posted = posted
		}

		resource.Name = r.Title
		resource.Link = r.AlternateLink
		resource.Desc = r.Description
		resource.ResLinks, e = resFromMaterials(r.Materials)
		if e != nil {
			err = errors.New(e.Error(), nil)
			return plat.Resource{}, errors.New("failed getting resource links from gclass resource", err)
		}
	}

	resource.Class, e = <-classChan, <-classErrChan
	if e != nil {
		err = errors.New(e.Error(), nil)
		return plat.Resource{}, errors.New("failed to get class name from ID", err)
	}

	return resource, nil
}
