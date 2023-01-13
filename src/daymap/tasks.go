package daymap

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"main/errors"
	"main/plat"
)

// Retrieve a webpage of all DayMap tasks for a user.
func tasksPage(creds User) (string, error) {
	tasksUrl := "https://gihs.daymap.net/daymap/student/assignments.aspx?View=0"
	client := &http.Client{}

	req, err := http.NewRequest("GET", tasksUrl, nil)
	if err != nil {
		newErr := errors.NewError("daymap.ListTasks", "GET request failed", err)
		return "", newErr
	}

	req.Header.Set("Cookie", creds.Token)
	resp, err := client.Do(req)
	if err != nil {
		newErr := errors.NewError("daymap.ListTasks", "failed to get resp", err)
		return "", newErr
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		newErr := errors.NewError("daymap.ListTasks", "failed to read resp.Body", err)
		return "", newErr
	}

	taskForm := url.Values{}
	b := string(respBody)
	i := strings.Index(b, "<input ")

	for i != -1 {
		var value string
		b = b[i:]
		i = strings.Index(b, ">")

		if i == -1 {
			return "", errInvalidResp
		}

		inputTag := b[:i]
		b = b[i:]
		i = strings.Index(inputTag, `type="`)

		if i == -1 {
			return "", errInvalidResp
		}

		i += len(`type="`)
		inputType := inputTag[i:]
		i = strings.Index(inputType, `"`)

		if i == -1 {
			return "", errInvalidResp
		}

		inputType = inputType[:i]

		if inputType != "hidden" {
			i = strings.Index(b, "<input ")
			continue
		}

		i = strings.Index(inputTag, `name="`)

		if i == -1 {
			return "", errInvalidResp
		}

		i += len(`name="`)
		name := inputTag[i:]
		i = strings.Index(name, `"`)

		if i == -1 {
			return "", errInvalidResp
		}

		name = name[:i]
		i = strings.Index(inputTag, `value="`)

		if i != -1 {
			i += len(`value="`)
			value = inputTag[i:]
			i = strings.Index(value, `"`)

			if i == -1 {
				return "", errInvalidResp
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
		newErr := errors.NewError("daymap.ListTasks", "POST request failed", err)
		return "", newErr
	}

	fullReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	fullReq.Header.Set("Cookie", creds.Token)

	full, err := client.Do(fullReq)
	if err != nil {
		newErr := errors.NewError("daymap.ListTasks", "failed to get full resp", err)
		return "", newErr
	}

	fullBody, err := io.ReadAll(full.Body)
	if err != nil {
		newErr := errors.NewError("daymap.ListTasks", "failed to real full.Body", err)
		return "", newErr
	}

	return string(fullBody), nil
}

// Retrieve a list of tasks from DayMap for a user.
func ListTasks(creds User, t chan map[string][]plat.Task, e chan error) {
	b, err := tasksPage(creds)
	if err != nil {
		t <- nil
		e <- err
		return
	}

	unsortedTasks := []plat.Task{}
	i := strings.Index(b, `href="javascript:ViewAssignment(`)

	for i != -1 {
		task := plat.Task{
			Platform: "daymap",
		}

		i += len(`href="javascript:ViewAssignment(`)
		b = b[i:]
		i = strings.Index(b, `)">`)

		if i == -1 {
			t <- nil
			e <- errInvalidResp
			return
		}

		task.Id = b[:i]
		b = b[i:]
		task.Link = "https://gihs.daymap.net/daymap/student/assignment.aspx?TaskID=" + task.Id
		i = strings.Index(b, `<td>`)

		if i == -1 {
			t <- nil
			e <- errInvalidResp
			return
		}

		i += len(`<td>`)
		b = b[i:]
		i = strings.Index(b, `</td>`)

		if i == -1 {
			t <- nil
			e <- errInvalidResp
			return
		}

		task.Class = b[:i]
		i += len(`</td>`)
		b = b[i:]
		i = strings.Index(b, `</td><td>`)

		if i == -1 {
			t <- nil
			e <- errInvalidResp
			return
		}

		i += len(`</td><td>`)
		b = b[i:]
		i = strings.Index(b, `</td><td>`)

		if i == -1 {
			t <- nil
			e <- errInvalidResp
			return
		}

		task.Name = b[:i]
		i += len(`</td><td>`)
		b = b[i:]
		i = strings.Index(b, `</td><td>`)

		if i == -1 {
			t <- nil
			e <- errInvalidResp
			return
		}

		postedString := b[:i]

		postedNoTimezone, err := time.Parse("2/01/06", postedString)
		if err != nil {
			newErr := errors.NewError("daymap.ListTasks", "failed to parse time (postedString)", err)
			t <- nil
			e <- newErr
			return
		}

		task.Posted = time.Date(
			postedNoTimezone.Year(),
			postedNoTimezone.Month(),
			postedNoTimezone.Day(),
			postedNoTimezone.Hour(),
			postedNoTimezone.Minute(),
			postedNoTimezone.Second(),
			postedNoTimezone.Nanosecond(),
			creds.Timezone,
		)

		i += len(`</td><td>`)
		b = b[i:]
		i = strings.Index(b, `</td><td>`)

		if i == -1 {
			t <- nil
			e <- errInvalidResp
			return
		}

		dueString := b[:i]

		dueNoTimezone, err := time.Parse("2/01/06", dueString)
		if err != nil {
			newErr := errors.NewError("daymap.ListTasks", "failed to parse time (dueString)", err)
			t <- nil
			e <- newErr
			return
		}

		task.Due = time.Date(
			dueNoTimezone.Year(),
			dueNoTimezone.Month(),
			dueNoTimezone.Day(),
			dueNoTimezone.Hour(),
			dueNoTimezone.Minute(),
			dueNoTimezone.Second(),
			dueNoTimezone.Nanosecond(),
			creds.Timezone,
		)

		i = strings.Index(b, "\n")

		if i == -1 {
			t <- nil
			e <- errInvalidResp
			return
		}

		taskLine := b[:i]
		i = strings.Index(taskLine, `Results have been published`)

		if i != -1 {
			task.Submitted = true
		}

		i = strings.Index(taskLine, `Your work has been received`)

		if i != -1 && !task.Submitted {
			task.Submitted = true
		}

		unsortedTasks = append(unsortedTasks, task)
		i = strings.Index(b, `href="javascript:ViewAssignment(`)
	}

	tasks := map[string][]plat.Task{
		"active":    {},
		"notDue":    {},
		"overdue":   {},
		"submitted": {},
	}

	for x := 0; x < len(unsortedTasks); x++ {
		if unsortedTasks[x].Submitted {
			tasks["submitted"] = append(
				tasks["submitted"],
				unsortedTasks[x],
			)
		} else if unsortedTasks[x].Due.Before(time.Now()) {
			tasks["overdue"] = append(
				tasks["overdue"],
				unsortedTasks[x],
			)
		} else {
			tasks["active"] = append(
				tasks["active"],
				unsortedTasks[x],
			)
		}
	}

	t <- tasks
	e <- nil
}
