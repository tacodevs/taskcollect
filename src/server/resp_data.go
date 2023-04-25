package server

import (
	"html/template"
)

// Primary (page, head, body)

type pageData struct {
	PageType string
	Head     headData
	Body     bodyData
	User     userData
}

type headData struct {
	Title string
	//CssFiles []string
}

type bodyData struct {
	ErrorData     errData
	LoginData     loginData
	TimetableData timetableData
	GradesData    gradesData
	ResourceData  resourceData
	ResData       resData
	TasksData     tasksData
	TaskData      taskData
}

type userData struct {
	Name string
}

// Error Page

type errData struct {
	Heading  string
	Message  string
	InfoLink string
}

// Login page

type loginData struct {
	Failed bool
}

// Timetable

type timetableData struct {
	CurrentDay int
	Days       []ttDay
}

type ttDay struct {
	Day     string
	Lessons []ttLesson
}

type ttLesson struct {
	Class         string
	FormattedTime string
	Height        float64
	TopOffset     float64
	Room          string
	Teacher       string
	Notice        string
	Color         string
	BGColor       string
}

// Resources (/res page)

type resData struct {
	Heading string
	Classes []resClass
}

type resClass struct {
	Name     string
	ResItems []resItem
}

type resItem struct {
	Id       string
	Name     string
	Platform string //e.g. daymap, gclass
	Posted   string
	URL      string
}

// Resource (single resource)

type resourceData struct {
	Name        string
	Class       string
	URL         string
	Desc        template.HTML
	Posted      string
	HasResLinks bool
	ResLinks    map[string]string
	Platform    string
	Id          string
}

// Tasks

type taskItem struct {
	Id       string
	Name     string
	Platform string
	Class    string
	DueDate  string
	Posted   string
	Grade    string
	URL      string
}

type taskType struct {
	Name     string
	NoteType string
	Tasks    []taskItem
}

type tasksData struct {
	Heading   string
	TaskTypes []taskType
}

// Task (single task)

type taskData struct {
	Id           string
	Name         string
	Platform     string
	Class        string
	URL          string
	IsDue        bool
	DueDate      string
	Submitted    bool
	Desc         template.HTML
	HasResLinks  bool
	ResLinks     map[string]string
	HasWorkLinks bool
	WorkLinks    map[string]string
	HasUpload    bool
	Comment      template.HTML
	TaskGrade    taskGrade
}

type taskGrade struct {
	Grade   string
	Mark    string
	Color   string
	BGColor string
}

// Grades

type gradesData struct {
	Heading string
	Tasks   []taskItem
}

var loginPageData = pageData{
	PageType: "login",
	Head: headData{
		Title: "Login",
	},
	Body: bodyData{
		LoginData: loginData{
			Failed: false,
		},
	},
}

// TODO: Create a function for fetching these status codes then constructing the pageData

var statusNotFoundData = pageData{
	PageType: "error",
	Head: headData{
		Title: "404 Not Found",
	},
	Body: bodyData{
		ErrorData: errData{
			Heading: "404 Not Found",
			Message: "The requested resource was not found on the server.",
		},
	},
}

var statusServerErrorData = pageData{
	PageType: "error",
	Head: headData{
		Title: "500 Internal Server Error",
	},
	Body: bodyData{
		ErrorData: errData{
			Heading: "500 Internal Server Error",
			Message: "The server encountered an unexpected error and cannot continue.",
		},
	},
}

var daymapErrMsg = `We're trying to figure out how Daymap uploads work, but in ` +
`the meantime, please go back and use "open task in source platform" to upload work.`

var statusDaymapErrorData = pageData{
	PageType: "error",
	Head: headData{
		Title: "500 Internal Server Error",
	},
	Body: bodyData{
		ErrorData: errData{
			Heading: "Daymap uploads don't currently work",
			Message: daymapErrMsg,
			InfoLink: "https://codeberg.org/kvo/taskcollect/issues/68",
		},
	},
}

var statusGclassErrorData = pageData{
	PageType: "error",
	Head: headData{
		Title: "500 Internal Server Error",
	},
	Body: bodyData{
		ErrorData: errData{
			Heading: "Google does not allow you to submit tasks, nor upload/remove work.",
			Message: "Why, you may ask? I don't know! You should ask Google, not me.",
            InfoLink: "https://codeberg.org/kvo/taskcollect/issues/3",
		},
	},
}

var statusForbiddenData = pageData{
	PageType: "error",
	Head: headData{
		Title: "403 Forbidden",
	},
	Body: bodyData{
		ErrorData: errData{
			Heading: "403 Forbidden",
			Message: "You do not have permission to access this resource.",
		},
	},
}
