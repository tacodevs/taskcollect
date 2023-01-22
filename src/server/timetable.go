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

	"codeberg.org/kvo/builtin"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"

	"main/errors"
	"main/logger"
	"main/plat"
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

	boldFont, err := freetype.ParseFont(gobold.TTF)
	if err != nil {
		logger.Error(errors.NewError("server.genLessonImg", "font (bold) parsing failed", err))
		*img = canvas
		daymapWG.Done()
		return
	}

	regFont, err := freetype.ParseFont(goregular.TTF)
	if err != nil {
		logger.Error(errors.NewError("server.genLessonImg", "font (reg) parsing failed", err))
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

	s := l.Start.Format("15:04") + "–" + l.End.Format("15:04")
	s += ", " + l.Room
	d.DrawString(s)

	d.Dot = fixed.Point26_6{
		X: fixed.I(5),
		Y: fixed.I(50 + offset),
	}

	d.DrawString(l.Teacher)
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
func genTimetableImg(creds User, w http.ResponseWriter) {
	lessons, err := getLessons(creds)
	if err != nil {
		logger.Error(errors.NewError("server.genTimetableImg", "failed to get lessons", err))
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
			if !builtin.Contains(classes, lessons[i][j].Class) {
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

		if i == weekday-1 {
			c = color.RGBA{0xc2, 0xcd, 0xfc, 0xff} // #c2cdfc = very light blue
		} else if i%2 == 0 {
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
		logger.Error(errors.NewError("server.genTimetableImg", "font (bold) parsing failed", err))
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
		logger.Error(errors.NewError("server.genTimetableImg", "timetable image encoding failed", err))
		w.WriteHeader(500)
		return
	}
}

// Generate data for the HTML timetable.
func genTimetable(creds User) (timetableData, error) {
	data := timetableData{}

	lessons, err := getLessons(creds)
	if err != nil {
		return data, errors.NewError("server.genTimetable", "failed to get lessons", err)
	}

	const numOfDays = 5

	//weekday := int(time.Now().Weekday())
	//dayOffset := (weekday - 1) * dayWidth
	classes := []string{}

	for i := 0; i < len(lessons); i++ {
		for j := 0; j < len(lessons[i]); j++ {
			if !builtin.Contains(classes, lessons[i][j].Class) {
				classes = append(classes, lessons[i][j].Class)
			}
		}
	}

	colorList := map[string]color.RGBA{}

	for i := 0; i < len(classes); i++ {
		idx := i % len(colors)
		colorList[classes[i]] = colors[idx]
	}

	today := int(time.Now().Weekday())
	monday := time.Now()
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
				Class:         day[j].Class,
				FormattedTime: day[j].Start.Format("15:04") + "–" + day[j].End.Format("15:04"),
				Height:        height,
				TopOffset:     topOffset,
				Room:          day[j].Room,
				Teacher:       day[j].Teacher,
				Notice:        day[j].Notice,
				Color:         textColor,
				BGColor:       hexColor,
			}

			d.Lessons = append(d.Lessons, lesson)
		}

		s := time.Weekday(i+1).String() + ", "
		s += monday.AddDate(0, 0, i).Format("2 January")
		d.Day = s

		data.Days = append(data.Days, d)
	}

	data.CurrentDay = today

	return data, nil
}