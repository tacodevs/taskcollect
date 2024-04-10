package server

import (
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	fp "path/filepath"
	"strings"

	"git.sr.ht/~kvo/libgo/errors"

	"main/logger"
	"main/plat"
)

type handler struct {
	templates *template.Template
	database  *authDB
}

// Handle things like submission and file uploads/removals.
func (h *handler) handleTask(r *http.Request, creds plat.User, platform, id, cmd string) (int, pageData, [][2]string) {
	data := pageData{}

	res := r.URL.EscapedPath()
	statusCode := 200
	var headers [][2]string

	if cmd == "submit" {
		err := submitTask(creds, platform, id)
		if err != nil {
			logger.Debug(errors.New("failed to submit task", err))
			data = statusServerErrorData
			statusCode = 500
		} else {
			index := strings.Index(res, "/submit")
			headers = [][2]string{{"Location", res[:index]}}
			statusCode = 302
		}
	} else if cmd == "upload" {
		err := uploadWork(creds, platform, id, r)
		if err != nil {
			logger.Debug(errors.New("failed to upload work", err))
			data = statusServerErrorData
			statusCode = 500
		} else {
			index := strings.Index(res, "/upload")
			headers = [][2]string{{"Location", res[:index]}}
			statusCode = 302
		}
	} else if cmd == "remove" {
		filenames := []string{}

		for name := range r.URL.Query() {
			filenames = append(filenames, name)
		}

		err := removeWork(creds, platform, id, filenames)
		if errors.Is(err, plat.ErrNoPlatform) {
			data = statusNotFoundData
			statusCode = 404
		} else if err != nil {
			logger.Debug(errors.New("failed to remove work", err))
			data = statusServerErrorData
			statusCode = 500
		} else {
			index := strings.Index(res, "/remove")
			headers = [][2]string{{"Location", res[:index]}}
			statusCode = 302
		}
	} else {
		data = statusNotFoundData
		statusCode = 404
	}

	return statusCode, data, headers
}

func (h *handler) handleTaskReq(r *http.Request, creds plat.User) (int, pageData, [][2]string) {
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
		assignment, err := getTask(platform, taskId, creds)
		if err != nil {
			logger.Debug(errors.New("failed to get task", err))
			data = statusServerErrorData
			statusCode = 500
			return statusCode, data, headers
		}

		data = genTaskPage(assignment, creds)
	} else {
		taskCmd := taskId[index+1:]
		taskId = taskId[:index]

		statusCode, data, headers = h.handleTask(
			r,
			creds,
			platform,
			taskId,
			taskCmd,
		)
	}

	data.User = userData{Name: creds.DispName}
	return statusCode, data, headers
}

// Generate the HTML page (and write that data to http.ResponseWriter).
func (h *handler) genPage(w http.ResponseWriter, data pageData) {
	err := h.templates.ExecuteTemplate(w, "page", data)
	if err != nil {
		logger.Debug(errors.New("template execution failed", err))
	}
}

// Responds to the client with the requested resources.
func dispatchAsset(w http.ResponseWriter, fullPath string, mimeType string) {
	w.Header().Set("Content-Type", mimeType+`, charset="utf-8"`)

	file, err := os.Open(fullPath)
	if err != nil {
		logger.Error(errors.New("could not open "+fullPath, err))
		w.WriteHeader(500)
	}
	defer file.Close()

	_, err = io.Copy(w, file)
	if err != nil {
		logger.Debug(errors.New("could not copy contents of "+fullPath, err))
		w.WriteHeader(500)
	}
}

// Handle assets - CSS, JS, fonts, etc.
func (h *handler) assetHandler(w http.ResponseWriter, r *http.Request) {
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
				h.genPage(w, data)
			} else {
				w.WriteHeader(404)
				data := statusNotFoundData
				data.User = userData{Name: "none"}
				h.genPage(w, data)
			}
			return
		}

		fullPath := fp.Join(h.database.path, "icons", fileStr)
		dispatchAsset(w, fullPath, mimeType)

	} else if res == "/taskcollect-wordmark.svg" {
		fullPath := fp.Join(h.database.path, "taskcollect-wordmark.svg")
		dispatchAsset(w, fullPath, "image/svg+xml")

	} else if res == "/manifest.webmanifest" {
		fullPath := fp.Join(h.database.path, "manifest.webmanifest")
		dispatchAsset(w, fullPath, "application/json")

	} else if res == "/styles.css" {
		//w.Header().Set("Cache-Control", "max-age=3600")
		fullPath := fp.Join(h.database.path, "styles.css")
		dispatchAsset(w, fullPath, "text/css")

	} else if res == "/script.js" {
		w.Header().Set("Cache-Control", "max-age=3600")
		fullPath := fp.Join(h.database.path, "script.js")
		dispatchAsset(w, fullPath, "text/javascript")

	} else if res == "/mainfont.woff2" {
		w.Header().Set("Cache-Control", "max-age=259200")
		fullPath := fp.Join(h.database.path, "mainfont.woff2")
		dispatchAsset(w, fullPath, "font/woff2")

	} else if res == "/navfont.woff2" {
		w.Header().Set("Cache-Control", "max-age=259200")
		fullPath := fp.Join(h.database.path, "navfont.woff2")
		dispatchAsset(w, fullPath, "font/woff2")

	} else {
		if res == "/" {
			w.WriteHeader(403)
			data := statusForbiddenData
			data.User = userData{Name: "none"}
			h.genPage(w, data)
		} else {
			w.WriteHeader(404)
			data := statusNotFoundData
			data.User = userData{Name: "none"}
			h.genPage(w, data)
		}
	}
}

// Handle login requests. If the user is already logged in, redirect to the timetable view.
func (h *handler) loginHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true
	redirect := r.URL.Query().Get("redirect")
	_, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, plat.ErrInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Debug(errors.New("failed to get creds", err))
		data := statusServerErrorData
		data.User = userData{Name: "none"}
		h.genPage(w, data)
		return
	}

	data := loginPageData
	if strings.HasPrefix(redirect, "/") {
		data.Body.LoginData.Redirect = "?" + r.URL.RawQuery
	}

	if !validAuth {
		if r.URL.Query().Get("auth") == "failed" {
			w.WriteHeader(401)
			data.Body.LoginData.Failed = true
			h.genPage(w, data)
		} else {
			h.genPage(w, data)
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
func (h *handler) authHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true
	_, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, plat.ErrInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Debug(errors.New("failed to get creds", err))
		data := statusServerErrorData
		data.User = userData{Name: "none"}
		h.genPage(w, data)
		return
	}

	if !validAuth {
		var cookie string
		err := r.ParseForm()

		// If err != nil, the "else" section of the next if/else block will
		// execute, which returns the "could not authenticate user" error.
		if err == nil {
			cookie, err = h.database.auth(r.PostForm)
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
			logger.Debug(errors.New("auth failed", err))
			w.Header().Set("Location", "/login?auth=failed")
			w.WriteHeader(302)
		}
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

func (h *handler) logoutHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true
	creds, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, plat.ErrInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Debug(errors.New("failed to get creds", err))
		data := statusServerErrorData
		data.User = userData{Name: "none"}
		h.genPage(w, data)
		return
	}

	if validAuth {
		err = h.database.logout(creds)
		if err == nil {
			w.Header().Set("Location", "/login")
			w.WriteHeader(302)
		} else {
			logger.Error(errors.New("failed to log out user", err))
			w.WriteHeader(500)
			data := statusServerErrorData
			data.User = userData{Name: creds.DispName}
			h.genPage(w, data)
		}
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

// Handle individual resource pages (located under "/res/").
func (h *handler) resourceHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true
	creds, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, plat.ErrInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Debug(errors.New("failed to get creds", err))
		data := statusServerErrorData
		data.User = userData{Name: "none"}
		h.genPage(w, data)
		return
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
			data.User = userData{Name: creds.DispName}
			h.genPage(w, data)
			return
		}

		resId := platform[index+1:]
		platform = platform[:index]

		res, err := getResource(platform, resId, creds)
		if err != nil {
			logger.Debug(errors.New("failed to get resource", err))
			w.WriteHeader(500)
			data := statusServerErrorData
			data.User = userData{Name: creds.DispName}
			h.genPage(w, data)
			return
		}

		respBody = genResPage(res, creds)
		w.WriteHeader(statusCode)
		h.genPage(w, respBody)
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

// Handle individual task pages (located under "/tasks/").
func (h *handler) taskHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true
	creds, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, plat.ErrInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Debug(errors.New("failed to get creds", err))
		data := statusServerErrorData
		data.User = userData{Name: "none"}
		h.genPage(w, data)
		return
	}

	if validAuth {
		statusCode, respBody, respHeaders := h.handleTaskReq(r, creds)

		for _, respHeader := range respHeaders {
			w.Header().Set(respHeader[0], respHeader[1])
		}

		w.WriteHeader(statusCode)
		h.genPage(w, respBody)
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

// Handle the "/tasks" page
func (h *handler) tasksHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true
	creds, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, plat.ErrInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Debug(errors.New("failed to get creds", err))
		data := statusServerErrorData
		data.User = userData{Name: "none"}
		h.genPage(w, data)
		return
	}

	if validAuth {
		webpageData, err := genRes(h.database.path, "/tasks", creds)
		if errors.Is(err, plat.ErrNotFound) {
			w.WriteHeader(404)
			data := statusNotFoundData
			data.User = userData{Name: creds.DispName}
			h.genPage(w, data)
		} else if err != nil {
			logger.Debug(errors.New("failed to generate resources", err))
			w.WriteHeader(500)
			data := statusServerErrorData
			data.User = userData{Name: creds.DispName}
			h.genPage(w, data)
		} else {
			w.Header().Set("Cache-Control", "max-age=2400")
			h.genPage(w, webpageData)
		}
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

// Handle the "/timetable" page
func (h *handler) timetableHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true
	creds, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, plat.ErrInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Debug(errors.New("failed to get creds", err))
		data := statusServerErrorData
		data.User = userData{Name: "none"}
		h.genPage(w, data)
		return
	}

	if validAuth {
		webpageData, err := genRes(h.database.path, "/timetable", creds)
		if errors.Is(err, plat.ErrNotFound) {
			w.WriteHeader(404)
			data := statusNotFoundData
			data.User = userData{Name: creds.DispName}
			h.genPage(w, data)
		} else if err != nil {
			logger.Debug(errors.New("failed to generate resources", err))
			w.WriteHeader(500)
			data := statusServerErrorData
			data.User = userData{Name: creds.DispName}
			h.genPage(w, data)
		} else {
			w.Header().Set("Cache-Control", "max-age=2400")
			h.genPage(w, webpageData)
		}
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

// Handle the "/grades" page
func (h *handler) gradesHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true
	creds, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, plat.ErrInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Debug(errors.New("failed to get creds", err))
		data := statusServerErrorData
		data.User = userData{Name: "none"}
		h.genPage(w, data)
		return
	}

	if validAuth {
		webpageData, err := genRes(h.database.path, "/grades", creds)
		if errors.Is(err, plat.ErrNotFound) {
			w.WriteHeader(404)
			data := statusNotFoundData
			data.User = userData{Name: creds.DispName}
			h.genPage(w, data)
		} else if err != nil {
			logger.Debug(errors.New("failed to generate resources", err))
			w.WriteHeader(500)
			data := statusServerErrorData
			data.User = userData{Name: creds.DispName}
			h.genPage(w, data)
		} else {
			w.Header().Set("Cache-Control", "max-age=2400")
			h.genPage(w, webpageData)
		}
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

// Handle the "/images"

// Handle the "/res" page
func (h *handler) resHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true
	creds, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, plat.ErrInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Debug(errors.New("failed to get creds", err))
		data := statusServerErrorData
		data.User = userData{Name: "none"}
		h.genPage(w, data)
		return
	}

	if validAuth {
		webpageData, err := genRes(h.database.path, "/res", creds)
		if errors.Is(err, plat.ErrNotFound) {
			w.WriteHeader(404)
			data := statusNotFoundData
			data.User = userData{Name: creds.DispName}
			h.genPage(w, data)
		} else if err != nil {
			logger.Debug(errors.New("failed to generate resources", err))
			w.WriteHeader(500)
			data := statusServerErrorData
			data.User = userData{Name: creds.DispName}
			h.genPage(w, data)
		} else {
			w.Header().Set("Cache-Control", "max-age=2400")
			h.genPage(w, webpageData)
		}
	} else {
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	}
}

func (h *handler) rootHandler(w http.ResponseWriter, r *http.Request) {
	res := r.URL.EscapedPath()
	validAuth := true
	creds, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, plat.ErrInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Debug(errors.New("failed to get creds", err))
		data := statusServerErrorData
		data.User = userData{Name: "none"}
		h.genPage(w, data)
		return
	}

	if res == "/favicon.ico" {
		fullPath := fp.Join(h.database.path, "/icons/favicon.ico")
		dispatchAsset(w, fullPath, "text/plain")
	} else if !validAuth {
		// User is not logged in (and is not on login page)
		redirect := "/login?redirect=" + url.QueryEscape(r.URL.String())
		w.Header().Set("Location", redirect)
		w.WriteHeader(302)
	} else if validAuth && res == "/timetable.png" {
		genTimetableImg(creds, w)
	} else if validAuth && res == "/" {
		w.Header().Set("Location", "/timetable")
		w.WriteHeader(302)
	} else if validAuth && res != "/" {
		// Logged in, and the requested URL is not handled by anything else (it's a 404)
		w.WriteHeader(404)
		h.genPage(w, statusNotFoundData)
	}
}
