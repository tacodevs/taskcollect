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
	osusr "os/user"
	"strings"
	"sync"
	_ "time/tzdata"
)

type authdb struct {
	lock	*sync.Mutex
	path	string
	pwd	[]byte
	gauth	[]byte
}

type postReader struct {
	div	[]byte
	reader	io.Reader
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
			c, err := reader.Peek(i+1)

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
		return "", nil, errors.New("taskcollect: corrupt MIME request")
	}

	idx := strings.Index(line, "form-data;")
	_, formvals, err := mime.ParseMediaType(line[idx:])

	if err != nil {
		return "", nil, err
	}

	for line != "\r\n" {
		line, err = reader.ReadString('\n')

		if err != nil {
			return "", nil, err
		}
	}

	filename := formvals["filename"]

	pr := postReader{
		div: []byte("\r\n" + div + "--"),
		reader: reader,
	}

	return filename, pr, nil
}

func handleTaskFunc(r *http.Request, c user, p, id, cmd string, gcid []byte) (int, []byte, [][2]string) {
	res := r.URL.EscapedPath()
	statCode := 200
	var webpage []byte
	var headers [][2]string

	if cmd == "submit" {
		err := submitTask(c, p, id, gcid)

		if err != nil {
			log.Println("main.go: 130:", err)
			webpage = []byte(srvErrPage)
			statCode = 500
		} else {
			index := strings.Index(res, "/submit")
			headers = [][2]string{{"Location", res[:index]}}
			statCode = 302
		}
	} else if cmd == "upload" {
		filename, reader, err := fileFromReq(r)

		if err != nil {
			log.Println("main.go: 142:", err)
			return 500, []byte(srvErrPage), nil
		}

		err = uploadWork(c, p, id, filename, &reader, gcid)

		if err != nil {
			log.Println("main.go: 149:", err)
			webpage = []byte(srvErrPage)
			statCode = 500
		} else {
			index := strings.Index(res, "/upload")
			headers = [][2]string{{"Location", res[:index]}}
			statCode = 302
		}
	} else if cmd == "remove" {
		filenames := []string{}

		for name, _ := range r.URL.Query() {
			filenames = append(filenames, name)
		}

		err := removeWork(c, p, id, filenames, gcid)

		if err == errNoPlatform {
			webpage = []byte(notFoundPage)
			statCode = 404
		} else if err != nil {
			log.Println("main.go: 170:", err)
			webpage = []byte(srvErrPage)
			statCode = 500
		} else {
			index := strings.Index(res, "/remove")
			headers = [][2]string{{"Location", res[:index]}}
			statCode = 302
		}
	} else {
		webpage = []byte(notFoundPage)
		statCode = 404
	}

	return statCode, webpage, headers
}

func handleTaskReq(r *http.Request, creds user, gcid []byte) (int, []byte, [][2]string) {
	res := r.URL.EscapedPath()
	statCode := 200
	var webpage []byte
	var headers [][2]string

	platform := res[7:]
	index := strings.Index(platform, "/")

	if index == -1 {
		webpage = []byte(notFoundPage)
		statCode = 404
	}

	taskId := platform[index+1:]
	platform = platform[:index]
	index = strings.Index(taskId, "/")

	if index == -1 {
		assignment, err := getTask(platform, taskId, creds, gcid)

		if err != nil {
			log.Println("main.go: 208:", err)
			webpage = []byte(srvErrPage)
			statCode = 500
		}

		title := html.EscapeString(assignment.Name)
		htmlBody := genHtmlTask(assignment, creds)
		taskHtml := genPage(title, htmlBody)
		webpage = []byte(taskHtml)
	} else {
		taskCmd := taskId[index+1:]
		taskId = taskId[:index]

		statCode, webpage, headers = handleTaskFunc(
			r,
			creds,
			platform,
			taskId,
			taskCmd,
			gcid,
		)
	}

	return statCode, webpage, headers
}

func (db *authdb) handler(w http.ResponseWriter, r *http.Request) {
	res := r.URL.EscapedPath()
	validAuth := true
	creds, err := getCreds(r.Header.Get("Cookie"), db.path, db.pwd)

	if errors.Is(err, errInvalidAuth) {
		validAuth = false
	} else if err != nil {
		log.Println("main.go: 248:", err)
		w.Write([]byte(srvErrPage))
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
		w.Write([]byte(css))
	} else if !validAuth && res == "/auth" {
		cookie, err := auth(
			r.URL.Query(),
			db.lock,
			db.path,
			db.pwd,
			db.gauth,
		)

		if err == nil {
			w.Header().Set("Location", "/tasks")
			w.Header().Set("Set-Cookie", cookie)
			w.WriteHeader(302)
			return
		} else if !errors.Is(err, needsGauth) {
			log.Println(err)
			w.Header().Set("Location", "/login?auth=failed")
			w.WriteHeader(302)
			return
		}
	
		gauthloc, err := genGauthloc(db.path)

		if err != nil {
			log.Println(err)
		} else {
			w.Header().Set("Location", gauthloc)
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
		err = gauth(creds, r.URL.Query(), db.lock, db.path, db.pwd)

		if err != nil {
			log.Println(err)
		}

		w.Header().Set("Location", "/tasks")
		w.WriteHeader(302)
	} else if validAuth && res == "/logout" {
		err = logout(creds, db.lock, db.path, db.pwd)

		if err == nil {
			w.Header().Set("Location", "/login")
			w.WriteHeader(302)
		} else {
			log.Println("main.go: 317:", err)
			w.WriteHeader(500)
			w.Write([]byte(srvErrPage))
		}
	} else if validAuth && res == "/timetable.png" {
		genTimetable(creds, w)
	} else if validAuth && strings.HasPrefix(res, "/tasks/") {
		statCode, respBody, respHeaders := handleTaskReq(
			r, creds, db.gauth,
		)

		for _, respHeader := range respHeaders {
			w.Header().Set(respHeader[0], respHeader[1])
		}

		w.WriteHeader(statCode)
		w.Write(respBody)
	} else if validAuth && invalidRes {
		w.Header().Set("Location", "/tasks")
		w.WriteHeader(302)
	} else if validAuth && !invalidRes {
		webpage, err := genRes(res, creds, db.gauth)

		if errors.Is(err, errNotFound) {
			w.WriteHeader(404)
			w.Write([]byte(notFoundPage))
		} else if err != nil {
			log.Println("main.go: 343:", err)
			w.WriteHeader(500)
			w.Write([]byte(srvErrPage))
		} else {
			w.Write(webpage)
		}
	}
}

func main() {
	curUser, err := osusr.Current()

	if err != nil {
		strerr := "taskcollect: Can't determine current user's home folder."
		os.Stderr.WriteString(strerr + "\n")
		os.Exit(1)
	}

	home := curUser.HomeDir
	respath := home + "/res/taskcollect/"
	//certfile := respath + "cert.pem"
	//keyfile := respath + "key.pem"

	var dbpwd string
	fmt.Print("Passphrase to user credentials file: ")
	fmt.Scanln(&dbpwd)
	pwdbytes := []byte(dbpwd)
	var dbp []byte

	if len(pwdbytes) == 32 {
		dbp = pwdbytes
	} else if len(pwdbytes) > 32 {
		dbp = pwdbytes[:32]
	} else {
		zerolen := 32 - len(pwdbytes)
		dbp = pwdbytes

		for i := 0; i < zerolen; i++ {
			dbp = append(dbp, 0x00)
		}
	}

	dbmutex := new(sync.Mutex)

	gcid, err := ioutil.ReadFile(respath + "gauth.json")

	if err != nil {
		strerr := "taskcollect: Can't read Google client ID file."
		os.Stderr.WriteString(strerr + "\n")
		os.Exit(1)
	}

	db := authdb{
		lock: dbmutex,
		path: respath,
		pwd: dbp,
		gauth: gcid,
	}

	http.HandleFunc("/", db.handler)
	http.ListenAndServe(":8080", nil)
	//http.ListenAndServeTLS(":443", certfile, keyfile, nil)
}
