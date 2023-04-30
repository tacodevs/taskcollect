package gclass

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/classroom/v1"
	"google.golang.org/api/option"

	"main/errors"
)

type User struct {
	ClientID []byte
	Timezone *time.Location
	Token    string
}

// Generate a Google OAuth 2.0 configuration from the provided Google client ID.
func AuthConfig(clientID []byte) (*oauth2.Config, error) {
	authConfig, err := google.ConfigFromJSON(
		clientID,
		classroom.ClassroomCoursesReadonlyScope,
		classroom.ClassroomStudentSubmissionsMeReadonlyScope,
		classroom.ClassroomCourseworkMeScope,
		classroom.ClassroomCourseworkmaterialsReadonlyScope,
		classroom.ClassroomAnnouncementsReadonlyScope,
	)

	return authConfig, err
}

// Authenticate to Google Classroom and return an API connection.
func Auth(creds User) (*classroom.Service, error) {
	ctx := context.Background()
	authConfig, err := AuthConfig(creds.ClientID)

	if err != nil {
		return nil, errors.NewError("gclass.Auth", "failed to get config from JSON", err)
	}

	r := strings.NewReader(creds.Token)
	oauthTok := &oauth2.Token{}

	err = json.NewDecoder(r).Decode(oauthTok)
	if err != nil {
		return nil, errors.NewError("gclass.Auth", "failed to decode JSON", err)
	}

	client := authConfig.Client(context.Background(), oauthTok)

	svc, err := classroom.NewService(
		ctx,
		option.WithHTTPClient(client),
	)

	if err != nil {
		return nil, errors.NewError("gclass.Auth", "failed to create new service", err)
	}

	return svc, nil
}

// Test if the provided Google credentials are valid.
func Test(gcid []byte, gTok string, e chan error) {
	svc, err := Auth(User{ClientID: gcid, Token: gTok})
	if err != nil {
		e <- errors.NewError("gclass.Test", "Google auth failed", err)
		return
	}

	_, err = svc.Courses.List().PageSize(1).Do()
	if err != nil {
		e <- errors.NewError("gclass.Test", "failed to get response", err)
		return
	}

	e <- nil
}
