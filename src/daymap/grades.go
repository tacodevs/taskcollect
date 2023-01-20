package daymap

import (
	"image/color"
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
func findGrade(page *string) (plat.TaskGrade, error) {

	var grade string
	var percent float64
	var gradeColor color.Color
	i := strings.Index(*page, "Grade:")

	if i != -1 {
		i = strings.Index(*page, "TaskGrade'>")

		if i == -1 {
			result := plat.TaskGrade{Grade: grade, Mark: percent, Color: gradeColor}
			return result, errInvalidTaskResp
		}

		*page = (*page)[i:]
		i = len("TaskGrade'>")
		*page = (*page)[i:]
		i = strings.Index(*page, "</div>")

		if i == -1 {
			result := plat.TaskGrade{Grade: grade, Mark: percent, Color: gradeColor}
			return result, errInvalidTaskResp
		}

		grade = (*page)[:i]
		*page = (*page)[i:]
	}

	i = strings.Index(*page, "Mark:")

	if i != -1 {
		i = strings.Index(*page, "TaskGrade'>")

		if i == -1 {
			result := plat.TaskGrade{Grade: grade, Mark: percent, Color: gradeColor}
			return result, errInvalidTaskResp
		}

		*page = (*page)[i:]
		i = len("TaskGrade'>")
		*page = (*page)[i:]
		i = strings.Index(*page, "</div>")

		if i == -1 {
			result := plat.TaskGrade{Grade: grade, Mark: percent, Color: gradeColor}
			return result, errInvalidTaskResp
		}

		markStr := (*page)[:i]
		*page = (*page)[i:]

		x := strings.Index(markStr, " / ")

		if x == -1 {
			result := plat.TaskGrade{Grade: grade, Mark: percent, Color: gradeColor}
			return result, errInvalidTaskResp
		}

		st := markStr[:x]
		sb := markStr[x+3:]

		it, err := strconv.ParseFloat(st, 64)
		if err != nil {
			result := plat.TaskGrade{Grade: grade, Mark: percent, Color: gradeColor}
			return result, errors.NewError("daymap.GetTask", "(1) string to float64 conversion failed", err)
		}

		ib, err := strconv.ParseFloat(sb, 64)
		if err != nil {
			result := plat.TaskGrade{Grade: grade, Mark: percent, Color: gradeColor}
			return result, errors.NewError("daymap.GetTask", "(2) string to float64 conversion failed", err)
		}

		percent = it / ib * 100

	}

	result := plat.TaskGrade{Grade: grade, Mark: percent, Color: gradeColor}

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
func GradedTasks(creds User, t chan []plat.Task, e chan error) {
	b, err := tasksPage(creds)
	if err != nil {
		t <- nil
		e <- err
		return
	}

	unsortedTasks := []plat.Task{}
	i := strings.Index(b, `href="javascript:ViewAssignment(`)
	graded := []string{}

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
			t <- nil
			e <- errors.NewError("daymap.ListTasks", "failed to parse time (postedString)", err)
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
			t <- nil
			e <- errors.NewError("daymap.ListTasks", "failed to parse time (dueString)", err)
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
	result := make([]plat.TaskGrade, len(graded))
	errs := make([]error, len(graded))

	for i, id := range graded {
		wg.Add(1)
		go taskGrade(creds, id, &result[i], &errs[i], &wg)
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
				unsortedTasks[i].Result.Grade = result[j].Grade
				unsortedTasks[i].Result.Mark = result[j].Mark
				unsortedTasks[i].Result.Color = result[j].Color
			}
		}
	}

	tasks := []plat.Task{}

	for _, task := range unsortedTasks {
		if task.Result.Grade != "" {
			tasks = append(tasks, task)
		}
	}

	t <- tasks
	e <- nil
}
