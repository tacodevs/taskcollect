package main

import (
	"bufio"
	"context"
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"main/daymap"
	"main/gclass"
	"math/rand"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/classroom/v1"
)

type user struct {
	Timezone   *time.Location
	School     string
	Username   string
	Password   string
	Token      string
	SiteTokens map[string]string
}

func tsvEscapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", `\\`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

func tsvUnescapeString(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\t`, "\t")
	s = strings.ReplaceAll(s, `\\`, "\\")
	return s
}

func decryptDb(dbPath string, pwd []byte) (*bufio.Reader, error) {
	ecrFile, err := ioutil.ReadFile(dbPath)

	if err != nil {
		return nil, err
	}

	var key []byte

	if len(pwd) == 32 {
		key = pwd
	} else if len(pwd) > 32 {
		key = pwd[:32]
	} else {
		zeroLen := 32 - len(pwd)
		key = pwd

		for i := 0; i != zeroLen; i++ {
			key = append(key, 0x00)
		}
	}

	aesCipher, err := aes.NewCipher(key)

	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(aesCipher)

	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()

	if len(ecrFile) < nonceSize {
		return nil, err
	}

	nonce, ecrFile := ecrFile[:nonceSize], ecrFile[nonceSize:]
	dcrFile, err := gcm.Open(nil, nonce, ecrFile, nil)

	if err != nil {
		return nil, err
	}

	s := string(dcrFile)
	sr := strings.NewReader(s)
	db := bufio.NewReader(sr)
	return db, nil
}

func getCreds(cookies string, resPath string, pwd []byte) (user, error) {
	dbPath := resPath + "creds"
	creds := user{}
	var token string
	start := strings.Index(cookies, "token=")

	if start == -1 {
		return user{}, errInvalidAuth
	}

	start += 6
	end := strings.Index(cookies[start:], ";")

	if end == -1 {
		token = cookies[start:]
	} else {
		token = cookies[start:end]
	}

	db, err := decryptDb(dbPath, pwd)

	if err != nil {
		return user{}, err
	}

	if db == nil {
		return user{}, errInvalidAuth
	}

	var ln []string

	for {
		line, err := db.ReadString('\n')

		if err != nil {
			return user{}, errInvalidAuth
		}

		ln = strings.Split(line, "\t")

		if token == tsvUnescapeString(ln[0]) {
			creds.Token = tsvUnescapeString(ln[0])
			creds.School = tsvUnescapeString(ln[1])
			creds.Username = tsvUnescapeString(ln[2])
			creds.Password = tsvUnescapeString(ln[3])
			break
		}
	}

	if creds.School == "gihs" {
		creds.Timezone, err = time.LoadLocation("Australia/Adelaide")

		if err != nil {
			return user{}, err
		}

		creds.SiteTokens = map[string]string{
			"daymap": tsvUnescapeString(ln[4]),
			"gclass": tsvUnescapeString(ln[5]),
		}
	} else {
		return user{}, errInvalidAuth
	}

	return creds, nil
}

func findUser(dbPath string, dbPwd []byte, usr, pwd string) (bool, error) {
	db, err := decryptDb(dbPath, dbPwd)

	if err != nil {
		return false, err
	}

	if db == nil {
		return false, errInvalidAuth
	}

	var ln []string
	exists := false

	for {
		line, err := db.ReadString('\n')

		if err != nil {
			return false, errInvalidAuth
		}

		ln = strings.Split(line, "\t")
		tsvUsr := tsvUnescapeString(ln[2])
		tsvPwd := tsvUnescapeString(ln[3])

		if usr == tsvUsr && pwd == tsvPwd {
			exists = true
			break
		}
	}

	return exists, nil
}

func genGAuthLoc(resPath string) (string, error) {
	gcid, err := ioutil.ReadFile(resPath + "gauth.json")

	if err != nil {
		panic(err)
	}

	gAuthConfig, err := google.ConfigFromJSON(
		gcid,
		classroom.ClassroomCoursesReadonlyScope,
		classroom.ClassroomStudentSubmissionsMeReadonlyScope,
		classroom.ClassroomCourseworkMeScope,
		classroom.ClassroomCourseworkmaterialsReadonlyScope,
	)

	if err != nil {
		return "", err
	}

	gAuthLoc := gAuthConfig.AuthCodeURL(
		"state-token",
		oauth2.ApprovalForce,
		oauth2.AccessTypeOffline,
	)

	return gAuthLoc, nil
}

func gAuth(creds user, query url.Values, authDb *sync.Mutex, resPath string, dbPwd []byte) error {
	dbPath := resPath + "creds"
	authCode := query.Get("code")

	clientId, err := ioutil.ReadFile(resPath + "gauth.json")

	if err != nil {
		return nil
	}

	gAuthConfig, err := google.ConfigFromJSON(
		clientId,
		classroom.ClassroomCoursesReadonlyScope,
		classroom.ClassroomStudentSubmissionsMeReadonlyScope,
		classroom.ClassroomCourseworkMeScope,
		classroom.ClassroomCourseworkmaterialsReadonlyScope,
	)

	if err != nil {
		return err
	}

	gTok, err := gAuthConfig.Exchange(context.TODO(), authCode)

	if err != nil {
		return err
	}

	token, err := json.Marshal(gTok)

	if err != nil {
		return err
	}

	creds.SiteTokens["gclass"] = string(token)

	authDb.Lock()
	err = writeCreds(creds, dbPath, dbPwd)

	if err != nil {
		return err
	}

	authDb.Unlock()
	return nil
}

func getGTok(dbPath string, dbPwd []byte, usr, pwd string) (string, error) {
	db, err := decryptDb(dbPath, dbPwd)

	if err != nil {
		return "", err
	}

	if db == nil {
		return "", errInvalidAuth
	}

	var ln []string
	var gTok string

	for {
		line, err := db.ReadString('\n')

		if err != nil {
			return "", nil
		}

		ln = strings.Split(line, "\t")
		tsvUsr := tsvUnescapeString(ln[2])
		tsvPwd := tsvUnescapeString(ln[3])

		if usr == tsvUsr && pwd == tsvPwd {
			gTok = tsvUnescapeString(ln[5])
			break
		}
	}

	if len(ln) != 6 {
		return "", errIncompleteCreds
	}

	return gTok, nil
}

func auth(query url.Values, authDb *sync.Mutex, resPath string, dbPwd, gcid []byte) (string, error) {
	dbPath := resPath + "creds"
	school := query.Get("school")

	if school != "gihs" {
		return "", errAuthFailed
	}

	usr := query.Get("usr")
	pwd := query.Get("pwd")

	gTok, err := getGTok(dbPath, dbPwd, usr, pwd)

	if !errors.Is(err, errInvalidAuth) && err != nil {
		return "", err
	}

	gTestErr := make(chan error)

	if gTok != "" {
		go gclass.Test(gcid, gTok, gTestErr)
	}

	if !strings.HasPrefix(usr, `CURRIC\`) {
		usr = `CURRIC\` + usr
	}

	dmCreds, err := daymap.Auth(school, usr, pwd)

	if err != nil {
		userExists, err := findUser(dbPath, dbPwd, usr, pwd)

		if err != nil {
			return "", err
		}

		if !userExists {
			return "", errAuthFailed
		}
	}

	siteTokens := map[string]string{
		"daymap": dmCreds.Token,
		"gclass": "",
	}

	b := make([]byte, 32)
	rand.Seed(time.Now().UnixNano())

	for i := range b {
		b[i] = byte(rand.Intn(255))
	}

	token := base64.StdEncoding.EncodeToString(b)
	cookie := "token=" + token + "; Expires="
	cookie += time.Now().UTC().AddDate(0, 0, 7).Format(time.RFC1123)
	timezone := dmCreds.Timezone

	gAuthStatus := needsGAauth

	if gTok != "" {
		err = <-gTestErr
		if err == nil {
			siteTokens["gclass"] = gTok
			gAuthStatus = nil
		}
	}

	creds := user{
		Timezone:   timezone,
		School:     school,
		Username:   usr,
		Password:   pwd,
		Token:      token,
		SiteTokens: siteTokens,
	}

	authDb.Lock()
	err = writeCreds(creds, dbPath, dbPwd)
	if err != nil {
		return "", err
	}

	authDb.Unlock()
	return cookie, gAuthStatus
}

func logout(creds user, authDb *sync.Mutex, resPath string, dbPwd []byte) error {
	dbPath := resPath + "creds"
	creds.Token = ""

	for k, _ := range creds.SiteTokens {
		creds.SiteTokens[k] = ""
	}

	authDb.Lock()
	err := writeCreds(creds, dbPath, dbPwd)
	if err != nil {
		return err
	}

	authDb.Unlock()
	return nil
}

func genCredLine(creds user) string {
	line := tsvEscapeString(creds.Token) + "\t"
	line += tsvEscapeString(creds.School) + "\t"
	line += tsvEscapeString(creds.Username) + "\t"
	line += tsvEscapeString(creds.Password) + "\t"

	platGihs := []string{"daymap", "gclass"}

	for _, plat := range platGihs {
		line += tsvEscapeString(creds.SiteTokens[plat]) + "\t"
	}

	buf := []rune(line)
	buf[len(buf)-1] = '\n'
	line = string(buf)
	return line
}

func writeCreds(creds user, dbpath string, pwd []byte) error {
	db, err := decryptDb(dbpath, pwd)

	if err != nil {
		return err
	}

	var new string
	exists := false

	if db != nil {
		for {
			line, err := db.ReadString('\n')

			if errors.Is(err, io.EOF) && new != "" {
				break
			} else if err != nil {
				return err
			}

			ln := strings.Split(line, "\t")
			tsvSchool := tsvUnescapeString(ln[1])
			tsvUsr := tsvUnescapeString(ln[2])

			if creds.School == tsvSchool && creds.Username == tsvUsr {
				line = genCredLine(creds)
				exists = true
				new += line
				break
			}

			new += line
		}
	}

	if !exists {
		line := genCredLine(creds)
		new += line
	}

	aesCipher, err := aes.NewCipher(pwd)

	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(cryptorand.Reader, nonce)

	if err != nil {
		return err
	}

	newfile := gcm.Seal(nonce, nonce, []byte(new), nil)
	err = ioutil.WriteFile(dbpath, newfile, 0640)

	if err != nil {
		return err
	}

	return nil
}
