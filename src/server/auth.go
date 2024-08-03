package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"git.sr.ht/~kvo/go-std/defs"
	"git.sr.ht/~kvo/go-std/errors"

	"main/logger"
	"main/site"
)

type Creds struct {
	Tokens map[string]site.Uid
	Users  map[site.Uid]site.User
	Mutex  sync.Mutex
}

func extract(cookie string) (string, error) {
	// TODO: simplify token extraction
	var token string

	start := strings.Index(cookie, "token=")
	if start == -1 {
		return "", errors.New("no token in session cookie", nil)
	}

	start += 6
	end := strings.Index(cookie[start:], ";")

	if end == -1 {
		token = cookie[start:]
	} else {
		token = cookie[start:end]
	}

	return token, nil
}

func (creds *Creds) LookupToken(cookie string) (site.User, error) {
	token, err := extract(cookie)
	if err != nil {
		return site.User{}, errors.New("cannot lookup token", err)
	}
	user := creds.Users[creds.Tokens[token]]
	if user.Username == "" {
		errstr := "no user with matching token: " + token
		return site.User{}, errors.New(errstr, nil)
	}
	return user, nil
}

func (creds *Creds) LookupUid(school, username string) (site.User, error) {
	uid := site.Uid{school, username}
	user := creds.Users[uid]
	if user.Username == "" {
		errstr := fmt.Sprintf(
			`no user with matching uid: {"%s", "%s"}`,
			school, username,
		)
		return site.User{}, errors.New(errstr, nil)
	}
	return user, nil
}

func (creds *Creds) Update(token string, user site.User) {
	uid := site.Uid{user.School, user.Username}
	creds.Mutex.Lock()
	if token != "" {
		creds.Tokens[token] = uid
	}
	creds.Users[uid] = user
	creds.Mutex.Unlock()
}

func auth(school, email, username, password string) (site.User, error) {
	var err error

	if school == "gihs" && !strings.HasPrefix(strings.ToUpper(username), `CURRIC\`) {
		username = `CURRIC\` + username
	} else if school == "gihs" {
		username = strings.ToUpper(username)
	}

	if defs.Has([]string{username, password}, "") {
		return site.User{}, errors.New("username or password is empty", nil)
	}

	user := site.User{
		School:     school,
		Email:      email,
		DispName:   username,
		Username:   username,
		Password:   password,
		SiteTokens: make(map[string]string),
	}

	switch school {
	case "gihs":
		user.Timezone, err = time.LoadLocation("Australia/Adelaide")
		if err != nil {
			return site.User{}, errors.New("cannot load timezone", err)
		}
		user.DispName = strings.TrimPrefix(username, `CURRIC\`)
		err = schools["gihs"].Auth(&user)
		if err != nil {
			return site.User{}, errors.New("", err)
		}
	case "uofa":
		user.Timezone, err = time.LoadLocation("Australia/Adelaide")
		if err != nil {
			return site.User{}, errors.New("cannot load timezone", err)
		}
		err = schools["uofa"].Auth(&user)
		if err != nil {
			return site.User{}, errors.New("", err)
		}
	case "example":
		user.Timezone = time.UTC
		err = schools["example"].Auth(&user)
		if err != nil {
			return site.User{}, errors.New("", err)
		}
	default:
		return site.User{}, errors.New(fmt.Sprintf("unsupported school: %s", school), nil)
	}

	return user, nil
}

func (creds *Creds) Expire(token string, expiry time.Time) {
	time.Sleep(time.Until(expiry))
	creds.Mutex.Lock()
	delete(creds.Tokens, token)
	creds.Mutex.Unlock()
}

func (creds *Creds) Login(query url.Values) (string, error) {
	school := query.Get("school")
	email := query.Get("email")
	username := query.Get("user")
	password := query.Get("password")

	user, err := auth(school, email, username, password)
	if err != nil {
		logger.Debug(err)
		user, err = creds.LookupUid(school, username)
		if err != nil {
			return "", errors.New("login failed", err)
		}
	}

	buf := make([]byte, 32)
	_, err = rand.Read(buf)
	if err != nil {
		return "", errors.New("login failed", err)
	}

	token := base64.StdEncoding.EncodeToString(buf)
	expiry := time.Now().UTC().AddDate(0, 0, 3)
	cookie := fmt.Sprintf(
		"token=%s; Expires=%s",
		token, expiry.Format(time.RFC1123),
	)

	go creds.Expire(token, expiry)
	creds.Update(token, user)
	return cookie, nil
}

func (creds *Creds) Logout(cookie string) error {
	token, err := extract(cookie)
	if err != nil {
		return errors.New("cannot logout user", err)
	}
	creds.Mutex.Lock()
	delete(creds.Tokens, token)
	creds.Mutex.Unlock()
	return nil
}
