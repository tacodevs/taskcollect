package classroom

import (
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
)

// fetch retrieves the webpage at URL link from Google Classroom using email,
// username, and password. Returns the desired webpage as a string, as well as
// the cookies provided with the webpage, and an error (if any occurs), in that
// order.
//
// fetch's primary purpose is to authenticate to Google Classroom. The cookies
// returned by fetch can be used as a web session token for further retrieval of
// resources.
//
// fetch is vulnerable to obsoletion due to frequent changes in the Google
// Classroom interface.
func fetch(link, email, user, password string) (string, string, error) {
	// Stage 1 - Knock on the door of accounts.google.com

	/*
		NOTE: There are 6 total stages to auth.
		(1) knock on the door of accounts.google.com
		(2) first post to google /batchexecute
		(3) second post to google /batchexecute
		(4) post to gihs saml
		(5) post to google /acs
		(6) press continue button
		Result is / on classroom.google.com
	*/

	// A persistent cookie jar is required for the entire process.
	// Do NOT forget to keep pretending to be one of the latest versions of Firefox!
	// TODO: Fetch a random valid user agent from a curated list.
	browser := "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/127.0"

	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", "", errors.New(err, "cannot create stage 1 cookie jar")
	}

	client := &http.Client{Jar: jar}

	s1link := "https://accounts.google.com/ServiceLogin?continue=%s&passive=true"
	s1link = fmt.Sprintf(s1link, url.QueryEscape(link))
	s1req, err := http.NewRequest("GET", s1link, nil)
	if err != nil {
		return "", "", errors.New(err, "cannot create stage 1 request")
	}

	s1req.Header.Set("User-Agent", browser)

	s1, err := client.Do(s1req)
	if err != nil {
		return "", "", errors.New(err, "cannot execute stage 1 request")
	}

	s1body, err := io.ReadAll(s1.Body)
	if err != nil {
		return "", "", errors.New(err, "cannot read stage 1 body")
	}

	s1page := string(s1body)

	// Stage 2 - First POST to Google's /batchexecute

	// Compile all needed regular expressions.
	s2dshr := regexp.MustCompile(`dsh=[_0-9-\.a-zA-Z:]*&amp`)
	s2flowr := regexp.MustCompile(`flowName=[_0-9-\.a-zA-Z]*&amp`)
	s2ifkvr := regexp.MustCompile(`ifkv=[_0-9-\.a-zA-Z]*&amp`)
	s2fsidr := regexp.MustCompile(`"FdrFJe":"[-0-9]*"`)
	s2blr := regexp.MustCompile(`"boq_identityfrontendauthuiserver[_0-9-\.a-zA-Z]*"`)
	s2atr := regexp.MustCompile(`null ,'[_0-9-\.a-zA-Z:]*','@font-face\\x7bfont-family:\\x27Noto Sans Myanmar UI`)

	// Search s1page for all needed regular expressions.
	s2dsh := s2dshr.FindString(s1page)
	s2flow := s2flowr.FindString(s1page)
	s2ifkv := s2ifkvr.FindString(s1page)
	s2fsid := s2fsidr.FindString(s1page)
	s2bl := s2blr.FindString(s1page)
	s2at := s2atr.FindString(s1page)

	// BUG: replace all the following with safe slicing operations in case len(x) = 0
	// Extract required strings from search results.
	s2dsh = s2dsh[4 : len(s2dsh)-4]
	s2flow = s2flow[9 : len(s2flow)-4]
	s2ifkv = s2ifkv[5 : len(s2ifkv)-4]
	s2fsid = s2fsid[10 : len(s2fsid)-1]
	s2bl = s2bl[1 : len(s2bl)-1]
	s2at = s2at[7 : len(s2at)-54]

	// Forge counterfeit request token.
	s2tm := time.Now()
	s2mb := 1 + s2tm.Hour()*3600 + s2tm.Minute()*60 + s2tm.Second()

	// Generate the stage 2 HTTP Google extension headers.
	s2gx2783 := fmt.Sprintf(`["%s"]`, s2flow)
	s2gx3915 := fmt.Sprintf(`["%s",null,null,"%s"]`, s2dsh, s2ifkv)

	// Generate the stage 2 POST body.
	s2form := url.Values{}
	s2form.Add("f.req", fmt.Sprintf(`[[["UEkKwb","[\"%s\"]",null,"generic"]]]`, s2dsh))
	s2sdata := s2form.Encode() + "&"
	s2form = url.Values{}
	s2form.Add("at", s2ifkv)
	s2sdata += s2form.Encode() + "&"
	s2data := strings.NewReader(s2sdata)

	// Generate the stage 2 URL query.
	s2query := url.Values{}
	s2query.Add("f.sid", s2fsid)
	s2qdata := s2query.Encode() + "&"
	s2query = url.Values{}
	s2query.Add("bl", s2bl)
	s2qdata += s2query.Encode() + "&hl=en-US&"
	s2query = url.Values{}
	s2query.Add("_reqid", strconv.Itoa(s2mb))
	s2qdata += s2query.Encode() + "&rt=c"

	// Generate stage 2 URL
	s2link := "https://accounts.google.com/v3/signin/_/AccountsSignInUi/data/batchexecute"
	s2link += "?rpcids=UEkKwb&source-path=%2Fv3%2Fsignin%2Fidentifier&"
	s2link += s2qdata

	s2req, err := http.NewRequest("POST", s2link, s2data)
	if err != nil {
		return "", "", errors.New(err, "cannot create stage 2 request")
	}

	s2req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	s2req.Header.Set("Alt-Used", "accounts.google.com")
	s2req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	s2req.Header.Set("Origin", "https://accounts.google.com")
	s2req.Header.Set("Referer", "https://accounts.google.com/")
	s2req.Header.Set("User-Agent", browser)
	s2req.Header.Set("x-goog-ext-278367001-jspb", s2gx2783)
	s2req.Header.Set("x-goog-ext-391502476-jspb", s2gx3915)
	s2req.Header.Set("X-Same-Domain", "1")

	s2, err := client.Do(s2req)
	if err != nil {
		return "", "", errors.New(err, "cannot execute stage 2 request")
	}

	s2body, err := io.ReadAll(s2.Body)
	if err != nil {
		return "", "", errors.New(err, "cannot read stage 2 body")
	}

	s2page := string(s2body)

	return s2page, "", nil
}
