package plat

import (
	"io"
	"time"
)

// User represents an authenticated TaskCollect user.
type User struct {
	Timezone   *time.Location
	School     string
	DispName   string
	Username   string
	Password   string
	Token      string
	SiteTokens map[string]string
	GAuthID    []byte
}

// Class represents a class into which the user is enrolled in.
type Class struct {
	Name     string
	Link     string
	Platform string
	Id       string
}

// Task represents a task assigned to the user.
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
	Graded    bool
	Grade     string
	Score     float64
	Comment   string
	Platform  string
	Id        string
}

// File represents a file submission to a platform.
type File struct {
	Name     string
	MimeType string
	Reader   io.Reader
}

// Grade represents a grade for a class in a report card.
type Grade struct {
	Class string
	Grade string
	Score float64
}

// Lesson represents a lesson.
type Lesson struct {
	Start   time.Time
	End     time.Time
	Class   string
	Room    string
	Teacher string
	Notice  string
}

// Report represents a report card.
type Report struct {
	Grades   []Grade
	Released time.Time
}

// Resource represents an educational resource provided by a teacher for a
// class.
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
