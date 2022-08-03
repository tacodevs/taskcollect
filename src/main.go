package main

import (
	"bufio"
	"errors"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	osUser "os/user"
	"strings"
	"sync"
	_ "time/tzdata"
)

var (
	errAuthFailed = errors.New("main: authentication failed")
	errCorruptMIME = errors.New("main: corrupt MIME request")
	errIncompleteCreds = errors.New("main: user has incomplete credentials")
	errInvalidAuth = errors.New("main: invalid session token")
	errNoPlatform = errors.New("main: unsupported platform")
	errNotFound = errors.New("main: cannot find resource")
	needsGAauth = errors.New("main: Google auth required")
)

type authDb struct {
	lock  *sync.Mutex
	path  string
	pwd   []byte
	gAuth []byte
}

type postReader struct {
	div    []byte
	reader io.Reader
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

func handleTask(r *http.Request, c user, p, id, cmd string, gcid []byte) (int, []byte, [][2]string) {
	res := r.URL.EscapedPath()
	statusCode := 200
	var webpage []byte
	var headers [][2]string

	if cmd == "submit" {
		err := submitTask(c, p, id, gcid)

		if err != nil {
			log.Println(err)
			webpage = []byte(serverErrorPage)
			statusCode = 500
		} else {
			index := strings.Index(res, "/submit")
			headers = [][2]string{{"Location", res[:index]}}
			statusCode = 302
		}
	} else if cmd == "upload" {
		filename, reader, err := fileFromReq(r)
		if err != nil {
			log.Println(err)
			return 500, []byte(serverErrorPage), nil
		}

		err = uploadWork(c, p, id, filename, &reader, gcid)
		if err != nil {
			log.Println(err)
			webpage = []byte(serverErrorPage)
			statusCode = 500
		} else {
			index := strings.Index(res, "/upload")
			headers = [][2]string{{"Location", res[:index]}}
			statusCode = 302
		}
	} else if cmd == "remove" {
		filenames := []string{}

		for name, _ := range r.URL.Query() {
			filenames = append(filenames, name)
		}

		err := removeWork(c, p, id, filenames, gcid)
		if err == errNoPlatform {
			webpage = []byte(notFoundPage)
			statusCode = 404
		} else if err != nil {
			log.Println(err)
			webpage = []byte(serverErrorPage)
			statusCode = 500
		} else {
			index := strings.Index(res, "/remove")
			headers = [][2]string{{"Location", res[:index]}}
			statusCode = 302
		}
	} else {
		webpage = []byte(notFoundPage)
		statusCode = 404
	}

	return statusCode, webpage, headers
}

func handleTaskReq(r *http.Request, creds user, gcid []byte) (int, []byte, [][2]string) {
	res := r.URL.EscapedPath()
	statusCode := 200
	var webpage []byte
	var headers [][2]string

	platform := res[7:]
	index := strings.Index(platform, "/")

	if index == -1 {
		webpage = []byte(notFoundPage)
		statusCode = 404
	}

	taskId := platform[index+1:]
	platform = platform[:index]
	index = strings.Index(taskId, "/")

	if index == -1 {
		assignment, err := getTask(platform, taskId, creds, gcid)

		if err != nil {
			log.Println(err)
			webpage = []byte(serverErrorPage)
			statusCode = 500
		}

		title := html.EscapeString(assignment.Name)
		htmlBody := genHtmlTask(assignment, creds)
		taskHtml := genPage(title, htmlBody)
		webpage = []byte(taskHtml)
	} else {
		taskCmd := taskId[index+1:]
		taskId = taskId[:index]

		statusCode, webpage, headers = handleTask(
			r,
			creds,
			platform,
			taskId,
			taskCmd,
			gcid,
		)
	}

	return statusCode, webpage, headers
}

func (db *authDb) handler(w http.ResponseWriter, r *http.Request) {
	res := r.URL.EscapedPath()
	validAuth := true
	creds, err := getCreds(r.Header.Get("Cookie"), db.path, db.pwd, db.gAuth)

	if errors.Is(err, errInvalidAuth) {
		validAuth = false
	} else if err != nil {
		log.Println(err)
		w.Write([]byte(serverErrorPage))
		return
	}

	resIsLogin := false
	invalidRes := false

	if res == "/login" || res == "/auth" {
		resIsLogin = true
	}

	if resIsLogin || res == "/" {
		invalidRes = true
	}

	if res == "/css" {
		w.Header().Set("Content-Type", `text/css, charset="utf-8"`)
		cssFile, err := os.Open(db.path + "styles.css")

		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
		}

		_, err = io.Copy(w, cssFile)

		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
		}

		cssFile.Close()
	} else if !validAuth && res == "/auth" {
		cookie, err := auth(
			r.URL.Query(),
			db.lock,
			db.path,
			db.pwd,
			db.gAuth,
		)

		if err == nil {
			w.Header().Set("Location", "/timetable")
			w.Header().Set("Set-Cookie", cookie)
			w.WriteHeader(302)
			return
		} else if !errors.Is(err, needsGAauth) {
			log.Println(err)
			w.Header().Set("Location", "/login?auth=failed")
			w.WriteHeader(302)
			return
		}

		gAuthLoc, err := genGAuthLoc(db.path)
		if err != nil {
			log.Println(err)
		} else {
			w.Header().Set("Location", gAuthLoc)
			w.Header().Set("Set-Cookie", cookie)
			w.WriteHeader(302)
		}

	} else if !validAuth && res == "/login" {
		if r.URL.Query().Get("auth") == "failed" {
			w.WriteHeader(401)
			w.Write([]byte(loginFailed))
		} else {
			w.Write([]byte(loginPage))
		}

	} else if !validAuth && !resIsLogin {
		w.Header().Set("Location", "/login")
		w.WriteHeader(302)
	} else if validAuth && res == "/gauth" {
		err = gAuth(creds, r.URL.Query(), db.lock, db.path, db.pwd)
		if err != nil {
			log.Println(err)
		}

		w.Header().Set("Location", "/timetable")
		w.WriteHeader(302)

	} else if validAuth && res == "/logout" {
		err = logout(creds, db.lock, db.path, db.pwd)
		if err == nil {
			w.Header().Set("Location", "/login")
			w.WriteHeader(302)
		} else {
			log.Println(err)
			w.WriteHeader(500)
			w.Write([]byte(serverErrorPage))
		}

	} else if validAuth && res == "/timetable.png" {
		genTimetable(creds, w)
	} else if validAuth && strings.HasPrefix(res, "/tasks/") {
		statusCode, respBody, respHeaders := handleTaskReq(
			r, creds, db.gAuth,
		)

		for _, respHeader := range respHeaders {
			w.Header().Set(respHeader[0], respHeader[1])
		}

		w.WriteHeader(statusCode)
		w.Write(respBody)
	} else if validAuth && invalidRes {
		w.Header().Set("Location", "/timetable")
		w.WriteHeader(302)
	} else if validAuth && !invalidRes {
		webpage, err := genRes(res, creds, db.gAuth)

		if errors.Is(err, errNotFound) {
			w.WriteHeader(404)
			w.Write([]byte(notFoundPage))
		} else if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			w.Write([]byte(serverErrorPage))
		} else {
			w.Write(webpage)
		}
	}
}

func main() {
	curUser, err := osUser.Current()

	if err != nil {
		errStr := "taskcollect: Can't determine current user's home folder."
		os.Stderr.WriteString(errStr + "\n")
		os.Exit(1)
	}

	home := curUser.HomeDir
	resPath := home + "/res/taskcollect/"
	//certFile := resPath + "cert.pem"
	//keyFile := resPath + "key.pem"

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
		zerolen := 32 - len(pwdBytes)
		dbPwd = pwdBytes

		for i := 0; i < zerolen; i++ {
			dbPwd = append(dbPwd, 0x00)
		}
	}

	dbMutex := new(sync.Mutex)

	gcid, err := ioutil.ReadFile(resPath + "gauth.json")
	if err != nil {
		strErr := "taskcollect: Can't read Google client ID file."
		os.Stderr.WriteString(strErr + "\n")
		os.Exit(1)
	}

	db := authDb{
		lock:  dbMutex,
		path:  resPath,
		pwd:   dbPwd,
		gAuth: gcid,
	}

	// TODO: Use http.NewServeMux

	http.HandleFunc("/", db.handler)
	http.ListenAndServe(":8080", nil)
	//http.ListenAndServeTLS(":443", certFile, keyFile, nil)
}
