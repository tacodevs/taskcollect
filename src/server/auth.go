package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	fp "path/filepath"
	"strings"
	"time"

	"codeberg.org/kvo/std"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"

	"main/daymap"
	"main/errors"
	"main/gclass"
	"main/logger"
	"main/plat"
)

// Global variable which holds an in-memory copy of the Google OAuth client ID
// file.
//
// This is a kluge. Google Classroom auth must be made to function in the same
// way as authentication for all other supported platforms.
var GAuthID []byte

// Attempt to get GIHS Daily Access home page using a username and password.
// Used for authenticating GIHS students.
func gihsAuth(username, password string) error {
	// Stage 1 - Get a Daily Access redirect to SAML.

	// A persistent cookie jar is required for the entire process.

	jar, err := cookiejar.New(nil)
	if err != nil {
		return errors.NewError("daymap.get", "failed to create cookiejar", err)
	}

	client := &http.Client{Jar: jar}

	s1, err := client.Get("https://da.gihs.sa.edu.au")
	if err != nil {
		return errors.NewError("daymap.get", "GET request failed", err)
	}

	s1body, err := io.ReadAll(s1.Body)
	if err != nil {
		return errors.NewError("daymap.get", "failed to read s1.Body", err)
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
		err = errors.NewError("server.gihsAuth", "could not find client request ID", nil)
		return err
	}

	idEnd := strings.Index(s1page[idIndex:], `"`)
	idEnd += idIndex

	if idEnd == -1 {
		err = errors.NewError("server.gihsAuth", "client request ID has no end", nil)
		return err
	}

	s2id := s1page[idIndex:idEnd]
	s2url := s1.Request.URL.String() + s2id

	// Send the POST request with the generated form data.

	s2req, err := http.NewRequest("POST", s2url, s2data)
	if err != nil {
		return errors.NewError("daymap.get", "(s2) POST request failed", err)
	}

	s2req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s2, err := client.Do(s2req)
	if err != nil {
		return errors.NewError("daymap.get", "s2req", err)
	}

	// Stage 3 - Check if authentication was successful.

	if s2.StatusCode == 200 && s2.Header.Get("X-Frame-Options") == "" {
		return nil
	}
	return errors.NewError("server.gihsAuth", "error authenticating to GIHS SAML", err)
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
		logger.Fatal(errors.NewError("redis", "incorrect password", res.Err()))
	}

	return redisDB
}

// Attempt to get pre-existing user credentials.
func (db *authDB) getCreds(cookies string) (plat.User, error) {
	creds := plat.User{}
	var token string

	start := strings.Index(cookies, "token=")
	if start == -1 {
		return plat.User{}, errInvalidAuth
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
		err := errors.NewError("server.getCreds", "failed to get student data via token", tokenExists.Err())
		return creds, err
	}
	exists, err := tokenExists.Result()
	if err != nil {
		return creds, errInvalidAuth
	}
	if exists != 1 {
		return creds, errInvalidAuth
	}

	studentID := db.client.HGetAll(ctx, userToken)
	if studentID.Err() != nil {
		//err := errors.NewError("server.getCreds", "a token does not exist for current user", studentID.Err())
		//return creds, err
		return creds, errInvalidAuth
	}
	res, err := studentID.Result()
	if err != nil {
		logger.Debug("studentID result ERROR: %v", studentID.Err().Error())
		return creds, errInvalidAuth
	}

	key := "school:" + res["school"] + ":studentID:" + res["studentID"]
	studentData := db.client.HGetAll(ctx, key)
	if studentData.Err() != nil {
		err := errors.NewError("server.getCreds", "failed to get student data", studentData.Err())
		return creds, err
	}
	result, err := studentData.Result()
	if err != nil {
		return creds, errors.NewError("server.getCreds", "resulting data could not be read", err)
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
		creds.Timezone, err = time.LoadLocation("Australia/Adelaide")
		if err != nil {
			return plat.User{}, errors.NewError("server.getCreds", "could not load timezone location data", err)
		}

		creds.SiteTokens = map[string]string{
			"daymap": result["daymap"],
			"gclass": result["gclass"],
		}
	} else {
		return plat.User{}, errors.NewError("server.getCreds", "invalid school", errInvalidAuth)
	}

	return creds, nil
}

// Get Google auth token. If a "gclass" token field is found, even if it is blank, it will be returned
// as an empty string.
func (db *authDB) getGTok(school, user, pwd string) (string, error) {
	ctx := context.Background()

	key := "school:" + school + ":studentID:" + user
	data := db.client.HGetAll(ctx, key)
	if data.Err() != nil {
		err := errors.NewError("server.getGTok", "could not fetch password for user", data.Err())
		return "", err
	}

	result, err := data.Result()
	if err != nil {
		return "", errors.NewError("server.getGTok", "resulting data could not be read", err)
	}

	res := db.client.HExists(ctx, key, "gclass")
	exists, err := res.Result()
	if err != nil {
		return "", errors.NewError("server.getGTok", "could not fetch user's gclass data", err)
	}
	if !exists {
		return "", nil
	}

	if result["password"] == pwd {
		return result["gclass"], nil
	}

	return "", nil
}

// Check if user exists in the database.
func (db *authDB) findUser(school, user, pwd string) (bool, error) {
	exists := false
	ctx := context.Background()

	key := "school:" + school + ":studentID:" + user
	data := db.client.HGetAll(ctx, key)
	if data.Err() != nil {
		err := errors.NewError("server.findUser", "could not fetch data for user", data.Err())
		return exists, err
	}

	result, err := data.Result()
	if err != nil {
		return exists, errors.NewError("server.findUser", "resulting data could not be read", err)
	}

	if result["password"] == pwd {
		exists = true
	}

	return exists, nil
}

// Create new user or update pre-existing user in the database.
func (db *authDB) writeCreds(creds plat.User) error {
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
func (db *authDB) auth(query url.Values) (string, error) {
	school := query.Get("school")

	// NOTE: Options for other schools could be added in the future
	if school != "gihs" {
		err := errors.NewError("server.auth", "school was not GIHS", errAuthFailed)
		return "", err
	}

	user := query.Get("usr")
	pwd := query.Get("pwd")

	if std.Contains([]string{user, pwd}, "") {
		err := errors.NewError("server.auth", "username or password is empty", errAuthFailed)
		return "", err
	}

	if school == "gihs" && !strings.HasPrefix(strings.ToUpper(user), `CURRIC\`) {
		user = `CURRIC\` + user
	} else if school == "gihs" {
		user = strings.ToUpper(user)
	}

	gTok, err := db.getGTok(school, user, pwd)
	if err != nil {
		return "", errors.NewError("server.auth", "gclass token was not found", err)
	}

	gTestErr := make(chan error)

	if gTok != "" {
		go gclass.Test(GAuthID, gTok, gTestErr)
	}

	if gihsAuth(user, pwd) != nil {
		userExists, err := db.findUser(school, user, pwd)
		if err != nil {
			return "", errors.NewError("server.auth", "could not determine if user exists", err)
		}
		if !userExists {
			return "", errors.NewError("server.auth", "user was not found", errAuthFailed)
		}
	}

	dmCreds, err := daymap.Auth(school, user, pwd)
	if err != nil {
		logger.Warn(errors.NewError("server.auth", "could not authenticate to Daymap", err))
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

	gAuthStatus := errNeedsGAuth.AsError()

	if gTok != "" {
		err = <-gTestErr
		if err == nil {
			siteTokens["gclass"] = gTok
			gAuthStatus = nil
		}
	}

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
		return "", errors.NewError("server.auth", "error writing creds", err)
	}

	return cookie, gAuthStatus
}

// Return the Google authentication endpoint URL.
func (db *authDB) gAuthEndpoint() (string, error) {
	gcid, err := os.ReadFile(fp.Join(db.path, "gauth.json"))
	if err != nil {
		return "", errors.NewError("server.gAuthEndpoint", "failed to read gauth.json", err)
	}

	gAuthConfig, err := gclass.AuthConfig(gcid)
	if err != nil {
		return "", errors.NewError("server.gAuthEndpoint", "creation of config failed", err)
	}

	gAuthLoc := gAuthConfig.AuthCodeURL(
		"state-token",
		oauth2.ApprovalForce,
		oauth2.AccessTypeOffline,
	)

	return gAuthLoc, nil
}

// Run the Google authentication flow for a user.
func (db *authDB) runGAuth(creds plat.User, query url.Values) error {
	authCode := query.Get("code")

	clientId, err := os.ReadFile(fp.Join(db.path, "gauth.json"))
	if err != nil {
		return errors.NewError("server.runGAuth", errors.ErrFileRead.Error(), err)
	}

	gAuthConfig, err := gclass.AuthConfig(clientId)
	if err != nil {
		return errors.NewError("server.runGAuth", "failed to get config from JSON", err)
	}

	gTok, err := gAuthConfig.Exchange(context.TODO(), authCode)
	if err != nil {
		return errors.NewError("server.runGAuth", "failed to convert auth code into token", err)
	}

	token, err := json.Marshal(gTok)
	if err != nil {
		return errors.NewError("server.runGAuth", "failed to encode into JSON", err)
	}

	creds.SiteTokens["gclass"] = string(token)
	err = db.writeCreds(creds)
	if err != nil {
		return errors.NewError("server.runGAuth", "failed to write creds", err)
	}

	return nil
}

// Logout a user from TaskCollect.
func (db *authDB) logout(creds plat.User) error {
	token := creds.Token

	// Clear tokens
	creds.Token = ""
	for k := range creds.SiteTokens {
		creds.SiteTokens[k] = ""
	}

	err := db.writeCreds(creds)
	if err != nil {
		return errors.NewError("server.logout", "could not write to database", err)
	}

	// NOTE: The student token needs to be deleted from the token list on logout
	// NOTE: This MUST be done after writeCreds() since that creates the studentToken entry
	ctx := context.Background()
	key := "studentToken:" + token
	db.client.Del(ctx, key)

	return nil
}
