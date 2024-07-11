package example

import (
	"crypto/rand"
	"encoding/base64"

	"git.sr.ht/~kvo/go-std/errors"

	"main/plat"
)

// Example fetch function
//
// fetch retrieves a webpage (specified by link) from the platform, using the
// given email, username, and password. The caller is always the Auth function.
// fetch returns the webpage as a string, then the final cookie saved from the
// fetch process, and an error, if any occurs. The final cookie is used as the
// session token for the platform in all future requests.
//
// fetch may be written by using browser devtools to reverse engineer the
// process behind retrieving a certain webpage, which means reverse engineering
// the auth process as well.
func fetch(link, email, username, password string) (string, string, error) {
	page := ""
	buf := make([]byte, 32)
	_, err := rand.Read(buf)
	if err != nil {
		return page, "", errors.New("rand reader", err)
	}
	token := base64.StdEncoding.EncodeToString(buf)
	return page, token, nil
}

// Example auth function
func Auth(user plat.User, c chan plat.Pair[[2]string, error]) {
	var result plat.Pair[[2]string, error]

	link := "https://example.com"
	_, token, err := fetch(link, user.Email, user.Username, user.Password)
	if err != nil {
		// result.Second is the error returned by Auth
		result.Second = errors.New("example login failed", err)
		c <- result
		return
	}

	// result.First is an array of two strings
	// first string is the platform codename ("example")
	// second string is the session token for the platform
	result.First = [2]string{"example", token}
	c <- result
}
