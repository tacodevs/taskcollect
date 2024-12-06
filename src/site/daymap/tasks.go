package daymap

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.sr.ht/~kvo/go-std/errors"

	"main/site"
)

func tasksPage(user site.User) (string, error) {
	link := "https://gihs.daymap.net/daymap/student/assignments.aspx?View=0"
	client := &http.Client{}

	s1req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return "", errors.New("cannot create stage 1 request", err)
	}

	s1req.Header.Set("Cookie", user.SiteTokens["daymap"])

	s1, err := client.Do(s1req)
	if err != nil {
		return "", errors.New("cannot execute stage 1 request", err)
	}

	s1body, err := io.ReadAll(s1.Body)
	if err != nil {
		return "", errors.New("cannot read stage 1 body", err)
	}

	form := url.Values{}
	s1page := string(s1body)
	i := strings.Index(s1page, "<input ")

	for i != -1 {
		var value string
		s1page = s1page[i:]
		i = strings.Index(s1page, ">")

		if i == -1 {
			return "", errors.New("invalid HTML response", nil)
		}

		inputTag := s1page[:i]
		s1page = s1page[i:]
		i = strings.Index(inputTag, `type="`)

		if i == -1 {
			return "", errors.New("invalid HTML response", nil)
		}

		i += len(`type="`)
		inputType := inputTag[i:]
		i = strings.Index(inputType, `"`)

		if i == -1 {
			return "", errors.New("invalid HTML response", nil)
		}

		inputType = inputType[:i]

		if inputType != "hidden" {
			i = strings.Index(s1page, "<input ")
			continue
		}

		i = strings.Index(inputTag, `name="`)

		if i == -1 {
			return "", errors.New("invalid HTML response", nil)
		}

		i += len(`name="`)
		name := inputTag[i:]
		i = strings.Index(name, `"`)

		if i == -1 {
			return "", errors.New("invalid HTML response", nil)
		}

		name = name[:i]
		i = strings.Index(inputTag, `value="`)

		if i != -1 {
			i += len(`value="`)
			value = inputTag[i:]
			i = strings.Index(value, `"`)

			if i == -1 {
				return "", errors.New("invalid HTML response", nil)
			}

			value = value[:i]
		}

		form.Set(name, value)
		i = strings.Index(s1page, "<input ")
	}

	for k, v := range auxValues {
		form.Set(k, v)
	}

	data := strings.NewReader(form.Encode())

	s2req, err := http.NewRequest("POST", link, data)
	if err != nil {
		return "", errors.New("cannot create stage 2 request", err)
	}

	s2req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s2req.Header.Set("Cookie", user.SiteTokens["daymap"])

	s2, err := client.Do(s2req)
	if err != nil {
		return "", errors.New("cannot execute stage 2 request", err)
	}

	s2body, err := io.ReadAll(s2.Body)
	if err != nil {
		return "", errors.New("cannot read stage 2 body", err)
	}

	return string(s2body), nil
}

func Tasks(user site.User, c chan site.Pair[[]site.Task, error], classes []site.Class) {
	var result site.Pair[[]site.Task, error]
	var tasks []site.Task

	page, err := tasksPage(user)
	if err != nil {
		result.Second = errors.New("cannot fetch tasks page", err)
		c <- result
		return
	}

	var unsorted []site.Task
	i := strings.Index(page, `href="javascript:ViewAssignment(`)

	for i != -1 {
		task := site.Task{
			Platform: "daymap",
		}

		i += len(`href="javascript:ViewAssignment(`)
		page = page[i:]
		i = strings.Index(page, `)">`)

		if i == -1 {
			result.Second = errors.New("invalid HTML response", nil)
			c <- result
			return
		}

		task.Id = page[:i]
		page = page[i:]
		task.Link = "https://gihs.daymap.net/daymap/student/assignment.aspx?TaskID=" + task.Id
		i = strings.Index(page, `<td>`)

		if i == -1 {
			result.Second = errors.New("invalid HTML response", nil)
			c <- result
			return
		}

		i += len(`<td>`)
		page = page[i:]
		i = strings.Index(page, `</td>`)

		if i == -1 {
			result.Second = errors.New("invalid HTML response", nil)
			c <- result
			return
		}

		task.Class = page[:i]
		i += len(`</td>`)
		page = page[i:]
		i = strings.Index(page, `</td><td>`)

		if i == -1 {
			result.Second = errors.New("invalid HTML response", nil)
			c <- result
			return
		}

		i += len(`</td><td>`)
		page = page[i:]
		i = strings.Index(page, `</td><td>`)

		if i == -1 {
			result.Second = errors.New("invalid HTML response", nil)
			c <- result
			return
		}

		task.Name = page[:i]
		i += len(`</td><td>`)
		page = page[i:]
		i = strings.Index(page, `</td><td>`)

		if i == -1 {
			result.Second = errors.New("invalid HTML response", nil)
			c <- result
			return
		}

		postedStr := page[:i]

		task.Posted, err = time.ParseInLocation("2/01/06", postedStr, user.Timezone)
		if err != nil {
			result.Second = errors.New("invalid HTML response", nil)
			c <- result
			return
		}

		i += len(`</td><td>`)
		page = page[i:]
		i = strings.Index(page, `</td><td>`)

		if i == -1 {
			result.Second = errors.New("invalid HTML response", nil)
			c <- result
			return
		}

		dueStr := page[:i]

		task.Due, err = time.ParseInLocation("2/01/06", dueStr, user.Timezone)
		if err != nil {
			result.Second = errors.New("invalid HTML response", nil)
			c <- result
			return
		}

		// Due time might not be 23:59:59, but if it is 00:00:00, the task will
		// not be considered an active task even if it is due after 00:00:00.
		// It's better to stay safe and mark overdue tasks as active rather than
		// mark active tasks as overdue.
		task.Due = time.Date(
			task.Due.Year(), task.Due.Month(), task.Due.Day(),
			23, 59, 59, 999999999,
			user.Timezone,
		)

		i = strings.Index(page, "\n")

		if i == -1 {
			result.Second = errors.New("invalid HTML response", nil)
			c <- result
			return
		}

		taskLine := page[:i]
		i = strings.Index(taskLine, `Results have been published`)

		if i != -1 {
			task.Submitted = true
			task.Graded = true
		}

		i = strings.Index(taskLine, `Your work has been received`)

		if i != -1 && !task.Submitted {
			task.Submitted = true
		}

		unsorted = append(unsorted, task)
		i = strings.Index(page, `href="javascript:ViewAssignment(`)
	}

	// BUG: matching tasks to classes by class name may lead to collisions if
	// several classes share a name. Sadly the site.Task struct does not store
	// class ID...
	for _, task := range unsorted {
		for _, class := range classes {
			if task.Class == class.Name {
				tasks = append(tasks, task)
			}
		}
	}

	result.First = tasks
	c <- result
}
