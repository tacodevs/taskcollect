package daymap

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Lesson struct {
	Start   time.Time
	End     time.Time
	Class   string
	Room    string
	Teacher string
	Notice  string
}

// TODO: What does "dmlent" actually mean?
type dmlent struct {
	Text   string
	Id     int
	Start  string
	Finish string
	Title  string
}

func GetLessons(creds User) ([][]Lesson, error) {
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
		return nil, err
	}

	req.Header.Set("Cookie", creds.Token)
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	dmJson := []dmlent{}
	err = json.NewDecoder(resp.Body).Decode(&dmJson)

	if err != nil {
		return nil, err
	}

	lessons := make([][]Lesson, 5)

	for _, l := range dmJson {
		start, err := time.Parse("2006-01-02T15:04:05.0000000", l.Start)

		if err != nil {
			return nil, err
		}

		finish, err := time.Parse("2006-01-02T15:04:05.0000000", l.Finish)

		if err != nil {
			return nil, err
		}

		class := l.Title

		for class[len(class)-1] == ' ' {
			class = strings.TrimSuffix(class, " ")
		}

		re, err := regexp.Compile("[0-9][A-Z]+[0-9]+")

		if err != nil {
			return nil, err
		}

		room := re.FindString(class)
		roomIdx := re.FindStringIndex(class)
		class = class[:roomIdx[0]-1]
		notice := ""

		if !strings.HasPrefix(l.Text, "<div") && len(l.Text) > 0 {
			if strings.Index(l.Text, "<div") != -1 {
				notice = l.Text[:strings.Index(l.Text, "<div")]
			} else {
				notice = l.Text
			}
		}

		if strings.Index(class, "Mentor") != -1 {
			continue
		}

		day := time.Date(
			start.Year(), start.Month(), start.Day(),
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

		lesson := Lesson{
			Start:   start,
			End:     finish,
			Class:   class,
			Room:    room,
			Teacher: "",
			Notice:  notice,
		}

		lessons[i] = append(lessons[i], lesson)
	}

	return lessons, nil
}
