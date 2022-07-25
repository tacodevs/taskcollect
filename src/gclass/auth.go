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
)

type User struct {
	Timezone *time.Location
	Token    string
}

func Test(gcid []byte, gTok string, e chan error) {
	ctx := context.Background()

	gauthConfig, err := google.ConfigFromJSON(
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

	client := gauthConfig.Client(context.Background(), oauthTok)

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
}
