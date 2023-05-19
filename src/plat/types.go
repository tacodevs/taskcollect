package plat

import (
	"image/color"
	"time"
)

// Class represents a class into which the user is enrolled in.
type Class struct {
	Name     string
	Link     string
	Platform string
	Id       string
}

// Event represents a calendar event.
type Event struct {
	Name     string
	Start    time.Time
	End      time.Time
	Location string
	Category string
	Color    color.Color
	Platform string
}

// Grade represents a grade for a class in a report card.
type Grade struct {
	Class string
	Grade string
	Score float64
}

// Item represents a generic task/resource item.
type Item Task

// Lesson represents a lesson.
type Lesson struct {
	Start   time.Time
	End     time.Time
	Class   string
	Room    string
	Teacher string
	Notice  string
}

// Message represents a parsed email-like message of a proprietary format.
type Message struct {
	From    string
	To      string
	Sent    time.Time
	Subject string
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

// User represents an authenticated TaskCollect user.
type User struct {
	Timezone   *time.Location
	School     string
	DispName   string
	Email      string
	Username   string
	Password   string
	Token      string
	SiteTokens map[string]string
}
