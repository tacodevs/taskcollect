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

	"git.sr.ht/~kvo/libgo/defs"
	"git.sr.ht/~kvo/libgo/errors"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"

	"main/logger"
	"main/plat"
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

func genLessonImg(daymapWG *sync.WaitGroup, c color.RGBA, img *image.Image, w, h int, l plat.Lesson) {
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

	boldFont, e := freetype.ParseFont(gobold.TTF)
	if e != nil {
		err := errors.New(e.Error(), nil)
		logger.Error(errors.New("font (bold) parsing failed", err))
		*img = canvas
		daymapWG.Done()
		return
	}

	regFont, e := freetype.ParseFont(goregular.TTF)
	if e != nil {
		err := errors.New(e.Error(), nil)
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

func genDayImg(wg *sync.WaitGroup, img *image.Image, w int, h int, c color.RGBA, colorList map[string]color.RGBA, day []plat.Lesson) {
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
func genTimetableImg(creds plat.User, w http.ResponseWriter) {
	lessons, err := getLessons(creds)
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

	boldFont, e := freetype.ParseFont(gobold.TTF)
	if e != nil {
		err := errors.New(e.Error(), nil)
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

	today := int(time.Now().In(creds.Timezone).Weekday())
	monday := time.Now().In(creds.Timezone)
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

	if e := png.Encode(w, canvas); e != nil {
		err = errors.New(e.Error(), nil)
		logger.Error(errors.New("timetable image encoding failed", err))
		w.WriteHeader(500)
		return
	}
}

// Generate data for the HTML timetable.
func genTimetable(creds plat.User) (timetableData, errors.Error) {
	data := timetableData{}

	lessons, err := getLessons(creds)
	if err != nil {
		return data, errors.New("failed to get lessons", err)
	}

	const numOfDays = 5

	//weekday := int(time.Now().In(creds.Timezone).Weekday())
	//dayOffset := (weekday - 1) * dayWidth
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

	today := int(time.Now().In(creds.Timezone).Weekday())
	monday := time.Now().In(creds.Timezone)
	v := -1

	if today == int(time.Sunday) || today > numOfDays {
		v = 1
	}

	for int(monday.Weekday()) != 1 {
		monday = monday.AddDate(0, 0, v)
	}

	for i := 0; i < numOfDays; i++ {
		day := lessons[i]
		d := ttDay{}

		dayStart := 800.0 // is 08:00

		for j := 0; j < len(day); j++ {
			day[j].Start = day[j].Start.In(creds.Timezone)
			day[j].End = day[j].End.In(creds.Timezone)

			startMins := day[j].Start.Hour()*60 + day[j].Start.Minute()
			endMins := day[j].End.Hour()*60 + day[j].End.Minute()
			duration := endMins - startMins

			c := colorList[day[j].Class]
			hexColor := fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)

			textColor := "#ffffff"
			luminance := (0.299*float32(c.R) + 0.587*float32(c.G) + 0.114*float32(c.B)) / 255
			if luminance > 0.5 {
				textColor = "#000000"
			}

			topOffset := math.Round(float64(startMins)*10/6 - dayStart)
			height := math.Round(float64(duration) * 10 / 6)

			lesson := ttLesson{
				Class:     day[j].Class,
				Height:    height,
				TopOffset: topOffset,
				Room:      day[j].Room,
				Teacher:   day[j].Teacher,
				Notice:    day[j].Notice,
				Color:     textColor,
				BGColor:   hexColor,
			}

			lesson.FormattedTime = day[j].Start.Format("15:04") + "–" + day[j].End.Format("15:04")
			lesson.Duration = fmt.Sprintf(
				"%d mins",
				int(day[j].End.Sub(day[j].Start).Minutes()),
			)
			d.Lessons = append(d.Lessons, lesson)
		}

		s := time.Weekday(i+1).String() + ", "
		s += monday.AddDate(0, 0, i).Format("2 January")
		d.Day = s

		data.Days = append(data.Days, d)
	}

	if time.Now().In(creds.Timezone).Before(monday) {
		data.CurrentDay = 0
	} else {
		data.CurrentDay = int(today)
	}

	return data, nil
}
