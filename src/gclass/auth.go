package gclass

import (
	"context"
	"errors"
	"encoding/json"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/classroom/v1"
	"google.golang.org/api/option"
)

var errInvalidTaskID = errors.New("gclass: invalid task ID")

type User struct {
	ClientID []byte
	Timezone *time.Location
	Token    string
}

func Test(gcid []byte, gTok string, e chan error) {
	ctx := context.Background()

	gAuthConfig, err := google.ConfigFromJSON(
		gcid,
		classroom.ClassroomCoursesReadonlyScope,
		classroom.ClassroomStudentSubmissionsMeReadonlyScope,
		classroom.ClassroomCourseworkMeScope,
		classroom.ClassroomCourseworkmaterialsReadonlyScope,
	)

	if err != nil {
		e <- err
		return
	}

	r := strings.NewReader(gTok)
	oauthTok := &oauth2.Token{}

	err = json.NewDecoder(r).Decode(oauthTok)
	if err != nil {
		e <- err
		return
	}

	client := gAuthConfig.Client(context.Background(), oauthTok)

	svc, err := classroom.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		e <- err
		return
	}

	_, err = svc.Courses.List().PageSize(1).Do()
	if err != nil {
		e <- err
		return
	}

	e <- nil
}
