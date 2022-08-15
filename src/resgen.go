package main

import (
	"html"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
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

func contains(strSlice []string, str string) bool {
	for i := 0; i < len(strSlice); i++ {
		if strSlice[i] == str {
			return true
		}
	}

	return false
}

func genDueStr(due time.Time, creds user) string {
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

func genTimetable(creds user, w http.ResponseWriter) {
	lessons, err := getLessons(creds)

	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

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

func genHtmlTasks(assignment task, noDue bool, creds user) string {
	dueDate := genDueStr(assignment.Due, creds)
	h := "<tr>\n"

	if !noDue {
		h += "<td>" + html.EscapeString(dueDate)
		h += "</td>\n"
	}

	// TODO: use HTML templates as this is messy.
	h += "<td>" + html.EscapeString(assignment.Class)
	h += "</td>\n<td>" + `<a href="/tasks/`
	h += html.EscapeString(assignment.Platform) + "/"
	h += html.EscapeString(assignment.Id) + `">`
	h += html.EscapeString(assignment.Name)
	h += "</a></td>\n" + `<td><a href="`
	h += html.EscapeString(assignment.Link) + `">`
	h += html.EscapeString(assignment.Link)
	h += "</a></td>\n</tr>\n"

	return h
}

func genHtmlResLinks(class string, res [][2]string) string {
	h := "<h2>" + html.EscapeString(class)
	h += "</h2>\n<ul>\n"

	for i := 0; i < len(res); i++ {
		h += "<li><a href=\""
		h += html.EscapeString(res[i][0]) + "\">"
		h += html.EscapeString(res[i][1])
		h += "</a></li>\n"
	}

	h += "</ul>\n"
	return h
}

func genHtmlTask(assignment task, creds user) string {
	h := "<h1>" + html.EscapeString(assignment.Name) + "</h1>\n<h3>"
	h += html.EscapeString(assignment.Class) + "</h3>\n" + `<a href="`
	h += html.EscapeString(assignment.Link)
	h += `">View task in source platform</a>` + "\n"

	if !assignment.Due.IsZero() || !assignment.Submitted {
		h += "<hr>\n"
	}

	if !assignment.Due.IsZero() {
		h += "<h4>Due "
		h += html.EscapeString(genDueStr(assignment.Due, creds))
		h += "</h4>\n"
	}

	if !assignment.Submitted {
		h += `<a href="/tasks/`
		h += html.EscapeString(assignment.Platform) + "/"
		h += html.EscapeString(assignment.Id)
		h += "/submit\">Submit work</a>\n"
	}

	if assignment.Desc != "" {
		taskDesc := html.EscapeString(assignment.Desc)
		taskDesc = strings.ReplaceAll(taskDesc, "\n", "<br>")
		h += "<hr>\n<h4>Task description</h4>\n<p>"
		h += taskDesc + "</p>\n"
	}

	if assignment.ResLinks != nil {
		h += "<hr>\n<h4>Linked resources</h4>\n<ul>\n"
		for i := 0; i < len(assignment.ResLinks); i++ {
			h += "<li><a href=\""
			h += html.EscapeString(assignment.ResLinks[i][0])
			h += "\">"
			h += html.EscapeString(assignment.ResLinks[i][1])
			h += "</a></li>\n"
		}
		h += "</ul>\n"
	}

	if assignment.Upload == true {
		h += "<hr>\n<h4>Upload file</h4>\n" + `<form method="POST" `
		h += `enctype="multipart/form-data" action="/tasks/`
		h += html.EscapeString(assignment.Platform) + "/"
		h += html.EscapeString(assignment.Id) + uploadHtml
		h += html.EscapeString(assignment.Platform) + "/"
		h += html.EscapeString(assignment.Id) + "/remove\">\n"

		for i := 0; i < len(assignment.WorkLinks); i++ {
			h += `<input type="checkbox" name="`
			h += html.EscapeString(assignment.WorkLinks[i][1])
			h += `">` + "\n" + `<label for="`
			h += html.EscapeString(assignment.WorkLinks[i][1])
			h += `"><a href="`
			h += html.EscapeString(assignment.WorkLinks[i][0])
			h += `">`
			h += html.EscapeString(assignment.WorkLinks[i][1])
			h += "</a></label><br>\n"
		}

		h += `<input type="submit" value="Remove file">` + "\n</form>\n"
	}

	if assignment.Grade != "" || assignment.Comment != "" {
		h += "<hr>\n"
	}

	if assignment.Grade != "" {
		h += "<h3>Grade: "
		h += html.EscapeString(assignment.Grade) + "</h3>\n"
	}

	if assignment.Comment != "" {
		taskCmt := html.EscapeString(assignment.Comment)
		taskCmt = strings.ReplaceAll(taskCmt, "\n", "<br>")
		h += "<h4>Teacher comment:</h4>\n<p>"
		h += taskCmt + "</p>\n"
	}

	return h
}

func genPage(title, htmlBody string) string {
	webpage := htmlHead + html.EscapeString(title) + htmlNav + htmlBody
	return webpage + htmlEnd
}

func genRes(resource string, creds user, gcid []byte) ([]byte, error) {
	var title string
	var htmlBody string

	if resource == "/timetable" {
		title = "Timetable"
		htmlBody = `<img id="timetable" src="/timetable.png" alt="timetable.png">`
		htmlBody += "\n"
	} else if resource == "/tasks" {
		title = "Tasks"
		htmlBody = tasksHeader
		tasks, err := getTasks(creds, gcid)

		if err != nil {
			return []byte{}, err
		}

		for i := 0; i < len(tasks["tasks"]); i++ {
			htmlBody += genHtmlTasks(
				tasks["tasks"][i],
				false,
				creds,
			)
		}

		htmlBody += notDueHeader

		for i := 0; i < len(tasks["notDue"]); i++ {
			htmlBody += genHtmlTasks(
				tasks["notDue"][i],
				true,
				creds,
			)
		}

		htmlBody += overdueHeader

		for i := 0; i < len(tasks["overdue"]); i++ {
			htmlBody += genHtmlTasks(
				tasks["overdue"][i],
				true,
				creds,
			)
		}

		htmlBody += submittedHeader

		for i := 0; i < len(tasks["submitted"]); i++ {
			htmlBody += genHtmlTasks(
				tasks["submitted"][i],
				true,
				creds,
			)
		}

		htmlBody += "</table>\n"
	} else if resource == "/res" {
		title = "Resources"
		htmlBody = "<h1>Resources</h1>\n"

		classes, resLinks, err := getResLinks(creds, gcid)
		if err != nil {
			return []byte{}, err
		}

		for i := 0; i < len(classes); i++ {
			htmlBody += genHtmlResLinks(
				classes[i],
				resLinks[classes[i]],
			)
		}
	} else {
		return []byte{}, errNotFound
	}

	return []byte(genPage(title, htmlBody)), nil
}
