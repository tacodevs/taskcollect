package daymap

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"git.sr.ht/~kvo/go-std/errors"
	"git.sr.ht/~kvo/go-std/slices"

	"main/site"
)

func fileRes(user site.User, id string, class site.Class) (site.Resource, error) {
	ch := make(chan site.Pair[[]site.Resource, error])
	resource := site.Resource{
		Link:     "https://gihs.daymap.net/daymap/attachment.ashx?ID=" + id,
		Platform: "daymap",
		Id:       class.Id + "-f" + id,
	}
	go classRes(user, ch, class)
	sent := <-ch
	resources, err := sent.First, sent.Second
	if err != nil {
		return site.Resource{}, errors.New(err, "cannot fetch resources list")
	}
	for _, res := range resources {
		if res.Id == resource.Id {
			resource.Posted = res.Posted
			resource.Name = res.Name
			resource.Class = res.Class
			break
		}
	}
	resource.ResLinks = [][2]string{{resource.Link, resource.Name}}
	return resource, nil
}

func planRes(user site.User, id string, class site.Class) (site.Resource, error) {
	ch := make(chan site.Pair[[]site.Resource, error])
	resource := site.Resource{
		Link:     "https://gihs.daymap.net/DayMap/curriculum/plan.aspx?id=" + id,
		Platform: "daymap",
		Id:       class.Id + "-" + id,
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", resource.Link, nil)
	if err != nil {
		return site.Resource{}, errors.New(err, "cannot create resource request")
	}

	req.Header.Set("Cookie", user.SiteTokens["daymap"])

	resp, err := client.Do(req)
	if err != nil {
		return site.Resource{}, errors.New(err, "cannot execute resource request")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return site.Resource{}, errors.New(err, "cannot read resource response body")
	}

	page := string(body)
	nameDiv := `<div id="ctl00_cp_divPlan"><div><h3>`
	i := strings.Index(page, nameDiv)

	if i == -1 {
		return site.Resource{}, errors.New(nil, "invalid HTML response")
	}

	i += len(nameDiv)
	page = page[i:]
	fileDiv := `</h3></div><br>`
	i = strings.Index(page, fileDiv)

	if i == -1 {
		return site.Resource{}, errors.New(nil, "invalid HTML response")
	}

	resource.Name = page[:i]
	i += len(fileDiv)
	page = page[i:]
	descDiv := fmt.Sprintf(`<div  ><div class="lpAll" id="Note%s">`, id)
	i = strings.Index(page, descDiv)

	if i == -1 {
		return site.Resource{}, errors.New(nil, "invalid HTML response")
	}

	fileSect := page[:i]
	i += len(descDiv)
	page = page[i:]

	for i = strings.Index(fileSect, "DMU.OpenAttachment("); i != -1; {
		i += len("DMU.OpenAttachment(")
		fileSect = fileSect[i:]
		i = strings.Index(fileSect, ");")

		if i == -1 {
			return site.Resource{}, errors.New(nil, "invalid HTML response")
		}

		rlLink := "https://gihs.daymap.net/daymap/attachment.ashx?ID=" + fileSect[:i]
		fileSect = fileSect[i:]
		i = strings.Index(fileSect, "&nbsp;")

		if i == -1 {
			return site.Resource{}, errors.New(nil, "invalid HTML response")
		}

		i += len("&nbsp;")
		fileSect = fileSect[i:]
		i = strings.Index(fileSect, "</a>")

		if i == -1 {
			return site.Resource{}, errors.New(nil, "invalid HTML response")
		}

		rlName := fileSect[:i]
		fileSect = fileSect[i:]
		resource.ResLinks = append(resource.ResLinks, [2]string{rlLink, rlName})

		i = strings.Index(fileSect, "DMU.OpenAttachment(")
	}

	endDiv := "</div></div></div>\r\n" + ` <div style="margin: 25px 0px; width:25%;">`
	endDiv += "\r\n    \r\n </div>\r\n\r\n    </form>\r\n\r\n    <script>\r\n"
	i = strings.Index(page, endDiv)
	if i == -1 {
		return site.Resource{}, errors.New(nil, "invalid HTML response")
	}

	resource.Desc = page[:i]
	go classRes(user, ch, class)
	sent := <-ch
	resources, err := sent.First, sent.Second
	if err != nil {
		return site.Resource{}, errors.New(err, "cannot fetch resources list")
	}

	for _, res := range resources {
		if res.Id == resource.Id {
			resource.Posted = res.Posted
			resource.Class = res.Class
			break
		}
	}

	return resource, nil
}

func Resource(user site.User, id string) (site.Resource, error) {
	var res site.Resource
	var err error
	ids := strings.Split(id, "-")
	class := site.Class{
		Platform: "daymap",
	}
	class.Id, err = slices.Get(ids, 0)
	if err != nil {
		return site.Resource{}, errors.New(err, "invalid resource ID")
	}
	class.Link = "https://gihs.daymap.net/daymap/student/plans/class.aspx?id=" + class.Id
	resId, err := slices.Get(ids, 1)
	if err != nil {
		return site.Resource{}, errors.New(err, "invalid resource ID")
	}
	if strings.HasPrefix(resId, "f") {
		res, err = fileRes(user, resId[1:], class)
	} else {
		res, err = planRes(user, resId, class)
	}
	return res, err
}
