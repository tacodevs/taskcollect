package daymap

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"git.sr.ht/~kvo/go-std/defs"
	"git.sr.ht/~kvo/go-std/errors"

	"main/site"
)

type resJson struct {
	D string
}

// Returns the index of the next school resource type, as well as a corresponding planDiv/fileDiv/linkDiv.
func nextRes(buf, planDiv, fileDiv, linkDiv string) (int, string) {
	planIdx := strings.Index(buf, planDiv)
	fileIdx := strings.Index(buf, fileDiv)
	linkIdx := strings.Index(buf, linkDiv)

	isFile := (fileIdx != -1) && (fileIdx < planIdx || planIdx == -1)

	if planIdx == -1 && fileIdx == -1 && linkIdx == -1 {
		return -1, ""
	} else if isFile {
		return fileIdx, fileDiv
	} else {
		return planIdx, planDiv
	}
}

// Return class name and secondary "courseId" from specified link to Daymap class page.
func auxClassInfo(user site.User, link string) (string, string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return "", "", errors.New("cannot create aux class request", err)
	}

	req.Header.Set("Cookie", user.SiteTokens["daymap"])

	resp, err := client.Do(req)
	if err != nil {
		return "", "", errors.New("cannot execute aux class request", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", errors.New("cannot read aux class response body", err)
	}

	page := string(body)
	exp, err := regexp.Compile(`new Classroom\([0-9]+,null,[0-9]+,`)
	if err != nil {
		return "", "", errors.New("cannot compile regex", err)
	}
	courseId, err := defs.Get(strings.Split(exp.FindString(page), ","), 2)
	if err != nil {
		return "", "", errors.New("missing secondary course ID", err)
	}

	classDiv := `<td><span id="ctl00_ctl00_cp_cp_divHeader" class="Header14" style="padding-left: 20px">`
	i := strings.Index(page, classDiv)
	if i == -1 {
		return "", "", errors.New("missing class name", nil)
	}
	i += len(classDiv)
	page = page[i:]
	i = strings.Index(page, "</span>")
	if i == -1 {
		return "", "", errors.New("unterminated class name", nil)
	}
	class := page[:i]

	return class, courseId, nil
}

func classRes(user site.User, c chan site.Pair[[]site.Resource, error], class site.Class) {
	var result site.Pair[[]site.Resource, error]
	resUrl := "https://gihs.daymap.net/daymap/student/plans/class.aspx/InitialiseResources"
	classUrl := "https://gihs.daymap.net/daymap/student/plans/class.aspx?id=" + class.Id
	var resources []site.Resource

	className, courseId, err := auxClassInfo(user, classUrl)
	if err != nil {
		result.Second = errors.New("cannot fetch secondary class ID", err)
		c <- result
		return
	}

	form := fmt.Sprintf(`{"classId":%s,"courseId":%s}`, class.Id, courseId)
	client := &http.Client{}

	req, err := http.NewRequest("POST", resUrl, strings.NewReader(form))
	if err != nil {
		result.Second = errors.New("cannot create resources request", err)
		c <- result
		return
	}

	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Cookie", user.SiteTokens["daymap"])
	req.Header.Set("Referer", classUrl)

	resp, err := client.Do(req)
	if err != nil {
		result.Second = errors.New("cannot execute resources request", err)
		c <- result
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Second = errors.New("cannot read resources response body", err)
		c <- result
		return
	}

	exp, err := regexp.Compile("[0-9]+/[0-9]+/[0-9]+")
	if err != nil {
		result.Second = errors.New("cannot compile regex", err)
		c <- result
		return
	}

	var data resJson
	err = json.Unmarshal(body, &data)
	if err != nil {
		result.Second = errors.New("cannot unmarshal JSON", err)
		c <- result
		return
	}

	page := data.D
	planDiv := `</td><td class='active itm' onclick="DMU.ViewPlan(`
	fileDiv := `<div class='fLinkDiv'><a href='#' onclick="DMU.OpenAttachment(`
	linkDiv := `<a href='javascript:DMU.OpenNewWindow("`
	i, div := nextRes(page, planDiv, fileDiv, linkDiv)

	for i != -1 {
		resource := site.Resource{
			Class:    className,
			Platform: "daymap",
		}
		dateRegion := page[:i]
		page = page[i:]
		dates := exp.FindAllString(dateRegion, -1)

		if len(dates) == 0 && strings.Index(page, planDiv) == -1 && strings.Index(page, fileDiv) == -1 {
			result.Second = errors.New("resource has no post date", nil)
			c <- result
			return
		} else if len(dates) > 0 {
			postStr := dates[len(dates)-1]
			resource.Posted, err = time.ParseInLocation("2/01/2006", postStr, user.Timezone)
			if err != nil {
				result.Second = errors.New("cannot parse time", err)
				c <- result
				return
			}
		} else {
			if _, err = defs.Get([]byte(page), len(div)); err != nil {
				result.Second = errors.New("invalid HTML response", err)
				c <- result
				return
			}
			page = page[len(div):]
			i, div = nextRes(page, planDiv, fileDiv, linkDiv)
			continue
		}

		i = len(div)
		if _, err = defs.Get([]byte(page), i); err != nil {
			result.Second = errors.New("invalid HTML response", err)
			c <- result
			return
		}
		page = page[i:]

		i = strings.Index(page, ");")

		if i == -1 {
			result.Second = errors.New("invalid HTML response", nil)
			c <- result
			return
		}

		resource.Id = page[:i]

		if div == fileDiv {
			i = strings.Index(page, "&nbsp;")
			if i == -1 {
				result.Second = errors.New("invalid HTML response", nil)
				c <- result
				return
			}

			i += len("&nbsp;")
			page = page[i:]
			i = strings.Index(page, "</a>")
			if i == -1 {
				result.Second = errors.New("invalid HTML response", nil)
				c <- result
				return
			}
		} else {
			i += len(`);;"><div class='lpTitle'>`)
			page = page[i:]
			i = strings.Index(page, "</div>")
			if i == -1 {
				result.Second = errors.New("invalid HTML response", nil)
				c <- result
				return
			}
		}

		if _, err = defs.Get([]byte(page), i); err != nil {
			result.Second = errors.New("invalid HTML response", err)
			c <- result
			return
		}
		resource.Name = page[:i]

		if div == fileDiv {
			resource.Link = "https://gihs.daymap.net/daymap/attachment.ashx?ID=" + resource.Id
			resource.Id = "f" + resource.Id
		} else {
			resource.Link = "https://gihs.daymap.net/DayMap/curriculum/plan.aspx?id=" + resource.Id
		}

		resource.Id = class.Id + "-" + resource.Id
		resources = append(resources, resource)

		if _, err = defs.Get([]byte(page), i); err != nil {
			result.Second = errors.New("invalid HTML response", err)
			c <- result
			return
		}
		page = page[i:]
		i, div = nextRes(page, planDiv, fileDiv, linkDiv)
	}
	result.First = resources
	c <- result
}

func Resources(user site.User, c chan site.Pair[[]site.Resource, error], classes []site.Class) {
	var result site.Pair[[]site.Resource, error]
	var resources []site.Resource
	ch := make(chan site.Pair[[]site.Resource, error])
	for _, class := range classes {
		go classRes(user, ch, class)
	}
	for range classes {
		sent := <-ch
		list, err := sent.First, sent.Second
		if err != nil {
			result.Second = errors.New("", err)
			c <- result
			return
		}
		resources = append(resources, list...)
	}
	result.First = resources
	c <- result
}
