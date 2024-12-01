package daymap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~kvo/go-std/errors"

	"main/site"
)

type Lesson struct {
	Text   string
	Type   string
	Id     int
	Start  string
	Finish string
	Title  string
}

func Lessons(user site.User, start, end time.Time) ([]site.Lesson, error) {
	client := &http.Client{}
	var fetched []Lesson
	var lessons []site.Lesson

	lessonsUrl := "https://gihs.daymap.net/daymap/DWS/Diary.ashx?cmd=EventList&from="
	lessonsUrl += start.Format("2006-01-02") + "&to=" + end.Format("2006-01-02")

	req, err := http.NewRequest("GET", lessonsUrl, nil)
	if err != nil {
		return nil, errors.New("cannot create lessons request", err)
	}

	req.Header.Set("Cookie", user.SiteTokens["daymap"])

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("cannot execute lessons request", err)
	}

	err = json.NewDecoder(resp.Body).Decode(&fetched)
	if err != nil {
		return nil, errors.New("cannot decode lessons JSON", err)
	}

	for _, l := range fetched {
		if l.Type != "Lesson" {
			continue
		}

		lesson := site.Lesson{}
		lesson.Start, err = time.ParseInLocation("2006-01-02T15:04:05.0000000", l.Start, user.Timezone)

		if err != nil {
			startIdx := strings.Index(l.Start, "(") + 1
			endIdx := strings.Index(l.Start, "000-")

			if startIdx == 0 || endIdx == -1 {
				return nil, errors.New("invalid lessons JSON", nil)
			}

			startStr := l.Start[startIdx:endIdx]
			startInt, err := strconv.Atoi(startStr)
			if err != nil {
				return nil, errors.New(fmt.Sprintf(`cannot convert "%s" to int`, startStr), err)
			}

			lesson.Start = time.Unix(int64(startInt), 0)

			startIdx = strings.Index(l.Finish, "(") + 1
			endIdx = strings.Index(l.Finish, "000-")

			if startIdx == 0 || endIdx == -1 {
				return nil, errors.New("invalid lessons JSON", nil)
			}

			finishStr := l.Finish[startIdx:endIdx]
			finishInt, err := strconv.Atoi(finishStr)
			if err != nil {
				return nil, errors.New(fmt.Sprintf(`cannot convert "%s" to int`, finishStr), err)
			}

			lesson.End = time.Unix(int64(finishInt), 0)
		} else {
			lesson.End, err = time.ParseInLocation("2006-01-02T15:04:05.0000000", l.Finish, user.Timezone)
			if err != nil {
				return nil, errors.New("cannot parse time", err)
			}
		}

		class := l.Title
		class = strings.TrimSpace(class)

		exp, err := regexp.Compile("[0-9][A-Z]+[0-9]+")
		if err != nil {
			return nil, errors.New("cannot compile regex", err)
		}

		lesson.Room = exp.FindString(class)
		roomIdx := exp.FindStringIndex(class)
		if len(class) > 0 && len(roomIdx) > 0 && roomIdx[0] > 0 {
			lesson.Class = class[:roomIdx[0]-1]
		} else {
			lesson.Class = class
		}

		if !strings.HasPrefix(l.Text, "<div") && len(l.Text) > 0 {
			if strings.Contains(l.Text, "<div") {
				lesson.Notice = l.Text[:strings.Index(l.Text, "<div")]
			} else {
				lesson.Notice = l.Text
			}
			lesson.Notice = strings.ReplaceAll(lesson.Notice, `<img src="/daymap/images/buttons/roomChange.gif"/>&nbsp;`, "")
		}

		if strings.Contains(class, "Mentor Group") {
			continue
		}

		lessons = append(lessons, lesson)
	}

	return lessons, nil
}
