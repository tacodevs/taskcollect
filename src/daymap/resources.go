package daymap

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"main/errors"
)

// Returns the next school resource type and its index, as well as a corresponding planDiv or fileDiv.
func nextRes(buf, planDiv, fileDiv string) (bool, int, string) {
	planIdx := strings.Index(buf, planDiv)
	fileIdx := strings.Index(buf, fileDiv)

	if planIdx == -1 && fileIdx == -1 {
		return false, -1, ""
	} else if fileIdx < planIdx && fileIdx != -1 || planIdx == -1 {
		return true, fileIdx, fileDiv
	} else {
		return false, planIdx, planDiv
	}
}

// Get a list of resources for a DayMap class.
func getClassRes(creds User, class, id string, res *[]Resource, wg *sync.WaitGroup, e chan error) {
	defer wg.Done()
	classUrl := "https://gihs.daymap.net/daymap/student/plans/class.aspx?id=" + id
	client := &http.Client{}

	req, err := http.NewRequest("GET", classUrl, nil)
	if err != nil {
		newErr := errors.NewError("daymap: getClassRes", "GET request failed", err)
		e <- newErr
		return
	}

	req.Header.Set("Cookie", creds.Token)
	resp, err := client.Do(req)
	if err != nil {
		newErr := errors.NewError("daymap: getClassRes", "failed to get resp", err)
		e <- newErr
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		newErr := errors.NewError("daymap: getClassRes", "failed to read resp.Body", err)
		e <- newErr
		return
	}

	re, err := regexp.Compile("[0-9]+/[0-9]+/[0-9]+")

	if err != nil {
		newErr := errors.NewError("daymap: getClassRes", "failed to compile regex", err)
		e <- newErr
		return
	}

	planDiv := "</div><div class='lpTitle'><a href='javascript:DMU.ViewPlan("
	fileDiv := `<div class='fLinkDiv'><a href='#' onclick="DMU.OpenAttachment(`
	b := string(respBody)
	isFile, i, div := nextRes(b, planDiv, fileDiv)
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
				newErr := errors.NewError("daymap: getClassRes", "failed to parse time", err)
				e <- newErr
				return
			}
		} else {
			b = b[len(div):]
			isFile, i, div = nextRes(b, planDiv, fileDiv)
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

		if isFile {
			i = strings.Index(b, "&nbsp;")

			if i == -1 {
				e <- errInvalidResp
				return
			}

			b = b[i+6:]
		} else {
			b = b[i+4:]
		}

		i = strings.Index(b, "</a>")
		if i == -1 {
			e <- errInvalidResp
			return
		}

		resource.Name = b[:i]
		b = b[i:]

		if isFile {
			resource.Link = "https://gihs.daymap.net/daymap/attachment.ashx?ID=" + resource.Id
			resource.Id = "f" + resource.Id
		} else {
			resource.Link = "https://gihs.daymap.net/DayMap/curriculum/plan.aspx?id=" + resource.Id
		}

		*res = append(*res, resource)
		b = b[i:]
		isFile, i, div = nextRes(b, planDiv, fileDiv)
	}
}

// Get a list of resources from DayMap for a user.
func ListRes(creds User, r chan []Resource, e chan error) {
	homeUrl := "https://gihs.daymap.net/daymap/student/dayplan.aspx"
	client := &http.Client{}

	req, err := http.NewRequest("GET", homeUrl, nil)
	if err != nil {
		newErr := errors.NewError("daymap: ListRes", "GET request failed", err)
		r <- nil
		e <- newErr
		return
	}

	req.Header.Set("Cookie", creds.Token)

	resp, err := client.Do(req)
	if err != nil {
		newErr := errors.NewError("daymap: ListRes", "failed to get resp", err)
		r <- nil
		e <- newErr
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		newErr := errors.NewError("daymap: ListRes", "failed to read resp.Body", err)
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

	for class, id := range classes {
		wg.Add(1)
		go getClassRes(creds, class, id, &unordered[x], &wg, errChan)
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
