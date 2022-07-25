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

var errInvalidAuth = errors.New("taskcollect: invalid session token")
var errAuthFailed = errors.New("taskcollect: authentication failed")
var needsGauth = errors.New("taskcollect: Google auth required")

type user struct {
	Timezone *time.Location
	School string
	Username string
	Password string
	Token string
	SiteTokens map[string]string
}

func decryptdb(dbpath string, pwd []byte) (*bufio.Reader, error) {
	ecrfile, err := ioutil.ReadFile(dbpath)

	if err != nil {
		return nil, err
	}

	var key []byte

	if len(pwd) == 32 {
		key = pwd
	} else if len(pwd) > 32 {
		key = pwd[:32]
	} else {
		zerolen := 32 - len(pwd)
		key = pwd

		for i := 0; i != zerolen; i++ {
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

func getCreds(cookies string, respath string, pwd []byte) (user, error) {
	dbpath := respath + "creds"
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

	db, err := decryptdb(dbpath, pwd)

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

func finduser(dbpath string, dbp []byte, usr, pwd string) (bool, error) {
	db, err := decryptdb(dbpath, dbp)

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

func genGauthloc(respath string) (string, error) {
	gcid, err := ioutil.ReadFile(respath + "gauth.json")

	if err != nil {
		panic(err)
	}

	gauthcnf, err := google.ConfigFromJSON(
		gcid,
		classroom.ClassroomCoursesReadonlyScope,
		classroom.ClassroomStudentSubmissionsMeReadonlyScope,
		classroom.ClassroomCourseworkMeScope,
		classroom.ClassroomCourseworkmaterialsReadonlyScope,
	)

	if err != nil {
		return "", err
	}

	gauthloc := gauthcnf.AuthCodeURL(
		"state-token",
		oauth2.AccessTypeOffline,
	)

	return gauthloc, nil
}

func gauth(creds user, query url.Values, authdb *sync.Mutex, respath string, dbp []byte) error {
	dbpath := respath + "creds"
	authcode := query.Get("code")

	clientId, err := ioutil.ReadFile(respath + "gauth.json")

	if err != nil {
		return nil
	}

	gauthcnf, err := google.ConfigFromJSON(
		clientId,
		classroom.ClassroomCoursesReadonlyScope,
		classroom.ClassroomStudentSubmissionsMeReadonlyScope,
		classroom.ClassroomCourseworkMeScope,
		classroom.ClassroomCourseworkmaterialsReadonlyScope,
	)

	if err != nil {
		return err
	}

	gtok, err := gauthcnf.Exchange(context.TODO(), authcode)

	if err != nil {
		return err
	}

	token, err := json.Marshal(gtok)

	if err != nil {
		return err
	}

	creds.SiteTokens["gclass"] = string(token)

	authdb.Lock()
	err = writeCreds(creds, dbpath, dbp)

	if err != nil {
		return err
	}

	authdb.Unlock()
	return nil
}


func getgtok(dbpath string, dbp []byte, usr, pwd string) (string, error) {
	db, err := decryptdb(dbpath, dbp)

	if err != nil {
		return "", err
	}

	if db == nil {
		return "", errInvalidAuth
	}

	var ln []string
	var gtok string

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

	return gtok, nil
}

func auth(query url.Values, authdb *sync.Mutex, respath string, dbp, gcid []byte) (string, error) {
	dbpath := respath + "creds"
	school := query.Get("school")

	if school != "gihs" {
		return "", errAuthFailed
	}

	usr := query.Get("usr")
	pwd := query.Get("pwd")

	gtok, err := getgtok(dbpath, dbp, usr, pwd)

	if !errors.Is(err, errInvalidAuth) && err != nil {
		return "", err
	}

	gtesterr := make(chan error)

	if gtok != "" {
		go gclass.Test(gcid, gtok, gtesterr)
	}

	dmcreds, err := daymap.Auth(school, usr, pwd)

	if errors.Is(err, daymap.ErrAuthFailed) {
		return "", errAuthFailed
	} else if err != nil {
		userExists, err := finduser(dbpath, dbp, usr, pwd)

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
	cookie += time.Now().UTC().AddDate(0,0,7).Format(time.RFC1123)
	timezone := dmcreds.Timezone

	if err != nil {
		return "", err
	}

	gauthStatus := needsGauth

	if gtok != "" {
		err = <-gtesterr

		if err == nil {
			siteTokens["gclass"] = gtok
			gauthStatus = nil
		}
	}

	creds := user{
		Timezone: timezone,
		School: school,
		Username: usr,
		Password: pwd,
		Token: token,
		SiteTokens: siteTokens,
	}

	authdb.Lock()
	err = writeCreds(creds, dbpath, dbp)

	if err != nil {
		return "", err
	}

	authdb.Unlock()
	return cookie, gauthStatus
}

func logout(creds user, authdb *sync.Mutex, respath string, dbp []byte) error {
	dbpath := respath + "creds"
	creds.Token = ""

	for k, _ := range creds.SiteTokens {
		creds.SiteTokens[k] = ""
	}

	authdb.Lock()
	err := writeCreds(creds, dbpath, dbp)

	if err != nil {
		return err
	}

	authdb.Unlock()
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

func writeCreds(creds user, dbpath string, pwd []byte) error {
	db, err := decryptdb(dbpath, pwd)

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

			if creds.School == ln[1] && creds.Username == ln[2] {
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
