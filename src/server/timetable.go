package server

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"net/http"
	"sync"
	"time"

	"git.sr.ht/~kvo/go-std/defs"
	"git.sr.ht/~kvo/go-std/errors"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"

	"main/logger"
	"main/site"
)

var colors = []color.RGBA{
	{0x00, 0x28, 0x70, 0xff}, // Dark blue
	{0x00, 0x70, 0x00, 0xff}, // Green
	{0x58, 0x09, 0x7e, 0xff}, // Purple
	{0xb4, 0x3a, 0x83, 0xff}, // Pink
	{0xaa, 0x00, 0x00, 0xff}, // Dark red
	{0xb4, 0x6a, 0x00, 0xff}, // Ochre
	{0x70, 0x26, 0x00, 0xff}, // Brown
	{0x00, 0x7a, 0xa8, 0xff}, // Cerulean blue
	{0x4f, 0x00, 0x2a, 0xff}, // Tyrian purple
	{0x00, 0x38, 0x34, 0xff}, // Myrtle green
}

func midnight(t time.Time) time.Time {
	return time.Date(
		t.Year(), t.Month(), t.Day(),
		0, 0, 0, 0,
		t.Location(),
	)
}

func genLessonImg(daymapWG *sync.WaitGroup, c color.RGBA, img *image.Image, w, h int, l site.Lesson) {
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
		logger.Error(errors.New("font (bold) parsing failed", err))
		*img = canvas
		daymapWG.Done()
		return
	}

	regFont, err := freetype.ParseFont(goregular.TTF)
	if err != nil {
		logger.Error(errors.New("font (reg) parsing failed", err))
		*img = canvas
		daymapWG.Done()
		return
	}

	headFace := truetype.NewFace(boldFont, &truetype.Options{
		Size:    16.0,
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
	d.Src = image.White
	d.Face = regFace

	d.Dot = fixed.Point26_6{
		X: fixed.I(5),
		Y: fixed.I(35),
	}

	s := l.Start.Format("15:04") + "–" + l.End.Format("15:04")
	s += fmt.Sprintf(
		" (%d mins)",
		int(l.End.Sub(l.Start).Minutes()),
	)
	d.DrawString(s)

	d.Dot = fixed.Point26_6{
		X: fixed.I(5),
		Y: fixed.I(50),
	}

	if l.Teacher != "" {
		d.DrawString(l.Teacher + ", " + l.Room)
	} else {
		d.DrawString(l.Room)
	}
	*img = canvas
}

func genDayImg(wg *sync.WaitGroup, img *image.Image, w int, h int, c color.RGBA, colorList map[string]color.RGBA, day []site.Lesson) {
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
		go genLessonImg(
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

// Generate a timetable image in PNG format.
func genTimetableImg(user site.User, w http.ResponseWriter) {
	lessons, err := getLessons(user)
	if err != nil {
		logger.Error(errors.New("failed to get lessons", err))
		w.WriteHeader(500)
		return
	}

	const width = 1135
	const height = 800
	const numOfDays = 5
	const dayWidth = 1135 / numOfDays

	classes := []string{}

	for i := 0; i < len(lessons); i++ {
		for j := 0; j < len(lessons[i]); j++ {
			if !defs.Has(classes, lessons[i][j].Class) {
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
		c := color.RGBA{0xe0, 0xe0, 0xe0, 0xff} // #e0e0e0 = very light grey
		if i%2 == 0 {
			c = color.RGBA{0xff, 0xff, 0xff, 0xff} // white
		}

		wg.Add(1)

		go genDayImg(
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
			canvas.Set(x, y, color.RGBA{0x30, 0x30, 0x30, 0xff}) // #303030 = grey
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
	}

	for y := 0; y < height; y++ {
		for i := 1; i < numOfDays; i++ {
			canvas.Set(dayWidth*i, y, color.White)
		}
	}

	boldFont, err := freetype.ParseFont(gobold.TTF)
	if err != nil {
		logger.Error(errors.New("font (bold) parsing failed", err))
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

	today := int(time.Now().In(user.Timezone).Weekday())
	monday := time.Now().In(user.Timezone)
	v := -1

	if today == int(time.Sunday) || today > numOfDays {
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
		logger.Error(errors.New("timetable image encoding failed", err))
		w.WriteHeader(500)
		return
	}
}

func TimetableHTML(user site.User) (timetableData, error) {
	data := timetableData{}

	var weekStartIdx, weekEndIdx int
	now := midnight(time.Now().In(user.Timezone))

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

	lessons, err := schools[user.School].Lessons(user, weekStart, weekEnd)
	if err != nil {
		return data, errors.New("failed to get lessons", err)
	}

	const numOfDays = 5
	data.Days = make([]ttDay, numOfDays)

	//weekday := int(time.Now().In(user.Timezone).Weekday())
	//dayOffset := (weekday - 1) * dayWidth
	classes := []string{}

	for _, lesson := range lessons {
		if !defs.Has(classes, lesson.Class) {
			classes = append(classes, lesson.Class)
		}
	}

	colorList := map[string]color.RGBA{}

	for i := 0; i < len(classes); i++ {
		idx := i % len(colors)
		colorList[classes[i]] = colors[idx]
	}

	today := int(time.Now().In(user.Timezone).Weekday())
	monday := now.In(user.Timezone)
	v := -1

	if today == int(time.Sunday) || today > numOfDays {
		v = 1
	}

	for int(monday.Weekday()) != 1 {
		monday = monday.AddDate(0, 0, v)
	}

	for i := range data.Days {
		s := time.Weekday(i+1).String() + ", "
		s += monday.AddDate(0, 0, i).Format("2 January")
		data.Days[i].Day = s
	}

	curDay := 0

	dayStart := 800.0 // is 08:00

	for _, lesson := range lessons {
		if lesson.Start.After(monday.AddDate(0, 0, curDay+1)) {
			curDay++
		}

		if curDay > numOfDays {
			break
		}

		lesson.Start = lesson.Start.In(user.Timezone)
		lesson.End = lesson.End.In(user.Timezone)

		startMins := lesson.Start.Hour()*60 + lesson.Start.Minute()
		endMins := lesson.End.Hour()*60 + lesson.End.Minute()
		duration := endMins - startMins

		c := colorList[lesson.Class]
		hexColor := fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)

		textColor := "#ffffff"
		luminance := (0.299*float32(c.R) + 0.587*float32(c.G) + 0.114*float32(c.B)) / 255
		if luminance > 0.5 {
			textColor = "#000000"
		}

		topOffset := math.Round(float64(startMins)*10/6 - dayStart)
		height := math.Round(float64(duration) * 10 / 6)

		classInfo := ttLesson{
			Class:     lesson.Class,
			Height:    height,
			TopOffset: topOffset,
			Room:      lesson.Room,
			Teacher:   lesson.Teacher,
			Notice:    lesson.Notice,
			Color:     textColor,
			BGColor:   hexColor,
		}

		classInfo.FormattedTime = lesson.Start.Format("15:04") + "–" + lesson.End.Format("15:04")
		classInfo.Duration = fmt.Sprintf(
			"%d mins",
			int(lesson.End.Sub(lesson.Start).Minutes()),
		)
		data.Days[curDay].Lessons = append(data.Days[curDay].Lessons, classInfo)
	}

	if time.Now().In(user.Timezone).Before(monday) {
		data.CurrentDay = 0
	} else {
		data.CurrentDay = int(today)
	}

	return data, nil
}
