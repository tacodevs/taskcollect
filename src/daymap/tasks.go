package daymap

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"main/errors"
)

const PAGE_SIZE = "1000000"

var taskFormValues = map[string]string{
	`ctl00_ctl00_cp_cp_grdAssignments_ctl00_ctl03_ctl01_PageSizeComboBox_ClientState`: `{"logEntries":[],"value":"50","text":"50","enabled":true,"checkedIndices":[],"checkedItemsTextOverflows":false}`,
	`ctl00$ctl00$cp$cp$grdAssignments$ctl00$ctl03$ctl01$PageSizeComboBox`:             PAGE_SIZE,
	`__EVENTTARGET`:                       `ctl00$ctl00$cp$cp$grdAssignments`,
	`__EVENTARGUMENT`:                     `FireCommand:ctl00$ctl00$cp$cp$grdAssignments$ctl00;PageSize;` + PAGE_SIZE,
	`ctl00_ctl00_cp_cp_ScriptManager_TSM`: `;;System.Web.Extensions, Version=4.0.0.0, Culture=neutral, PublicKeyToken=31bf3856ad364e35:en-AU:9ddf364d-d65d-4f01-a69e-8b015049e026:ea597d4b:b25378d2;Telerik.Web.UI, Version=2020.1.219.45, Culture=neutral, PublicKeyToken=121fae78165ba3d4:en-AU:bb184598-9004-47ca-9e82-5def416be84b:16e4e7cd:33715776:58366029:f7645509:24ee1bba:f46195d3:2003d0b8:c128760b:88144a7a:1e771326:aa288e2d:258f1c72`,
}

// Retrieve a list of tasks from DayMap for a user.
func ListTasks(creds User, t chan map[string][]Task, e chan error) {
	tasksUrl := "https://gihs.daymap.net/daymap/student/assignments.aspx?View=0"
	client := &http.Client{}

	req, err := http.NewRequest("GET", tasksUrl, nil)
	if err != nil {
		newErr := errors.NewError("daymap: ListTasks", "GET request failed", err)
		t <- nil
		e <- newErr
		return
	}

	req.Header.Set("Cookie", creds.Token)
	resp, err := client.Do(req)
	if err != nil {
		newErr := errors.NewError("daymap: ListTasks", "failed to get resp", err)
		t <- nil
		e <- newErr
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		newErr := errors.NewError("daymap: ListTasks", "failed to read resp.Body", err)
		t <- nil
		e <- newErr
		return
	}

	taskForm := url.Values{}
	b := string(respBody)
	i := strings.Index(b, "<input ")

	for i != -1 {
		var value string
		b = b[i:]
		i = strings.Index(b, ">")

		if i == -1 {
			t <- nil
			e <- errInvalidResp
			return
		}

		inputTag := b[:i]
		b = b[i:]
		i = strings.Index(inputTag, `type="`)

		if i == -1 {
			t <- nil
			e <- errInvalidResp
			return
		}

		i += len(`type="`)
		inputType := inputTag[i:]
		i = strings.Index(inputType, `"`)

		if i == -1 {
			t <- nil
			e <- errInvalidResp
			return
		}

		inputType = inputType[:i]

		if inputType != "hidden" {
			i = strings.Index(b, "<input ")
			continue
		}

		i = strings.Index(inputTag, `name="`)

		if i == -1 {
			t <- nil
			e <- errInvalidResp
			return
		}

		i += len(`name="`)
		name := inputTag[i:]
		i = strings.Index(name, `"`)

		if i == -1 {
			t <- nil
			e <- errInvalidResp
			return
		}

		name = name[:i]
		i = strings.Index(inputTag, `value="`)

		if i != -1 {
			i += len(`value="`)
			value = inputTag[i:]
			i = strings.Index(value, `"`)

			if i == -1 {
				t <- nil
				e <- errInvalidResp
				return
			}

			value = value[:i]
		}

		taskForm.Set(name, value)
		i = strings.Index(b, "<input ")
	}

	for val := range taskFormValues {
		taskForm.Set(val, taskFormValues[val])
	}

	tdata := strings.NewReader(taskForm.Encode())

	fullReq, err := http.NewRequest("POST", tasksUrl, tdata)
	if err != nil {
		newErr := errors.NewError("daymap: ListTasks", "POST request failed", err)
		t <- nil
		e <- newErr
		return
	}

	fullReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	fullReq.Header.Set("Cookie", creds.Token)

	full, err := client.Do(fullReq)
	if err != nil {
		newErr := errors.NewError("daymap: ListTasks", "failed to get full resp", err)
		t <- nil
		e <- newErr
		return
	}

	fullBody, err := io.ReadAll(full.Body)
	if err != nil {
		newErr := errors.NewError("daymap: ListTasks", "failed to real full.Body", err)
		t <- nil
		e <- newErr
		return
	}

	b = string(fullBody)
	unsortedTasks := []Task{}
	i = strings.Index(b, `href="javascript:ViewAssignment(`)

	for i != -1 {
		task := Task{
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
			newErr := errors.NewError("daymap: ListTasks", "failed to parse time (postedString)", err)
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
			newErr := errors.NewError("daymap: ListTasks", "failed to parse time (dueString)", err)
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

	tasks := map[string][]Task{
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
