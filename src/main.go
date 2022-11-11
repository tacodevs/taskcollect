package main

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"os/user"
	fp "path/filepath"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"main/errors"
	"main/logger"
)

var (
	errAuthFailed      = errors.NewError("main", nil, errors.ErrAuthFailed.Error())
	errCorruptMIME     = errors.NewError("main", nil, errors.ErrCorruptMIME.Error())
	errIncompleteCreds = errors.NewError("main", nil, errors.ErrIncompleteCreds.Error())
	errInvalidAuth     = errors.NewError("main", nil, errors.ErrInvalidAuth.Error())
	errNoPlatform      = errors.NewError("main", nil, errors.ErrNoPlatform.Error())
	errNotFound        = errors.NewError("main", nil, errors.ErrNotFound.Error())
	errNeedsGAuth      = errors.NewError("main", nil, errors.ErrNeedsGAuth.Error())
)

type authDb struct {
	lock      *sync.Mutex
	path      string
	pwd       []byte
	gAuth     []byte
	templates *template.Template
}

type postReader struct {
	div    []byte
	reader io.Reader
}

type tcUser struct {
	Timezone   *time.Location
	School     string
	Username   string
	Password   string
	Token      string
	SiteTokens map[string]string
	GAuthID    []byte
}

func (pr postReader) Read(p []byte) (int, error) {
	n := 0
	reader := bufio.NewReader(pr.reader)

	for n < len(p) {
		b, err := reader.ReadByte()
		if err != nil {
			return n, err
		}

		i := 0

		for i < len(pr.div) {
			c, err := reader.Peek(i + 1)
			if err != nil {
				return n, err
			}
			if c[i] != pr.div[i] {
				break
			}
			i++
		}

		p[n] = b
		n++

		if i == len(pr.div) {
			for x := 0; x < i; x++ {
				_, err := reader.ReadByte()
				if err != nil {
					return n, err
				}
			}
			return n, nil
		}
	}

	return n, nil
}

func fileFromReq(r *http.Request) (string, io.Reader, error) {
	reader := bufio.NewReader(r.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", nil, err
	}

	div := strings.ReplaceAll(line, "\n", "")
	div = strings.ReplaceAll(div, "\r", "")

	for line != "\n" {
		line, err = reader.ReadString('\n')
		if err != nil {
			return "", nil, err
		}

		if strings.HasPrefix(line, "Content-Disposition: form-data;") {
			break
		}
	}

	if line == "\n" {
		return "", nil, errCorruptMIME
	}

	idx := strings.Index(line, "form-data;")
	_, formVals, err := mime.ParseMediaType(line[idx:])

	if err != nil {
		return "", nil, err
	}

	for line != "\r\n" {
		line, err = reader.ReadString('\n')
		if err != nil {
			return "", nil, err
		}
	}

	filename := formVals["filename"]

	pr := postReader{
		div:    []byte("\r\n" + div + "--"),
		reader: reader,
	}

	return filename, pr, nil
}

// Handle things like submission and file uploads
func handleTask(r *http.Request, c tcUser, p, id, cmd string) (int, pageData, [][2]string) {
	data := pageData{}

	res := r.URL.EscapedPath()
	statusCode := 200
	var headers [][2]string

	if cmd == "submit" {
		err := submitTask(c, p, id)

		if err != nil {
			logger.Error(err)
			data = statusServerErrorData
			statusCode = 500
		} else {
			index := strings.Index(res, "/submit")
			headers = [][2]string{{"Location", res[:index]}}
			statusCode = 302
		}
	} else if cmd == "upload" {
		filename, reader, err := fileFromReq(r)
		if err != nil {
			logger.Error(err)
			return 500, statusServerErrorData, nil
		}

		err = uploadWork(c, p, id, filename, &reader)
		if err != nil {
			logger.Error(err)
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

		err := removeWork(c, p, id, filenames)
		if err == errNoPlatform {
			data = statusNotFoundData
			statusCode = 404
		} else if err != nil {
			logger.Error(err)
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

func handleTaskReq(tmpls *template.Template, r *http.Request, creds tcUser) (int, pageData, [][2]string) {
	res := r.URL.EscapedPath()
	//logger.Info("%v\n", res)

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
			logger.Error(err)
			data = statusServerErrorData
			statusCode = 500
			return statusCode, data, headers
		}

		data = genTaskPage(assignment, creds)
	} else {
		taskCmd := taskId[index+1:]
		taskId = taskId[:index]

		statusCode, data, headers = handleTask(
			r,
			creds,
			platform,
			taskId,
			taskCmd,
		)
	}

	return statusCode, data, headers
}

// The main handler function
func (db *authDb) handler(w http.ResponseWriter, r *http.Request) {
	res := r.URL.EscapedPath()
	validAuth := true
	creds, err := getCreds(r.Header.Get("Cookie"), db.path, db.pwd)
	if err != nil {
		logger.Error(err)
	}

	if errors.Is(err, errInvalidAuth) {
		validAuth = false
	} else if err != nil {
		logger.Error(err)
		genPage(w, db.templates, statusServerErrorData)
		return
	}

	creds.GAuthID = db.gAuth
	resIsLogin := false
	invalidRes := false

	// TODO: Have separate functions for handling validAuth and !validAuth outcomes

	if res == "/login" || res == "/auth" {
		resIsLogin = true
	}

	if resIsLogin || res == "/" {
		invalidRes = true
	}

	if res == "/css" {
		w.Header().Set("Content-Type", `text/css, charset="utf-8"`)

		cssFile, err := os.Open(fp.Join(db.path, "styles.css"))
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
		}

		_, err = io.Copy(w, cssFile)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
		}

		cssFile.Close()
	} else if res == "/mainfont.ttf" {
		w.Header().Set("Content-Type", `font/ttf`)

		fontFile, err := os.Open(fp.Join(db.path, "mainfont.ttf"))
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
		}

		_, err = io.Copy(w, fontFile)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
		}

		fontFile.Close()
	} else if res == "/navfont.ttf" {
		w.Header().Set("Content-Type", `font/ttf`)

		fontFile, err := os.Open(fp.Join(db.path, "navfont.ttf"))
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
		}

		_, err = io.Copy(w, fontFile)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
		}

		fontFile.Close()
	} else if !validAuth && res == "/auth" {
		var cookie string

		err = r.ParseForm()
		if err == nil {
			cookie, err = auth(
				r.PostForm,
				db.lock,
				db.path,
				db.pwd,
				db.gAuth,
			)
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

		gAuthLoc, err := genGAuthLoc(db.path)
		if err != nil {
			logger.Error(err)
		} else {
			w.Header().Set("Location", gAuthLoc)
			w.Header().Set("Set-Cookie", cookie)
			w.WriteHeader(302)
		}

	} else if !validAuth && res == "/login" {
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
			genPage(w, db.templates, data)
		} else {
			genPage(w, db.templates, loginPageData)
		}

	} else if !validAuth && !resIsLogin {
		w.Header().Set("Location", "/login")
		w.WriteHeader(302)

	} else if validAuth && res == "/gauth" {
		err = gAuth(creds, r.URL.Query(), db.lock, db.path, db.pwd)
		if err != nil {
			logger.Error(err)
		}

		w.Header().Set("Location", "/timetable")
		w.WriteHeader(302)
	} else if validAuth && res == "/logout" {
		err = logout(creds, db.lock, db.path, db.pwd)
		if err == nil {
			w.Header().Set("Location", "/login")
			w.WriteHeader(302)
		} else {
			logger.Error(err)
			w.WriteHeader(500)
			genPage(w, db.templates, statusServerErrorData)
		}

		// Timetable image
		// NOTE: Perhaps still keep the png generation even though the main timetable will
		// be replaced by a table, rather than image
	} else if validAuth && res == "/timetable.png" {
		genTimetable(creds, w)

		// View a single task
	} else if validAuth && strings.HasPrefix(res, "/tasks/") {
		statusCode, respBody, respHeaders := handleTaskReq(db.templates, r, creds)

		for _, respHeader := range respHeaders {
			w.Header().Set(respHeader[0], respHeader[1])
		}

		w.WriteHeader(statusCode)
		genPage(w, db.templates, respBody)

		// Invalid URL while logged in redirects to /timetable
	} else if validAuth && invalidRes {
		//logger.Info("invalidRes -- redirect")
		w.Header().Set("Location", "/timetable")
		w.WriteHeader(302)

		// Logged in, and the requested URL is valid
	} else if validAuth && !invalidRes {
		webpageData, err := genRes(db.path, res, creds)

		if errors.Is(err, errNotFound) {
			w.WriteHeader(404)
			genPage(w, db.templates, statusNotFoundData)
		} else if err != nil {
			logger.Error(err.Error())
			w.WriteHeader(500)
			genPage(w, db.templates, statusServerErrorData)
		} else {
			genPage(w, db.templates, webpageData)
		}
	}
}

// Create and manage necessary HTML files from template files
func initTemplates(resPath string) (*template.Template, error) {
	// Create "./templates/" dir if it does not exist
	tmplPath := fp.Join(resPath, "templates")
	err := os.MkdirAll(tmplPath, os.ModePerm)
	if err != nil {
		return nil, err
	}

	tmplResPath := tmplPath

	var files []string
	err = fp.WalkDir(tmplResPath, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			logger.Fatal(err)
		}
		// Skip the directory name itself from being appended, although its children won't be affected
		// Excluding via info.IsDir() will exclude files that are under a subdirectory so it cannot be used
		if info.Name() == "components" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		logger.Fatal(errors.NewError("main", err, "error walking the template/ directory"))
	}

	files = remove(files, tmplResPath)

	var requiredFiles []string
	rf := []string{
		"page", "head", "error", "login",
		"components/nav", "components/footer",
		"main", "res", "task", "tasks", "timetable",
	}
	for _, f := range rf {
		requiredFiles = append(requiredFiles, fp.Join(tmplResPath, f+".tmpl"))
	}

	filesMissing := false
	var missingFiles []string
	for _, f := range requiredFiles {
		if !contains(files, f) {
			filesMissing = true
			missingFiles = append(missingFiles, f)
		}
	}
	if filesMissing {
		errStr := fmt.Errorf("%v:\n%+v", errors.ErrMissingFiles.Error(), missingFiles)
		logger.Fatal(errors.NewError("main", nil, errStr.Error()))
	}

	// Find page.tmpl and put it at the front; NOTE: (only needed when not using ExecuteTemplate)
	//var sortedFiles []string
	//pageTmpl := fp.Join(tmplResPath, "page.tmpl")
	//for _, f := range files {
	//	if f == pageTmpl {
	//		sortedFiles = append([]string{f}, sortedFiles...)
	//	} else {
	//		sortedFiles = append(sortedFiles, f)
	//	}
	//}
	sortedFiles := files

	templates := template.Must(template.ParseFiles(sortedFiles...))
	//fmt.Printf("%+v\n", sortedFiles)
	return templates, nil
}

func readConfig() {

}

func main() {
	tlsConn := true

	if len(os.Args) > 2 || len(os.Args) == 2 && os.Args[1] != "-w" {
		logger.Info(errors.ErrBadCommandUsage)
		os.Exit(1)
	}

	if contains(os.Args, "-w") {
		tlsConn = false
	}

	curUser, err := user.Current()
	if err != nil {
		errStr := "taskcollect: Cannot determine current user's home folder"
		logger.Fatal(errStr)
	}

	home := curUser.HomeDir
	configFile := fp.Join(home, "config.json")
	resPath := fp.Join(home, "res/taskcollect")
	certFile := fp.Join(resPath, "cert.pem")
	keyFile := fp.Join(resPath, "key.pem")

	//readConfig()

	// TODO: check the error type and then set different levels e.g. WARN, ERROR, etc.
	err = logger.UseConfig(configFile)
	if err != nil {
		logger.Error("Logging configurations were not initialized")
	} else {
		logger.Info("Logging configurations were initialized successfully")
	}

	// TODO: Hide password input
	var dbPwdInput string
	fmt.Print("Passphrase to user credentials file: ")
	fmt.Scanln(&dbPwdInput)
	pwdBytes := []byte(dbPwdInput)
	var dbPwd []byte

	if len(pwdBytes) == 32 {
		dbPwd = pwdBytes
	} else if len(pwdBytes) > 32 {
		dbPwd = pwdBytes[:32]
	} else {
		zeroLen := 32 - len(pwdBytes)
		dbPwd = pwdBytes

		for i := 0; i < zeroLen; i++ {
			dbPwd = append(dbPwd, 0x00)
		}
	}

	dbMutex := new(sync.Mutex)

	gcid, err := os.ReadFile(fp.Join(resPath, "gauth.json"))
	if err != nil {
		strErr := errors.NewError("main", err, "Google client ID "+errors.ErrFileRead.Error())
		//strErr := fmt.Errorf("taskcollect: Cannot read Google client ID file: %w", err)
		logger.Fatal(strErr.Error())
	}

	templates, err := initTemplates(resPath)
	if err != nil {
		strErr := errors.NewError("main", err, errors.ErrInitFailed.Error()+" for HTML templates")
		logger.Fatal(strErr)
	}
	logger.Info("Successfully initialized HTML templates")

	db := authDb{
		lock:      dbMutex,
		path:      resPath,
		pwd:       dbPwd,
		gAuth:     gcid,
		templates: templates,
	}

	// TODO: Use http.NewServeMux
	http.HandleFunc("/", db.handler)

	if tlsConn {
		logger.Info("Running on port 443")
		err = http.ListenAndServeTLS(":443", certFile, keyFile, nil)
	} else {
		logger.Info("Running on port 8080 (without TLS). DO NOT USE THIS IN PRODUCTION!")
		err = http.ListenAndServe(":8080", nil)
	}

	if err != nil {
		logger.Fatal("%w", err)
	}
}
