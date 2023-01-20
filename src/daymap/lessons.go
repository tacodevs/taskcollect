package daymap

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"main/errors"
	"main/plat"
)

type dmJsonEntry struct {
	Text   string
	Id     int
	Start  string
	Finish string
	Title  string
}

// Get a list of lessons for the week from DayMap for a user.
func GetLessons(creds User) ([][]plat.Lesson, error) {
	var weekStartIdx, weekEndIdx int
	t := time.Now().In(creds.Timezone)

	now := time.Date(
		t.Year(), t.Month(), t.Day(),
		0, 0, 0, 0,
		creds.Timezone,
	)

	weekday := now.Weekday()

	switch weekday {
	case 6:
		weekStartIdx = 2
		weekEndIdx = 6
	default:
		weekStartIdx = 1 - int(weekday)
		weekEndIdx = 5 - int(weekday)
	}

	weekStart := now.AddDate(0, 0, weekStartIdx)
	weekEnd := now.AddDate(0, 0, weekEndIdx)

	lessonsUrl := "https://gihs.daymap.net/daymap/DWS/Diary.ashx"
	lessonsUrl += "?cmd=EventList&from="
	lessonsUrl += weekStart.Format("2006-01-02") + "&to="
	lessonsUrl += weekEnd.Format("2006-01-02")

	client := &http.Client{}

	req, err := http.NewRequest("GET", lessonsUrl, nil)
	if err != nil {
		return nil, errors.NewError("daymap.GetLessons", "GET request for lessonsUrl failed", err)
	}

	req.Header.Set("Cookie", creds.Token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.NewError("daymap.GetLessons", "failed to get resp", err)
	}

	dmJson := []dmJsonEntry{}

	err = json.NewDecoder(resp.Body).Decode(&dmJson)
	if err != nil {
		return nil, errors.NewError("daymap.GetLessons", "failed to decode JSON", err)
	}

	lessons := make([][]plat.Lesson, 5)

	for _, l := range dmJson {
		lesson := plat.Lesson{}
		lesson.Start, err = time.Parse("2006-01-02T15:04:05.0000000", l.Start)

		if err != nil {
			startIdx := strings.Index(l.Start, "(") + 1
			endIdx := strings.Index(l.Start, "000-")

			if startIdx == 0 || endIdx == -1 {
				return nil, errInvalidDmJson
			}

			startStr := l.Start[startIdx:endIdx]
			startInt, err := strconv.Atoi(startStr)
			if err != nil {
				return nil, errors.NewError("daymap.GetLessons", "(1) string -> int conversion failed", err)
			}

			lesson.Start = time.Unix(int64(startInt), 0)

			startIdx = strings.Index(l.Finish, "(") + 1
			endIdx = strings.Index(l.Finish, "000-")

			if startIdx == 0 || endIdx == -1 {
				return nil, errInvalidDmJson
			}

			finishStr := l.Finish[startIdx:endIdx]
			finishInt, err := strconv.Atoi(finishStr)
			if err != nil {
				return nil, errors.NewError("daymap.GetLessons", "(2) string -> int conversion failed", err)
			}

			lesson.End = time.Unix(int64(finishInt), 0)
		} else {
			lesson.End, err = time.Parse("2006-01-02T15:04:05.0000000", l.Finish)
			if err != nil {
				return nil, errors.NewError("daymap.GetLessons", "failed to parse time", err)
			}
		}

		class := l.Title

		for class[len(class)-1] == ' ' {
			class = strings.TrimSuffix(class, " ")
		}

		re, err := regexp.Compile("[0-9][A-Z]+[0-9]+")
		if err != nil {
			return nil, errors.NewError("daymap.GetLessons", "failed to compile regex", err)
		}

		lesson.Room = re.FindString(class)
		roomIdx := re.FindStringIndex(class)
		lesson.Class = class[:roomIdx[0]-1]

		if !strings.HasPrefix(l.Text, "<div") && len(l.Text) > 0 {
			if strings.Contains(l.Text, "<div") {
				lesson.Notice = l.Text[:strings.Index(l.Text, "<div")]
			} else {
				lesson.Notice = l.Text
			}
		}

		if strings.Contains(class, "Mentor") {
			continue
		}

		day := time.Date(
			lesson.Start.Year(),
			lesson.Start.Month(),
			lesson.Start.Day(),
			0, 0, 0, 0,
			creds.Timezone,
		)

		i := 0

		for day.After(weekStart.AddDate(0, 0, i)) {
			i++
		}

		if i > 4 {
			continue
		}

		lessons[i] = append(lessons[i], lesson)
	}

	return lessons, nil
}
