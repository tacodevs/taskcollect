package plat

import (
	"sort"
	"time"

	"git.sr.ht/~kvo/libgo/errors"
)

// Mux is a platform multiplexer. Methods can be invoked on it to select the
// platform functions to multiplex, and alternatively to create a multi-platform
// function call.
type Mux struct {
	auth     []func(User, chan [2]string, chan errors.Error, *int)
	classes  []func(User, chan []Class, chan errors.Error, *int)
	duetasks []func(User, chan []Task, chan errors.Error, *int)
	events   []func(User, chan []Event, chan errors.Error, *int)
	graded   []func(User, chan []Task, chan errors.Error, *int)
	items    []func(User, chan []Item, chan errors.Error, []Class, *int)
	lessons  func(User, time.Time, time.Time) ([]Lesson, errors.Error)
	messages []func(User, chan []Message, chan errors.Error, *int)
	reports  func(User) ([]Report, errors.Error)
}

// Return a new instance of Mux.
func NewMux() *Mux {
	return &Mux{}
}

// AddAuth adds the authentication function f to m for platform authentication
// multiplexing.
func (m *Mux) AddAuth(f func(User, chan [2]string, chan errors.Error, *int)) {
	m.auth = append(m.auth, f)
}

// AddClasses adds the class list retrieval function f to m for platform
// multiplexing.
func (m *Mux) AddClasses(f func(User, chan []Class, chan errors.Error, *int)) {
	m.classes = append(m.classes, f)
}

// AddDueTasks adds the active tasks retrieval function f to m for platform
// multiplexing.
func (m *Mux) AddDueTasks(f func(User, chan []Task, chan errors.Error, *int)) {
	m.duetasks = append(m.duetasks, f)
}

// AddEvents adds the calendar events retrieval function f to m for platform
// multiplexing.
func (m *Mux) AddEvents(f func(User, chan []Event, chan errors.Error, *int)) {
	m.events = append(m.events, f)
}

// AddGraded adds the graded tasks retrieval function f to m for platform
// mulitplexing.
func (m *Mux) AddGraded(f func(User, chan []Task, chan errors.Error, *int)) {
	m.graded = append(m.graded, f)
}

// AddItems adds the class tasks/resources retrieval function f to m for
// platform multiplexing.
func (m *Mux) AddItems(f func(User, chan []Item, chan errors.Error, []Class, *int)) {
	m.items = append(m.items, f)
}

// AddMessages adds the unread messages retrieval function f to m for platform
// multiplexing.
func (m *Mux) AddMessages(f func(User, chan []Message, chan errors.Error, *int)) {
	m.messages = append(m.messages, f)
}

// SetLessons sets the lessons retrieval function for m as f for platform
// multiplexing.
func (m *Mux) SetLessons(f func(User, time.Time, time.Time) ([]Lesson, errors.Error)) {
	m.lessons = f
}

// SetReports sets the report card retrieval function for *m as f for platform
// multiplexing.
func (m *Mux) SetReports(f func(User) ([]Report, errors.Error)) {
	m.reports = f
}

// Auth attempts to authenticate to all platforms multiplexed by m using the
// provided *creds. Each new platform authentication token returned by each
// successful authentication attempt is added to *creds.SiteTokens
//
// An error is returned if no platform multiplexed by m can verify the
// authenticity of the provided *creds.
func (m *Mux) Auth(creds *User) errors.Error {
	ch := make(chan [2]string)
	errs := make(chan errors.Error)
	var finished int
	for _, f := range m.auth {
		finished--
		go f(*creds, ch, errs, &finished)
	}
	var err errors.Error
	valid := false
	for e := range errs {
		errors.Join(err, e)
		if err == nil {
			valid = true
		}
	}
	if !valid {
		return err
	}
	for token := range ch {
		(*creds).SiteTokens[token[0]] = token[1]
	}
	return nil
}

// Classes returns a list of classes from all platforms multiplexed by m.
func (m *Mux) Classes(creds User) ([]Class, errors.Error) {
	var classes []Class
	ch := make(chan []Class)
	errs := make(chan errors.Error)
	var finished int
	for _, f := range m.classes {
		finished--
		go f(creds, ch, errs, &finished)
	}
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}
	for list := range ch {
		classes = append(classes, list...)
	}
	sort.SliceStable(classes, func(i, j int) bool {
		return classes[i].Name < classes[j].Name
	})
	return classes, nil
}

// DueTasks returns a list of active tasks from all platforms multiplexed by m.
func (m *Mux) DueTasks(creds User) ([]Task, errors.Error) {
	var active []Task
	ch := make(chan []Task)
	errs := make(chan errors.Error)
	var finished int
	for _, f := range m.duetasks {
		finished--
		go f(creds, ch, errs, &finished)
	}
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}
	for list := range ch {
		active = append(active, list...)
	}
	sort.SliceStable(active, func(i, j int) bool {
		return active[i].Due.Before(active[j].Posted)
	})
	return active, nil
}

// Events returns a list of calendar events from all platforms multiplexed by m.
func (m *Mux) Events(creds User) ([]Event, errors.Error) {
	var events []Event
	ch := make(chan []Event)
	errs := make(chan errors.Error)
	var finished int
	for _, f := range m.events {
		finished--
		go f(creds, ch, errs, &finished)
	}
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}
	for list := range ch {
		events = append(events, list...)
	}
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Start.Before(events[j].Start)
	})
	return events, nil
}

// Graded returns a list of graded tasks from all platforms multiplexed by m.
func (m *Mux) Graded(creds User) ([]Task, errors.Error) {
	var graded []Task
	ch := make(chan []Task)
	errs := make(chan errors.Error)
	var finished int
	for _, f := range m.graded {
		finished--
		go f(creds, ch, errs, &finished)
	}
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}
	for list := range ch {
		graded = append(graded, list...)
	}
	sort.SliceStable(graded, func(i, j int) bool {
		return graded[i].Posted.After(graded[j].Posted)
	})
	return graded, nil
}

// Items returns a list of tasks and resources for all specified classes.
func (m *Mux) Items(creds User, classes ...Class) ([]Item, errors.Error) {
	var items []Item
	ch := make(chan []Item)
	errs := make(chan errors.Error)
	var finished int
	for _, f := range m.items {
		finished--
		go f(creds, ch, errs, classes, &finished)
	}
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}
	for list := range ch {
		items = append(items, list...)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Posted.After(items[j].Posted)
	})
	return items, nil
}

// Lessons returns a list of lessons occuring from start to end.
func (m *Mux) Lessons(creds User, start, end time.Time) ([]Lesson, errors.Error) {
	lessons, err := m.lessons(creds, start, end)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(lessons, func(i, j int) bool {
		return lessons[i].Start.Before(lessons[j].Start)
	})
	return lessons, nil
}

// Messages returns all unread messages from all platforms multiplexed by m.
func (m *Mux) Messages(creds User) ([]Message, errors.Error) {
	var messages []Message
	ch := make(chan []Message)
	errs := make(chan errors.Error)
	var finished int
	for _, f := range m.messages {
		finished--
		go f(creds, ch, errs, &finished)
	}
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}
	for list := range ch {
		messages = append(messages, list...)
	}
	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].Sent.After(messages[j].Sent)
	})
	return messages, nil
}

// Reports returns a series of report cards.
func (m *Mux) Reports(creds User) ([]Report, errors.Error) {
	reports, err := m.reports(creds)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(reports, func(i, j int) bool {
		return reports[i].Released.After(reports[j].Released)
	})
	return reports, nil
}

// Deliver sends *t to channel c, and closes c if the calling goroutine is the
// last goroutine to use the channel (i.e. when *done == 0).
func Deliver[T any](c chan T, t *T, done *int) {
	if *done == 0 {
		defer close(c)
	}
	c <- *t
}

// Done specifies that the calling goroutine is finished using a channel shared
// by multiple goroutines.
func Done(done *int) {
	*done++
}
