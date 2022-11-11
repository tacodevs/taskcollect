package main

import (
	"bufio"
	"context"
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/rand"
	"net/url"
	"os"
	fp "path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/classroom/v1"

	"main/daymap"
	"main/errors"
	"main/gclass"
)

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
	//logger.Debug("db path: %+v\n", dbPath)
	ecrFile, err := os.ReadFile(dbPath)

	if err != nil {
		//log.Println("decryptDb ERR 1")
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
		//log.Println("decryptDb ERR 2")
		return nil, err
	}

	gcm, err := cipher.NewGCM(aesCipher)

	if err != nil {
		//log.Println("decryptDb ERR 3")
		return nil, err
	}

	nonceSize := gcm.NonceSize()

	if len(ecrFile) < nonceSize {
		//log.Println("decryptDb ERR 4")
		return nil, err
	}

	nonce, ecrFile := ecrFile[:nonceSize], ecrFile[nonceSize:]
	dcrFile, err := gcm.Open(nil, nonce, ecrFile, nil)

	if err != nil {
		//log.Println("decryptDb ERR 5")
		return nil, err
	}

	s := string(dcrFile)
	sr := strings.NewReader(s)
	db := bufio.NewReader(sr)
	return db, nil
}

func getCreds(cookies string, resPath string, pwd []byte) (tcUser, error) {
	// TODO: Error handling needs to be improved

	dbPath := fp.Join(resPath, "creds")
	creds := tcUser{}
	var token string

	start := strings.Index(cookies, "token=")

	if start == -1 {
		//log.Println("ERROR 1")

		return tcUser{}, errInvalidAuth
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
		//log.Println("ERROR 2")
		return tcUser{}, err
	}

	if db == nil {
		//log.Println("ERROR 3")
		return tcUser{}, errors.ErrInvalidAuth
	}

	var ln []string

	for {
		line, err := db.ReadString('\n')

		if err != nil {
			//log.Println("ERROR 4")
			return tcUser{}, errors.ErrInvalidAuth
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
			//log.Println("ERROR 5")
			return tcUser{}, err
		}

		creds.SiteTokens = map[string]string{
			"daymap": tsvUnescapeString(ln[4]),
			"gclass": tsvUnescapeString(ln[5]),
		}
	} else {
		//log.Println("ERROR 6")
		return tcUser{}, errors.ErrInvalidAuth
	}

	return creds, nil
}

func findUser(dbPath string, dbPwd []byte, usr, pwd string) (bool, error) {
	db, err := decryptDb(dbPath, dbPwd)

	if err != nil {
		return false, err
	}

	if db == nil {
		return false, errors.ErrInvalidAuth
	}

	var ln []string
	exists := false

	for {
		line, err := db.ReadString('\n')

		if err != nil {
			return false, errors.ErrInvalidAuth
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
	gcid, err := os.ReadFile(fp.Join(resPath, "gauth.json"))

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

func gAuth(creds tcUser, query url.Values, authDb *sync.Mutex, resPath string, dbPwd []byte) error {
	dbPath := fp.Join(resPath, "creds")
	authCode := query.Get("code")

	clientId, err := os.ReadFile(fp.Join(resPath, "gauth.json"))

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
		return "", errors.ErrInvalidAuth
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
		return "", errors.ErrIncompleteCreds
	}

	return gTok, nil
}

func auth(query url.Values, authDb *sync.Mutex, resPath string, dbPwd, gcid []byte) (string, error) {
	dbPath := fp.Join(resPath, "creds")
	school := query.Get("school")

	if school != "gihs" {
		return "", errors.ErrAuthFailed
	}

	usr := query.Get("usr")
	pwd := query.Get("pwd")

	gTok, err := getGTok(dbPath, dbPwd, usr, pwd)

	if !errors.Is(err, errors.ErrInvalidAuth) && err != nil {
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
			return "", errors.ErrAuthFailed
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

	gAuthStatus := errors.ErrNeedsGAuth

	if gTok != "" {
		err = <-gTestErr
		if err == nil {
			siteTokens["gclass"] = gTok
			gAuthStatus = nil
		}
	}

	creds := tcUser{
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

func logout(creds tcUser, authDb *sync.Mutex, resPath string, dbPwd []byte) error {
	dbPath := fp.Join(resPath, "creds")
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

func genCredLine(creds tcUser) string {
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

func writeCreds(creds tcUser, dbPath string, pwd []byte) error {
	db, err := decryptDb(dbPath, pwd)

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

	newFile := gcm.Seal(nonce, nonce, []byte(new), nil)
	err = os.WriteFile(dbPath, newFile, 0640)

	if err != nil {
		return err
	}

	return nil
}
