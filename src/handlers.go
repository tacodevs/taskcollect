package main

import (
	"html/template"
	"io"
	"net/http"
	"os"
	fp "path/filepath"
	"strings"

	"main/errors"
	"main/logger"
)

type handler struct {
	templates *template.Template
	database  *authDB
}

// Generate the HTML page (and write that data to http.ResponseWriter).
func (h *handler) genPage(w http.ResponseWriter, data pageData) {
	err := h.templates.ExecuteTemplate(w, "page", data)
	if err != nil {
		logger.Error(err)
	}

	// TESTING CODE:
	// NOTE: It seems that when fetching data (res or tasks) it fetches the data and writes to
	// the file but that gets overridden by a 404 page instead.

	/*
		var processed bytes.Buffer
		err := templates.ExecuteTemplate(&processed, "page", data)
		outputPath := "./result.txt"
		f, _ := os.Create(outputPath)
		a := bufio.NewWriter(f)
		a.WriteString(processed.String())
		a.Flush()
		if err != nil {
			fmt.Println("Errors:")
			logger.Error(err)
		}
	*/
}

// Handle assets - CSS, JS, fonts, etc.
func (h *handler) assetHandler(w http.ResponseWriter, r *http.Request) {
	res := strings.Replace(r.URL.EscapedPath(), "/assets", "", 1)

	if res == "/styles.css" {
		w.Header().Set("Content-Type", `text/css, charset="utf-8"`)

		cssFile, err := os.Open(fp.Join(h.database.path, "styles.css"))
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
		}
		defer cssFile.Close()

		_, err = io.Copy(w, cssFile)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
		}
	} else if res == "/mainfont.ttf" {
		w.Header().Set("Content-Type", `font/ttf`)

		fontFile, err := os.Open(fp.Join(h.database.path, "mainfont.ttf"))
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
		}
		defer fontFile.Close()

		_, err = io.Copy(w, fontFile)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
		}
	} else if res == "/navfont.ttf" {
		w.Header().Set("Content-Type", `font/ttf`)

		fontFile, err := os.Open(fp.Join(h.database.path, "navfont.ttf"))
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
		}
		defer fontFile.Close()

		_, err = io.Copy(w, fontFile)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
		}
	}
}

// Handle login requests. If the user is already logged in, redirect to the timetable view.
func (h *handler) loginHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true
	creds, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, errInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Error(err)
		h.genPage(w, statusServerErrorData)
		return
	}

	creds.GAuthID = h.database.gAuth

	if !validAuth {
		if r.URL.Query().Get("auth") == "failed" {
			w.WriteHeader(401)
			data := pageData{
				PageType: "login",
				Head: headData{
					Title: "Login",
				},
				Body: bodyData{
					LoginData: loginData{
						Failed: true,
					},
				},
			}
			h.genPage(w, data)
		} else {
			h.genPage(w, loginPageData)
		}
	} else {
		w.Header().Set("Location", "/timetable")
		w.WriteHeader(302)
	}
}

// Handle authentication requests. If the user is already logged in, redirect to the timetable view.
func (h *handler) authHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true
	creds, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, errInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Error(err)
		h.genPage(w, statusServerErrorData)
		return
	}

	creds.GAuthID = h.database.gAuth

	if !validAuth {
		var cookie string

		err = r.ParseForm()
		if err == nil {
			cookie, err = h.database.auth(r.PostForm)
		}
		if err == nil {
			w.Header().Set("Location", "/timetable")
			w.Header().Set("Set-Cookie", cookie)
			w.WriteHeader(302)
			return
		} else if !errors.Is(err, errNeedsGAuth) {
			logger.Error(err)
			w.Header().Set("Location", "/login?auth=failed")
			w.WriteHeader(302)
			return
		}

		gAuthLoc, err := h.database.genGAuthLoc()
		if err != nil {
			logger.Error(err)
		} else {
			w.Header().Set("Location", gAuthLoc)
			w.Header().Set("Set-Cookie", cookie)
			w.WriteHeader(302)
		}
	} else {
		w.Header().Set("Location", "/timetable")
		w.WriteHeader(302)
	}
}

func (h *handler) logoutHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true
	creds, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, errInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Error(err)
		h.genPage(w, statusServerErrorData)
		return
	}

	creds.GAuthID = h.database.gAuth

	if validAuth {
		err = h.database.logout(creds)
		if err == nil {
			w.Header().Set("Location", "/login")
			w.WriteHeader(302)
		} else {
			logger.Error(err)
			w.WriteHeader(500)
			h.genPage(w, statusServerErrorData)
		}
	} else {
		w.Header().Set("Location", "/login")
		w.WriteHeader(302)
	}
}

// Handle individual task pages (located under "/tasks/").
func (h *handler) taskHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true
	creds, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, errInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Error(err)
		h.genPage(w, statusServerErrorData)
		return
	}

	creds.GAuthID = h.database.gAuth

	if validAuth {
		statusCode, respBody, respHeaders := handleTaskReq(h.templates, r, creds)

		for _, respHeader := range respHeaders {
			w.Header().Set(respHeader[0], respHeader[1])
		}

		w.WriteHeader(statusCode)
		h.genPage(w, respBody)
	} else {
		w.Header().Set("Location", "/login")
		w.WriteHeader(302)
	}
}

// Handle the "/tasks" page
func (h *handler) tasksHandler(w http.ResponseWriter, r *http.Request) {
	validAuth := true
	creds, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, errInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Error(err)
		h.genPage(w, statusServerErrorData)
		return
	}

	creds.GAuthID = h.database.gAuth

	if validAuth {
		webpageData, err := genRes(h.database.path, "/tasks", creds)
		if errors.Is(err, errNotFound) {
			w.WriteHeader(404)
			h.genPage(w, statusNotFoundData)
		} else if err != nil {
			logger.Error(err.Error())
			w.WriteHeader(500)
			h.genPage(w, statusServerErrorData)
		} else {
			h.genPage(w, webpageData)
		}
	} else {
		w.Header().Set("Location", "/login")
		w.WriteHeader(302)
	}
}

func (h *handler) rootHandler(w http.ResponseWriter, r *http.Request) {
	res := r.URL.EscapedPath()
	validAuth := true
	creds, err := h.database.getCreds(r.Header.Get("Cookie"))

	if errors.Is(err, errInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Error(err)
		h.genPage(w, statusServerErrorData)
		return
	}

	creds.GAuthID = h.database.gAuth
	invalidRes := false

	if res == "/" {
		invalidRes = true
	}

	// User is not logged in (and is not on login page)
	if !validAuth {
		w.Header().Set("Location", "/login")
		w.WriteHeader(302)

	} else if validAuth && res == "/gauth" {
		err = h.database.runGAuth(creds, r.URL.Query())
		if err != nil {
			logger.Error(err)
		}
		w.Header().Set("Location", "/timetable")
		w.WriteHeader(302)

		// Timetable image
		// NOTE: Perhaps still keep the png generation even though the main timetable will
		// be replaced by a table, rather than image
	} else if validAuth && res == "/timetable.png" {
		genTimetable(creds, w)

		// Invalid URL while logged in redirects to /timetable
	} else if validAuth && invalidRes {
		w.Header().Set("Location", "/timetable")
		w.WriteHeader(302)

		// Logged in, and the requested URL is valid
	} else if validAuth && !invalidRes {
		webpageData, err := genRes(h.database.path, res, creds)

		if errors.Is(err, errNotFound) {
			w.WriteHeader(404)
			h.genPage(w, statusNotFoundData)
		} else if err != nil {
			logger.Error(err.Error())
			w.WriteHeader(500)
			h.genPage(w, statusServerErrorData)
		} else {
			h.genPage(w, webpageData)
		}
	}
}
