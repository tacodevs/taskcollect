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

	"main/errors"
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
		newErr := errors.NewError("daymap.auxClassInfo", "GET request failed", err)
		return "", "", newErr
	}

	req.Header.Set("Cookie", creds.Token)
	resp, err := client.Do(req)
	if err != nil {
		newErr := errors.NewError("daymap.auxClassInfo", "failed to get resp", err)
		return "", "", newErr
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		newErr := errors.NewError("daymap.auxClassInfo", "failed to read resp.Body", err)
		return "", "", newErr
	}

	page := string(respBody)
	re, err := regexp.Compile(`new Classroom\([0-9]+,null,[0-9]+,`)
	if err != nil {
		newErr := errors.NewError("daymap.auxClassInfo", "failed to compile regex", err)
		return "", "", newErr
	}
	courseId := strings.Split(re.FindString(page), ",")[2]

	classDiv := `<td><span id="ctl00_ctl00_cp_cp_divHeader" class="Header14" style="padding-left: 20px">`
	i := strings.Index(page, classDiv)
	if i == -1 {
		newErr := errors.NewError("daymap.auxClassInfo", "can't find class name", errInvalidResp)
		return "", "", newErr
	}
	i += len(classDiv)
	page = page[i:]
	i = strings.Index(page, "</span>")
	if i == -1 {
		newErr := errors.NewError("daymap.auxClassInfo", "can't find class name end", errInvalidResp)
		return "", "", newErr
	}
	class := page[:i]

	return class, courseId, nil
}

// Get a list of resources for a DayMap class.
func getClassRes(creds User, id string, res *[]Resource, wg *sync.WaitGroup, e chan error) {
	defer wg.Done()
	resUrl := "https://gihs.daymap.net/daymap/student/plans/class.aspx/InitialiseResources"
	classUrl := "https://gihs.daymap.net/daymap/student/plans/class.aspx?id=" + id

	class, courseId, err := auxClassInfo(creds, classUrl)
	if err != nil {
		newErr := errors.NewError("daymap.getClassRes", "failed retrieving secondary class ID", err)
		e <- newErr
		return
	}

	jsonReq := fmt.Sprintf(`{"classId":%s,"courseId":%s}`, id, courseId)
	client := &http.Client{}

	req, err := http.NewRequest("POST", resUrl, strings.NewReader(jsonReq))
	if err != nil {
		newErr := errors.NewError("daymap.getClassRes", "GET request failed", err)
		e <- newErr
		return
	}

	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Cookie", creds.Token)
	req.Header.Set("Referer", classUrl)
	resp, err := client.Do(req)
	if err != nil {
		newErr := errors.NewError("daymap.getClassRes", "failed to get resp", err)
		e <- newErr
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		newErr := errors.NewError("daymap.getClassRes", "failed to read resp.Body", err)
		e <- newErr
		return
	}

	re, err := regexp.Compile("[0-9]+/[0-9]+/[0-9]+")

	if err != nil {
		newErr := errors.NewError("daymap.getClassRes", "failed to compile regex", err)
		e <- newErr
		return
	}

	var data resJson
	err = json.Unmarshal(respBody, &data)
	if err != nil {
		newErr := errors.NewError("daymap.getClassRes", "failed to unmarshal JSON", err)
		e <- newErr
		return
	}

	b := data.D
	planDiv := `</td><td class='active itm' onclick="DMU.ViewPlan(`
	fileDiv := `<div class='fLinkDiv'><a href='#' onclick="DMU.OpenAttachment(`
	linkDiv := `<a href='javascript:DMU.OpenNewWindow("`
	i, div := nextRes(b, planDiv, fileDiv, linkDiv)
	var posted time.Time

	for i != -1 {
		resource := Resource{}
		resource.Class = class
		resource.Platform = "daymap"

		dateRegion := b[:i]
		b = b[i:]
		dates := re.FindAllString(dateRegion, -1)

		if dates == nil && strings.Index(b, planDiv) == -1 && strings.Index(b, fileDiv) == -1 {
			e <- errNoDateFound
			return
		} else if dates != nil {
			postStr := dates[len(dates)-1]
			posted, err = time.Parse("2/01/2006", postStr)
			if err != nil {
				newErr := errors.NewError("daymap.getClassRes", "failed to parse time", err)
				e <- newErr
				return
			}
		} else {
			b = b[len(div):]
			i, div = nextRes(b, planDiv, fileDiv, linkDiv)
			continue
		}

		i = len(div)
		b = b[i:]

		resource.Posted = time.Date(
			posted.Year(), posted.Month(), posted.Day(),
			0, 0, 0, 0,
			creds.Timezone,
		)

		i = strings.Index(b, ");")

		if i == -1 {
			e <- errInvalidResp
			return
		}

		resource.Id = b[:i]

		if div == fileDiv {
			i = strings.Index(b, "&nbsp;")
			if i == -1 {
				e <- errInvalidResp
				return
			}

			i += len("&nbsp;")
			b = b[i:]
			i = strings.Index(b, "</a>")
			if i == -1 {
				e <- errInvalidResp
				return
			}
		} else {
			i += len(`);;"><div class='lpTitle'>`)
			b = b[i:]
			i = strings.Index(b, "</div>")

			if i == -1 {
				e <- errInvalidResp
				return
			}
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
		b = b[i:]
		i, div = nextRes(b, planDiv, fileDiv, linkDiv)
	}
}

// Get a list of resources from DayMap for a user.
func ListRes(creds User, r chan []Resource, e chan error) {
	homeUrl := "https://gihs.daymap.net/daymap/student/dayplan.aspx"
	client := &http.Client{}

	req, err := http.NewRequest("GET", homeUrl, nil)
	if err != nil {
		newErr := errors.NewError("daymap.ListRes", "GET request failed", err)
		r <- nil
		e <- newErr
		return
	}

	req.Header.Set("Cookie", creds.Token)

	resp, err := client.Do(req)
	if err != nil {
		newErr := errors.NewError("daymap.ListRes", "failed to get resp", err)
		r <- nil
		e <- newErr
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		newErr := errors.NewError("daymap.ListRes", "failed to read resp.Body", err)
		r <- nil
		e <- newErr
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
			e <- errInvalidResp
			return
		}

		id := b[:i]
		b = b[i+2:]
		i = strings.Index(b, "</a>")

		if i == -1 {
			r <- nil
			e <- errInvalidResp
			return
		}

		class := b[:i]
		b = b[i:]
		classes[class] = id
		i = strings.Index(b, "plans/class.aspx?id=")
	}

	unordered := make([][]Resource, len(classes))
	errChan := make(chan error)
	var wg sync.WaitGroup
	x := 0

	for _, id := range classes {
		wg.Add(1)
		go getClassRes(creds, id, &unordered[x], &wg, errChan)
		x++
	}

	wg.Wait()

	select {
	case err = <-errChan:
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
