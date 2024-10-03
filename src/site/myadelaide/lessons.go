package myadelaide

import (
	"encoding/json"
	"main/site"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~kvo/go-std/errors"
)

type jsonSTRM struct {
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

type jsonLesson struct {
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

type jsonLessonWeek struct {
	Status string `json:"status"`
	Data   struct {
		Query struct {
			NumRows int          `json:"numrows"`
			Name    string       `json:"queryname="`
			Rows    []jsonLesson `json:"rows"`
		} `json:"query"`
	} `json:"data"`
}

type jsonLessonList struct {
	Status string `json:"status"`
	Data   struct {
		Query struct {
			NumRows int              `json:"numrows"`
			Name    string           `json:"queryname="`
			Rows    []jsonLessonItem `json:"rows"`
		} `json:"query"`
	} `json:"data"`
}
type jsonLessonItem struct {
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
	numWeeks := int(math.Floor(float64((endWeek.Unix()-startWeek.Unix())/(60*60*24*7)))) + 1
	if numWeeks > 2 {
		lessons, err = listResp(user)
		if err != nil {
			return nil, errors.New("failed to get lessons", err)
		}
		_ = lessons

	} else {
		var weeks [2]int
		nowIndex := int(time.Now().Weekday())
		if nowIndex == 0 {
			nowIndex = 7
		}
		nowWeek := time.Now().AddDate(0, 0, -(nowIndex - 1))
		nowWeek = time.Date(nowWeek.Year(), nowWeek.Month(), nowWeek.Day(), 0, 0, 0, 0, user.Timezone)
		weeks[0] = int(math.Ceil(float64((startWeek.Unix() - nowWeek.Unix()) / (60 * 60 * 24))))

		for i := 1; i < numWeeks; i++ {
			weeks[i] = weeks[0] + i*7
		}
		lessons, err = weekResp(user, weeks)
		if err != nil {
			return nil, errors.New("failed to get lessons", err)
		}
		_ = lessons
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
	end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, user.Timezone)
	for _, lesson := range lessons {
		if lesson.Start.Before(end) {
			point2++
		} else {
			break
		}
	}
	lessons = lessons[point1:point2]

	return lessons, nil
}

func listResp(user site.User) ([]site.Lesson, error) {
	strmURL := "https://api.adelaide.edu.au/api/generic-query-structured/v1/?target=/system/TIMETABLE_TERMS/queryx/" + user.Username[1:] + "&MaxRows=9999"
	var lessons []site.Lesson
	client := &http.Client{}
	req, err := http.NewRequest("GET", strmURL, nil)
	if err != nil {
		return nil, errors.New("GET request for STRM info failed", err)
	}
	req.Header.Set("Authorization", "Bearer "+user.SiteTokens["myadelaide"])
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://myadelaide.uni.adelaide.edu.au/")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Host", "api.adelaide.edu.au")
	resp, err := client.Do(req)

	var strmJSON jsonSTRM
	err = json.NewDecoder(resp.Body).Decode(&strmJSON)
	if err != nil {
		return nil, errors.New("failed to decode json", err)
	}

	STRM := strmJSON.Data.Query.Rows[0].Strm

	listURL := "https://api.adelaide.edu.au/api/generic-query-structured/v1/?target=/system/TIMETABLE_LIST/queryx/" + user.Username[1:] + "," + STRM + "&MaxRows=9999"

	req, err = http.NewRequest("GET", listURL, nil)
	if err != nil {
		return nil, errors.New("GET request for STRM info failed", err)
	}
	req.Header.Set("Authorization", "Bearer "+user.SiteTokens["myadelaide"])
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://myadelaide.uni.adelaide.edu.au/")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Host", "api.adelaide.edu.au")
	resp, err = client.Do(req)
	var listJSON jsonLessonList
	err = json.NewDecoder(resp.Body).Decode(&listJSON)
	if err != nil {
		return nil, errors.New("failed to decode json", err)
	}
	for _, lesson := range listJSON.Data.Query.Rows {
		now := time.Now()
		now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		start, err := time.ParseInLocation("2006-01-02", lesson.StartDate, user.Timezone)
		if err != nil {
			return nil, errors.New("failed to parse date", err)
		}
		end, err := time.ParseInLocation("2006-01-02", lesson.EndDate, user.Timezone)
		if err != nil {
			return nil, errors.New("failed to parse date", err)
		}
		if now.After(end) {
			continue
		}
		// n is the number of lesson instances
		time_start, err := time.ParseInLocation("2006-01-02 3:04 PM", strings.Join([]string{lesson.StartDate, lesson.StartTime}, " "), user.Timezone)
		if err != nil {
			return nil, errors.New("failed to parse time", err)
		}
		time_end, err := time.ParseInLocation("2006-01-02 3:04 PM", strings.Join([]string{lesson.StartDate, lesson.EndTime}, " "), user.Timezone)
		if err != nil {
			return nil, errors.New("failed to parse time", err)
		}
		n := int(end.UnixMilli()-start.UnixMilli())/(7*24*60*60*1000) + 1
		for i := 0; i < n; i++ {
			lessons = append(lessons, site.Lesson{
				Start:   time_start.AddDate(0, 0, 7*i),
				End:     time_end.AddDate(0, 0, 7*i),
				Class:   lesson.SubjectDescription,
				Teacher: lesson.Type,
				Notice:  "",
				Room:    lesson.Location + " " + lesson.Room + " " + lesson.RoomDescription,
			})
		}
	}
	return lessons, nil
}

func weekResp(user site.User, displacements [2]int) ([]site.Lesson, error) {
	var lessons []site.Lesson
	for i, value := range displacements {
		if i != 0 && displacements[i] <= displacements[i-1] {
			break
		}
		url := "https://api.adelaide.edu.au/api/generic-query-structured/v1/?target=/system/TIMETABLE_WEEKLY/queryx/" + user.Username[1:] + "," + strconv.Itoa(value) + "&MaxRows=9999"
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, errors.New("GET request for lesson info failed", err)
		}
		req.Header.Set("Authorization", "Bearer "+user.SiteTokens["myadelaide"])
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Referer", "https://myadelaide.uni.adelaide.edu.au/")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Host", "api.adelaide.edu.au")
		resp, err := client.Do(req)
		var respjson jsonLessonWeek
		err = json.NewDecoder(resp.Body).Decode(&respjson)
		if err != nil {
			return nil, errors.New("failed to decode json", err)
		}
		for _, lesson := range respjson.Data.Query.Rows {
			// n is the number of lesson instances
			time_start, err := time.ParseInLocation("02 Jan 2006 15.04", strings.Join([]string{lesson.Date, lesson.StartTime}, " "), user.Timezone)

			if err != nil {
				return nil, errors.New("failed to parse date", err)
			}

			time_end, err := time.ParseInLocation("02 Jan 2006 15.04", strings.Join([]string{lesson.Date, lesson.EndTime}, " "), user.Timezone)
			if err != nil {
				return nil, errors.New("failed to parse date", err)
			}
			lessons = append(lessons, site.Lesson{
				Start:   time_start,
				End:     time_end,
				Class:   lesson.SubjectDescription,
				Teacher: lesson.Type,
				Notice:  "",
				Room:    lesson.Location + " " + lesson.Room + " " + lesson.RoomDescription,
			})
		}
	}
	return lessons, nil
}
