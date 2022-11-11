package main

import (
	"html/template"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"

	"main/errors"
	"main/logger"
)

var colors = []color.RGBA{
	{0x00, 0x28, 0x70, 0xff}, // Dark blue
	{0x00, 0x70, 0x00, 0xff}, // Green
	{0x58, 0x09, 0x7e, 0xff}, // Purple
	{0xab, 0x31, 0x7b, 0xff}, // Fuchsia
	{0xaa, 0x00, 0x00, 0xff}, // Dark red
	{0xab, 0x63, 0x00, 0xff}, // Tan
	{0x70, 0x26, 0x00, 0xff}, // Brown
	{0x00, 0x58, 0x70, 0xff}, // Dark azure
	{0x1b, 0x86, 0x69, 0xff}, // Teal
	{0x60, 0x60, 0x60, 0xff}, // Grey
}

var _colors = map[string]color.RGBA{
	"Dark Blue":  {0x00, 0x28, 0x70, 0xff},
	"Green":      {0x00, 0x70, 0x00, 0xff},
	"Purple":     {0x58, 0x09, 0x7e, 0xff},
	"Fuchsia":    {0xab, 0x31, 0x7b, 0xff},
	"Dark Red":   {0xaa, 0x00, 0x00, 0xff},
	"Tan":        {0xab, 0x63, 0x00, 0xff},
	"Brown":      {0x70, 0x26, 0x00, 0xff},
	"Dark Azure": {0x00, 0x58, 0x70, 0xff},
	"Teal":       {0x1b, 0x86, 0x69, 0xff},
	"Grey":       {0x60, 0x60, 0x60, 0xff},
}

func genDueStr(due time.Time, creds tcUser) string {
	var dueDate string
	now := time.Now().In(creds.Timezone)
	localDueDate := due.In(creds.Timezone)

	todayStart := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0, 0, 0, 0,
		creds.Timezone,
	)

	todayEnd := todayStart.AddDate(0, 0, 1)
	tmrEnd := todayStart.AddDate(0, 0, 2)
	weekEnd := todayStart.AddDate(0, 0, 7)

	if localDueDate.Before(todayStart) {
		dueDate = strconv.Itoa(localDueDate.Day())
		dueDate += " " + localDueDate.Month().String()
		if localDueDate.Year() != now.Year() {
			dueDate += " " + strconv.Itoa(localDueDate.Year())
		}
	} else if localDueDate.Before(todayEnd) {
		dueDate = "Today"
	} else if localDueDate.Before(tmrEnd) {
		dueDate = "Tomorrow"
	} else if localDueDate.Before(weekEnd) {
		dueDate = localDueDate.Weekday().String()
	} else {
		dueDate = strconv.Itoa(localDueDate.Day())
		dueDate += " " + localDueDate.Month().String()
		if localDueDate.Year() != now.Year() {
			dueDate += " " + strconv.Itoa(localDueDate.Year())
		}
	}

	if localDueDate.Hour() != 0 || localDueDate.Minute() != 0 {
		strHour := strconv.Itoa(localDueDate.Hour())
		if len(strHour) == 1 {
			strHour = "0" + strHour
		}

		strMinute := strconv.Itoa(localDueDate.Minute())
		if len(strMinute) == 1 {
			strMinute = "0" + strMinute
		}

		dueDate += ", " + strHour + ":" + strMinute
	}

	return dueDate
}

func genLesson(daymapWG *sync.WaitGroup, c color.RGBA, img *image.Image, w, h int, l lesson) {
	defer daymapWG.Done()

	canvas := image.NewRGBA(
		image.Rectangle{
			image.Point{0, 0},
			image.Point{w, h},
		},
	)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			canvas.Set(x, y, c)
		}
	}

	boldFont, err := freetype.ParseFont(gobold.TTF)
	if err != nil {
		*img = canvas
		daymapWG.Done()
		return
	}

	regFont, err := freetype.ParseFont(goregular.TTF)
	if err != nil {
		*img = canvas
		daymapWG.Done()
		return
	}

	headFace := truetype.NewFace(boldFont, &truetype.Options{
		Size:    16.0,
		DPI:     72,
		Hinting: font.HintingNone,
	})

	boldface := truetype.NewFace(boldFont, &truetype.Options{
		Size:    12.0,
		DPI:     72,
		Hinting: font.HintingNone,
	})

	regFace := truetype.NewFace(regFont, &truetype.Options{
		Size:    12.0,
		DPI:     72,
		Hinting: font.HintingNone,
	})

	d := font.Drawer{
		Dst:  canvas,
		Src:  image.White,
		Face: headFace,
	}

	d.Dot = fixed.Point26_6{
		X: fixed.I(5),
		Y: fixed.I(20),
	}

	d.DrawString(l.Class)
	d.Face = boldface
	offset := 0

	if l.Notice != "" {
		d.Dot = fixed.Point26_6{
			X: fixed.I(5),
			Y: fixed.I(35),
		}

		d.DrawString(l.Notice)
		offset = 15
	}

	d.Src = image.White
	d.Face = regFace

	d.Dot = fixed.Point26_6{
		X: fixed.I(5),
		Y: fixed.I(35 + offset),
	}

	s := l.Start.Format("15:04") + "-" + l.End.Format("15:04")
	s += ", " + l.Room
	d.DrawString(s)

	d.Dot = fixed.Point26_6{
		X: fixed.I(5),
		Y: fixed.I(50 + offset),
	}

	d.DrawString(l.Teacher)
	*img = canvas
}

func genDay(wg *sync.WaitGroup, img *image.Image, w int, h int, c color.RGBA, colorList map[string]color.RGBA, day []lesson) {
	defer wg.Done()

	minPerDay := float64(600)
	pxPerMin := float64(h) / minPerDay
	lessonStartPx := []int{}
	lessonDurationPx := []int{}

	for i := 0; i < len(day); i++ {
		minOffset := day[i].Start.Hour()*60 + day[i].Start.Minute()
		minOffset -= 8 * 60
		pxOffset := float64(minOffset) * pxPerMin
		lessonStartPx = append(lessonStartPx, int(pxOffset))
	}

	for i := 0; i < len(day); i++ {
		maxOffset := day[i].End.Hour()*60 + day[i].End.Minute()
		maxOffset -= day[i].Start.Hour()*60 + day[i].Start.Minute()
		pxOffset := float64(maxOffset) * pxPerMin
		lessonDurationPx = append(lessonDurationPx, int(pxOffset))
	}

	imgNil := image.NewRGBA(image.Rectangle{
		image.Point{0, 0}, image.Point{0, 0},
	})

	lessons := []image.Image{}
	var daymapWG sync.WaitGroup

	for i := 0; i < len(day); i++ {
		lessons = append(lessons, imgNil)
	}

	for i := 0; i < len(day); i++ {
		daymapWG.Add(1)
		go genLesson(
			&daymapWG,
			colorList[day[i].Class],
			&lessons[i],
			w,
			lessonDurationPx[i],
			day[i],
		)
	}

	canvas := image.NewRGBA(
		image.Rectangle{
			image.Point{0, 0},
			image.Point{w, h},
		},
	)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			canvas.Set(x, y, c)
		}
	}

	daymapWG.Wait()

	for i := 0; i < len(day); i++ {
		sr := lessons[i].Bounds()
		dp := image.Point{0, lessonStartPx[i]}
		r := sr.Sub(sr.Min).Add(dp)
		draw.Draw(canvas, r, lessons[i], sr.Min, draw.Src)
	}

	*img = canvas
}

func genTimetable(creds tcUser, w http.ResponseWriter) {
	lessons, err := getLessons(creds)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	// TODO: Use named colors instead of plain RGBA values
	// Using c = color.RGBA{...} doesn't tell you what color you're using (at a glance)
	// Refer to _colors variable for future implementation

	const width = 1135
	const height = 800
	const numOfDays = 5
	const dayWidth = 1135 / numOfDays

	weekday := int(time.Now().Weekday())
	dayOffset := (weekday - 1) * dayWidth
	classes := []string{}

	for i := 0; i < len(lessons); i++ {
		for j := 0; j < len(lessons[i]); j++ {
			if !contains(classes, lessons[i][j].Class) {
				classes = append(classes, lessons[i][j].Class)
			}
		}
	}

	colorList := map[string]color.RGBA{}

	for i := 0; i < len(classes); i++ {
		idx := i % len(colors)
		colorList[classes[i]] = colors[idx]
	}

	days := [numOfDays]image.Image{}
	var wg sync.WaitGroup

	for i := 0; i < numOfDays; i++ {
		c := color.RGBA{0xe0, 0xe0, 0xe0, 0xff}

		if i == weekday-1 {
			c = color.RGBA{0xc2, 0xcd, 0xfc, 0xff}
		} else if i%2 == 0 {
			c = color.RGBA{0xff, 0xff, 0xff, 0xff}
		}

		wg.Add(1)

		go genDay(
			&wg,
			&days[i],
			dayWidth-40,
			height-80,
			c,
			colorList,
			lessons[i],
		)
	}

	canvas := image.NewRGBA(
		image.Rectangle{
			image.Point{0, 0},
			image.Point{width, height},
		},
	)

	for y := 0; y < 40; y++ {
		for x := 0; x < width; x++ {
			canvas.Set(x, y, color.RGBA{0x30, 0x30, 0x30, 0xff})
		}

		for x := dayOffset; x < (dayWidth + dayOffset); x++ {
			canvas.Set(x, y, color.RGBA{0x32, 0x57, 0xad, 0xff})
		}
	}

	for y := 40; y < height; y++ {
		for x := 0; x < width; x++ {
			if (x/dayWidth)%2 == 0 {
				canvas.Set(x, y, color.White)
			} else {
				canvas.Set(x, y, color.RGBA{
					0xe0, 0xe0, 0xe0, 0xff,
				})
			}
		}

		for x := dayOffset; x < (dayWidth + dayOffset); x++ {
			canvas.Set(x, y, color.RGBA{0xc2, 0xcd, 0xfc, 0xff})
		}
	}

	for y := 0; y < height; y++ {
		for i := 1; i < numOfDays; i++ {
			canvas.Set(dayWidth*i, y, color.White)
		}
	}

	boldFont, err := freetype.ParseFont(gobold.TTF)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	face := truetype.NewFace(boldFont, &truetype.Options{
		Size:    16.0,
		DPI:     72,
		Hinting: font.HintingNone,
	})

	d := font.Drawer{
		Dst:  canvas,
		Src:  image.White,
		Face: face,
	}

	today := int(time.Now().Weekday())
	monday := time.Now()
	v := -1

	if today == 0 || today > numOfDays {
		v = 1
	}

	for int(monday.Weekday()) != 1 {
		monday = monday.AddDate(0, 0, v)
	}

	for i := 0; i < numOfDays; i++ {
		s := time.Weekday(i+1).String() + ", "
		s += monday.AddDate(0, 0, i).Format("2 January")

		strLen := font.MeasureString(face, s).Round()
		xpt := dayWidth * i
		xpt += (dayWidth - strLen) / 2

		d.Dot = fixed.Point26_6{
			X: fixed.I(xpt),
			Y: fixed.I(24),
		}

		d.DrawString(s)
	}

	wg.Wait()

	for i := 0; i < numOfDays; i++ {
		xp := dayWidth*i + 20
		sr := days[i].Bounds()
		dp := image.Point{xp, 60}
		r := sr.Sub(sr.Min).Add(dp)
		draw.Draw(canvas, r, days[i], sr.Min, draw.Src)
	}

	if err := png.Encode(w, canvas); err != nil {
		w.WriteHeader(500)
		return
	}
}

// Generate a single task and format it in HTML (for the list of tasks)
func genTask(assignment task, hasDueDate bool, creds tcUser) taskItem {
	task := taskItem{
		Id:       assignment.Id,
		Name:     assignment.Name,
		Platform: assignment.Platform,
		Class:    assignment.Class,
		URL:      assignment.Link,
	}

	dueDate := genDueStr(assignment.Due, creds)
	if hasDueDate {
		task.DueDate = dueDate
	}

	return task
}

// Generate the HTML page for viewing a single task
func genTaskPage(assignment task, creds tcUser) pageData {
	data := pageData{
		PageType: "task",
		Head: headData{
			Title: assignment.Name,
		},
		Body: bodyData{
			TaskData: taskData{
				Id:          assignment.Id,
				Name:        assignment.Name,
				Platform:    assignment.Platform,
				Class:       assignment.Class,
				URL:         assignment.Link,
				IsDue:       false,
				Desc:        "",
				ResLinks:    nil,
				WorkLinks:   nil,
				HasResLinks: false,
			},
		},
	}

	if !assignment.Due.IsZero() {
		data.Body.TaskData.IsDue = true
		data.Body.TaskData.DueDate = genDueStr(assignment.Due, creds)
	}

	if !assignment.Submitted {
		data.Body.TaskData.Submitted = false
	}

	if assignment.Desc != "" {
		taskDesc := assignment.Desc
		taskDesc = strings.ReplaceAll(taskDesc, "\n", "<br>")
		data.Body.TaskData.Desc = taskDesc
	}

	if assignment.ResLinks != nil {
		data.Body.TaskData.HasResLinks = true

		data.Body.TaskData.ResLinks = make(map[string]string)
		for i := 0; i < len(assignment.ResLinks); i++ {
			url := assignment.ResLinks[i][0]
			name := assignment.ResLinks[i][1]
			data.Body.TaskData.ResLinks[name] = url
		}
	}

	//logger.Info("%+v\n", data.Body.TaskData.ResLinks)

	if assignment.Upload {
		data.Body.TaskData.HasUpload = true

		data.Body.TaskData.WorkLinks = make(map[string]string)
		for i := 0; i < len(assignment.WorkLinks); i++ {
			url := assignment.WorkLinks[i][0]
			name := assignment.WorkLinks[i][1]
			data.Body.TaskData.WorkLinks[name] = url
		}
	}

	if assignment.Grade != "" {
		data.Body.TaskData.Grade = assignment.Grade
	}

	if assignment.Comment != "" {
		taskCmt := assignment.Comment
		taskCmt = strings.ReplaceAll(taskCmt, "\n", "<br>")
		data.Body.TaskData.Comment = taskCmt
	}

	return data
}

// Generate a resource link
func genHtmlResLink(className string, res [][2]string) resClass {
	class := resClass{
		Name: className,
	}

	for i := 0; i < len(res); i++ {
		class.ResItems = append(class.ResItems, resItem{
			Name: res[i][1],
			URL:  res[i][0],
		})
	}

	return class
}

// Generate the HTML page (and write that data to http.ResponseWriter)
func genPage(w http.ResponseWriter, templates *template.Template, data pageData) {
	//fmt.Printf("%+v\n", data)
	err := templates.ExecuteTemplate(w, "page", data)
	if err != nil {
		logger.Error(err)
	}

	// TESTING CODE:
	// NOTE: It seems that when fetching data (res or tasks) it fetches the data and writes to
	// the file but that gets overridden by a 404 page instead.

	//var processed bytes.Buffer
	//err := templates.ExecuteTemplate(&processed, "page", data)
	//outputPath := "./result.txt"
	//f, _ := os.Create(outputPath)
	//a := bufio.NewWriter(f)
	//a.WriteString(processed.String())
	//a.Flush()
	//if err != nil {
	//	fmt.Println("Errors:")
	//	logger.Error(err)
	//}
}

// Generate resources and components for the webpage
func genRes(resPath string, resURL string, creds tcUser) (pageData, error) {
	var data pageData

	if resURL == "/timetable" {
		data.PageType = "timetable"
		data.Head.Title = "Timetable"

	} else if resURL == "/tasks" {
		data.PageType = "tasks"
		data.Head.Title = "Tasks"
		data.Body.TasksData.Heading = "Tasks"

		tasks, err := getTasks(creds)
		if err != nil {
			return data, err
		}

		activeTasks := taskType{
			Name:       "Active tasks",
			HasDueDate: true,
		}
		for i := 0; i < len(tasks["active"]); i++ {
			activeTasks.Tasks = append(activeTasks.Tasks, genTask(
				tasks["active"][i],
				true,
				creds,
			))
		}
		data.Body.TasksData.TaskTypes = append(data.Body.TasksData.TaskTypes, activeTasks)

		notDueTasks := taskType{
			Name:       "No due date",
			HasDueDate: false,
		}
		for i := 0; i < len(tasks["notDue"]); i++ {
			notDueTasks.Tasks = append(notDueTasks.Tasks, genTask(
				tasks["notDue"][i],
				false,
				creds,
			))
		}
		data.Body.TasksData.TaskTypes = append(data.Body.TasksData.TaskTypes, notDueTasks)

		overdueTasks := taskType{
			Name:       "Overdue tasks",
			HasDueDate: false,
		}
		for i := 0; i < len(tasks["overdue"]); i++ {
			overdueTasks.Tasks = append(overdueTasks.Tasks, genTask(
				tasks["overdue"][i],
				false,
				creds,
			))
		}
		data.Body.TasksData.TaskTypes = append(data.Body.TasksData.TaskTypes, overdueTasks)

		submittedTasks := taskType{
			Name:       "Submitted tasks",
			HasDueDate: false,
		}
		for i := 0; i < len(tasks["submitted"]); i++ {
			submittedTasks.Tasks = append(submittedTasks.Tasks, genTask(
				tasks["submitted"][i],
				false,
				creds,
			))
		}
		data.Body.TasksData.TaskTypes = append(data.Body.TasksData.TaskTypes, submittedTasks)

		return data, nil

	} else if resURL == "/res" {
		data.PageType = "res"
		data.Head.Title = "Resources"
		data.Body.ResData.Heading = "Resources"

		classes, resLinks, err := getResLinks(creds)
		if err != nil {
			return data, err
		}

		for i := 0; i < len(classes); i++ {
			data.Body.ResData.Classes = append(data.Body.ResData.Classes, genHtmlResLink(
				classes[i],
				resLinks[classes[i]],
			))
		}

	} else {
		return data, errors.ErrNotFound
	}

	return data, nil
}
