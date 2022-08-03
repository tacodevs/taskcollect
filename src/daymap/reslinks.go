package daymap

import (
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

/*
ISSUE: For some reason, updating *r from classRes does not seem to affect res in
ResLinks.
*/

func classRes(creds User, id string, r *[][2]string, wg *sync.WaitGroup, e chan error) {
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
		b = b[i:]
		i = len(div)
		b = b[i:]
		i = strings.Index(b, ");")

		if i == -1 {
			e <- errInvalidResp
			return
		}

		resId := b[:i]
		b = b[i+4:]
		i = strings.Index(b, "</a>")

		if i == -1 {
			e <- errInvalidResp
			return
		}

		name := b[:i]
		b = b[i:]
		link := "https://gihs.daymap.net/DayMap/curriculum/plan.aspx?id=" + resId
		*r = append(*r, [2]string{link, name})
		b = b[i:]
		i = strings.Index(b, div)
	}
}

func ResLinks(creds User, r chan map[string][][2]string, e chan error) {
	homeUrl := "https://gihs.daymap.net/daymap/student/dayplan.aspx"
	resLinks := map[string][][2]string{}

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

	res := make([][][2]string, len(classes))
	errChan := make(chan error)
	var wg sync.WaitGroup
	x := 0

	for _, id := range classes {
		wg.Add(1)
		go classRes(creds, id, &res[x], &wg, errChan)
		x++
	}

	x = 0

	for class := range classes {
		resLinks[class] = res[x]
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

	r <- resLinks
	e <- err
}
