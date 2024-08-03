package myadelaide

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~kvo/go-std/errors"

	"main/hotp"
	"main/site"
)

// Auxiliary structures for the fetch function.

type s5json struct {
	Nonce string
}

type s5struct struct {
	Identifier  string `json:"identifier"`
	StateHandle string `json:"stateHandle"`
}

type s6json struct {
	Credentials s6creds `json:"credentials"`
	StateHandle string  `json:"stateHandle"`
}

type s6creds struct {
	Passcode string `json:"passcode"`
}

type s8json struct {
	Success s8success `json:"success"`
}

type s8success struct {
	Href string `json:"href"`
}

type s10json struct {
	Token string `json:"access_token"`
}

// Auxiliary functions for the fetch function.

func mkcode(key string) (string, error) {
	b, err := base32.StdEncoding.DecodeString(strings.ToUpper(key))
	if err != nil {
		return "", err
	}
	digits := 6
	code := hotp.New(b, uint64(time.Now().UnixNano())/30e9, digits)
	return fmt.Sprintf("%0*d", digits, code), nil
}

func randstr(charset []byte, size int) (string, error) {
	buf := make([]byte, size)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}
	for i := range buf {
		randnum := int(buf[i]) % len(charset)
		buf[i] = charset[randnum]
	}
	return string(buf), nil
}

func randhex(size int) (string, error) {
	buf := make([]byte, size)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(buf)[:size], nil
}

// The next two functions are an attempt to appease MyAdelaide by providing
// values that look like values it expects.
//
// Okta's requirements for the S256 code challenge method can be found here:
// https://developer.okta.com/docs/guides/implement-grant-type/authcodepkce/main

func mkverifier() (string, error) {
	charset := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_")
	return randstr(charset, 64)
}

func mkchallenge(verifier string) (string, error) {
	h := sha256.New()
	h.Write([]byte(verifier))
	hash := h.Sum(nil)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash), nil
}

// fetch retrieves a webpage (specified by link) from MyAdelaide, using the
// given username and password. Returns the desired webpage as a string, as well
// as the cookies provided with the webpage, and an error (if any occurs), in
// that order.
//
// fetch's primary purpose is to authenticate to MyAdelaide. The cookies
// returned by fetch can be used as a web session token for further retrieval of
// resources.
//
// fetch is vulnerable to obsoletion due to changes in the MyAdelaide interface.
// More importantly fetch should NOT be run more frequently than once in 300s or
// errors may be encountered.
func fetch(link, username, password, key string) (string, string, error) {
	// Stage 1 - Request redirect info from MyAdelaide.

	// A persistent cookie jar is required for the entire process.
	// Do NOT forget to keep pretending to be one of the latest versions of Firefox!
	// TODO: Fetch a random valid user agent from a curated list.
	browser := "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/127.0"

	myadelaideUrl := url.URL{
		Scheme: "https",
		Host:   "myadelaide.uni.adelaide.edu.au",
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", "", errors.New("cannot create stage 1 cookie jar", err)
	}

	client := &http.Client{Jar: jar}

	_, err = client.Get(link)
	if err != nil {
		return "", "", errors.New("stage 1 request failed", err)
	}

	// Stage 2 - Manually self-redirect to Okta.

	s2charset := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	s2verifier, err := mkverifier()
	if err != nil {
		return "", "", errors.New("cannot make stage 2 verifier token", nil)
	}
	s2challenge, err := mkchallenge(s2verifier)
	if err != nil {
		return "", "", errors.New("cannot make stage 2 code_challenge token", nil)
	}
	s2nonce, err := randstr(s2charset, 64)
	if err != nil {
		return "", "", errors.New("cannot make stage 2 nonce token", nil)
	}
	s2state, err := randstr(s2charset, 64)
	if err != nil {
		return "", "", errors.New("cannot make stage 2 state token", nil)
	}

	s2url := "https://id.adelaide.edu.au/oauth2/default/v1/authorize?"
	s2url += "client_id=0oaiku3xxvUYEFpAR3l6&"
	s2url += fmt.Sprintf("code_challenge=%s&", s2challenge)
	s2url += "code_challenge_method=S256&"
	s2url += fmt.Sprintf("nonce=%s&", s2nonce)
	s2url += "redirect_uri=https%3A%2F%2Fmyadelaide.uni.adelaide.edu.au&"
	s2url += "response_mode=fragment&"
	s2url += "response_type=code&"
	s2url += fmt.Sprintf("state=%s&", s2state)
	s2url += "scope=openid+email+profile"

	s2req, err := http.NewRequest("GET", s2url, nil)
	if err != nil {
		return "", "", errors.New("cannot create stage 2 request", err)
	}

	s2req.Header.Set("Referer", "https://myadelaide.uni.adelaide.edu.au/")
	s2req.Header.Set("User-Agent", browser)

	s2, err := client.Do(s2req)
	if err != nil {
		return "", "", errors.New("cannot execute stage 2 request", err)
	}

	s2body, err := io.ReadAll(s2.Body)
	if err != nil {
		return "", "", errors.New("cannot read stage 2 response body", err)
	}

	s2page := string(s2body)

	// Stage 3 - POST to Okta introspect.

	s3rstate := regexp.MustCompile(`"stateToken":"[_0-9\\\-\.a-zA-Z]+","helpLinks"`)
	s3state := s3rstate.FindString(s2page)
	// BUG: replace the following with safe slicing operation in case len(x) = 0
	s3state, err = strconv.Unquote(fmt.Sprintf(`"%s"`, s3state[14:len(s3state)-13]))
	if err != nil {
		return "", "", errors.New("cannot unquote stage 3 state token", err)
	}

	s3url := "https://id.adelaide.edu.au/idp/idx/introspect"
	s3tmpl := `{"stateToken":%s}`
	s3jstate, err := json.Marshal(s3state)
	if err != nil {
		return "", "", errors.New("cannot marshal stage 3 state token as JSON", err)
	}
	s3data := strings.NewReader(fmt.Sprintf(s3tmpl, string(s3jstate)))

	s3req, err := http.NewRequest("POST", s3url, s3data)
	if err != nil {
		return "", "", errors.New("cannot create stage 3 request", err)
	}

	s3req.Header.Set("Accept", `application/ion+json; okta-version=1.0.0`)
	s3req.Header.Set("Content-Type", `application/ion+json; okta-version=1.0.0`)
	s3req.Header.Set("Origin", "https://id.adelaide.edu.au")
	s3req.Header.Set("User-Agent", browser)
	s3req.Header.Set("X-Okta-User-Agent-Extended", "okta-auth-js/7.7.0 okta-signin-widget-7.20.1")

	_, err = client.Do(s3req)
	if err != nil {
		return "", "", errors.New("cannot execute stage 3 request", err)
	}

	// Stage 4 - POST to Okta nonce.

	s4url := "https://id.adelaide.edu.au/api/v1/internal/device/nonce"
	s4req, err := http.NewRequest("POST", s4url, nil)
	if err != nil {
		return "", "", errors.New("cannot create stage 4 request", err)
	}

	s4req.Header.Set("Accept", `*/*`)
	s4req.Header.Set("Content-Type", `application/json`)
	s4req.Header.Set("Origin", `https://id.adelaide.edu.au`)
	s4req.Header.Set("Referer", `https://id.adelaide.edu.au/auth/services/devicefingerprint`)
	s4req.Header.Set("User-Agent", browser)
	s4req.Header.Set("X-Requested-With", `XMLHttpRequest`)

	s4, err := client.Do(s4req)
	if err != nil {
		return "", "", errors.New("cannot execute stage 4 request", err)
	}

	s4body, err := io.ReadAll(s4.Body)
	if err != nil {
		return "", "", errors.New("cannot read stage 4 response body", err)
	}

	// Stage 5 - POST to Okta identify.

	s5nonce := s5json{}
	err = json.Unmarshal(s4body, &s5nonce)
	if err != nil {
		return "", "", errors.New("cannot unmarshal stage 5 nonce", err)
	}
	s5finger1, err := randhex(64)
	if err != nil {
		return "", "", errors.New("cannot make counterfeit fingerprint field 1", err)
	}
	s5finger2, err := randhex(32)
	if err != nil {
		return "", "", errors.New("cannot make counterfeit fingerprint field 2", err)
	}
	s5finger := fmt.Sprintf("%s|%s|%s", s5nonce.Nonce, s5finger1, s5finger2)

	s5form := s5struct{username, s3state}
	s5data, err := json.Marshal(s5form)
	if err != nil {
		return "", "", errors.New("cannot marshal stage 5 form", err)
	}

	s5url := "https://id.adelaide.edu.au/idp/idx/identify"
	s5req, err := http.NewRequest("POST", s5url, bytes.NewReader(s5data))
	if err != nil {
		return "", "", errors.New("cannot create stage 5 request", err)
	}

	s5req.Header.Set("Accept", `application/json; okta-version=1.0.0`)
	s5req.Header.Set("Content-Type", `application/json`)
	s5req.Header.Set("Origin", `https://id.adelaide.edu.au`)
	s5req.Header.Set("Referer", `https://id.adelaide.edu.au/auth/services/devicefingerprint`)
	s5req.Header.Set("User-Agent", browser)
	s5req.Header.Set("X-Device-Fingerprint", s5finger)
	s5req.Header.Set("X-Okta-User-Agent-Extended", `okta-auth-js/7.7.0 okta-signin-widget-7.20.1`)

	s5, err := client.Do(s5req)
	if err != nil {
		return "", "", errors.New("cannot execute stage 5 request", err)
	}

	s5body, err := io.ReadAll(s5.Body)
	if err != nil {
		return "", "", errors.New("cannot read stage 5 response body", err)
	}

	// Stage 6 - POST to Okta answer.

	s6state := s5struct{}
	err = json.Unmarshal(s5body, &s6state)
	if err != nil {
		return "", "", errors.New("cannot unmarshal stage 6 state token", err)
	}
	s6form := s6json{s6creds{password}, s6state.StateHandle}
	s6data, err := json.Marshal(s6form)
	if err != nil {
		return "", "", errors.New("cannot marshal stage 6 form", err)
	}

	s6url := "https://id.adelaide.edu.au/idp/idx/challenge/answer"
	s6req, err := http.NewRequest("POST", s6url, bytes.NewReader(s6data))
	if err != nil {
		return "", "", errors.New("cannot create stage 6 request", err)
	}

	s6req.Header.Set("Accept", `application/json; okta-version=1.0.0`)
	s6req.Header.Set("Content-Type", `application/json`)
	s6req.Header.Set("Origin", `https://id.adelaide.edu.au`)
	s6req.Header.Set("User-Agent", browser)
	s6req.Header.Set("X-Device-Fingerprint", s5finger)
	s6req.Header.Set("X-Okta-User-Agent-Extended", `okta-auth-js/7.7.0 okta-signin-widget-7.20.1`)

	_, err = client.Do(s6req)
	if err != nil {
		return "", "", errors.New("cannot execute stage 6 request", err)
	}

	// Stage 7 - POST to Okta answer (again).

	s7mfa, err := mkcode(key)
	if err != nil {
		return "", "", errors.New("cannot make 2fa code", err)
	}
	s7form := s6json{s6creds{s7mfa}, s6state.StateHandle}
	s7data, err := json.Marshal(s7form)
	if err != nil {
		return "", "", errors.New("cannot marshal stage 7 form", err)
	}

	s7url := "https://id.adelaide.edu.au/idp/idx/challenge/answer"
	s7req, err := http.NewRequest("POST", s7url, bytes.NewReader(s7data))
	if err != nil {
		return "", "", errors.New("cannot create stage 7 request", err)
	}

	s7req.Header.Set("Accept", `application/json; okta-version=1.0.0`)
	s7req.Header.Set("Content-Type", `application/json`)
	s7req.Header.Set("Origin", `https://id.adelaide.edu.au`)
	s7req.Header.Set("User-Agent", browser)
	s7req.Header.Set("X-Device-Fingerprint", s5finger)
	s7req.Header.Set("X-Okta-User-Agent-Extended", `okta-auth-js/7.7.0 okta-signin-widget-7.20.1`)

	s7, err := client.Do(s7req)
	if err != nil {
		return "", "", errors.New("cannot execute stage 7 request", err)
	}

	s7body, err := io.ReadAll(s7.Body)
	if err != nil {
		return "", "", errors.New("cannot read stage 7 response body", err)
	}

	// Stage 8 - Get redirect from Okta.

	s8cookies := make([]*http.Cookie, 0, 3)
	s8params, s8nonce, s8state := new(http.Cookie), new(http.Cookie), new(http.Cookie)
	s8params.Name = `okta-oauth-redirect-params`
	s8params.Value = `{%22responseType%22:%22code%22%2C%22state%22:%22`
	s8params.Value += s2state + `%22%2C%22nonce%22:%22` + s2nonce
	s8params.Value += `%22%2C%22scopes%22:[%22openid%22%2C%22email%22%2C%22profile%22]%2C%22clientId`
	s8params.Value += `%22:%220oaiku3xxvUYEFpAR3l6%22%2C%22urls%22:{%22issuer%22:%22`
	s8params.Value += `https://adelaide.okta.com/oauth2/default%22%2C%22authorizeUrl%22:%22`
	s8params.Value += `https://id.adelaide.edu.au/oauth2/default/v1/authorize%22%2C%22userinfoUrl%22`
	s8params.Value += `:%22https://id.adelaide.edu.au/oauth2/default/v1/userinfo%22%2C%22tokenUrl%22:`
	s8params.Value += `%22https://adelaide.okta.com/oauth2/default/v1/token%22%2C%22revokeUrl%22:%22`
	s8params.Value += `https://adelaide.okta.com/oauth2/default/v1/revoke%22%2C%22logoutUrl%22:%22`
	s8params.Value += `https://adelaide.okta.com/oauth2/default/v1/logout%22}%2C%22ignoreSignature%22:false}`
	s8nonce.Name, s8nonce.Value = `okta-oauth-nonce`, s2nonce
	s8state.Name, s8state.Value = `okta-oauth-state`, s2state
	s8cookies = append(s8cookies, s8params, s8nonce, s8state)
	jar.SetCookies(&myadelaideUrl, s8cookies)

	noredirect := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar: jar,
	}

	s8url := s8json{}
	err = json.Unmarshal(s7body, &s8url)
	if err != nil {
		return "", "", errors.New("cannot unmarshal stage 8 url", err)
	}

	s8req, err := http.NewRequest("GET", s8url.Success.Href, nil)
	if err != nil {
		return "", "", errors.New("cannot create stage 8 request", err)
	}

	s8req.Header.Set("User-Agent", browser)

	s8, err := noredirect.Do(s8req)
	if err != nil {
		return "", "", errors.New("cannot execute stage 8 request", err)
	}
	s8loc := s8.Header.Get("location")

	s8req, err = http.NewRequest("GET", link, nil)
	if err != nil {
		return "", "", errors.New("cannot create redirected stage 8 request", err)
	}

	s8req.Header.Set("User-Agent", browser)

	_, err = client.Do(s8req)
	if err != nil {
		return "", "", errors.New("cannot execute redirected stage 8 request", err)
	}

	// Stage 9 - Request token options from Adelaide Okta.

	s9url := "https://adelaide.okta.com/oauth2/default/v1/token"
	s9req, err := http.NewRequest("OPTIONS", s9url, nil)
	if err != nil {
		return "", "", errors.New("cannot create stage 9 request", err)
	}

	s9req.Header.Set("Accept", `*/*`)
	s9req.Header.Set("Access-Control-Request-Headers", `x-okta-user-agent-extended`)
	s9req.Header.Set("Access-Control-Request-Method", `POST`)
	s9req.Header.Set("Origin", `https://myadelaide.uni.adelaide.edu.au`)
	s9req.Header.Set("Referer", `https://myadelaide.uni.adelaide.edu.au/`)
	s9req.Header.Set("User-Agent", browser)

	_, err = client.Do(s9req)
	if err != nil {
		return "", "", errors.New("cannot execute stage 9 request", err)
	}

	// Stage 10 - POST to Adelaide Okta token.

	s10rloc := regexp.MustCompile("#code=[-A-Za-z0-9]+&state=")
	s10loc := s10rloc.FindString(s8loc)
	// BUG: replace the following with safe slicing operation in case len(x) = 0
	s10loc = s10loc[6 : len(s10loc)-7]

	s10form := url.Values{}
	s10form.Add("client_id", "0oaiku3xxvUYEFpAR3l6")
	s10form.Add("redirect_uri", link)
	s10form.Add("grant_type", "authorization_code")
	s10form.Add("code_verifier", s2verifier)
	s10form.Add("code", s10loc)
	s10data := strings.NewReader(s10form.Encode())

	s10req, err := http.NewRequest("POST", s9url, s10data)
	if err != nil {
		return "", "", errors.New("cannot create stage 10 request", err)
	}

	s10req.Header.Set("Accept", `application/json`)
	s10req.Header.Set("Content-Type", `application/x-www-form-urlencoded`)
	s10req.Header.Set("Origin", `https://myadelaide.uni.adelaide.edu.au`)
	s10req.Header.Set("Referer", `https://myadelaide.uni.adelaide.edu.au/`)
	s10req.Header.Set("User-Agent", browser)
	s10req.Header.Set("X-Okta-User-Agent-Extended", `@okta/okta-vue/3.1.0 okta-auth-js/4.9.2`)

	s10, err := client.Do(s10req)
	if err != nil {
		return "", "", errors.New("cannot execute stage 10 request", err)
	}

	s10body, err := io.ReadAll(s10.Body)
	if err != nil {
		return "", "", errors.New("cannot read stage 10 response body", err)
	}

	s10page := string(s10body)

	// Return the bearer token for future requests to MyAdelaide.

	s10bearer := s10json{}
	err = json.Unmarshal(s10body, &s10bearer)
	if err != nil {
		return "", "", errors.New("cannot unmarshal stage 10 bearer token", err)
	}

	return s10page, s10bearer.Token, nil
}

func Auth(user site.User, c chan site.Pair[[2]string, error]) {
	var result site.Pair[[2]string, error]
	cfg, ok := user.Config["myadelaide"]
	if !ok {
		result.Second = errors.New("no user settings for myadelaide", nil)
		c <- result
		return
	}
	link := "https://myadelaide.uni.adelaide.edu.au"
	_, token, err := fetch(link, user.Username, user.Password, cfg.HotpKey)
	if err != nil {
		result.Second = errors.New("myadelaide login failed", err)
		c <- result
		return
	}
	result.First = [2]string{"myadelaide", token}
	c <- result
}
