package main

import (
	"bufio"
	"context"
	"crypto/aes"
	"crypto/cipher"
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

var errInvalidAuth = errors.New("taskcollect: invalid session token")
var errAuthFailed = errors.New("taskcollect: authentication failed")
var needsGAauth = errors.New("taskcollect: Google auth required")

type user struct {
	Timezone   *time.Location
	School     string
	Username   string
	Password   string
	Token      string
	SiteTokens map[string]string
}

func decryptDb(dbPath string, pwd []byte) (*bufio.Reader, error) {
	ecrfile, err := ioutil.ReadFile(dbPath)

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

	if len(ecrfile) < nonceSize {
		return nil, err
	}

	nonce, ecrfile := ecrfile[:nonceSize], ecrfile[nonceSize:]
	dcrfile, err := gcm.Open(nil, nonce, ecrfile, nil)

	if err != nil {
		return nil, err
	}

	s := string(dcrfile)
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

	var ln []string

	for {
		line, err := db.ReadString('\n')

		if err != nil {
			return user{}, errInvalidAuth
		}

		ln = strings.Split(line, "\t")

		if token == ln[0] {
			creds.Token = ln[0]
			creds.School = ln[1]
			creds.Username = ln[2]
			creds.Password = ln[3]
			break
		}
	}

	if creds.School == "gihs" {
		creds.Timezone, err = time.LoadLocation("Australia/Adelaide")

		if err != nil {
			return user{}, err
		}

		creds.SiteTokens = map[string]string{
			"daymap": ln[4],
			"gclass": ln[5],
		}
	} else {
		return user{}, errInvalidAuth
	}

	return creds, nil
}

func finduser(dbPath string, dbPwd []byte, usr, pwd string) (bool, error) {
	db, err := decryptDb(dbPath, dbPwd)

	if err != nil {
		return false, err
	}

	var ln []string
	exists := false

	for {
		line, err := db.ReadString('\n')

		if err != nil {
			return false, errInvalidAuth
		}

		ln = strings.Split(line, "\t")

		if usr == ln[2] && pwd == ln[3] {
			exists = true
			break
		}
	}

	return exists, nil
}

func genGauthloc(resPath string) (string, error) {
	gcid, err := ioutil.ReadFile(resPath + "gauth.json")

	if err != nil {
		panic(err)
	}

	gauthConfig, err := google.ConfigFromJSON(
		gcid,
		classroom.ClassroomCoursesReadonlyScope,
		classroom.ClassroomStudentSubmissionsMeReadonlyScope,
		classroom.ClassroomCourseworkMeScope,
		classroom.ClassroomCourseworkmaterialsReadonlyScope,
	)

	if err != nil {
		return "", err
	}

	gauthloc := gauthConfig.AuthCodeURL(
		"state-token",
		oauth2.AccessTypeOffline,
	)

	return gauthloc, nil
}

func gAuth(creds user, query url.Values, authDb *sync.Mutex, resPath string, dbPwd []byte) error {
	dbPath := resPath + "creds"
	authcode := query.Get("code")

	clientId, err := ioutil.ReadFile(resPath + "gauth.json")

	if err != nil {
		return nil
	}

	gauthConfig, err := google.ConfigFromJSON(
		clientId,
		classroom.ClassroomCoursesReadonlyScope,
		classroom.ClassroomStudentSubmissionsMeReadonlyScope,
		classroom.ClassroomCourseworkMeScope,
		classroom.ClassroomCourseworkmaterialsReadonlyScope,
	)

	if err != nil {
		return err
	}

	gTok, err := gauthConfig.Exchange(context.TODO(), authcode)

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

func getgtok(dbPath string, dbPwd []byte, usr, pwd string) (string, error) {
	db, err := decryptDb(dbPath, dbp)

	if err != nil {
		return "", err
	}

	var ln []string
	var gTok string

	for {
		line, err := db.ReadString('\n')

		if err != nil {
			return "", nil
		}

		ln = strings.Split(line, "\t")

		if usr == ln[2] && pwd == ln[3] {
			break
		}
	}

	if len(ln) != 6 {
		return "", errors.New("main: user has incomplete credentials")
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

	gTok, err := getgtok(dbPath, dbPwd, usr, pwd)

	if err != nil {
		return "", err
	}

	gtesterr := make(chan error)

	if gTok != "" {
		go gclass.Test(gcid, gTok, gtesterr)
	}

	dmcreds, err := daymap.Auth(school, usr, pwd)

	if errors.Is(err, daymap.ErrAuthFailed) {
		return "", errAuthFailed
	} else if err != nil {
		userExists, err := finduser(dbPath, dbPwd, usr, pwd)

		if err != nil {
			return "", err
		}

		if !userExists {
			return "", errAuthFailed
		}
	}

	siteTokens := map[string]string{
		"daymap": dmcreds.Token,
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
	timezone := dmcreds.Timezone

	if err != nil {
		return "", err
	}

	gauthStatus := needsGAauth

	if gTok != "" {
		err = <-gtesterr

		if err == nil {
			siteTokens["gclass"] = gTok
			gauthStatus = nil
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
	return cookie, gauthStatus
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
	line := creds.Token + "\t"
	line += creds.School + "\t"
	line += creds.Username + "\t"
	line += creds.Password + "\t"

	platGihs := []string{"daymap", "gclass"}

	for _, plat := range platGihs {
		line += creds.SiteTokens[plat] + "\t"
	}

	buf := []rune(line)
	buf[len(buf)-1] = '\n'
	line = string(buf)
	return line
}

func writeCreds(creds user, dbPath string, pwd []byte) error {
	ecrfile, err := ioutil.ReadFile(dbPath)

	if err != nil {
		return err
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
		return err
	}

	gcm, err := cipher.NewGCM(aesCipher)

	if err != nil {
		return err
	}

	nonceSize := gcm.NonceSize()

	if len(ecrfile) < nonceSize {
		return err
	}

	nonce, ecrfile := ecrfile[:nonceSize], ecrfile[nonceSize:]
	dcrfile, err := gcm.Open(nil, nonce, ecrfile, nil)

	if err != nil {
		return err
	}

	s := string(dcrfile)
	sr := strings.NewReader(s)
	db := bufio.NewReader(sr)
	var new string
	exists := false

	for {
		line, err := db.ReadString('\n')

		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}

		ln := strings.Split(line, "\t")

		if creds.School == ln[1] && creds.Username == ln[2] {
			line = genCredLine(creds)
			exists = true
			new += line
			break
		}

		new += line
	}

	if !exists {
		line := genCredLine(creds)
		new += line
	}

	newfile := gcm.Seal(nonce, nonce, []byte(new), nil)
	err = ioutil.WriteFile(dbPath, newfile, 0640)

	if err != nil {
		return err
	}

	return nil
}
