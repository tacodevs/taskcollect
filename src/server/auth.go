package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"git.sr.ht/~kvo/go-std/errors"
	"git.sr.ht/~kvo/go-std/slices"

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
		return "", errors.New(nil, "no token in session cookie")
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
		return site.User{}, errors.New(err, "cannot lookup token")
	}
	user := creds.Users[creds.Tokens[token]]
	if user.Username == "" {
		return site.User{}, errors.New(nil, "no user with matching token: %s", token)
	}
	return user, nil
}

func (creds *Creds) LookupUid(school, username string) (site.User, error) {
	uid := site.Uid{school, username}
	user := creds.Users[uid]
	if user.Username == "" {
		return site.User{}, errors.New(nil, `no user with matching uid: {"%s", "%s"}`, school, username)
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

	if slices.Has([]string{username, password}, "") {
		return site.User{}, errors.New(nil, "username or password is empty")
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
			return site.User{}, errors.New(err, "cannot load timezone")
		}
		user.DispName = strings.TrimPrefix(username, `CURRIC\`)
		err = schools["gihs"].Auth(&user)
		if err != nil {
			return site.User{}, errors.Wrap(err)
		}
	case "uofa":
		user.Timezone, err = time.LoadLocation("Australia/Adelaide")
		if err != nil {
			return site.User{}, errors.New(err, "cannot load timezone")
		}
		err = schools["uofa"].Auth(&user)
		if err != nil {
			return site.User{}, errors.Wrap(err)
		}
	case "example":
		user.Timezone = time.UTC
		err = schools["example"].Auth(&user)
		if err != nil {
			return site.User{}, errors.Wrap(err)
		}
	default:
		return site.User{}, errors.New(nil, "unsupported school: %s", school)
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
			return "", errors.New(err, "login failed")
		}
	}

	buf := make([]byte, 32)
	_, err = rand.Read(buf)
	if err != nil {
		return "", errors.New(err, "login failed")
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
		return errors.New(err, "cannot logout user")
	}
	creds.Mutex.Lock()
	delete(creds.Tokens, token)
	creds.Mutex.Unlock()
	return nil
}
