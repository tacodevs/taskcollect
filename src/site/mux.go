package site

import (
	"mime/multipart"
	"net/http"
	"sort"
	"time"

	"git.sr.ht/~kvo/go-std/errors"

	"main/logger"
)

// Mux is a platform multiplexer. Methods can be invoked on it to select the
// platform functions to multiplex, and alternatively to create a multi-platform
// function call.
type Mux struct {
	auth      []func(User, chan Pair[[2]string, error])
	classes   []func(User, chan Pair[[]Class, error])
	duetasks  []func(User, chan Pair[[]Task, error])
	events    []func(User, chan Pair[[]Event, error])
	graded    []func(User, chan Pair[[]Task, error])
	lessons   func(User, time.Time, time.Time) ([]Lesson, error)
	messages  []func(User, chan Pair[[]Message, error])
	remove    map[string]func(User, string, []string) error
	reports   func(User) ([]Report, error)
	resource  map[string]func(User, string) (Resource, error)
	resources map[string]func(User, chan Pair[[]Resource, error], []Class)
	submit    map[string]func(User, string) error
	task      map[string]func(User, string) (Task, error)
	tasks     map[string]func(User, chan Pair[[]Task, error], []Class)
	upload    map[string]func(User, string, *multipart.Reader) error
}

// Return a new instance of Mux.
func NewMux() *Mux {
	m := new(Mux)
	m.remove = make(map[string]func(User, string, []string) error)
	m.resource = make(map[string]func(User, string) (Resource, error))
	m.resources = make(map[string]func(User, chan Pair[[]Resource, error], []Class))
	m.submit = make(map[string]func(User, string) error)
	m.task = make(map[string]func(User, string) (Task, error))
	m.tasks = make(map[string]func(User, chan Pair[[]Task, error], []Class))
	m.upload = make(map[string]func(User, string, *multipart.Reader) error)
	return m
}

// AddAuth adds the authentication function f to m for platform authentication
// multiplexing.
func (m *Mux) AddAuth(f func(User, chan Pair[[2]string, error])) {
	m.auth = append(m.auth, f)
}

// AddClasses adds the class list retrieval function f to m for platform
// multiplexing.
func (m *Mux) AddClasses(f func(User, chan Pair[[]Class, error])) {
	m.classes = append(m.classes, f)
}

// AddDueTasks adds the active tasks retrieval function f to m for platform
// multiplexing.
func (m *Mux) AddDueTasks(f func(User, chan Pair[[]Task, error])) {
	m.duetasks = append(m.duetasks, f)
}

// AddEvents adds the calendar events retrieval function f to m for platform
// multiplexing.
func (m *Mux) AddEvents(f func(User, chan Pair[[]Event, error])) {
	m.events = append(m.events, f)
}

// AddGraded adds the graded tasks retrieval function f to m for platform
// mulitplexing.
func (m *Mux) AddGraded(f func(User, chan Pair[[]Task, error])) {
	m.graded = append(m.graded, f)
}

// AddMessages adds the unread messages retrieval function f to m for platform
// multiplexing.
func (m *Mux) AddMessages(f func(User, chan Pair[[]Message, error])) {
	m.messages = append(m.messages, f)
}

// AddRemoveWork adds the work submission removal function f to m for platform
// multiplexing.
func (m *Mux) AddRemoveWork(platform string, f func(User, string, []string) error) {
	m.remove[platform] = f
}

// AddResource adds the resource information retrieval function f to m for
// platform multiplexing.
func (m *Mux) AddResource(platform string, f func(User, string) (Resource, error)) {
	m.resource[platform] = f
}

// AddResources adds the class resources retrieval function f to m for platform
// multiplexing.
func (m *Mux) AddResources(platform string, f func(User, chan Pair[[]Resource, error], []Class)) {
	m.resources[platform] = f
}

// AddSubmit adds the task submission function f to m for platform multiplexing.
func (m *Mux) AddSubmit(platform string, f func(User, string) error) {
	m.submit[platform] = f
}

// AddTask adds the task information retrieval function f to m for platform
// multiplexing.
func (m *Mux) AddTask(platform string, f func(User, string) (Task, error)) {
	m.task[platform] = f
}

// AddTasks adds the class tasks retrieval function f to m for platform
// multiplexing.
func (m *Mux) AddTasks(platform string, f func(User, chan Pair[[]Task, error], []Class)) {
	m.tasks[platform] = f
}

// AddUploadWork adds the work submission upload function f to m for platform
// multiplexing.
func (m *Mux) AddUploadWork(platform string, f func(User, string, *multipart.Reader) error) {
	m.upload[platform] = f
}

// SetLessons sets the lessons retrieval function for m as f for platform
// multiplexing.
func (m *Mux) SetLessons(f func(User, time.Time, time.Time) ([]Lesson, error)) {
	m.lessons = f
}

// SetReports sets the report card retrieval function for *m as f for platform
// multiplexing.
func (m *Mux) SetReports(f func(User) ([]Report, error)) {
	m.reports = f
}

// Auth attempts to authenticate to all platforms multiplexed by m using the
// provided *user. Each new platform authentication token returned by each
// successful authentication attempt is added to *user.SiteTokens
//
// An error is returned if no platform multiplexed by m can verify the
// authenticity of the provided *user. Each platform authentication attempt that
// fails is logged at debug level.
func (m *Mux) Auth(user *User) error {
	ch := make(chan Pair[[2]string, error])
	if user == nil {
		return errors.New(nil, "user is nil")
	}
	err := LoadConfig(user)
	if err != nil {
		return errors.Wrap(err)
	}
	for _, f := range m.auth {
		go f(*user, ch)
	}
	var errs error
	valid := false
	for range m.auth {
		result := <-ch
		token, err := result.First, result.Second
		if err != nil {
			logger.Debug(err)
			errors.Join(errs, err)
		} else if !valid {
			valid = true
		}
		if err == nil {
			user.SiteTokens[token[0]] = token[1]
		}
	}
	return nil
}

// Classes returns a list of classes from all platforms multiplexed by m.
func (m *Mux) Classes(user User) ([]Class, error) {
	var classes []Class
	ch := make(chan Pair[[]Class, error])
	for _, f := range m.classes {
		go f(user, ch)
	}
	for range m.classes {
		result := <-ch
		list, err := result.First, result.Second
		if err != nil {
			return nil, errors.New(err, "cannot get class list")
		}
		classes = append(classes, list...)
	}
	sort.SliceStable(classes, func(i, j int) bool {
		return classes[i].Name < classes[j].Name
	})
	return classes, nil
}

// DueTasks returns a list of active tasks from all platforms multiplexed by m.
func (m *Mux) DueTasks(user User) ([]Task, error) {
	var active []Task
	ch := make(chan Pair[[]Task, error])
	for _, f := range m.duetasks {
		go f(user, ch)
	}
	for range m.duetasks {
		result := <-ch
		list, err := result.First, result.Second
		if err != nil {
			return nil, errors.New(err, "cannot get active task list")
		}
		active = append(active, list...)
	}
	sort.SliceStable(active, func(i, j int) bool {
		return active[i].Due.Before(active[j].Posted)
	})
	return active, nil
}

// Events returns a list of calendar events from all platforms multiplexed by m.
func (m *Mux) Events(user User) ([]Event, error) {
	var events []Event
	ch := make(chan Pair[[]Event, error])
	for _, f := range m.events {
		go f(user, ch)
	}
	for range m.events {
		result := <-ch
		list, err := result.First, result.Second
		if err != nil {
			return nil, errors.New(err, "cannot get event list")
		}
		events = append(events, list...)
	}
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Start.Before(events[j].Start)
	})
	return events, nil
}

// Graded returns a list of graded tasks from all platforms multiplexed by m.
func (m *Mux) Graded(user User) ([]Task, error) {
	var graded []Task
	ch := make(chan Pair[[]Task, error])
	for _, f := range m.graded {
		go f(user, ch)
	}
	for range m.graded {
		result := <-ch
		list, err := result.First, result.Second
		if err != nil {
			return nil, errors.New(err, "cannot get graded task list")
		}
		graded = append(graded, list...)
	}
	sort.SliceStable(graded, func(i, j int) bool {
		return graded[i].Posted.After(graded[j].Posted)
	})
	return graded, nil
}

// Lessons returns a list of lessons occuring from start to end.
func (m *Mux) Lessons(user User, start, end time.Time) ([]Lesson, error) {
	if m.lessons == nil {
		return nil, errors.New(nil, "lessons function not set")
	}
	lessons, err := m.lessons(user, start, end)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(lessons, func(i, j int) bool {
		return lessons[i].Start.Before(lessons[j].Start)
	})
	return lessons, nil
}

// Messages returns all unread messages from all platforms multiplexed by m.
func (m *Mux) Messages(user User) ([]Message, error) {
	var messages []Message
	ch := make(chan Pair[[]Message, error])
	for _, f := range m.messages {
		go f(user, ch)
	}
	for range m.messages {
		result := <-ch
		list, err := result.First, result.Second
		if err != nil {
			return nil, errors.New(err, "cannot get unread message list")
		}
		messages = append(messages, list...)
	}
	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].Sent.After(messages[j].Sent)
	})
	return messages, nil
}

// RemoveWork removes the work submissions specified by filenames from the task
// with given id from the specified platform. An error is returned if either the
// removal process fails or the platform is not supported by the platform
// multiplexer m.
func (m *Mux) RemoveWork(user User, platform, id string, filenames []string) error {
	f, ok := m.remove[platform]
	if !ok {
		return errors.New(nil, "unsupported platform")
	}
	return f(user, id, filenames)
}

// Reports returns a series of report cards.
func (m *Mux) Reports(user User) ([]Report, error) {
	if m.reports == nil {
		return nil, errors.New(nil, "reports function not set")
	}
	reports, err := m.reports(user)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(reports, func(i, j int) bool {
		return reports[i].Released.After(reports[j].Released)
	})
	return reports, nil
}

// Resource returns the resource specified by id from the given platform. An
// error is returned if either the task information could not be retrieved or
// the platform is not supported by the platform multiplexer m.
func (m *Mux) Resource(user User, platform, id string) (Resource, error) {
	f, ok := m.resource[platform]
	if !ok {
		return Resource{}, errors.New(nil, "unsupported platform")
	}
	return f(user, id)
}

// Resources returns a list of resources for all specified classes.
func (m *Mux) Resources(user User, classes ...Class) ([]Resource, error) {
	var resources []Resource
	ch := make(chan Pair[[]Resource, error])
	classMap := make(map[string][]Class)
	for _, class := range classes {
		classMap[class.Platform] = append(classMap[class.Platform], class)
	}
	for platform, courses := range classMap {
		go m.resources[platform](user, ch, courses)
	}
	for range m.tasks {
		result := <-ch
		list, err := result.First, result.Second
		if err != nil {
			return nil, errors.New(err, "cannot get resources list")
		}
		resources = append(resources, list...)
	}
	sort.SliceStable(resources, func(i, j int) bool {
		return resources[i].Posted.After(resources[j].Posted)
	})
	return resources, nil
}

// Submit submits the task with given id from the specified platform. An error
// is returned if either the submission process fails or the platform is not
// supported by the platform multiplexer m.
func (m *Mux) Submit(user User, platform, id string) error {
	f, ok := m.submit[platform]
	if !ok {
		return errors.New(nil, "unsupported platform")
	}
	return f(user, id)
}

// Task returns the task specified by id from the given platform. An error is
// returned if either the task could not be retrieved or the platform is not
// supported by the platform multiplexer m.
func (m *Mux) Task(user User, platform, id string) (Task, error) {
	f, ok := m.task[platform]
	if !ok {
		return Task{}, errors.New(nil, "unsupported platform")
	}
	return f(user, id)
}

// Tasks returns a list of tasks for all specified classes.
func (m *Mux) Tasks(user User, classes ...Class) ([]Task, error) {
	var tasks []Task
	ch := make(chan Pair[[]Task, error])
	classMap := make(map[string][]Class)
	for _, class := range classes {
		classMap[class.Platform] = append(classMap[class.Platform], class)
	}
	for platform, courses := range classMap {
		go m.tasks[platform](user, ch, courses)
	}
	for range m.tasks {
		result := <-ch
		list, err := result.First, result.Second
		if err != nil {
			return nil, errors.New(err, "cannot get tasks list")
		}
		tasks = append(tasks, list...)
	}
	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].Posted.After(tasks[j].Posted)
	})
	return tasks, nil
}

// UploadWork uploads all files in request r as work submissions for the task
// with given id for the specified platform. An error is returned if either the
// upload process fails or the platform is not supported by the platform
// multiplexer m.
func (m *Mux) UploadWork(user User, platform, id string, r *http.Request) error {
	f, ok := m.upload[platform]
	if !ok {
		return errors.New(nil, "unsupported platform")
	}
	files, err := r.MultipartReader()
	if err != nil {
		return errors.New(err, "cannot parse multipart MIME")
	}
	return f(user, id, files)
}
