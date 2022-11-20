package daymap

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
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

// Get a list of resources for a DayMap class.
func getClassRes(creds User, class, id string, res *[]Resource, wg *sync.WaitGroup, e chan error) {
	defer wg.Done()
	classUrl := "https://gihs.daymap.net/daymap/student/plans/class.aspx?id=" + id
	client := &http.Client{}

	req, err := http.NewRequest("GET", classUrl, nil)
	if err != nil {
		e <- err
		return
	}

	req.Header.Set("Cookie", creds.Token)
	resp, err := client.Do(req)
	if err != nil {
		e <- err
		return
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		e <- err
		return
	}

	div := "</div><div class='lpTitle'><a href='javascript:DMU.ViewPlan("
	b := string(respBody)
	i := strings.Index(b, div)

	for i != -1 {
		resource := Resource{}
		resource.Class = class
		resource.Platform = "daymap"

		dateRegion := b[:i]
		b = b[i:]

		re, err := regexp.Compile("[0-9]+/[0-9]+/[0-9]+")

		if err != nil {
			e <- err
			return
		}

		dates := re.FindAllString(dateRegion, -1)

		if dates == nil {
			e <- errNoDateFound
			return
		}

		postStr := dates[len(dates)-1]
		posted, err := time.Parse("2/01/2006", postStr)

		if err != nil {
			e <- err
			return
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
		b = b[i+4:]
		i = strings.Index(b, "</a>")

		if i == -1 {
			e <- errInvalidResp
			return
		}

		resource.Name = b[:i]
		b = b[i:]
		resource.Link = "https://gihs.daymap.net/DayMap/curriculum/plan.aspx?id=" + resource.Id
		*res = append(*res, resource)
		b = b[i:]
		i = strings.Index(b, div)
	}
}

// Public function to get a list of resources from DayMap for a user.
func ListRes(creds User, r chan []Resource, e chan error) {
	homeUrl := "https://gihs.daymap.net/daymap/student/dayplan.aspx"
	client := &http.Client{}

	req, err := http.NewRequest("GET", homeUrl, nil)
	if err != nil {
		r <- nil
		e <- err
		return
	}

	req.Header.Set("Cookie", creds.Token)

	resp, err := client.Do(req)
	if err != nil {
		r <- nil
		e <- err
		return
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		r <- nil
		e <- err
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
		for _, r := range resList {
			resources = append(resources, r)
		}
	}

	r <- resources
	e <- err
}
