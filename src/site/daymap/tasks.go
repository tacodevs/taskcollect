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

// Retrieve a webpage of all DayMap tasks for a user.
func tasksPage(creds site.User) (string, error) {
	tasksUrl := "https://gihs.daymap.net/daymap/student/assignments.aspx?View=0"
	client := &http.Client{}

	req, err := http.NewRequest("GET", tasksUrl, nil)
	if err != nil {
		return "", errors.New("GET request failed", err)
	}

	req.Header.Set("Cookie", creds.SiteTokens["daymap"])
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("failed to get resp", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("failed to read resp.Body", err)
	}

	taskForm := url.Values{}
	b := string(respBody)
	i := strings.Index(b, "<input ")

	for i != -1 {
		var value string
		b = b[i:]
		i = strings.Index(b, ">")

		if i == -1 {
			return "", errors.Raise(site.ErrInvalidResp)
		}

		inputTag := b[:i]
		b = b[i:]
		i = strings.Index(inputTag, `type="`)

		if i == -1 {
			return "", errors.Raise(site.ErrInvalidResp)
		}

		i += len(`type="`)
		inputType := inputTag[i:]
		i = strings.Index(inputType, `"`)

		if i == -1 {
			return "", errors.Raise(site.ErrInvalidResp)
		}

		inputType = inputType[:i]

		if inputType != "hidden" {
			i = strings.Index(b, "<input ")
			continue
		}

		i = strings.Index(inputTag, `name="`)

		if i == -1 {
			return "", errors.Raise(site.ErrInvalidResp)
		}

		i += len(`name="`)
		name := inputTag[i:]
		i = strings.Index(name, `"`)

		if i == -1 {
			return "", errors.Raise(site.ErrInvalidResp)
		}

		name = name[:i]
		i = strings.Index(inputTag, `value="`)

		if i != -1 {
			i += len(`value="`)
			value = inputTag[i:]
			i = strings.Index(value, `"`)

			if i == -1 {
				return "", errors.Raise(site.ErrInvalidResp)
			}

			value = value[:i]
		}

		taskForm.Set(name, value)
		i = strings.Index(b, "<input ")
	}

	for val := range tasksFormValues {
		taskForm.Set(val, tasksFormValues[val])
	}

	tdata := strings.NewReader(taskForm.Encode())

	fullReq, err := http.NewRequest("POST", tasksUrl, tdata)
	if err != nil {
		return "", errors.New("POST request failed", err)
	}

	fullReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	fullReq.Header.Set("Cookie", creds.SiteTokens["daymap"])

	full, err := client.Do(fullReq)
	if err != nil {
		return "", errors.New("failed to get full resp", err)
	}

	fullBody, err := io.ReadAll(full.Body)
	if err != nil {
		return "", errors.New("failed to read full.Body", err)
	}

	return string(fullBody), nil
}

// Retrieve a list of tasks from DayMap for a user.
func ListTasks(creds site.User, t chan map[string][]site.Task, e chan [][]error) {
	b, err := tasksPage(creds)
	if err != nil {
		t <- nil
		e <- [][]error{{errors.New("failed retrieving tasks page", err)}}
		return
	}

	unsorted := []site.Task{}
	i := strings.Index(b, `href="javascript:ViewAssignment(`)

	for i != -1 {
		task := site.Task{
			Platform: "daymap",
		}

		i += len(`href="javascript:ViewAssignment(`)
		b = b[i:]
		i = strings.Index(b, `)">`)

		if i == -1 {
			t <- nil
			e <- [][]error{{errors.Raise(site.ErrInvalidResp)}}
			return
		}

		task.Id = b[:i]
		b = b[i:]
		task.Link = "https://gihs.daymap.net/daymap/student/assignment.aspx?TaskID=" + task.Id
		i = strings.Index(b, `<td>`)

		if i == -1 {
			t <- nil
			e <- [][]error{{errors.Raise(site.ErrInvalidResp)}}
			return
		}

		i += len(`<td>`)
		b = b[i:]
		i = strings.Index(b, `</td>`)

		if i == -1 {
			t <- nil
			e <- [][]error{{errors.Raise(site.ErrInvalidResp)}}
			return
		}

		task.Class = b[:i]
		i += len(`</td>`)
		b = b[i:]
		i = strings.Index(b, `</td><td>`)

		if i == -1 {
			t <- nil
			e <- [][]error{{errors.Raise(site.ErrInvalidResp)}}
			return
		}

		i += len(`</td><td>`)
		b = b[i:]
		i = strings.Index(b, `</td><td>`)

		if i == -1 {
			t <- nil
			e <- [][]error{{errors.Raise(site.ErrInvalidResp)}}
			return
		}

		task.Name = b[:i]
		i += len(`</td><td>`)
		b = b[i:]
		i = strings.Index(b, `</td><td>`)

		if i == -1 {
			t <- nil
			e <- [][]error{{errors.Raise(site.ErrInvalidResp)}}
			return
		}

		postedString := b[:i]

		task.Posted, err = time.ParseInLocation("2/01/06", postedString, creds.Timezone)
		if err != nil {
			t <- nil
			e <- [][]error{{errors.New("failed to parse time (postedString)", err)}}
			return
		}

		i += len(`</td><td>`)
		b = b[i:]
		i = strings.Index(b, `</td><td>`)

		if i == -1 {
			t <- nil
			e <- [][]error{{errors.Raise(site.ErrInvalidResp)}}
			return
		}

		dueString := b[:i]

		task.Due, err = time.ParseInLocation("2/01/06", dueString, creds.Timezone)
		if err != nil {
			t <- nil
			e <- [][]error{{errors.New("failed to parse time (dueString)", err)}}
			return
		}

		// Due time might not be 23:59:59, but if it is 00:00:00, the task will
		// not be considered an active task even if it is due after 00:00:00.
		// It's better to stay safe and mark overdue tasks as active rather than
		// mark active tasks as overdue.
		task.Due = time.Date(
			task.Due.Year(), task.Due.Month(), task.Due.Day(),
			23, 59, 59, 999999999,
			creds.Timezone,
		)

		i = strings.Index(b, "\n")

		if i == -1 {
			t <- nil
			e <- [][]error{{errors.Raise(site.ErrInvalidResp)}}
			return
		}

		taskLine := b[:i]
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
		i = strings.Index(b, `href="javascript:ViewAssignment(`)
	}

	tasks := map[string][]site.Task{
		"active":    {},
		"notDue":    {},
		"overdue":   {},
		"submitted": {},
	}

	for _, utask := range unsorted {
		if utask.Graded {
			continue
		} else if utask.Submitted {
			tasks["submitted"] = append(tasks["submitted"], utask)
		} else if utask.Due.Before(time.Now()) {
			tasks["overdue"] = append(tasks["overdue"], utask)
		} else {
			tasks["active"] = append(tasks["active"], utask)
		}
	}

	t <- tasks
	e <- nil
}
