package server

import (
	"io"
	"net/http"
	"net/url"
	"os"
	path "path/filepath"
	"strings"

	"git.sr.ht/~kvo/go-std/errors"

	"main/logger"
	"main/site"
)

// Handle things like submission and file uploads/removals.
func handleTask(r *http.Request, user site.User, platform, id, cmd string) (int, pageData, [][2]string) {
	data := pageData{}

	res := r.URL.EscapedPath()
	statusCode := 200
	var headers [][2]string

	if cmd == "submit" {
		school, ok := schools[user.School]
		if !ok {
			logger.Debug(errors.New(nil, "unsupported platform"))
			data = statusServerErrorData
			statusCode = 500
			return statusCode, data, headers
		}
		err := school.Submit(user, platform, id)
		if err != nil {
			logger.Debug(errors.New(err, "cannot submit task"))
			data = statusServerErrorData
			statusCode = 500
			return statusCode, data, headers
		}
		index := strings.Index(res, "/submit")
		headers = [][2]string{{"Location", res[:index]}}
		statusCode = 302
	} else if cmd == "upload" {
		school, ok := schools[user.School]
		if !ok {
			logger.Debug(errors.New(nil, "unsupported platform"))
			data = statusServerErrorData
			statusCode = 500
			return statusCode, data, headers
		}
		err := school.UploadWork(user, platform, id, r)
		if err != nil {
			logger.Debug(errors.New(err, "cannot upload work"))
			data = statusServerErrorData
			statusCode = 500
			return statusCode, data, headers
		}
		index := strings.Index(res, "/upload")
		headers = [][2]string{{"Location", res[:index]}}
		statusCode = 302
	} else if cmd == "remove" {
		filenames := []string{}
		for name := range r.URL.Query() {
			filenames = append(filenames, name)
		}
		school, ok := schools[user.School]
		if !ok {
			logger.Debug(errors.New(nil, "unsupported platform"))
			data = statusServerErrorData
			statusCode = 500
			return statusCode, data, headers
		}
		err := school.RemoveWork(user, platform, id, filenames)
		if err != nil {
			logger.Debug(errors.New(err, "cannot remove worklink"))
			data = statusServerErrorData
			statusCode = 500
			return statusCode, data, headers
		}
		index := strings.Index(res, "/remove")
		headers = [][2]string{{"Location", res[:index]}}
		statusCode = 302
	} else {
		data = statusNotFoundData
		statusCode = 404
	}

	return statusCode, data, headers
}

func handleTaskReq(r *http.Request, user site.User) (int, pageData, [][2]string) {
	res := r.URL.EscapedPath()

	statusCode := 200
	var data pageData
	var headers [][2]string

	platform := res[7:]
	index := strings.Index(platform, "/")

	if index == -1 {
		data = statusNotFoundData
		statusCode = 404
		return statusCode, data, headers
	}

	taskId := platform[index+1:]
	platform = platform[:index]
	index = strings.Index(taskId, "/")

	if index == -1 {
		school, ok := schools[user.School]
		if !ok {
			logger.Debug(errors.New(nil, "unsupported platform"))
			data = statusServerErrorData
			statusCode = 500
			return statusCode, data, headers
		}
		assignment, err := school.Task(user, platform, taskId)
		if err != nil {
			logger.Debug(errors.New(err, "cannot fetch task"))
			data = statusServerErrorData
			statusCode = 500
			return statusCode, data, headers
		}

		data = genTaskPage(assignment, user)
	} else {
		taskCmd := taskId[index+1:]
		taskId = taskId[:index]

		statusCode, data, headers = handleTask(
			r,
			user,
			platform,
			taskId,
			taskCmd,
		)
	}

	data.User = userData{Name: user.DispName}
	return statusCode, data, headers
}

// Generate the HTML page (and write that data to http.ResponseWriter).
func genPage(w http.ResponseWriter, data pageData) {
	err := templates.ExecuteTemplate(w, "page", data)
	if err != nil {
		logger.Debug(errors.New(err, "template execution failed"))
	}
}

// Responds to the client with the requested resources.
func dispatchAsset(w http.ResponseWriter, fullPath string, mimeType string) {
	w.Header().Set("Content-Type", mimeType+`, charset="utf-8"`)

	file, err := os.Open(fullPath)
	if err != nil {
		logger.Error(errors.New(err, "could not open %s", fullPath))
		w.WriteHeader(500)
	}
	defer file.Close()

	_, err = io.Copy(w, file)
	if err != nil {
		logger.Debug(errors.New(err, "could not copy contents of %s", fullPath))
		w.WriteHeader(500)
	}
}

// Handle assets - CSS, JS, fonts, etc.
func assetHandler(w http.ResponseWriter, r *http.Request) {
	res := strings.Replace(r.URL.EscapedPath(), "/assets", "", 1)

	if strings.HasPrefix(res, "/icons") {
		fileStr := ""
		mimeType := ""

		switch name := strings.Replace(res, "/icons", "", 1); name {
		case "/apple-touch-icon.png":
			mimeType = "image/png"
			fileStr = "apple-touch-icon.png"
		case "/icon.svg":
			mimeType = "image/svg+xml"
			fileStr = "icon.svg"
		case "/icon-192.png":
			mimeType = "image/png"
			fileStr = "icon-512.png"
		case "/icon-512.png":
			mimeType = "image/png"
			fileStr = "icon-512.png"
		default:
			if name == "/" {
				w.WriteHeader(403)
				data := statusForbiddenData
				data.User = userData{Name: "none"}
				genPage(w, data)
			} else {
				w.WriteHeader(404)
				data := statusNotFoundData
				data.User = userData{Name: "none"}
				genPage(w, data)
			}
			return
		}

		fullPath := path.Join(respath, "icons", fileStr)
		dispatchAsset(w, fullPath, mimeType)

	} else if res == "/wordmark.svg" {
		w.Header().Set("Cache-Control", "max-age=3600")
		fullPath := path.Join(respath, "brand/wordmark.svg")
		dispatchAsset(w, fullPath, "image/svg+xml")

	} else if res == "/manifest.webmanifest" {
		fullPath := path.Join(respath, "manifest.webmanifest")
		dispatchAsset(w, fullPath, "application/json")

	} else if res == "/styles.css" {
		//w.Header().Set("Cache-Control", "max-age=3600")
		fullPath := path.Join(respath, "styles.css")
		dispatchAsset(w, fullPath, "text/css")

	} else if res == "/script.js" {
		w.Header().Set("Cache-Control", "max-age=3600")
		fullPath := path.Join(respath, "script.js")
		dispatchAsset(w, fullPath, "text/javascript")

	} else if res == "/mainfont.woff2" {
		w.Header().Set("Cache-Control", "max-age=259200")
		fullPath := path.Join(respath, "fonts/lato/mainfont.woff2")
		dispatchAsset(w, fullPath, "font/woff2")

	} else if res == "/navfont.woff2" {
		w.Header().Set("Cache-Control", "max-age=259200")
		fullPath := path.Join(respath, "fonts/redhat/navfont.woff2")
		dispatchAsset(w, fullPath, "font/woff2")

	} else {
		if res == "/" {
			w.WriteHeader(403)
			data := statusForbiddenData
			data.User = userData{Name: "none"}
			genPage(w, data)
		} else {
			w.WriteHeader(404)
			data := statusNotFoundData
			data.User = userData{Name: "none"}
			genPage(w, data)
		}
	}
}

// Handle login requests. If the user is already logged in, redirect to the timetable view.
func loginHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true
	redirect := r.URL.Query().Get("redirect")

	_, err := creds.LookupToken(r.Header.Get("Cookie"))
	if err != nil {
		validAuth = false
	}

	data := loginPageData
	if strings.HasPrefix(redirect, "/") {
		data.Body.LoginData.Redirect = "?" + r.URL.RawQuery
	}

	if !validAuth {
		if r.URL.Query().Get("auth") == "failed" {
			w.WriteHeader(401)
			data.Body.LoginData.Failed = true
			genPage(w, data)
		} else {
			genPage(w, data)
		}
	} else if !strings.HasPrefix(redirect, "/") {
		w.Header().Set("Location", "/timetable")
		w.WriteHeader(302)
	} else {
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

// Handle authentication requests. If the user is already logged in, redirect to the timetable view.
func authHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true

	_, err := creds.LookupToken(r.Header.Get("Cookie"))
	if err != nil {
		validAuth = false
	}

	if !validAuth {
		var cookie string
		err := r.ParseForm()

		// If err != nil, the "else" section of the next if/else block will
		// execute, which returns the "could not authenticate user" error.
		if err == nil {
			cookie, err = creds.Login(r.PostForm)
		}

		redirect := r.URL.Query().Get("redirect")
		if !strings.HasPrefix(redirect, "/") {
			redirect = "/timetable"
		}

		if err == nil {
			w.Header().Set("Location", redirect)
			w.Header().Set("Set-Cookie", cookie)
			w.WriteHeader(302)
		} else {
			logger.Debug(errors.New(err, "auth failed"))
			w.Header().Set("Location", "/login?auth=failed")
			w.WriteHeader(302)
		}
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true

	user, err := creds.LookupToken(r.Header.Get("Cookie"))
	if err != nil {
		validAuth = false
	}

	if validAuth {
		err = creds.Logout(r.Header.Get("Cookie"))
		if err == nil {
			w.Header().Set("Location", "/login")
			w.WriteHeader(302)
		} else {
			logger.Error(errors.New(err, "failed to log out user"))
			w.WriteHeader(500)
			data := statusServerErrorData
			data.User = userData{Name: user.DispName}
			genPage(w, data)
		}
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

// Handle individual resource pages (located under "/res/").
func resourceHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true

	user, err := creds.LookupToken(r.Header.Get("Cookie"))
	if err != nil {
		validAuth = false
	}

	if validAuth {
		reqRes := r.URL.EscapedPath()

		statusCode := 200
		var respBody pageData
		platform := reqRes[5:]
		index := strings.Index(platform, "/")

		if index == -1 {
			w.WriteHeader(404)
			data := statusNotFoundData
			data.User = userData{Name: user.DispName}
			genPage(w, data)
			return
		}

		resId := platform[index+1:]
		platform = platform[:index]

		school, ok := schools[user.School]
		if !ok {
			logger.Debug(errors.New(nil, "unsupported platform"))
			w.WriteHeader(500)
			data := statusServerErrorData
			data.User = userData{Name: user.DispName}
			genPage(w, data)
			return
		}
		res, err := school.Resource(user, platform, resId)
		if err != nil {
			logger.Debug(errors.New(err, "cannot fetch task"))
			w.WriteHeader(500)
			data := statusServerErrorData
			data.User = userData{Name: user.DispName}
			genPage(w, data)
			return
		}

		respBody = genResPage(res, user)
		w.WriteHeader(statusCode)
		genPage(w, respBody)
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

// Handle individual task pages (located under "/tasks/").
func taskHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true

	user, err := creds.LookupToken(r.Header.Get("Cookie"))
	if err != nil {
		validAuth = false
	}

	if validAuth {
		statusCode, respBody, respHeaders := handleTaskReq(r, user)

		for _, respHeader := range respHeaders {
			w.Header().Set(respHeader[0], respHeader[1])
		}

		w.WriteHeader(statusCode)
		genPage(w, respBody)
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

// Handle the "/tasks" page
func tasksHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true

	user, err := creds.LookupToken(r.Header.Get("Cookie"))
	if err != nil {
		validAuth = false
	}

	if validAuth {
		webpageData, err := genRes("/tasks", user)
		if errors.Is(err, errors.New(nil, "cannot find resource")) {
			w.WriteHeader(404)
			data := statusNotFoundData
			data.User = userData{Name: user.DispName}
			genPage(w, data)
		} else if err != nil {
			logger.Debug(errors.New(err, "failed to generate resources"))
			w.WriteHeader(500)
			data := statusServerErrorData
			data.User = userData{Name: user.DispName}
			genPage(w, data)
		} else {
			w.Header().Set("Cache-Control", "max-age=2400")
			genPage(w, webpageData)
		}
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

// Handle the "/timetable" page
func timetableHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true

	user, err := creds.LookupToken(r.Header.Get("Cookie"))
	if err != nil {
		validAuth = false
	}

	if validAuth {
		webpageData, err := genRes("/timetable", user)
		if errors.Is(err, errors.New(nil, "cannot find resource")) {
			w.WriteHeader(404)
			data := statusNotFoundData
			data.User = userData{Name: user.DispName}
			genPage(w, data)
		} else if err != nil {
			logger.Debug(errors.New(err, "failed to generate resources"))
			w.WriteHeader(500)
			data := statusServerErrorData
			data.User = userData{Name: user.DispName}
			genPage(w, data)
		} else {
			w.Header().Set("Cache-Control", "max-age=2400")
			genPage(w, webpageData)
		}
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

// Handle the "/grades" page
func gradesHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true

	user, err := creds.LookupToken(r.Header.Get("Cookie"))
	if err != nil {
		validAuth = false
	}

	if validAuth {
		webpageData, err := genRes("/grades", user)
		if errors.Is(err, errors.New(nil, "cannot find resource")) {
			w.WriteHeader(404)
			data := statusNotFoundData
			data.User = userData{Name: user.DispName}
			genPage(w, data)
		} else if err != nil {
			logger.Debug(errors.New(err, "failed to generate resources"))
			w.WriteHeader(500)
			data := statusServerErrorData
			data.User = userData{Name: user.DispName}
			genPage(w, data)
		} else {
			w.Header().Set("Cache-Control", "max-age=2400")
			genPage(w, webpageData)
		}
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

// Handle the "/images"

// Handle the "/res" page
func resHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true

	user, err := creds.LookupToken(r.Header.Get("Cookie"))
	if err != nil {
		validAuth = false
	}

	if validAuth {
		webpageData, err := genRes("/res", user)
		if errors.Is(err, errors.New(nil, "cannot find resource")) {
			w.WriteHeader(404)
			data := statusNotFoundData
			data.User = userData{Name: user.DispName}
			genPage(w, data)
		} else if err != nil {
			logger.Debug(errors.New(err, "failed to generate resources"))
			w.WriteHeader(500)
			data := statusServerErrorData
			data.User = userData{Name: user.DispName}
			genPage(w, data)
		} else {
			w.Header().Set("Cache-Control", "max-age=2400")
			genPage(w, webpageData)
		}
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	res := r.URL.EscapedPath()
	validAuth := true

	user, err := creds.LookupToken(r.Header.Get("Cookie"))
	if err != nil {
		validAuth = false
	}

	if res == "/favicon.ico" {
		fullPath := path.Join(respath, "/icons/favicon.ico")
		dispatchAsset(w, fullPath, "text/plain")
	} else if !validAuth {
		// User is not logged in (and is not on login page)
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	} else if validAuth && res == "/timetable.png" {
		err := TimetablePNG(user, w)
		if err != nil {
			logger.Error(err)
		}
	} else if validAuth && res == "/" {
		w.Header().Set("Location", "/timetable")
		w.WriteHeader(302)
	} else if validAuth && res != "/" {
		// Logged in, and the requested URL is not handled by anything else (it's a 404)
		w.WriteHeader(404)
		genPage(w, statusNotFoundData)
	}
}
