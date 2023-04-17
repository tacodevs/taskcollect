package plat

import (
	"sort"
	"time"
)

// Mux is a platform multiplexer. Methods can be invoked on it to select the
// platform functions to multiplex, and alternatively to create a multi-platform
// function call.
type Mux struct {
	classes []func(User, chan []Class, chan error)
	graded  []func(User, chan []Task, chan error)
	items   []func(User, chan []Task, chan error, []Class)
	lessons func(User, time.Time, time.Time) ([]Lesson, error)
	reports func(User) ([]Report, error)
}

// AddClasses adds the class list retrieval function f to *m for platform
// multiplexing.
func (m *Mux) AddClasses(f func(User, chan []Class, chan error)) {
	m.classes = append(m.classes, f)
}

// AddGraded adds the graded tasks retrieval function f to *m for platform
// mulitplexing.
func (m *Mux) AddGraded(f func(User, chan []Task, chan error)) {
	m.graded = append(m.graded, f)
}

// AddItems adds the class tasks/resources retrieval function f to *m for
// platform multiplexing.
func (m *Mux) AddItems(f func(User, chan []Task, chan error, []Class)) {
	m.items = append(m.items, f)
}

// AddLessons sets the lessons retrieval function for *m as f for platform
// multiplexing.
func (m *Mux) AddLessons(f func(User, time.Time, time.Time) ([]Lesson, error)) {
	m.lessons = f
}

// AddReports sets the report card retrieval function for *m as f for platform
// multiplexing.
func (m *Mux) AddReports(f func(User) ([]Report, error)) {
	m.reports = f
}

// Classes returns a list of classes from all platforms multiplexed by m.
func (m Mux) Classes(creds User) ([]Class, error) {
	var classes []Class
	c := make(chan []Class)
	errs := make(chan error)
	for _, f := range m.classes {
		go f(creds, c, errs)
	}
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}
	for list := range c {
		classes = append(classes, list...)
	}
	sort.SliceStable(classes, func(i, j int) bool {
		return classes[i].Name < classes[j].Name
	})
	return classes, nil
}

// Graded returns a list of graded tasks from all platforms multiplexed by m.
func (m Mux) Graded(creds User) ([]Task, error) {
	var graded []Task
	c := make(chan []Task)
	errs := make(chan error)
	for _, f := range m.graded {
		go f(creds, c, errs)
	}
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}
	for list := range c {
		graded = append(graded, list...)
	}
	sort.SliceStable(graded, func(i, j int) bool {
		return graded[i].Posted.After(graded[j].Posted)
	})
	return graded, nil
}

// Items returns a list of tasks and resources for all specified classes.
func (m Mux) Items(creds User, classes ...Class) ([]Task, error) {
	var items []Task
	c := make(chan []Task)
	errs := make(chan error)
	for _, f := range m.items {
		go f(creds, c, errs, classes)
	}
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}
	for list := range c {
		items = append(items, list...)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Posted.After(items[j].Posted)
	})
	return items, nil
}

// Lessons returns a list of lessons occuring from start to end.
func (m Mux) Lessons(creds User, start, end time.Time) ([]Lesson, error) {
	lessons, err := m.lessons(creds, start, end)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(lessons, func(i, j int) bool {
		return lessons[i].Start.Before(lessons[j].Start)
	})
	return lessons, nil
}

// Reports returns a series of report cards.
func (m Mux) Reports(creds User) ([]Report, error) {
	reports, err := m.reports(creds)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(reports, func(i, j int) bool {
		return reports[i].Released.After(reports[j].Released)
	})
	return reports, nil
}
