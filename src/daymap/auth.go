package daymap

import (
	"errors"
	"html"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

var ErrAuthFailed = errors.New("daymap: authentication failed")

type User struct {
	Timezone *time.Location
	Token string
}

func get(weburl, username, password string) (string, string, error) {
	// Stage 1 - Get a DayMap redirect to SAML.

	// A persistent cookie jar is required for the entire process.

	jar, err := cookiejar.New(nil)

	if err != nil {
		return "", "", err
	}

	client := &http.Client{Jar: jar}

	s1, err := client.Get(weburl)

	if err != nil {
		return "", "", err
	}

	s1body, err := ioutil.ReadAll(s1.Body)

	if err != nil {
		return "", "", err
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
		err = errors.New("daymap: Could not find client request ID.")
		return "", "", err
	}

	idEnd := strings.Index(s1page[idIndex:], `"`)
	idEnd += idIndex

	if idEnd == -1 {
		err = errors.New("daymap: Client request ID has no end.")
		return "", "", err
	}

	s2id := s1page[idIndex:idEnd]
	s2url := s1.Request.URL.String() + s2id

	// Send the POST request with the generated form data.

	s2req, err := http.NewRequest("POST", s2url, s2data)

	if err != nil {
		return "", "", err
	}

	s2req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s2, err := client.Do(s2req)

	if err != nil {
		return "", "", err
	}

	s2body, err := ioutil.ReadAll(s2.Body)

	if err != nil {
		return "", "", err
	}

	s2page := string(s2body)

	// Stage 3 - Parse gigantic SAML form and redirect to OpenIdConnect.

	// Parse the SAML HTML form.

	s3form := url.Values{}
	s3index := strings.Index(s2page, "action=")

	if s3index == -1 {
		err := errors.New(`daymap: Canot find "action=" in SAML form.`)
		return "", "", err
	}

	s3search := s2page[s3index:]

	for {
		s3index = strings.Index(s3search, `name="`)

		if s3index == -1 {
			break
		}

		s3search = s3search[s3index+6:]
		s3index = strings.Index(s3search, `"`)

		if s3index == -1 {
			err := errors.New("daymap: Invalid HTML form.")
			return "", "", err
		}

		key := s3search[:s3index]
		s3index = strings.Index(s3search, `value="`)

		if s3index == -1 {
			err := ErrAuthFailed
			return "", "", err
		}

		s3search = s3search[s3index+7:]
		s3index = strings.Index(s3search, `"`)

		if s3index == -1 {
			err := errors.New("daymap: Invalid HTML form.")
			return "", "", err
		}

		value := s3search[:s3index]
		s3form.Add(key, value)
	}

	s3wr := s3form.Get("wresult")
	s3wr = html.UnescapeString(s3wr)
	s3form.Set("wresult", s3wr)
	s3data := strings.NewReader(s3form.Encode())

	// Send the POST request with the payload.

	s3url := "https://portal.daymap.net/daymapidentity/adfs/gihs/"
	s3req, err := http.NewRequest("POST", s3url, s3data)

	if err != nil {
		return "", "", err
	}

	s3req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s3, err := client.Do(s3req)

	if err != nil {
		return "", "", err
	}

	s3body, err := ioutil.ReadAll(s3.Body)

	if err != nil {
		return "", "", err
	}

	s3page := string(s3body)

	// Stage 4 - Parse OpenIdConnect form and POST to "/DayMap".

	// Parse the OpenIdConnect HTML form.

	s4form := url.Values{}
	s4index := strings.Index(s3page, "action=")

	if s4index == -1 {
		err := errors.New(`daymap: Canot find "action=" in SAML form.`)
		return "", "", err
	}

	s4search := s3page[s4index:]

	for {
		s4index = strings.Index(s4search, `name='`)

		if s4index == -1 {
			break
		}

		s4search = s4search[s4index+6:]
		s4index = strings.Index(s4search, `'`)

		if s4index == -1 {
			err := errors.New("daymap: Invalid HTML form.")
			return "", "", err
		}

		key := s4search[:s4index]
		s4index = strings.Index(s4search, `value='`)

		if s4index == -1 {
			err := errors.New("daymap: Invalid HTML form.")
			return "", "", err
		}

		s4search = s4search[s4index+7:]
		s4index = strings.Index(s4search, `'`)

		if s4index == -1 {
			err := errors.New("daymap: Invalid HTML form.")
			return "", "", err
		}

		value := s4search[:s4index]
		s4form.Set(key, value)
	}

	s4data := strings.NewReader(s4form.Encode())

	// Send the POST request with the payload.

	s4url := "https://gihs.daymap.net/Daymap/"
	s4req, err := http.NewRequest("POST", s4url, s4data)

	if err != nil {
		return "", "", err
	}

	s4req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s4, err := client.Do(s4req)

	if err != nil {
		return "", "", err
	}

	s4body, err := ioutil.ReadAll(s4.Body)

	if err != nil {
		return "", "", err
	}

	s4page := string(s4body)

	daymapUrl := url.URL {
		Scheme: "https",
		Host: "gihs.daymap.net",
	}

	cookies := jar.Cookies(&daymapUrl)
	authtok := ""

	for i := 0; i < len(cookies); i++ {
		authtok += cookies[i].String()

		if i < len(cookies)-1 {
			authtok += "; "
		}
	}

	return s4page, authtok, nil
}

func Auth(school, usr, pwd string) (User, error) {
	timezone, err := time.LoadLocation("Australia/Adelaide")

	if err != nil {
		panic(err)
	}

	page := "https://gihs.daymap.net/daymap/student/dayplan.aspx"
	_, authtok, err := get(page, usr, pwd)

	if err != nil {
		return User{}, err
	}

	creds := User{
		Timezone: timezone,
		Token: authtok,
	}

	return creds, nil
}
