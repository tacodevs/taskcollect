package daymap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"

	"git.sr.ht/~kvo/go-std/errors"

	"main/site"
)

// Auxiliary structures for the fetch function.

type s7struct struct {
	Status       string
	SessionToken string
}

// fetch retrieves the webpage at URL link from Daymap using username and
// password. Returns the desired webpage as a string, as well as the cookies
// provided with the webpage, and an error (if any occurs), in that order.
//
// fetch's primary purpose is to authenticate to Daymap. The cookies returned by
// fetch can be used as a web session token for further retrieval of resources.
//
// fetch is vulnerable to obsoletion as the authentication mechanism for Daymap
// frequently changes.
func fetch(link, username, password string) (string, string, error) {
	// Stage 1 - Get a Daymap redirect to EdPass.

	// A persistent cookie jar is required for the entire process.

	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", "", errors.New(err, "cannot create stage 1 cookie jar")
	}

	client := &http.Client{Jar: jar}

	s1, err := client.Get(link)
	if err != nil {
		return "", "", errors.New(err, "stage 1 request failed")
	}

	s1body, err := io.ReadAll(s1.Body)
	if err != nil {
		return "", "", errors.New(err, "cannot read stage 1 body")
	}

	s1page := string(s1body)

	// Stage 2 - Send POST request at school selection page.

	// Extract EdPass redirect URL.

	s2index := strings.Index(s1page, `"redirectUri":"`)
	if s2index == -1 {
		return "", "", errors.New(nil, "cannot find stage 2 redirect URL")
	}
	s1page = s1page[s2index+15:]

	s2index = strings.Index(s1page, `"`)
	if s2index == -1 {
		return "", "", errors.New(nil, "stage 2 redirect URL has no end")
	}
	s2redirect, err := strconv.Unquote(fmt.Sprintf(`"%s"`, s1page[:s2index]))
	if err != nil {
		return "", "", errors.New(err, "cannot unquote stage 2 redirect URL")
	}

	// Extract Okta key from EdPass redirect URL.

	s2index = strings.Index(s2redirect, "?okta_key=")
	if s2index == -1 {
		return "", "", errors.New(nil, "cannot find Okta key in stage 2 redirect URL")
	}
	s2okta := s2redirect[s2index+10:]

	// Generate Okta relay URL.

	s2relay := "/oauth2/v1/authorize/redirect?okta_key=" + s2okta

	// Extract EdPass state token.

	s2index = strings.Index(s1page, `"stateToken":"`)
	if s2index == -1 {
		return "", "", errors.New(nil, "cannot find stage 2 state token")
	}
	s1page = s1page[s2index+14:]

	s2index = strings.Index(s1page, `"`)
	if s2index == -1 {
		return "", "", errors.New(nil, "stage 2 state token has no end")
	}
	s2token, err := strconv.Unquote(fmt.Sprintf(`"%s"`, s1page[:s2index]))
	if err != nil {
		return "", "", errors.New(err, "cannot unquote stage 2 state token")
	}

	// Bake new cookies for stage 2.

	s2dom, err := url.Parse("https://portal.edpass.sa.edu.au/")
	if err != nil {
		return "", "", errors.New(err, "cannot parsing stage 2 target domain")
	}

	s2cookieRd := http.Cookie{Name: "redirecturi", Value: s2redirect}

	s2cookieRl := http.Cookie{
		Name:  "relaystate",
		Value: s2relay,
	}

	jar.SetCookies(
		s2dom, []*http.Cookie{&s2cookieRd, &s2cookieRl},
	)

	// Send the POST request with the state token.

	s2jstok, err := json.Marshal(s2token)
	if err != nil {
		return "", "", errors.New(err, "cannot marshal stage 2 token as JSON")
	}
	s2data := bytes.NewReader([]byte(
		fmt.Sprintf(`{"stateToken":%s}`, string(s2jstok)),
	))

	s2req, err := http.NewRequest(
		"POST",
		"https://portal.edpass.sa.edu.au/api/v1/authn/introspect",
		s2data,
	)
	if err != nil {
		return "", "", errors.New(err, "cannot create stage 2 request")
	}

	s2req.Header.Set("Accept", "application/json")
	s2req.Header.Set("Content-Type", "application/json")
	s2req.Header.Set("Origin", "https://portal.edpass.sa.edu.au")
	s2req.Header.Set(
		"Referer",
		"https://portal.edpass.sa.edu.au/signin/refresh-auth-state/"+s2token,
	)
	s2req.Header.Set(
		"X-Okta-User-Agent-Extended",
		"okta-auth-js/5.8.0 okta-signin-widget-5.16.1",
	)

	_, err = client.Do(s2req)
	if err != nil {
		return "", "", errors.New(err, "cannot execute stage 2 request")
	}

	// Stage 3 - Send POST request to HRD EdPass IDPDiscovery.

	s3req, err := http.NewRequest(
		"POST", "https://hrd.edpass.sa.edu.au/api/IDPDiscovery", nil,
	)
	if err != nil {
		return "", "", errors.New(err, "cannot create stage 3 request")
	}

	s3req.Header.Set("Origin", "https://portal.edpass.sa.edu.au")
	s3req.Header.Set("Referer", "https://portal.edpass.sa.edu.au/")

	_, err = client.Do(s3req)
	if err != nil {
		return "", "", errors.New(err, "cannot execute stage 3 request")
	}

	// Stage 4 - Get SAML details from EdPass.

	s4form := url.Values{}
	s4form.Add("fromURI", s2relay)
	s4url := "https://portal.edpass.sa.edu.au/sso/saml2/0oamc0sv2IbQE6VD33l6/?" + s4form.Encode()

	s4req, err := http.NewRequest("GET", s4url, nil)
	if err != nil {
		return "", "", errors.New(err, "cannot create stage 4 request")
	}

	s4req.Header.Set("Origin", "https://portal.edpass.sa.edu.au")
	s4req.Header.Set("Referer", "https://portal.edpass.sa.edu.au/")

	s4, err := client.Do(s4req)
	if err != nil {
		return "", "", errors.New(err, "cannot execute stage 4 request")
	}

	s4body, err := io.ReadAll(s4.Body)
	if err != nil {
		return "", "", errors.New(err, "cannot read stage 4 body")
	}

	s4page := string(s4body)

	// Stage 5 - Initial SAML transaction via EdPass.

	// Parse the HTML form from the previous HTTP response.

	s5form := url.Values{}

	s5index := strings.Index(s4page, `action="`)
	if s5index == -1 {
		return "", "", errors.New(nil, "cannot find 'action' form attribute for stage 5")
	}
	s5search := s4page[s5index+8:]
	s5url := s5search

	s5index = strings.Index(s5url, `"`)
	if s5index == -1 {
		return "", "", errors.New(nil, "'action' attribute has no end in stage 5")
	}
	s5url = html.UnescapeString(s5url[:s5index])

	for {
		s5index = strings.Index(s5search, `name="`)
		if s5index == -1 {
			break
		}

		s5search = s5search[s5index+6:]
		s5index = strings.Index(s5search, `"`)
		if s5index == -1 {
			return "", "", errors.New(nil, "'name' attribute has no end in stage 5 form")
		}
		key := s5search[:s5index]

		s5index = strings.Index(s5search, `value="`)
		if s5index == -1 {
			return "", "", errors.New(nil, "no value matches key in stage 5 form")
		}

		s5search = s5search[s5index+7:]
		s5index = strings.Index(s5search, `"`)
		if s5index == -1 {
			return "", "", errors.New(nil, "'value' attribute has no end in stage 5 form")
		}
		value := s5search[:s5index]

		s5form.Add(key, html.UnescapeString(value))
	}

	s5data := strings.NewReader(s5form.Encode())

	// Send the POST request with the payload.

	s5req, err := http.NewRequest("POST", s5url, s5data)
	if err != nil {
		return "", "", errors.New(err, "cannot create stage 5 request")
	}

	s5req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s5req.Header.Set("Origin", "https://portal.edpass.sa.edu.au")
	s5req.Header.Set("Referer", "https://portal.edpass.sa.edu.au/")

	_, err = client.Do(s5req)
	if err != nil {
		return "", "", errors.New(err, "cannot execute stage 5 request")
	}

	// Stage 6 - Request a nonce from EdPass.

	s6url := "https://edpass-0927.okta.com/api/v1/internal/device/nonce"
	s6req, err := http.NewRequest("POST", s6url, nil)
	if err != nil {
		return "", "", errors.New(err, "cannot create stage 6 request")
	}

	s6req.Header.Set("Origin", "https://edpass-0927.okta.com/api/v1/internal/device/nonce")
	s6req.Header.Set("Referer", "https://edpass-0927.okta.com/auth/services/devicefingerprint")

	_, err = client.Do(s6req)
	if err != nil {
		return "", "", errors.New(err, "cannot execute stage 6 request")
	}

	// Stage 7 - Authenticate to EdPass.

	s7tmpl := `{"password":%s,"username":%s,"options":{"warnBeforePasswordExpired":true,"multiOptionalFactorEnroll":true}}`
	s7usr, err := json.Marshal(strings.TrimPrefix(username, `CURRIC\`))
	if err != nil {
		return "", "", errors.New(err, "cannot marshal username as JSON")
	}
	s7pwd, err := json.Marshal(password)
	if err != nil {
		return "", "", errors.New(err, "cannot marshal password as JSON")
	}
	s7data := strings.NewReader(
		fmt.Sprintf(s7tmpl, string(s7pwd), string(s7usr)),
	)

	s7url := "https://edpass-0927.okta.com/api/v1/authn"
	s7req, err := http.NewRequest("POST", s7url, s7data)
	if err != nil {
		return "", "", errors.New(err, "cannot create stage 7 request")
	}

	s7req.Header.Set("Accept", "application/json")
	s7req.Header.Set("Content-Type", "application/json")
	s7req.Header.Set("Origin", "https://edpass-0927.okta.com")
	s7req.Header.Set("Referer", s5url)

	s7, err := client.Do(s7req)
	if err != nil {
		return "", "", errors.New(err, "cannot execute stage 7 request")
	}

	s7body, err := io.ReadAll(s7.Body)
	if err != nil {
		return "", "", errors.New(err, "cannot read stage 7 body")
	}

	s7json := s7struct{}
	err = json.Unmarshal(s7body, &s7json)

	// Stage 8 - Request session cookie redirect from EdPass.

	s8rdform := s5form
	s8rdform.Add("OKTA_INVALID_SESSION_REPOST", "true")
	s8redirect := strings.TrimPrefix(s5url, "https://edpass-0927.okta.com")
	s8redirect += "?" + s8rdform.Encode()

	s8form := url.Values{}
	s8form.Add("checkAccountSetupComplete", "true")
	s8form.Add("repost", "true")
	s8form.Add("token", s7json.SessionToken)
	s8form.Add("redirectUrl", s8redirect)
	s8data := strings.NewReader(s8form.Encode())

	s8url := "https://edpass-0927.okta.com/login/sessionCookieRedirect"
	s8req, err := http.NewRequest("POST", s8url, s8data)
	if err != nil {
		return "", "", errors.New(err, "cannot create stage 8 request")
	}

	s8req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s8req.Header.Set("Origin", "https://edpass-0927.okta.com")
	s8req.Header.Set("Referer", s5url)

	s8, err := client.Do(s8req)
	if err != nil {
		return "", "", errors.New(err, "cannot execute stage 8 request")
	}

	s8body, err := io.ReadAll(s8.Body)
	if err != nil {
		return "", "", errors.New(err, "cannot read stage 8 body")
	}

	s8page := string(s8body)

	// Stage 9 - Make final POST request to EdPass.

	// Parse the HTML form from the previous HTTP response.

	s9form := url.Values{}

	s9index := strings.Index(s8page, `action="`)
	if s9index == -1 {
		return "", "", errors.New(nil, "cannot find 'action' form attribute for stage 9")
	}
	s9search := s8page[s9index+8:]
	s9url := s9search

	s9index = strings.Index(s9url, `"`)
	if s9index == -1 {
		return "", "", errors.New(nil, "'action' attribute has no end in stage 9")
	}
	s9url = html.UnescapeString(s9url[:s9index])

	for {
		s9index = strings.Index(s9search, `name="`)
		if s9index == -1 {
			break
		}

		s9search = s9search[s9index+6:]
		s9index = strings.Index(s9search, `"`)
		if s9index == -1 {
			return "", "", errors.New(nil, "'name' attribute has no end in stage 9 form")
		}
		key := s9search[:s9index]

		s9index = strings.Index(s9search, `value="`)
		if s9index == -1 {
			return "", "", errors.New(nil, "no value matches key in stage 9 form")
		}

		s9search = s9search[s9index+7:]
		s9index = strings.Index(s9search, `"`)
		if s9index == -1 {
			return "", "", errors.New(nil, "'value' attribute has no end in stage 9 form")
		}
		value := s9search[:s9index]

		s9form.Add(key, html.UnescapeString(value))
	}

	s9data := strings.NewReader(s9form.Encode())

	// Send the POST request with the payload.

	s9req, err := http.NewRequest("POST", s9url, s9data)
	if err != nil {
		return "", "", errors.New(err, "cannot create stage 9 request")
	}

	s9req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s9req.Header.Set("Origin", "https://edpass-0927.okta.com")
	s9req.Header.Set("Referer", "https://edpass-0927.okta.com/")

	s9, err := client.Do(s9req)
	if err != nil {
		return "", "", errors.New(err, "cannot execute stage 9 request")
	}

	s9body, err := io.ReadAll(s9.Body)
	if err != nil {
		return "", "", errors.New(err, "cannot read stage 9 body")
	}

	s9page := string(s9body)

	// Stage 10 - Make POST request to Daymap for session identification.

	// Parse the HTML form from the previous HTTP response.

	s10form := url.Values{}

	s10index := strings.Index(s9page, `action="`)
	if s10index == -1 {
		return "", "", errors.New(nil, "cannot find 'action' form attribute for stage 10")
	}
	s10search := s9page[s10index+8:]
	s10url := s10search

	s10index = strings.Index(s10url, `"`)
	if s10index == -1 {
		return "", "", errors.New(nil, "'action' attribute has no end in stage 10")
	}
	s10url = html.UnescapeString(s10url[:s10index])

	for {
		s10index = strings.Index(s10search, `name="`)
		if s10index == -1 {
			break
		}

		s10search = s10search[s10index+6:]
		s10index = strings.Index(s10search, `"`)
		if s10index == -1 {
			return "", "", errors.New(nil, "'name' attribute has no end in stage 10 form")
		}
		key := s10search[:s10index]

		s10index = strings.Index(s10search, `value="`)
		if s10index == -1 {
			return "", "", errors.New(nil, "no value matches key in stage 10 form")
		}

		s10search = s10search[s10index+7:]
		s10index = strings.Index(s10search, `"`)
		if s10index == -1 {
			return "", "", errors.New(nil, "'value' attribute has no end in stage 10 form")
		}
		value := s10search[:s10index]

		s10form.Add(key, html.UnescapeString(value))
	}

	s10data := strings.NewReader(s10form.Encode())

	// Send the POST request with the payload.

	s10req, err := http.NewRequest("POST", s10url, s10data)
	if err != nil {
		return "", "", errors.New(err, "cannot create stage 10 request")
	}

	s10req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	s10, err := client.Do(s10req)
	if err != nil {
		return "", "", errors.New(err, "cannot execute stage 10 request")
	}

	s10body, err := io.ReadAll(s10.Body)
	if err != nil {
		return "", "", errors.New(err, "cannot read stage 10 body")
	}

	s10page := string(s10body)

	// Stage 11 - Parse OpenIdConnect form and POST to "/Daymap".

	// Parse the OpenIdConnect HTML form.

	s11form := url.Values{}
	s11index := strings.Index(s10page, "action=")
	if s11index == -1 {
		return "", "", errors.New(nil, "cannot find 'action' form attribute for stage 11")
	}
	s11search := s10page[s11index:]

	for {
		s11index = strings.Index(s11search, `name='`)
		if s11index == -1 {
			break
		}

		s11search = s11search[s11index+6:]
		s11index = strings.Index(s11search, `'`)
		if s11index == -1 {
			return "", "", errors.New(nil, "'name' attribute has no end in stage 11 form")
		}
		key := s11search[:s11index]

		s11index = strings.Index(s11search, `value='`)
		if s11index == -1 {
			return "", "", errors.New(nil, "no value matches key in stage 11 form")
		}

		s11search = s11search[s11index+7:]
		s11index = strings.Index(s11search, `'`)
		if s11index == -1 {
			return "", "", errors.New(nil, "'value' attribute has no end in stage 11 form")
		}
		value := s11search[:s11index]

		s11form.Set(key, value)
	}

	s11data := strings.NewReader(s11form.Encode())

	// Send the POST request with the payload.

	s11url := "https://gihs.daymap.net/Daymap/"
	s11req, err := http.NewRequest("POST", s11url, s11data)
	if err != nil {
		return "", "", errors.New(err, "cannot create stage 11 request")
	}

	s11req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	s11, err := client.Do(s11req)
	if err != nil {
		return "", "", errors.New(err, "cannot execute stage 11 request")
	}

	s11body, err := io.ReadAll(s11.Body)
	if err != nil {
		return "", "", errors.New(err, "cannot read stage 11 body")
	}

	s11page := string(s11body)

	// Retrieve all cookies associated with Daymap from cookie jar.

	daymapUrl := url.URL{
		Scheme: "https",
		Host:   "gihs.daymap.net",
	}

	cookies := jar.Cookies(&daymapUrl)
	authToken := ""

	for i, cookie := range cookies {
		authToken += cookie.String()
		if i < len(cookies)-1 {
			authToken += "; "
		}
	}

	return s11page, authToken, nil
}

func Auth(user site.User, c chan site.Pair[[2]string, error]) {
	var result site.Pair[[2]string, error]
	link := "https://gihs.daymap.net/daymap/student/dayplan.aspx"
	_, token, err := fetch(link, user.Username, user.Password)
	if err != nil {
		result.Second = errors.New(err, "daymap login failed")
		c <- result
		return
	}
	result.First = [2]string{"daymap", token}
	c <- result
}
