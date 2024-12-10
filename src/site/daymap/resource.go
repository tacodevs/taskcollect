package daymap

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"git.sr.ht/~kvo/go-std/defs"
	"git.sr.ht/~kvo/go-std/errors"

	"main/site"
)

// TODO: delete
type User struct {
	Timezone *time.Location
	Token    string
}

// Get a file resource from DayMap for a user.
func fileRes(creds User, courseId, id string) (site.Resource, error) {
	res := site.Resource{}
	res.Platform = "daymap"
	res.Id = courseId + "-f" + id
	res.Link = "https://gihs.daymap.net/daymap/attachment.ashx?ID=" + id

	user := site.User{
		SiteTokens: map[string]string{"daymap": creds.Token},
		Timezone: creds.Timezone,
	}
	class := site.Class{
		Link: "https://gihs.daymap.net/daymap/student/plans/class.aspx?id=" + courseId,
		Platform: "daymap",
		Id: courseId,
	}
	//var resources []site.Resource
	//var err error
	//var wg sync.WaitGroup
	//wg.Add(1)
	//go getClassRes(creds, courseId, &resources, &err, &wg)
	//wg.Wait()
	ch := make(chan site.Pair[[]site.Resource, error])
	go classRes(user, ch, class)
	sent := <-ch
	resources, err := sent.First, sent.Second
	if err != nil {
		return site.Resource{}, errors.New("failed retrieving class resources", err)
	}

	for _, r := range resources {
		if r.Id == res.Id {
			res.Posted = r.Posted
			res.Name = r.Name
			res.Class = r.Class
		}
	}

	res.ResLinks = [][2]string{{res.Link, res.Name}}

	return res, nil
}

// Get a plan resource from DayMap for a user.
func planRes(creds User, courseId, id string) (site.Resource, error) {
	res := site.Resource{}
	res.Platform = "daymap"
	res.Id = courseId + "-" + id
	res.Link = "https://gihs.daymap.net/DayMap/curriculum/plan.aspx?id=" + id

	client := &http.Client{}
	req, err := http.NewRequest("GET", res.Link, nil)
	if err != nil {
		return site.Resource{}, errors.New("GET request failed", err)
	}

	req.Header.Set("Cookie", creds.Token)
	resp, err := client.Do(req)
	if err != nil {
		return site.Resource{}, errors.New("failed to get resp", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return site.Resource{}, errors.New("failed to read resp.Body", err)
	}

	page := string(respBody)
	nameDiv := `<div id="ctl00_cp_divPlan"><div><h3>`
	i := strings.Index(page, nameDiv)
	if i == -1 {
		return site.Resource{}, errors.Raise(site.ErrInvalidResp)
	}

	i += len(nameDiv)
	page = page[i:]
	fileDiv := `</h3></div><br>`
	i = strings.Index(page, fileDiv)
	if i == -1 {
		return site.Resource{}, errors.Raise(site.ErrInvalidResp)
	}

	res.Name = page[:i]
	i += len(fileDiv)
	page = page[i:]
	descDiv := fmt.Sprintf(`<div  ><div class="lpAll" id="Note%s">`, id)
	i = strings.Index(page, descDiv)
	if i == -1 {
		return site.Resource{}, errors.Raise(site.ErrInvalidResp)
	}

	fileSect := page[:i]
	i += len(descDiv)
	page = page[i:]
	for i = strings.Index(fileSect, "DMU.OpenAttachment("); i != -1; {
		i += len("DMU.OpenAttachment(")
		fileSect = fileSect[i:]
		i = strings.Index(fileSect, ");")
		if i == -1 {
			return site.Resource{}, errors.Raise(site.ErrInvalidResp)
		}
		rlLink := "https://gihs.daymap.net/daymap/attachment.ashx?ID=" + fileSect[:i]
		fileSect = fileSect[i:]

		i = strings.Index(fileSect, "&nbsp;")
		if i == -1 {
			return site.Resource{}, errors.Raise(site.ErrInvalidResp)
		}
		i += len("&nbsp;")
		fileSect = fileSect[i:]
		i = strings.Index(fileSect, "</a>")
		if i == -1 {
			return site.Resource{}, errors.Raise(site.ErrInvalidResp)
		}
		rlName := fileSect[:i]
		fileSect = fileSect[i:]

		res.ResLinks = append(res.ResLinks, [2]string{rlLink, rlName})
		i = strings.Index(fileSect, "DMU.OpenAttachment(")
	}

	endDiv := fmt.Sprintf(
		"</div></div></div>\r\n%s\r\n    \r\n </div>\r\n\r\n%s",
		` <div style="margin: 25px 0px; width:25%;">`,
		"    </form>\r\n\r\n    <script>\r\n",
	)
	i = strings.Index(page, endDiv)
	if i == -1 {
		return site.Resource{}, errors.Raise(site.ErrInvalidResp)
	}

	res.Desc = page[:i]

	user := site.User{
		SiteTokens: map[string]string{"daymap": creds.Token},
		Timezone: creds.Timezone,
	}
	class := site.Class{
		Link: "https://gihs.daymap.net/daymap/student/plans/class.aspx?id=" + courseId,
		Platform: "daymap",
		Id: courseId,
	}
	//var resources []site.Resource
	//var err error
	//var wg sync.WaitGroup
	//wg.Add(1)
	//go getClassRes(creds, courseId, &resources, &err, &wg)
	//wg.Wait()
	ch := make(chan site.Pair[[]site.Resource, error])
	go classRes(user, ch, class)
	sent := <-ch
	resources, err := sent.First, sent.Second
	if err != nil {
		return site.Resource{}, errors.New("failed retrieving class resources", err)
	}

	for _, r := range resources {
		if r.Id == res.Id {
			res.Posted = r.Posted
			res.Class = r.Class
		}
	}

	return res, nil
}

// Get a resource from DayMap for a user.
func GetResource(creds User, id string) (site.Resource, error) {
	idSlice := strings.Split(id, "-")
	var res site.Resource
	var err error

	courseId, err := defs.Get(idSlice, 0)
	if err != nil {
		return site.Resource{}, errors.New("invalid resource ID", err)
	}
	resId, err := defs.Get(idSlice, 1)
	if err != nil {
		return site.Resource{}, errors.New("invalid resource ID", err)
	}

	if strings.HasPrefix(resId, "f") {
		res, err = fileRes(creds, courseId, resId[1:])
	} else {
		res, err = planRes(creds, courseId, resId)
	}

	return res, err
}
