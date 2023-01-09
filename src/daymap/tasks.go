package daymap

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"main/errors"
)

// Retrieve the grade given to a student for a particular DayMap task.
func taskGrade(creds User, id string, grade *string, e *error, wg *sync.WaitGroup) {
	defer wg.Done()
	taskUrl := "https://gihs.daymap.net/daymap/student/assignment.aspx?TaskID=" + id
	client := &http.Client{}

	req, err := http.NewRequest("GET", taskUrl, nil)
	if err != nil {
		*e = errors.NewError("daymap: GetTask", "GET request failed", err)
		return
	}

	req.Header.Set("Cookie", creds.Token)

	resp, err := client.Do(req)
	if err != nil {
		*e = errors.NewError("daymap: GetTask", "failed to get resp", err)
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		*e = errors.NewError("daymap: GetTask", "failed to read resp.Body", err)
		return
	}

	b := string(respBody)
	i := strings.Index(b, "Grade:")

	if i != -1 {
		i = strings.Index(b, "TaskGrade'>")

		if i == -1 {
			*e = errInvalidTaskResp
			return
		}

		b = b[i:]
		i = len("TaskGrade'>")
		b = b[i:]
		i = strings.Index(b, "</div>")

		if i == -1 {
			*e = errInvalidTaskResp
			return
		}

		*grade = b[:i]
		b = b[i:]
	}

	i = strings.Index(b, "Mark:")

	if i != -1 {
		i = strings.Index(b, "TaskGrade'>")

		if i == -1 {
			*e = errInvalidTaskResp
			return
		}

		b = b[i:]
		i = len("TaskGrade'>")
		b = b[i:]
		i = strings.Index(b, "</div>")

		if i == -1 {
			*e = errInvalidTaskResp
			return
		}

		markStr := b[:i]
		b = b[i:]

		x := strings.Index(markStr, " / ")

		if x == -1 {
			*e = errInvalidTaskResp
			return
		}

		st := markStr[:x]
		sb := markStr[x+3:]

		it, err := strconv.ParseFloat(st, 64)
		if err != nil {
			*e = errors.NewError("daymap: GetTask", "(1) string to float64 conversion failed", err)
			return
		}

		ib, err := strconv.ParseFloat(sb, 64)
		if err != nil {
			*e = errors.NewError("daymap: GetTask", "(2) string to float64 conversion failed", err)
			return
		}

		percent := it/ib * 100

		if *grade == "" {
			*grade = fmt.Sprintf("%.f%%", percent)
		} else {
			*grade += fmt.Sprintf(" (%.f%%)", percent)
		}
	}

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

	for val := range tasksFormValues {
		taskForm.Set(val, tasksFormValues[val])
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
	graded := []string{}

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
			graded = append(graded, task.Id)
		}

		i = strings.Index(taskLine, `Your work has been received`)

		if i != -1 && !task.Submitted {
			task.Submitted = true
		}

		unsortedTasks = append(unsortedTasks, task)
		i = strings.Index(b, `href="javascript:ViewAssignment(`)
	}

	wg := sync.WaitGroup{}
	grades := make([]string, len(graded))
	errs := make([]error, len(graded))

	for i, id := range graded {
		wg.Add(1)
		go taskGrade(creds, id, &grades[i], &errs[i], &wg)
	}

	wg.Wait()

	if !errors.HasOnly(errs, nil) {
		t <- nil
		// TODO: Return all errs to higher call frame.
		e <- errGetGradesFailed
		return
	}

	for i, task := range unsortedTasks {
		for j, id := range graded {
			if task.Id == id {
				unsortedTasks[i].Grade = grades[j]
			}
		}
	}

	tasks := map[string][]Task{
		"active":    {},
		"notDue":    {},
		"overdue":   {},
		"submitted": {},
		"graded":    {},
	}

	for x := 0; x < len(unsortedTasks); x++ {
		if unsortedTasks[x].Grade != "" {
			tasks["graded"] = append(
				tasks["graded"],
				unsortedTasks[x],
			)
		} else if unsortedTasks[x].Submitted {
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
