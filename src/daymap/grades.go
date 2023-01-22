package daymap

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"main/errors"
	"main/plat"
)

// Return the grade for a DayMap task from a DayMap task webpage.
func findGrade(webpage *string) (plat.TaskGrade, error) {
	var grade string
	var percent float64
	i := strings.Index(*webpage, "Grade:")

	if i != -1 {
		i = strings.Index(*webpage, "TaskGrade'>")

		if i == -1 {
			return plat.TaskGrade{}, errInvalidTaskResp
		}

		*webpage = (*webpage)[i:]
		i = len("TaskGrade'>")
		*webpage = (*webpage)[i:]
		i = strings.Index(*webpage, "</div>")

		if i == -1 {
			return plat.TaskGrade{}, errInvalidTaskResp
		}

		grade = (*webpage)[:i]
		*webpage = (*webpage)[i:]
	}

	i = strings.Index(*webpage, "Mark:")

	if i != -1 {
		i = strings.Index(*webpage, "TaskGrade'>")

		if i == -1 {
			return plat.TaskGrade{}, errInvalidTaskResp
		}

		*webpage = (*webpage)[i:]
		i = len("TaskGrade'>")
		*webpage = (*webpage)[i:]
		i = strings.Index(*webpage, "</div>")

		if i == -1 {
			return plat.TaskGrade{}, errInvalidTaskResp
		}

		markStr := (*webpage)[:i]
		*webpage = (*webpage)[i:]

		x := strings.Index(markStr, " / ")

		if x == -1 {
			return plat.TaskGrade{}, errInvalidTaskResp
		}

		st := markStr[:x]
		sb := markStr[x+3:]

		it, err := strconv.ParseFloat(st, 64)
		if err != nil {
			return plat.TaskGrade{}, errors.NewError("daymap.GetTask", "(1) string to float64 conversion failed", err)
		}

		ib, err := strconv.ParseFloat(sb, 64)
		if err != nil {
			return plat.TaskGrade{}, errors.NewError("daymap.GetTask", "(2) string to float64 conversion failed", err)
		}

		percent = it / ib * 100
	}

	result := plat.TaskGrade{Exists: true, Grade: grade, Mark: percent}
	return result, nil
}

// Retrieve the grade given to a student for a particular DayMap task.
func taskGrade(creds User, id string, result *plat.TaskGrade, e *error, wg *sync.WaitGroup) {
	defer wg.Done()
	taskUrl := "https://gihs.daymap.net/daymap/student/assignment.aspx?TaskID=" + id
	client := &http.Client{}

	req, err := http.NewRequest("GET", taskUrl, nil)
	if err != nil {
		*e = errors.NewError("daymap.GetTask", "GET request failed", err)
		return
	}
	req.Header.Set("Cookie", creds.Token)

	resp, err := client.Do(req)
	if err != nil {
		*e = errors.NewError("daymap.GetTask", "failed to get resp", err)
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		*e = errors.NewError("daymap.GetTask", "failed to read resp.Body", err)
		return
	}
	page := string(respBody)
	*result, *e = findGrade(&page)
}

// Retrieve a list of graded tasks from DayMap for a user.
func GradedTasks(creds User, t chan []plat.Task, e chan [][]error) {
	webpage, err := tasksPage(creds)
	if err != nil {
		t <- nil
		e <- [][]error{{err}}
		return
	}

	page := Page(webpage)
	unsorted := []plat.Task{}
	graded := []string{}
	strErr := ""
	err = page.Advance(`href="javascript:ViewAssignment(`)

	for err == nil {
		var postStr, dueStr, taskLine string
		var local time.Time

		task := plat.Task{
			Platform: "daymap",
		}

		task.Id, err = page.UpTo(`)">`)
			if err != nil { strErr = "failed getting task ID"; break }
		task.Link = "https://gihs.daymap.net/daymap/student/assignment.aspx?TaskID=" + task.Id

		err = page.Advance(`<td>`)
			if err != nil { strErr = "failed advancing past task ID"; break }

		task.Class, err = page.UpTo(`</td>`)
			if err != nil { strErr = "failed getting class name"; break }

		err = page.Advance(`</td>`)
			if err != nil { strErr = "failed advancing past class name"; break }

		err = page.Advance(`</td><td>`)
			if err != nil { strErr = "failed advancing past summative/formative info"; break }

		task.Name, err = page.UpTo(`</td><td>`)
			if err != nil { strErr = "failed getting task name"; break }

		err = page.Advance(`</td><td>`)
			if err != nil { strErr ="failed advancing to post date"; break }

		postStr, err = page.UpTo(`</td><td>`)
		local, err = time.Parse("2/01/06", postStr)
			if err != nil { strErr = "failed to parse post date"; break }
		task.Posted = time.Date(
			local.Year(),
			local.Month(),
			local.Day(),
			local.Hour(),
			local.Minute(),
			local.Second(),
			local.Nanosecond(),
			creds.Timezone,
		)

		err = page.Advance(`</td><td>`)
			if err != nil { strErr = "failed advancing to due date" ; break }

		dueStr, err = page.UpTo(`</td><td>`)
		local, err = time.Parse("2/01/06", dueStr)
			if err != nil { strErr = "failed to parse due date"; break }
		task.Due = time.Date(
			local.Year(),
			local.Month(),
			local.Day(),
			local.Hour(),
			local.Minute(),
			local.Second(),
			local.Nanosecond(),
			creds.Timezone,
		)

		taskLine, err = page.UpTo("\n")
			if err != nil { strErr = "failed getting task info line"; break }

		i := strings.Index(taskLine, `Results have been published`)
		if i != -1 {
			task.Submitted = true
			graded = append(graded, task.Id)
		}

		i = strings.Index(taskLine, `Your work has been received`)
		if i != -1 && !task.Submitted {
			task.Submitted = true
		}

		unsorted = append(unsorted, task)
		err = page.Advance(`href="javascript:ViewAssignment(`)
	}

	if strErr != "" {
		t <- nil
		e <- [][]error{{errors.NewError("daymap.GradedTasks", strErr, err)}}
		return
	}

	wg := sync.WaitGroup{}
	results := make([]plat.TaskGrade, len(graded))
	errs := make([]error, len(graded))

	for i, id := range graded {
		wg.Add(1)
		taskGrade(creds, id, &results[i], &errs[i], &wg)
	}

	wg.Wait()

	if !errors.HasOnly(errs, nil) {
		t <- nil
		e <- [][]error{errs}
		return
	}

	for i, task := range unsorted {
		for j, id := range graded {
			if task.Id == id {
				unsorted[i].Result = results[j]
			}
		}
	}

	tasks := []plat.Task{}

	for _, task := range unsorted {
		if (task.Result != plat.TaskGrade{}) {
			tasks = append(tasks, task)
		}
	}

	t <- tasks
	e <- nil
}
