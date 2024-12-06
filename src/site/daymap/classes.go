package daymap

import (
	"io"
	"net/http"
	"strings"

	"git.sr.ht/~kvo/go-std/errors"

	"main/site"
)

func Classes(user site.User, c chan site.Pair[[]site.Class, error]) {
	var result site.Pair[[]site.Class, error]

	homeUrl := "https://gihs.daymap.net/daymap/student/dayplan.aspx"
	client := &http.Client{}

	req, err := http.NewRequest("GET", homeUrl, nil)
	if err != nil {
		result.Second = errors.New("cannot create classes request", err)
		c <- result
		return
	}

	req.Header.Set("Cookie", user.SiteTokens["daymap"])

	resp, err := client.Do(req)
	if err != nil {
		result.Second = errors.New("cannot execute classes request", err)
		c <- result
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Second = errors.New("cannot read classes response body", err)
		c <- result
		return
	}

	var classes []site.Class
	page := string(body)
	i := strings.Index(page, "plans/class.aspx?id=")

	for i != -1 {
		var class site.Class
		page = page[i:]
		i = len("plans/class.aspx?id=")
		page = page[i:]
		i = strings.Index(page, "'>")

		if i == -1 {
			result.Second = errors.New("invalid HTML response", nil)
			c <- result
			return
		}

		class.Id = page[:i]
		page = page[i+2:]
		i = strings.Index(page, "</a>")

		if i == -1 {
			result.Second = errors.New("invalid HTML response", nil)
			c <- result
			return
		}

		class.Name = page[:i]
		page = page[i:]
		class.Link = "https://gihs.daymap.net/daymap/student/plans/class.aspx?id=" + class.Id
		class.Platform = "daymap"
		classes = append(classes, class)
		i = strings.Index(page, "plans/class.aspx?id=")
	}

	result.First = classes
	c <- result
}
