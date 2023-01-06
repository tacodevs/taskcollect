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

var errInvalidTaskID = errors.NewError("gclass", "invalid task ID", nil)

type User struct {
	ClientID []byte
	Timezone *time.Location
	Token    string
}

// Test if the provided Google credentials are valid.
func Test(gcid []byte, gTok string, e chan error) {
	ctx := context.Background()

	gAuthConfig, err := google.ConfigFromJSON(
		gcid,
		classroom.ClassroomCoursesReadonlyScope,
		classroom.ClassroomStudentSubmissionsMeReadonlyScope,
		classroom.ClassroomCourseworkMeScope,
		classroom.ClassroomCourseworkmaterialsReadonlyScope,
		classroom.ClassroomAnnouncementsReadonlyScope,
	)

	if err != nil {
		newErr := errors.NewError("gclass: Test", "could not get JSON config", err)
		e <- newErr
		return
	}

	r := strings.NewReader(gTok)
	oauthTok := &oauth2.Token{}

	err = json.NewDecoder(r).Decode(oauthTok)
	if err != nil {
		newErr := errors.NewError("gclass: Test", "failed to decode JSON", err)
		e <- newErr
		return
	}

	client := gAuthConfig.Client(context.Background(), oauthTok)

	svc, err := classroom.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		newErr := errors.NewError("gclass: Test", "failed create new service", err)
		e <- newErr
		return
	}

	_, err = svc.Courses.List().PageSize(1).Do()
	if err != nil {
		newErr := errors.NewError("gclass: Test", "failed to get response", err)
		e <- newErr
		return
	}

	e <- nil
}
