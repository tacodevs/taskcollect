package server

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"net/http"
	"time"

	"io"
	"crypto/rand"

	"git.sr.ht/~kvo/go-std/defs"
	"git.sr.ht/~kvo/go-std/errors"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"

	"main/site"
)

type UUID [16]byte

var (
	colors   = make(map[string]color.RGBA)
	charcoal = color.RGBA{0x30, 0x30, 0x30, 0xff}
	silver   = color.RGBA{0xe0, 0xe0, 0xe0, 0xff}
)

var palette = []color.RGBA{
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

func since(day1, day2 time.Time) int {
	y, m, d := day2.Date()
	u2 := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	y, m, d = day1.In(day2.Location()).Date()
	u1 := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	days := u2.Sub(u1) / (24 * time.Hour)
	return int(days)
}

func fillrect(img *image.RGBA, rect image.Rectangle, c color.Color) {
	for x := rect.Min.X; x < rect.Max.X; x++ {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			img.Set(x, y, c)
		}
	}
}

func imprint(dest *image.RGBA, text string, face font.Face, pos image.Point) {
	pen := font.Drawer{
		Dst:  dest,
		Src:  image.White,
		Face: face,
	}
	pen.Dot = fixed.Point26_6{
		X: fixed.I(pos.X),
		Y: fixed.I(pos.Y),
	}
	pen.DrawString(text)
}

// TODO: add ability to scale text within blocks if block height is less than min height
func mkblock(canvas *image.RGBA, lesson site.Lesson, day, y, h int) error {
	bg := colors[lesson.Class]
	x, w := 20+(day*227), 187
	block := image.NewRGBA(
		image.Rectangle{
			image.Point{0, 0},
			image.Point{w, h},
		},
	)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			block.Set(x, y, bg)
		}
	}
	boldttf, err := freetype.ParseFont(gobold.TTF)
	if err != nil {
		return errors.New("cannot parse bold font", nil)
	}
	regttf, err := freetype.ParseFont(goregular.TTF)
	if err != nil {
		return errors.New("cannot parse regular font", nil)
	}
	headface := truetype.NewFace(boldttf, &truetype.Options{
		Size:    16.0,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	regface := truetype.NewFace(regttf, &truetype.Options{
		Size:    12.0,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	timeln := fmt.Sprintf(
		"%s–%s (%d mins)",
		lesson.Start.Format("15:04"),
		lesson.End.Format("15:04"),
		int(lesson.End.Sub(lesson.Start).Minutes()),
	)
	roomln := lesson.Room
	if lesson.Teacher != "" {
		roomln = lesson.Teacher + ", " + lesson.Room
	}
	imprint(block, lesson.Class, headface, image.Pt(5, 20))
	imprint(block, timeln, regface, image.Pt(5, 35))
	imprint(block, roomln, regface, image.Pt(5, 50))
	draw.Draw(canvas, image.Rect(x, y, x+w, y+h), block, image.Pt(0, 0), draw.Src)
	return nil
}

// TODO: scale y and h differently depending on day start/end times
func mkblocks(canvas *image.RGBA, lessons []site.Lesson, monday time.Time, days int) error {
	minPerDay := float64(600)
	pxPerMin := float64(800-80) / minPerDay
	for _, lesson := range lessons {
		day := since(monday, lesson.Start)
		if day > days || lesson.Start.Before(monday) {
			continue
		}
		ymins := (lesson.Start.Hour()-8)*60 + lesson.Start.Minute()
		hmins := lesson.End.Sub(lesson.Start).Minutes()
		y := int(float64(ymins)*pxPerMin) + 60
		h := int(float64(hmins) * pxPerMin)
		err := mkblock(canvas, lesson, day, y, h)
		if err != nil {
			return err
		}
	}
	return nil
}

func mkcanvas(width, height int, monday time.Time, days int) (*image.RGBA, error) {
	dayWidth := width / days
	canvas := image.NewRGBA(
		image.Rectangle{
			image.Point{0, 0},
			image.Point{width, height},
		},
	)
	fillrect(canvas, image.Rect(0, 0, width, 40), charcoal)
	for n := 0; n < days; n++ {
		c := color.Color(silver)
		if n%2 == 0 {
			c = color.White
		}
		fillrect(canvas, image.Rect(n*dayWidth, 40, (n+1)*dayWidth, height), c)
	}
	for n := 1; n < days; n++ {
		for y := 0; y < height; y++ {
			canvas.Set(dayWidth*n, y, color.White)
		}
	}
	bold, err := freetype.ParseFont(gobold.TTF)
	if err != nil {
		return nil, errors.New("cannot parse bold font", err)
	}
	face := truetype.NewFace(bold, &truetype.Options{
		Size:    16.0,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	for n := 0; n < days; n++ {
		date := monday.AddDate(0, 0, n).Format("Monday, 2 January")
		len := font.MeasureString(face, date).Round()
		x := (dayWidth * n) + ((dayWidth - len) / 2)
		imprint(canvas, date, face, image.Pt(x, 24))
	}
	return canvas, nil
}

// TODO: add function to calc end from start
// TODO: allow user to specify start date
func TimetablePNG(user site.User, w http.ResponseWriter) error {
	var days = 5
	var width, height = 1135, 800
	var classes []string
	var start, end time.Time
	var now = midnight(time.Now().In(user.Timezone))
	var weekday = now.Weekday()

	switch weekday {
	case time.Saturday:
		start = now.AddDate(0, 0, 2)
		end = now.AddDate(0, 0, 6)
	default:
		start = now.AddDate(0, 0, 1-int(weekday))
		end = now.AddDate(0, 0, 5-int(weekday))
	}
	school, ok := schools[user.School]
	if !ok {
		w.WriteHeader(500)
		return errors.New("unsupported platform", nil)
	}
	lessons, err := school.Lessons(user, start, end)
	if err != nil {
		w.WriteHeader(500)
		return errors.New("cannot get lessons", err)
	}
	for _, lesson := range lessons {
		if !defs.Has(classes, lesson.Class) {
			classes = append(classes, lesson.Class)
		}
	}
	for i, class := range classes {
		colors[class] = palette[i%len(palette)]
	}
	canvas, err := mkcanvas(width, height, start, days)
	if err != nil {
		w.WriteHeader(500)
		return errors.New("cannot make timetable canvas", err)
	}
	if err := mkblocks(canvas, lessons, start, days); err != nil {
		w.WriteHeader(500)
		return errors.New("cannot draw lesson blocks", err)
	}
	if err := png.Encode(w, canvas); err != nil {
		w.WriteHeader(500)
		return errors.New("cannot render PNG timetable", err)
	}
	return nil
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

	school, ok := schools[user.School]
	if !ok {
		return data, errors.New("unsupported platform", nil)
	}
	lessons, err := school.Lessons(user, weekStart, weekEnd)
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

	colors := map[string]color.RGBA{}
	for i := 0; i < len(classes); i++ {
		colors[classes[i]] = palette[i%len(palette)]
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
		for lesson.Start.After(monday.AddDate(0, 0, curDay+1)) {
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

		c := colors[lesson.Class]
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

// Export the user calendar as a .ics file

func TimetableIcal(user site.User, start, end time.Time, w http.ResponseWriter) error {
	var err error
	iCalString := ""

	school, ok := schools[user.School]
	if !ok {
		w.WriteHeader(500)
		return errors.New("unsupported platform", nil)
	}
	lessons, err := school.Lessons(user, start, end)

	if err != nil {
		w.WriteHeader(500)
		return errors.New("failed to get lessons", err)
	}
	//build the start of the string
	iCalString += "BEGIN:VCALENDAR\n"
	iCalString += "VERSION:2.0\n"
	iCalString += "PRODID:taco/calendar\n"
	iCalString += "CALSCALE:GREGORIAN\n"
	iCalString += "METHOD:PUBLISH\n"
	for _, lesson := range lessons {
		uuid, err := GenerateUUID()
		if err != nil {
			w.WriteHeader(500)
			return errors.New("failed to generate UUID", err)
		}

		iCalString += "BEGIN:VEVENT\n"
		iCalString += string(uuid[:]) + "\n"
		iCalString += "DTSTAMP:" + time.Now().Format("20060102T150405Z") + "\n"
		iCalString += "DTSTART:" + lesson.Start.Format("20060102T150405Z") + "\n"
		iCalString += "DTEND:" + lesson.End.Format("20060102T150405Z") + "\n"
		iCalString += "SUMMARY:" + lesson.Class + "\n"
		iCalString += "DESCRIPTION:" + lesson.Teacher + "\n"
		iCalString += "LOCATION:" + lesson.Room + "\n"
		iCalString += "END:VEVENT\n"
	}
	iCalString += "END:VCALENDAR\n"

	_, err = io.WriteString(w, iCalString);
	if err!=nil{
		w.WriteHeader(500)
		return errors.New("Failed to write calendar file", err)
	}
	return nil
}

func GenerateUUID() (u *UUID, err error){
	// generates a version 4 UUID and returns the byte string
	u = new(UUID)
	_, err = rand.Read(u[:])
	if err != nil {
		return nil, errors.New("failed to generate UUID", err)
	}
	u[8] = (u[8] | 0x40) & 0x7F
	u[6] = (u[6] & 0xF) | (4 << 4)

	return u, nil
}
