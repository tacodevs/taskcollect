package main

import "html/template"

// Error Page

type errData struct {
	Heading string
	Message string
}

// Login page

type loginData struct {
	Failed bool
}

// Timetable

type timetableData struct {
	Img string
}

// Resources (/res page)

type resItem struct {
	Name string
	URL  string
}

type resClass struct {
	Name     string
	ResItems []resItem
}

type resData struct {
	Heading string
	Classes []resClass
}

// Tasks

type taskItem struct {
	Id       string
	Name     string
	Platform string // e.g. DayMap, Google Classroom
	Class    string
	DueDate  string
	URL      string
}

type taskType struct {
	Name       string
	HasDueDate bool
	Tasks      []taskItem
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
	Grade        string
	Comment      template.HTML
}

// Primary (page, head, body)

type headData struct {
	Title string
	//CssFiles []string
}

type bodyData struct {
	ErrorData     errData
	LoginData     loginData
	TimetableData timetableData
	ResData       resData
	TasksData     tasksData
	TaskData      taskData
}

type pageData struct {
	PageType string
	Head     headData
	Body     bodyData
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
