package daymap

import (
	"bufio"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~kvo/go-std/errors"

	"main/site"
)

type taskGrade struct {
	Exists bool
	Grade  string
	Mark   float64
}

// Return the grade for a DayMap task from a DayMap task webpage.
func findGrade(webpage *string) (taskGrade, error) {
	var grade string
	var percent float64
	i := strings.Index(*webpage, "Grade:")

	if i != -1 {
		i = strings.Index(*webpage, "TaskGrade'>")

		if i == -1 {
			return taskGrade{}, errors.Raise(site.ErrInvalidTaskResp)
		}

		*webpage = (*webpage)[i:]
		i = len("TaskGrade'>")
		*webpage = (*webpage)[i:]
		i = strings.Index(*webpage, "</div>")

		if i == -1 {
			return taskGrade{}, errors.Raise(site.ErrInvalidTaskResp)
		}

		grade = (*webpage)[:i]
		*webpage = (*webpage)[i:]
	}

	i = strings.Index(*webpage, "Mark:")

	if i != -1 {
		i = strings.Index(*webpage, "TaskGrade'>")

		if i == -1 {
			return taskGrade{}, errors.Raise(site.ErrInvalidTaskResp)
		}

		*webpage = (*webpage)[i:]
		i = len("TaskGrade'>")
		*webpage = (*webpage)[i:]
		i = strings.Index(*webpage, "</div>")

		if i == -1 {
			return taskGrade{}, errors.Raise(site.ErrInvalidTaskResp)
		}

		markStr := (*webpage)[:i]
		*webpage = (*webpage)[i:]

		x := strings.Index(markStr, " / ")

		if x == -1 {
			return taskGrade{}, errors.Raise(site.ErrInvalidTaskResp)
		}

		st := markStr[:x]
		sb := markStr[x+3:]

		it, err := strconv.ParseFloat(st, 64)
		if err != nil {
			return taskGrade{}, errors.New("(1) string to float64 conversion failed", err)
		}

		ib, err := strconv.ParseFloat(sb, 64)
		if err != nil {
			return taskGrade{}, errors.New("(2) string to float64 conversion failed", err)
		}

		percent = it / ib * 100
	}

	result := taskGrade{Exists: true, Grade: grade, Mark: percent}
	return result, nil
}

// Retrieve a list of graded tasks from DayMap for a user.
func Graded(creds site.User, c chan site.Pair[[]site.Task, error]) {
	var result site.Pair[[]site.Task, error]

	client := &http.Client{}
	taskUrl := "https://gihs.daymap.net/daymap/student/assignment.aspx?TaskID="
	link := "https://gihs.daymap.net/daymap/student/portfolio.aspx/AssessmentReport"
	referrer := "https://gihs.daymap.net/daymap/student/portfolio.aspx?tab=Assessment_Results"
	form := `{"id":5303,"classId":0,"viewMode":"tabular","allCompleted":false,"taskType":0,`
	times := `"fromDate":"YYYY-01-01T00:00:00.000Z","toDate":"YYYY-12-31T23:59:59.999Z"}`
	year := strconv.Itoa(time.Now().In(creds.Timezone).Year())
	times = strings.ReplaceAll(times, "YYYY", year)
	data := strings.NewReader(form + times)

	req, err := http.NewRequest("POST", link, data)
	if err != nil {
		result.Second = errors.New("GET request failed", err)
		c <- result
		return
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Cookie", creds.SiteTokens["daymap"])
	req.Header.Set("Origin", "https://gihs.daymap.net")
	req.Header.Set("Referer", referrer)

	resp, err := client.Do(req)
	if err != nil {
		result.Second = errors.New("failed to get resp", err)
		c <- result
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	class := ""
	for scanner.Scan() {
		line := scanner.Text()
		i := strings.Index(line, `<tr><th colspan='5' align='left'>`)
		if i != -1 {
			i += len(`<tr><th colspan='5' align='left'>`)
			line = line[i:]
			i = strings.Index(line, " (")
			if i == -1 {
				result.Second = errors.Raise(site.ErrInvalidResp)
				c <- result
				return
			}
			class = line[:i]
			continue
		}

		if !strings.HasPrefix(line, `<tr><td><a href="javascript:OpenTask(`) {
			continue
		}
		task := site.Task{Class: class, Platform: "daymap"}
		i = len(`<tr><td><a href="javascript:OpenTask(`)
		line = line[i:]

		i = strings.Index(line, `);">`)
		if i == -1 {
			result.Second = errors.Raise(site.ErrInvalidResp)
			c <- result
			return
		}
		task.Id = line[:i]
		task.Link = taskUrl + task.Id
		i += len(`);">`)
		line = line[i:]

		i = strings.Index(line, `</a>`)
		if i == -1 {
			result.Second = errors.Raise(site.ErrInvalidResp)
			c <- result
			return
		}
		task.Name = line[:i]

		for j := 0; j < 2; j++ {
			i = strings.Index(line, `<td nowrap>`)
			if i == -1 {
				result.Second = errors.Raise(site.ErrInvalidResp)
				c <- result
				return
			}
			i += len(`<td nowrap>`)
			line = line[i:]
		}

		i = strings.Index(line, `</td>`)
		if i == -1 {
			result.Second = errors.Raise(site.ErrInvalidResp)
			c <- result
			return
		}
		task.Grade = line[:i]

		i = strings.Index(line, `<td nowrap>`)
		if i == -1 {
			result.Second = errors.Raise(site.ErrInvalidResp)
			c <- result
			return
		}
		i += len(`<td nowrap>`)
		line = line[i:]

		i = strings.Index(line, `</td>`)
		if i == -1 {
			result.Second = errors.Raise(site.ErrInvalidResp)
			c <- result
			return
		}
		mark := line[:i]
		marks := strings.Split(mark, "/")

		if len(marks) == 2 {
			top, err := strconv.ParseFloat(marks[0], 64)
			if err != nil {
				result.Second = errors.New("numerator float64 conversion failed", err)
				c <- result
				return
			}
			bottom, err := strconv.ParseFloat(marks[1], 64)
			if err != nil {
				result.Second = errors.New("denominator float64 conversion failed", err)
				c <- result
				return
			}
			task.Score = top / bottom * 100
		}

		result.First = append(result.First, task)
	}
	if err := scanner.Err(); err != nil {
		result.Second = errors.New("error reading response body", err)
		c <- result
		return
	}
	c <- result
}
