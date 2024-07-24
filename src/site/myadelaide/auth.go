package example

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/http/cookiejar"

	"git.sr.ht/~kvo/go-std/errors"
)

// Auxiliary code for the fetch function.

// The following functions are translated from their JS source:
// https://myadelaide.uni.adelaide.edu.au/etc.clientlibs/uoa-myadelaide/clientlibs/clientlib-myadelaide/resources.min.ACSHASHf0ccad0f177a89ba7ac16055f5b88cf8.js

// mknonce is the Go equivalent of G(64) in the JS source.
func mknonce() string {
	e := 64
	t := []byte("abcdefghijklnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	n := make([]byte, e)
	r := 0
	for o := len(t); r < e; r++ {
		random := rand.Float64() * float64(o)
		floored := math.Floor(random)
		n = append(n, t[int(floored)])
	}
	return string(n)
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
func fetch(link, username, password string) (string, string, error) {
	// Stage 1 - Request redirect info from MyAdelaide.

	/*
		NOTE: There are X known stages to auth.
		(1) request redirect info from myadelaide
		(2) manually self-redirect to okta
		(x) post to okta introspect
		(x) post to okta nonce
		(x) post to okta identify
		(x) post to okta answer
		(x) post to okta answer (again)
		Result is / on myadelaide.uni.adelaide.edu.au
	*/

	// A persistent cookie jar is required for the entire process.

	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", "", errors.New("cannot create stage 1 cookie jar", err)
	}

	client := &http.Client{Jar: jar}

	s1, err := client.Get(link)
	if err != nil {
		return "", "", errors.New("stage 1 request failed", err)
	}

	s1body, err := io.ReadAll(s1.Body)
	if err != nil {
		return "", "", errors.New("cannot read stage 1 response body", err)
	}

	_ = string(s1body)

	// Stage 2 - Manually self-redirect to Okta.

	// TODO: find out how to generate each of these
	s2challenge := ""
	s2nonce := mknonce()
	s2state := ""

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

	// How does the browser already know the cookie for okta??
	s2req.Header.Set("Referer", "https://myadelaide.uni.adelaide.edu.au/")

	s2, err := client.Do(s2req)
	if err != nil {
		return "", "", errors.New("cannot execute stage 2 request", err)
	}

	s2body, err := io.ReadAll(s2.Body)
	if err != nil {
		return "", "", errors.New("cannot read stage 2 response body", err)
	}

	s2page := string(s2body)
	return s2page, "", nil
}
