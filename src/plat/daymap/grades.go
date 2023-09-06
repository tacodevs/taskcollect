package daymap

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~kvo/libgo/errors"

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

// Retrieve a list of graded tasks from DayMap for a user.
func Graded(creds plat.User, c chan []plat.Task, ok chan errors.Error, done *int) {
	var tasks []plat.Task
	var err errors.Error

	defer plat.Deliver(c, &tasks, done)
	defer plat.Deliver(ok, &err, done)
	defer plat.Done(done)

	client := &http.Client{}
	taskUrl := "https://gihs.daymap.net/daymap/student/assignment.aspx?TaskID="
	link := "https://gihs.daymap.net/daymap/student/portfolio.aspx/AssessmentReport"
	referrer := "https://gihs.daymap.net/daymap/student/portfolio.aspx?tab=Assessment_Results"
	form := `{"id":5303,"classId":0,"viewMode":"tabular","allCompleted":false,"taskType":0,`
	times := `"fromDate":"YYYY-01-01T00:00:00.000Z","toDate":"YYYY-12-31T23:59:59.999Z"}`
	year := strconv.Itoa(time.Now().In(creds.Timezone).Year())
	times = strings.ReplaceAll(times, "YYYY", year)
	data := strings.NewReader(form + times)

	req, e := http.NewRequest("POST", link, data)
	if e != nil {
		err = errors.New(
			"GET request failed",
			errors.New(e.Error(), nil),
		)
		return
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Cookie", creds.SiteTokens["daymap"])
	req.Header.Set("Origin", "https://gihs.daymap.net")
	req.Header.Set("Referer", referrer)

	resp, e := client.Do(req)
	if e != nil {
		err = errors.New(
			"failed to get resp",
			errors.New(e.Error(), nil),
		)
		return
	}

	respBody, e := io.ReadAll(resp.Body)
	if e != nil {
		err = errors.New(
			"failed to read resp.Body",
			errors.New(e.Error(), nil),
		)
		return
	}

	page := string(respBody)
	lines := strings.Split(page, "\n")
	class := ""

	for _, line := range lines {
		i := strings.Index(line, `<tr><th colspan='5' align='left'>`)
		if i != -1 {
			i += len(`<tr><th colspan='5' align='left'>`)
			line = line[i:]
			i = strings.Index(line, " (")
			if i == -1 {
				err = plat.ErrInvalidResp.Here()
				return
			}
			class = line[:i]
			continue
		}

		if !strings.HasPrefix(line, `<tr><td><a href="javascript:OpenTask(`) {
			continue
		}
		task := plat.Task{Class: class, Platform: "daymap"}
		i = len(`<tr><td><a href="javascript:OpenTask(`)
		line = line[i:]

		i = strings.Index(line, `);">`)
		if i == -1 {
			err = plat.ErrInvalidResp.Here()
			return
		}
		task.Id = line[:i]
		task.Link = taskUrl + task.Id
		i += len(`);">`)
		line = line[i:]

		i = strings.Index(line, `</a>`)
		if i == -1 {
			err = plat.ErrInvalidResp.Here()
			return
		}
		task.Name = line[:i]

		for j := 0; j < 2; j++ {
			i = strings.Index(line, `<td nowrap>`)
			if i == -1 {
				err = plat.ErrInvalidResp.Here()
				return
			}
			i += len(`<td nowrap>`)
			line = line[i:]
		}

		i = strings.Index(line, `</td>`)
		if i == -1 {
			err = plat.ErrInvalidResp.Here()
			return
		}
		task.Grade = line[:i]

		i = strings.Index(line, `<td nowrap>`)
		if i == -1 {
			err = plat.ErrInvalidResp.Here()
			return
		}
		i += len(`<td nowrap>`)
		line = line[i:]

		i = strings.Index(line, `</td>`)
		if i == -1 {
			err = plat.ErrInvalidResp.Here()
			return
		}
		mark := line[:i]
		marks := strings.Split(mark, "/")

		top, err := strconv.ParseFloat(marks[0], 64)
		if err != nil {
			err = errors.New("numerator float64 conversion failed", nil)
			return
		}

		bottom, err := strconv.ParseFloat(marks[1], 64)
		if err != nil {
			err = errors.New("denominator float64 conversion failed", nil)
			return
		}

		task.Score = top / bottom * 100
		tasks = append(tasks, task)
	}
}
