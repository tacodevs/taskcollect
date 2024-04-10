package daymap

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"git.sr.ht/~kvo/libgo/defs"
	"git.sr.ht/~kvo/libgo/errors"

	"main/plat"
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

// Return auxillary class info from a link to a DayMap class page.
func auxClassInfo(creds User, link string) (string, string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return "", "", errors.New("GET request failed", err)
	}

	req.Header.Set("Cookie", creds.Token)
	resp, err := client.Do(req)
	if err != nil {
		return "", "", errors.New("failed to get resp", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", errors.New("failed to read resp.Body", err)
	}

	page := string(respBody)
	re, err := regexp.Compile(`new Classroom\([0-9]+,null,[0-9]+,`)
	if err != nil {
		return "", "", errors.New("failed to compile regex", err)
	}
	courseId, err := defs.Get(strings.Split(re.FindString(page), ","), 2)
	if err != nil {
		return "", "", errors.New("cannot get class ID", err)
	}

	classDiv := `<td><span id="ctl00_ctl00_cp_cp_divHeader" class="Header14" style="padding-left: 20px">`
	i := strings.Index(page, classDiv)
	if i == -1 {
		return "", "", errors.New("can't find class name", errors.Raise(plat.ErrInvalidResp))
	}
	i += len(classDiv)
	page = page[i:]
	i = strings.Index(page, "</span>")
	if i == -1 {
		return "", "", errors.New("can't find class name end", errors.Raise(plat.ErrInvalidResp))
	}
	class := page[:i]

	return class, courseId, nil
}

// Get a list of resources for a DayMap class.
func getClassRes(creds User, id string, res *[]plat.Resource, e *error, wg *sync.WaitGroup) {
	defer wg.Done()
	resUrl := "https://gihs.daymap.net/daymap/student/plans/class.aspx/InitialiseResources"
	classUrl := "https://gihs.daymap.net/daymap/student/plans/class.aspx?id=" + id

	class, courseId, err := auxClassInfo(creds, classUrl)
	if err != nil {
		*e = errors.New("failed retrieving secondary class ID", err)
		return
	}

	jsonReq := fmt.Sprintf(`{"classId":%s,"courseId":%s}`, id, courseId)
	client := &http.Client{}

	req, err := http.NewRequest("POST", resUrl, strings.NewReader(jsonReq))
	if err != nil {
		*e = errors.New("GET request failed", err)
		return
	}

	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Cookie", creds.Token)
	req.Header.Set("Referer", classUrl)
	resp, err := client.Do(req)
	if err != nil {
		*e = errors.New("failed to get resp", err)
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		*e = errors.New("failed to read resp.Body", err)
		return
	}

	re, err := regexp.Compile("[0-9]+/[0-9]+/[0-9]+")
	if err != nil {
		*e = errors.New("failed to compile regex", err)
		return
	}

	var data resJson
	err = json.Unmarshal(respBody, &data)
	if err != nil {
		*e = errors.New("failed to unmarshal JSON", err)
		return
	}

	b := data.D
	planDiv := `</td><td class='active itm' onclick="DMU.ViewPlan(`
	fileDiv := `<div class='fLinkDiv'><a href='#' onclick="DMU.OpenAttachment(`
	linkDiv := `<a href='javascript:DMU.OpenNewWindow("`
	i, div := nextRes(b, planDiv, fileDiv, linkDiv)

	for i != -1 {
		resource := plat.Resource{}
		resource.Class = class
		resource.Platform = "daymap"
		dateRegion := b[:i]
		b = b[i:]
		dates := re.FindAllString(dateRegion, -1)

		if len(dates) == 0 && strings.Index(b, planDiv) == -1 && strings.Index(b, fileDiv) == -1 {
			*e = errors.Raise(plat.ErrNoDateFound)
			return
		} else if len(dates) > 0 {
			postStr := dates[len(dates)-1]
			resource.Posted, err = time.ParseInLocation("2/01/2006", postStr, creds.Timezone)
			if err != nil {
				*e = errors.New("failed to parse time", err)
				return
			}
		} else {
			if _, err = defs.Get([]byte(b), len(div)); err != nil {
				*e = errors.New("invalid HTML response", err)
				return
			}
			b = b[len(div):]
			i, div = nextRes(b, planDiv, fileDiv, linkDiv)
			continue
		}

		i = len(div)
		if _, err = defs.Get([]byte(b), i); err != nil {
			*e = errors.New("invalid HTML response", err)
			return
		}
		b = b[i:]

		i = strings.Index(b, ");")

		if i == -1 {
			*e = errors.Raise(plat.ErrInvalidResp)
			return
		}

		resource.Id = b[:i]

		if div == fileDiv {
			i = strings.Index(b, "&nbsp;")
			if i == -1 {
				*e = errors.Raise(plat.ErrInvalidResp)
				return
			}

			i += len("&nbsp;")
			b = b[i:]
			i = strings.Index(b, "</a>")
			if i == -1 {
				*e = errors.Raise(plat.ErrInvalidResp)
				return
			}
		} else {
			i += len(`);;"><div class='lpTitle'>`)
			b = b[i:]
			i = strings.Index(b, "</div>")
			if i == -1 {
				*e = errors.Raise(plat.ErrInvalidResp)
				return
			}
		}

		if _, err = defs.Get([]byte(b), i); err != nil {
			*e = errors.New("invalid HTML response", err)
			return
		}
		resource.Name = b[:i]

		if div == fileDiv {
			resource.Link = "https://gihs.daymap.net/daymap/attachment.ashx?ID=" + resource.Id
			resource.Id = "f" + resource.Id
		} else {
			resource.Link = "https://gihs.daymap.net/DayMap/curriculum/plan.aspx?id=" + resource.Id
		}

		resource.Id = id + "-" + resource.Id
		*res = append(*res, resource)
		if _, err = defs.Get([]byte(b), i); err != nil {
			*e = errors.New("invalid HTML response", err)
			return
		}
		b = b[i:]
		i, div = nextRes(b, planDiv, fileDiv, linkDiv)
	}
}

// Get a list of resources from DayMap for a user.
func ListRes(creds User, r chan []plat.Resource, e chan []error) {
	homeUrl := "https://gihs.daymap.net/daymap/student/dayplan.aspx"
	client := &http.Client{}

	req, err := http.NewRequest("GET", homeUrl, nil)
	if err != nil {
		r <- nil
		e <- []error{errors.New("GET request failed", err)}
		return
	}

	req.Header.Set("Cookie", creds.Token)

	resp, err := client.Do(req)
	if err != nil {
		r <- nil
		e <- []error{errors.New("failed to get resp", err)}
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		r <- nil
		e <- []error{errors.New("failed to read resp.Body", err)}
		return
	}

	classes := map[string]string{}
	b := string(respBody)
	i := strings.Index(b, "plans/class.aspx?id=")

	for i != -1 {
		b = b[i:]
		i = len("plans/class.aspx?id=")
		b = b[i:]
		i = strings.Index(b, "'>")

		if i == -1 {
			r <- nil
			e <- []error{errors.Raise(plat.ErrInvalidResp)}
			return
		}

		id := b[:i]
		b = b[i+2:]
		i = strings.Index(b, "</a>")

		if i == -1 {
			r <- nil
			e <- []error{errors.Raise(plat.ErrInvalidResp)}
			return
		}

		class := b[:i]
		b = b[i:]
		classes[class] = id
		i = strings.Index(b, "plans/class.aspx?id=")
	}

	unordered := make([][]plat.Resource, len(classes))
	errs := make([]error, len(classes))
	var wg sync.WaitGroup
	x := 0

	for _, id := range classes {
		wg.Add(1)
		go getClassRes(creds, id, &unordered[x], &errs[x], &wg)
		x++
	}

	wg.Wait()

	if errors.Join(errs...) != nil {
		r <- nil
		e <- errs
		return
	}

	resources := []plat.Resource{}

	for _, resList := range unordered {
		resources = append(resources, resList...)
	}

	r <- resources
	e <- nil
}
