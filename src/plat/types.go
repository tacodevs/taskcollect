package plat

import (
	"io"
	"time"
)

type Task struct {
	Name      string
	Class     string
	Link      string
	Desc      string
	Due       time.Time
	Posted    time.Time
	ResLinks  [][2]string
	Upload    bool
	WorkLinks [][2]string
	Submitted bool
	Result    TaskGrade
	Comment   string
	Platform  string
	Id        string
}

type TaskGrade struct {
	Exists bool
	Grade  string
	Mark   float64
}

type File struct {
	Name     string
	MimeType string
	Reader   io.Reader
}

type Lesson struct {
	Start   time.Time
	End     time.Time
	Class   string
	Room    string
	Teacher string
	Notice  string
}

type Resource struct {
	Name     string
	Class    string
	Link     string
	Desc     string
	Posted   time.Time
	ResLinks [][2]string
	Platform string
	Id       string
}
