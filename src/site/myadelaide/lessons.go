package myadelaide

import (
	"encoding/json"
	"main/site"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	"git.sr.ht/~kvo/go-std/errors"
)

func midnight(t time.Time) time.Time {
	return time.Date(
		t.Year(), t.Month(), t.Day(),
		0, 0, 0, 0,
		t.Location(),
	)
}

type s1struct struct {
	Status string `json:"status"`
	Data   struct {
		Query struct {
			NumRows int    `json:"numrows"`
			Name    string `json:"queryname="`
			Rows    []struct {
				Strm        string `json:"STRM"`
				RowNum      int    `json:"attr:rownumber"`
				ID          string `json:"EMPLID"`
				Future      string `json:"CURRENT_FUTURE"`
				Description string `json:"DESCR"`
			} `json:"rows"`
		} `json:"query"`
	} `json:"data"`
}

type s2lesson struct {
	RowNum             int    `json:"attr:rownumber"`
	Type               string `json:"D.XLATLONGNAME"`
	StartTime          string `json:"START_TIME"`
	EndTime            string `json:"END_TIME"`
	Subject            string `json:"B.SUBJECT"`
	CatalogNumber      string `json:"B.CATALOG_NBR"`
	SubjectDescription string `json:"B.DESCR"`
	Location           string `json:"G.DESCR"`
	Room               string `json:"F.ROOM"`
	RoomDescription    string `json:"F.DESCR"`
	Day                string `json:"C.WEEKDAY_NAME"`
	StartDate          string `json:"E.START_DT"`
	EndDate            string `json:"E.END_DT"`
	CourseID           string `json:"B.CRSE_ID"`
}

type s2struct struct {
	Status string `json:"status"`
	Data   struct {
		Query struct {
			NumRows int        `json:"numrows"`
			Name    string     `json:"queryname="`
			Rows    []s2lesson `json:"rows"`
		} `json:"query"`
	} `json:"data"`
}

func semester(user site.User) ([]site.Lesson, error) {
	var lessons []site.Lesson
	client := &http.Client{}
	s1link := "https://api.adelaide.edu.au/api/generic-query-structured/v1/?target=/system/TIMETABLE_TERMS/queryx/" + user.Username[1:] + "&MaxRows=9999"

	s1req, err := http.NewRequest("GET", s1link, nil)
	if err != nil {
		return nil, errors.New(err, "cannot create stage 1 request")
	}

	s1req.Header.Set("Accept", "application/json")
	s1req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	s1req.Header.Set("Authorization", "Bearer "+user.SiteTokens["myadelaide"])
	s1req.Header.Set("Connection", "keep-alive")
	s1req.Header.Set("Host", "api.adelaide.edu.au")
	s1req.Header.Set("Referer", "https://myadelaide.uni.adelaide.edu.au/")

	s1, err := client.Do(s1req)
	if err != nil {
		return nil, errors.New(err, "cannot execute stage 1 request")
	}

	s1json := s1struct{}
	err = json.NewDecoder(s1.Body).Decode(&s1json)
	if err != nil {
		return nil, errors.New(err, "cannot decode stage 1 json")
	}

	s1strm := s1json.Data.Query.Rows[0].Strm

	s2link := "https://api.adelaide.edu.au/api/generic-query-structured/v1/?target=/system/TIMETABLE_LIST/queryx/" + user.Username[1:] + "," + s1strm + "&MaxRows=9999"
	s2req, err := http.NewRequest("GET", s2link, nil)
	if err != nil {
		return nil, errors.New(err, "cannot create stage 2 request")
	}

	s2req.Header.Set("Accept", "application/json")
	s2req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	s2req.Header.Set("Authorization", "Bearer "+user.SiteTokens["myadelaide"])
	s2req.Header.Set("Connection", "keep-alive")
	s2req.Header.Set("Host", "api.adelaide.edu.au")
	s2req.Header.Set("Referer", "https://myadelaide.uni.adelaide.edu.au/")

	s2, err := client.Do(s2req)
	if err != nil {
		return nil, errors.New(err, "cannot execute stage 2 request")
	}

	s2lessons := s2struct{}
	err = json.NewDecoder(s2.Body).Decode(&s2lessons)
	if err != nil {
		return nil, errors.New(err, "cannot decode stage 2 json")
	}

	for _, lesson := range s2lessons.Data.Query.Rows {
		today := midnight(time.Now())
		startStr := lesson.StartDate + " " + lesson.StartTime
		start, err := time.ParseInLocation("2006-01-02 3:04 PM", startStr, user.Timezone)
		if err != nil {
			return nil, errors.New(err, "cannot parse time")
		}
		endStr := lesson.StartDate + " " + lesson.EndTime
		end, err := time.ParseInLocation("2006-01-02 3:04 PM", endStr, user.Timezone)
		if err != nil {
			return nil, errors.New(err, "cannot parse time")
		}
		finalDate, err := time.ParseInLocation("2006-01-02", lesson.EndDate, user.Timezone)
		if err != nil {
			return nil, errors.New(err, "failed to parse date")
		}
		if today.After(finalDate) {
			continue
		}
		numLessons := int(finalDate.UnixMilli()-midnight(start).UnixMilli())/(7*24*60*60*1000) + 1
		for i := 0; i < numLessons; i++ {
			lessons = append(lessons, site.Lesson{
				Start:   start.AddDate(0, 0, 7*i),
				End:     end.AddDate(0, 0, 7*i),
				Class:   lesson.SubjectDescription,
				Teacher: lesson.Type,
				Notice:  "",
				Room:    lesson.Location + " " + lesson.Room + " " + lesson.RoomDescription,
			})
		}
	}
	sort.SliceStable(lessons, func(i, j int) bool {
		return lessons[i].Start.Before(lessons[j].Start)
	})
	return lessons, nil
}

type Lesson struct {
	RowNum             int    `json:"attr:rownumber"`
	Type               string `json:"D.XLATLONGNAME"`
	StartTime          string `json:"START_TIME"`
	EndTime            string `json:"END_TIME"`
	Subject            string `json:"B.SUBJECT"`
	CatalogNumber      string `json:"B.CATALOG_NBR"`
	SubjectDescription string `json:"B.DESCR"`
	Location           string `json:"F.DESCR"`
	Room               string `json:"E.ROOM"`
	RoomDescription    string `json:"E.DESCR"`
	Day                string `json:"C.WEEKDAY_NAME"`
	Date               string `json:"DATE"`
	CourseID           string `json:"B.CRSE_ID"`
}

type Week struct {
	Status string `json:"status"`
	Data   struct {
		Query struct {
			NumRows int      `json:"numrows"`
			Name    string   `json:"queryname="`
			Rows    []Lesson `json:"rows"`
		} `json:"query"`
	} `json:"data"`
}

// weeks returns all lessons for user that occur on the weeks corresponding to
// each delta. A delta is an offset (in days) that points to the start of the
// required week (Monday). An error is returned instead if one occurs.
func weeks(user site.User, deltas ...int) ([]site.Lesson, error) {
	var lessons []site.Lesson

	for i, value := range deltas {
		client := &http.Client{}
		if i != 0 && deltas[i] <= deltas[i-1] {
			break
		}

		link := "https://api.adelaide.edu.au/api/generic-query-structured/v1/?target=/system/TIMETABLE_WEEKLY/queryx/" + user.Username[1:] + "," + strconv.Itoa(value) + "&MaxRows=9999"
		req, err := http.NewRequest("GET", link, nil)
		if err != nil {
			return nil, errors.New(err, "cannot create week lessons request")
		}

		req.Header.Set("Accept", "application/json")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		req.Header.Set("Authorization", "Bearer "+user.SiteTokens["myadelaide"])
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Host", "api.adelaide.edu.au")
		req.Header.Set("Referer", "https://myadelaide.uni.adelaide.edu.au/")

		resp, err := client.Do(req)
		if err != nil {
			return nil, errors.New(err, "cannot execute week lessons request")
		}

		var week Week
		err = json.NewDecoder(resp.Body).Decode(&week)
		if err != nil {
			return nil, errors.New(err, "cannot decode json")
		}

		for _, lesson := range week.Data.Query.Rows {
			startStr := lesson.Date + lesson.StartTime
			start, err := time.ParseInLocation("02 Jan 2006 15.04", startStr, user.Timezone)
			if err != nil {
				return nil, errors.New(err, "cannot parse date")
			}
			endStr := lesson.Date + lesson.EndTime
			end, err := time.ParseInLocation("02 Jan 2006 15.04", endStr, user.Timezone)
			if err != nil {
				return nil, errors.New(err, "cannot parse date")
			}
			lessons = append(lessons, site.Lesson{
				Start:   start,
				End:     end,
				Class:   lesson.SubjectDescription,
				Teacher: lesson.Type,
				Notice:  "",
				Room:    lesson.Location + " " + lesson.Room + " " + lesson.RoomDescription,
			})
		}
	}

	return lessons, nil
}

func Lessons(user site.User, start, end time.Time) ([]site.Lesson, error) {
	var lessons []site.Lesson

	var err error
	startIndex := int(start.Weekday())
	if startIndex == 0 {
		startIndex = 7
	}
	startWeek := start.AddDate(0, 0, -(startIndex - 1))
	endIndex := int(end.Weekday())
	if endIndex == 0 {
		endIndex = 7
	}
	endWeek := end.AddDate(0, 0, -(endIndex - 1))
	numWeeks := int(float64((endWeek.Unix()-startWeek.Unix())/(60*60*24*7))) + 1

	if numWeeks > 2 {
		lessons, err = semester(user)
		if err != nil {
			return nil, errors.New(err, "cannot fetch semester lessons")
		}
	} else {
		deltas := make([]int, numWeeks)
		nowIndex := int(time.Now().Weekday())
		if nowIndex == 0 {
			nowIndex = 7
		}
		nowWeek := midnight(time.Now().AddDate(0, 0, -(nowIndex - 1)))
		deltas[0] = int(math.Ceil(float64((startWeek.Unix() - nowWeek.Unix()) / (60 * 60 * 24))))
		for i := 1; i < numWeeks; i++ {
			deltas[i] = deltas[0] + i*7
		}
		lessons, err = weeks(user, deltas...)
		if err != nil {
			return nil, errors.New(err, "cannot fetch lessons")
		}
	}
	point1 := 0
	point2 := 0
	for _, lesson := range lessons {
		if lesson.Start.Before(start) {
			point1++
		} else {
			break
		}
	}
	end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 999999999, user.Timezone)
	for i := point1; i < len(lessons); i++ {
		if lessons[i].Start.Before(end) {
			point2++
		} else {
			break
		}
	}
	lessons = lessons[point1:point2]

	return lessons, nil
}
