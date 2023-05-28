package daymap

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"codeberg.org/kvo/std/errors"

	"main/plat"
)

type taskGrade struct {
	Exists bool
	Grade  string
	Mark   float64
}

// Return the grade for a DayMap task from a DayMap task webpage.
func findGrade(webpage *string) (taskGrade, errors.Error) {
	var grade string
	var percent float64
	i := strings.Index(*webpage, "Grade:")

	if i != -1 {
		i = strings.Index(*webpage, "TaskGrade'>")

		if i == -1 {
			return taskGrade{}, plat.ErrInvalidTaskResp.Here()
		}

		*webpage = (*webpage)[i:]
		i = len("TaskGrade'>")
		*webpage = (*webpage)[i:]
		i = strings.Index(*webpage, "</div>")

		if i == -1 {
			return taskGrade{}, plat.ErrInvalidTaskResp.Here()
		}

		grade = (*webpage)[:i]
		*webpage = (*webpage)[i:]
	}

	i = strings.Index(*webpage, "Mark:")

	if i != -1 {
		i = strings.Index(*webpage, "TaskGrade'>")

		if i == -1 {
			return taskGrade{}, plat.ErrInvalidTaskResp.Here()
		}

		*webpage = (*webpage)[i:]
		i = len("TaskGrade'>")
		*webpage = (*webpage)[i:]
		i = strings.Index(*webpage, "</div>")

		if i == -1 {
			return taskGrade{}, plat.ErrInvalidTaskResp.Here()
		}

		markStr := (*webpage)[:i]
		*webpage = (*webpage)[i:]

		x := strings.Index(markStr, " / ")

		if x == -1 {
			return taskGrade{}, plat.ErrInvalidTaskResp.Here()
		}

		st := markStr[:x]
		sb := markStr[x+3:]

		it, err := strconv.ParseFloat(st, 64)
		if err != nil {
			return taskGrade{}, errors.New(
				"(1) string to float64 conversion failed",
				errors.New(err.Error(), nil),
			)
		}

		ib, err := strconv.ParseFloat(sb, 64)
		if err != nil {
			return taskGrade{}, errors.New(
				"(2) string to float64 conversion failed",
				errors.New(err.Error(), nil),
			)
		}

		percent = it / ib * 100
	}

	result := taskGrade{Exists: true, Grade: grade, Mark: percent}
	return result, nil
}

// Retrieve the grade given to a student for a particular DayMap task.
func getGrade(creds plat.User, id string, result *taskGrade, e *errors.Error, wg *sync.WaitGroup) {
	defer wg.Done()
	taskUrl := "https://gihs.daymap.net/daymap/student/assignment.aspx?TaskID=" + id
	client := &http.Client{}

	req, err := http.NewRequest("GET", taskUrl, nil)
	if err != nil {
		*e = errors.New(
			"GET request failed",
			errors.New(err.Error(), nil),
		)
		return
	}
	req.Header.Set("Cookie", creds.SiteTokens["daymap"])

	resp, err := client.Do(req)
	if err != nil {
		*e = errors.New(
			"failed to get resp",
			errors.New(err.Error(), nil),
		)
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		*e = errors.New(
			"failed to read resp.Body",
			errors.New(err.Error(), nil),
		)
		return
	}
	page := string(respBody)
	*result, *e = findGrade(&page)
}

// Retrieve a list of graded tasks from DayMap for a user.
func Graded(creds plat.User, c chan []plat.Task, ok chan errors.Error, done *int) {
	var tasks []plat.Task
	var err errors.Error

	defer plat.Deliver(c, &tasks, done)
	defer plat.Deliver(ok, &err, done)
	defer plat.Done(done)

	webpage, err := tasksPage(creds)
	if err != nil {
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
		if err != nil {
			strErr = "failed getting task ID"
			break
		}
		task.Link = "https://gihs.daymap.net/daymap/student/assignment.aspx?TaskID=" + task.Id

		err = page.Advance(`<td>`)
		if err != nil {
			strErr = "failed advancing past task ID"
			break
		}

		task.Class, err = page.UpTo(`</td>`)
		if err != nil {
			strErr = "failed getting class name"
			break
		}

		err = page.Advance(`</td>`)
		if err != nil {
			strErr = "failed advancing past class name"
			break
		}

		err = page.Advance(`</td><td>`)
		if err != nil {
			strErr = "failed advancing past summative/formative info"
			break
		}

		task.Name, err = page.UpTo(`</td><td>`)
		if err != nil {
			strErr = "failed getting task name"
			break
		}

		err = page.Advance(`</td><td>`)
		if err != nil {
			strErr = "failed advancing to post date"
			break
		}

		postStr, err = page.UpTo(`</td><td>`)
		var e error
		local, e = time.Parse("2/01/06", postStr)
		if e != nil {
			strErr = "failed to parse post date"
			break
		}
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
		if err != nil {
			strErr = "failed advancing to due date"
			break
		}

		dueStr, err = page.UpTo(`</td><td>`)
		local, e = time.Parse("2/01/06", dueStr)
		if e != nil {
			strErr = "failed to parse due date"
			break
		}
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
		if err != nil {
			strErr = "failed getting task info line"
			break
		}

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
		err = errors.New(strErr, err)
		return
	}

	wg := sync.WaitGroup{}
	results := make([]taskGrade, len(graded))
	errs := make([]errors.Error, len(graded))

	for i, id := range graded {
		wg.Add(1)
		getGrade(creds, id, &results[i], &errs[i], &wg)
	}

	wg.Wait()

	if errors.Join(errs...) != nil {
		err = errors.Join(errs...)
		return
	}

	for i, task := range unsorted {
		for j, id := range graded {
			if task.Id == id {
				unsorted[i].Graded = results[j].Exists
				unsorted[i].Grade = results[j].Grade
				unsorted[i].Score = results[j].Mark
			}
		}
	}

	for _, task := range unsorted {
		if task.Graded == true {
			tasks = append(tasks, task)
		}
	}

	err = nil
}
