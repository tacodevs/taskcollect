package server

import (
	"context"
	"encoding/base64"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"git.sr.ht/~kvo/libgo/defs"
	"git.sr.ht/~kvo/libgo/errors"
	"github.com/redis/go-redis/v9"

	"main/logger"
	"main/plat"
	"main/plat/daymap"
)

// Attempt to get GIHS Daily Access home page using a username and password.
// Used for authenticating GIHS students.
func gihsAuth(username, password string) errors.Error {
	// Stage 1 - Get a Daily Access redirect to SAML.

	// A persistent cookie jar is required for the entire process.

	jar, e := cookiejar.New(nil)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return errors.New("error creating cookiejar", err)
	}

	client := &http.Client{Jar: jar}

	s1, e := client.Get("https://da.gihs.sa.edu.au")
	if e != nil {
		err := errors.New(e.Error(), nil)
		return errors.New("GET request failed", err)
	}

	s1body, e := io.ReadAll(s1.Body)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return errors.New("error reading s1.Body", err)
	}

	s1page := string(s1body)

	// Stage 2 - POST credentials to SAML.

	// Generate POST form data with provided credentials.

	s2form := url.Values{}
	s2form.Set("UserName", username)
	s2form.Set("Password", password)
	s2form.Set("AuthMethod", "FormsAuthentication")
	s2data := strings.NewReader(s2form.Encode())

	// Get SAML request ID. This must be extracted to make a valid login.

	idIndex := strings.Index(s1page, "&client-request-id=")
	if idIndex == -1 {
		err := errors.New("missing client request ID", nil)
		return err
	}

	idEnd := strings.Index(s1page[idIndex:], `"`)
	idEnd += idIndex

	if idEnd == -1 {
		err := errors.New("unterminated client request ID", nil)
		return err
	}

	s2id := s1page[idIndex:idEnd]
	s2url := s1.Request.URL.String() + s2id

	// Send the POST request with the generated form data.

	s2req, e := http.NewRequest("POST", s2url, s2data)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return errors.New("malformed stage 2 POST", err)
	}

	s2req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s2, e := client.Do(s2req)
	if e != nil {
		err := errors.New(e.Error(), nil)
		return errors.New("stage 2 POST failed", err)
	}

	// Stage 3 - Check if authentication was successful.

	if s2.StatusCode == 200 && s2.Header.Get("X-Frame-Options") == "" {
		return nil
	}
	return errors.New("saml returned non-200 response", nil)
}

type authDB struct {
	path   string
	client *redis.Client
}

// Initializes the database and returns the created instance.
func initDB(addr string, pwd string, idx int) *redis.Client {
	// TODO: check that idx is between 0-15 inclusive
	redisDB := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pwd,
		DB:       idx,
	})

	ctx := context.Background()
	res := redisDB.Ping(ctx)
	if res.Err() != nil {
		err := errors.New(res.Err().Error(), nil)
		logger.Fatal(errors.New("incorrect password", err))
	}

	return redisDB
}

// Attempt to get pre-existing user credentials.
func (db *authDB) getCreds(cookies string) (plat.User, errors.Error) {
	creds := plat.User{}
	var token string

	start := strings.Index(cookies, "token=")
	if start == -1 {
		return plat.User{}, plat.ErrInvalidAuth.Here()
	}

	start += 6
	end := strings.Index(cookies[start:], ";")

	if end == -1 {
		token = cookies[start:]
	} else {
		token = cookies[start:end]
	}

	ctx := context.Background()

	userToken := "studentToken:" + token
	tokenExists := db.client.Exists(ctx, userToken)
	if tokenExists.Err() != nil {
		err := errors.New(tokenExists.Err().Error(), nil)
		err = errors.New("failed to get student data via token", err)
		return creds, err
	}
	exists, e := tokenExists.Result()
	if e != nil {
		return creds, plat.ErrInvalidAuth.Here()
	}
	if exists != 1 {
		return creds, plat.ErrInvalidAuth.Here()
	}

	studentID := db.client.HGetAll(ctx, userToken)
	if studentID.Err() != nil {
		//err := errors.New(studentID.Err().Error(), nil)
		//err = errors.New("a token does not exist for current user", err)
		//return creds, err
		return creds, plat.ErrInvalidAuth.Here()
	}
	res, e := studentID.Result()
	if e != nil {
		err := errors.New(e.Error(), nil)
		logger.Debug(err)
		return creds, plat.ErrInvalidAuth.Here()
	}

	key := "school:" + res["school"] + ":studentID:" + res["studentID"]
	studentData := db.client.HGetAll(ctx, key)
	if studentData.Err() != nil {
		err := errors.New(studentData.Err().Error(), nil)
		err = errors.New("failed to get student data", err)
		return creds, err
	}
	result, e := studentData.Result()
	if e != nil {
		err := errors.New(e.Error(), nil)
		return creds, errors.New("resulting data could not be read", err)
	}

	creds.Token = token // equivalent to result["token"]
	creds.School = result["school"]
	creds.Username = result["username"]
	creds.Password = result["password"]

	if creds.School == "gihs" {
		creds.DispName = strings.TrimPrefix(creds.Username, `CURRIC\`)
	} else {
		creds.DispName = creds.Username
	}

	if creds.School == "gihs" {
		creds.Timezone, e = time.LoadLocation("Australia/Adelaide")
		if e != nil {
			err := errors.New(e.Error(), nil)
			return plat.User{}, errors.New("cannot load timezone data", err)
		}

		creds.SiteTokens = map[string]string{
			"daymap": result["daymap"],
			"gclass": result["gclass"],
		}
	} else {
		return plat.User{}, errors.New("invalid school", plat.ErrInvalidAuth.Here())
	}

	return creds, nil
}

// Check if user exists in the database.
func (db *authDB) findUser(school, user, pwd string) (bool, errors.Error) {
	exists := false
	ctx := context.Background()

	key := "school:" + school + ":studentID:" + user
	data := db.client.HGetAll(ctx, key)
	if data.Err() != nil {
		err := errors.New(data.Err().Error(), nil)
		err = errors.New("could not fetch data for user", err)
		return exists, err
	}

	result, e := data.Result()
	if e != nil {
		err := errors.New(e.Error(), nil)
		return exists, errors.New("resulting data could not be read", err)
	}

	if result["password"] == pwd {
		exists = true
	}

	return exists, nil
}

// Create new user or update pre-existing user in the database.
func (db *authDB) writeCreds(creds plat.User) errors.Error {
	ctx := context.Background()

	studentIDKey := "school:" + creds.School + ":studentID:" + creds.Username
	platGihs := []string{"daymap", "gclass"}
	hashMap := map[string]string{
		"token":    creds.Token,
		"school":   creds.School,
		"username": creds.Username,
		"password": creds.Password,
	}
	for _, plat := range platGihs {
		hashMap[plat] = creds.SiteTokens[plat]
	}

	db.client.HSet(ctx, studentIDKey, hashMap) // returns # of fields added, not req at the moment

	// Add to list of students
	student := "school:" + creds.School + ":studentList"
	db.client.SAdd(ctx, student, creds.Username)

	// Add to token list
	key := "studentToken:" + creds.Token
	var duration time.Duration

	duration = time.Until(time.Now().AddDate(0, 0, 3))

	info := map[string]string{
		"studentID": creds.Username,
		"school":    creds.School,
	}

	db.client.HSet(ctx, key, info)
	db.client.Expire(ctx, key, duration)

	return nil
}

// Authenticate a user to TaskCollect.
func (db *authDB) auth(query url.Values) (string, errors.Error) {
	school := query.Get("school")

	// NOTE: Options for other schools could be added in the future
	if school != "gihs" {
		err := errors.New("school was not GIHS", plat.ErrAuthFailed.Here())
		return "", err
	}

	user := query.Get("usr")
	pwd := query.Get("pwd")

	if defs.Has([]string{user, pwd}, "") {
		err := errors.New("username or password is empty", plat.ErrAuthFailed.Here())
		return "", err
	}

	if school == "gihs" && !strings.HasPrefix(strings.ToUpper(user), `CURRIC\`) {
		user = `CURRIC\` + user
	} else if school == "gihs" {
		user = strings.ToUpper(user)
	}

	var dmCreds daymap.User
	err := gihsAuth(user, pwd)
	if err != nil {
		logger.Debug(errors.New("gihs auth failed", err))
		dmCreds, err = daymap.Auth(school, user, pwd)
		if err != nil {
			logger.Debug(errors.New("daymap auth failed", err))
			userExists, err := db.findUser(school, user, pwd)
			if err != nil {
				return "", errors.New("cannot check if user exists", err)
			}
			if !userExists {
				return "", errors.New("user not found", err)
			}
		}
	}

	if (dmCreds == daymap.User{}) {
		dmCreds, err = daymap.Auth(school, user, pwd)
		if err != nil {
			logger.Debug(errors.New("could not authenticate to Daymap", err))
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
	cookie += time.Now().UTC().AddDate(0, 0, 3).Format(time.RFC1123)
	timezone := dmCreds.Timezone

	creds := plat.User{
		Timezone:   timezone,
		School:     school,
		Username:   user,
		Password:   pwd,
		Token:      token,
		SiteTokens: siteTokens,
	}

	err = db.writeCreds(creds)
	if err != nil {
		return "", errors.New("error writing creds", err)
	}

	return cookie, nil
}

// Logout a user from TaskCollect.
func (db *authDB) logout(creds plat.User) errors.Error {
	token := creds.Token

	// Clear tokens
	creds.Token = ""
	for k := range creds.SiteTokens {
		creds.SiteTokens[k] = ""
	}

	err := db.writeCreds(creds)
	if err != nil {
		return errors.New("could not write to database", err)
	}

	// NOTE: The student token needs to be deleted from the token list on logout
	// NOTE: This MUST be done after writeCreds() since that creates the studentToken entry
	ctx := context.Background()
	key := "studentToken:" + token
	db.client.Del(ctx, key)

	return nil
}
