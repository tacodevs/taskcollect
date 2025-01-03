package saml

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"git.sr.ht/~kvo/go-std/errors"

	"main/site"
)

// Attempt to get GIHS Daily Access home page using a username and password.
// Used for authenticating GIHS students.
func fetch(username, password string) error {
	// Stage 1 - Get a Daily Access redirect to SAML.

	// A persistent cookie jar is required for the entire process.

	jar, err := cookiejar.New(nil)
	if err != nil {
		return errors.New(err, "cannot create cookiejar")
	}

	client := &http.Client{Jar: jar}

	s1, err := client.Get("https://da.gihs.sa.edu.au")
	if err != nil {
		return errors.New(err, "stage 1 request failed")
	}

	s1body, err := io.ReadAll(s1.Body)
	if err != nil {
		return errors.New(err, "cannot read stage 1 body")
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
		err := errors.New(nil, "missing client request ID")
		return err
	}

	idEnd := strings.Index(s1page[idIndex:], `"`)
	idEnd += idIndex

	if idEnd == -1 {
		err := errors.New(nil, "unterminated client request ID")
		return err
	}

	s2id := s1page[idIndex:idEnd]
	s2url := s1.Request.URL.String() + s2id

	// Send the POST request with the generated form data.

	s2req, err := http.NewRequest("POST", s2url, s2data)
	if err != nil {
		return errors.New(err, "cannot create stage 2 request")
	}

	s2req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	s2, err := client.Do(s2req)
	if err != nil {
		return errors.New(err, "cannot execute stage 2 request")
	}

	// Check if authentication was successful.

	if s2.StatusCode == 200 && s2.Header.Get("X-Frame-Options") == "" {
		return nil
	}
	return errors.New(nil, "saml returned non-200 response")
}

func Auth(user site.User, c chan site.Pair[[2]string, error]) {
	var result site.Pair[[2]string, error]
	err := fetch(user.Username, user.Password)
	if err != nil {
		result.Second = errors.New(err, "saml login failed")
		c <- result
		return
	}
	result.First = [2]string{"saml", ""}
	c <- result
}
