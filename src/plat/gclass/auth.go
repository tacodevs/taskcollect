package gclass

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"codeberg.org/kvo/std/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/classroom/v1"
	"google.golang.org/api/option"
)

type User struct {
	ClientID []byte
	Timezone *time.Location
	Token    string
}

// Generate a Google OAuth 2.0 configuration from the provided Google client ID.
func AuthConfig(clientID []byte) (*oauth2.Config, errors.Error) {
	authConfig, err := google.ConfigFromJSON(
		clientID,
		classroom.ClassroomCoursesReadonlyScope,
		classroom.ClassroomStudentSubmissionsMeReadonlyScope,
		classroom.ClassroomCourseworkMeScope,
		classroom.ClassroomCourseworkmaterialsReadonlyScope,
		classroom.ClassroomAnnouncementsReadonlyScope,
	)

	if err != nil {
		return authConfig, errors.New(err.Error(), nil)
	}
	return authConfig, nil
}

// Authenticate to Google Classroom and return an API connection.
func Auth(creds User) (*classroom.Service, errors.Error) {
	ctx := context.Background()
	authConfig, err := AuthConfig(creds.ClientID)

	if err != nil {
		return nil, errors.New("failed to get config from JSON", err)
	}

	r := strings.NewReader(creds.Token)
	oauthTok := &oauth2.Token{}

	e := json.NewDecoder(r).Decode(oauthTok)
	if e != nil {
		return nil, errors.New(
			"failed to decode JSON",
			errors.New(err.Error(), nil),
		)
	}

	client := authConfig.Client(context.Background(), oauthTok)

	svc, e := classroom.NewService(
		ctx,
		option.WithHTTPClient(client),
	)
	if e != nil {
		err = errors.New(e.Error(), nil)
		return nil, errors.New("failed to create new service", err)
	}

	return svc, nil
}

// Test if the provided Google credentials are valid.
func Test(gcid []byte, gTok string, e chan errors.Error) {
	svc, err := Auth(User{ClientID: gcid, Token: gTok})
	if err != nil {
		e <- errors.New("Google auth failed", err)
		return
	}

	_, er := svc.Courses.List().PageSize(1).Do()
	if er != nil {
		err = errors.New(er.Error(), nil)
		e <- errors.New("failed to get response", err)
		return
	}

	e <- nil
}
