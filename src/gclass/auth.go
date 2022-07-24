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
	Token string
}

func Test(gcid []byte, gtok string, e chan error) {
	ctx := context.Background()

	gauthcnf, err := google.ConfigFromJSON(
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

	r := strings.NewReader(gtok)
	oatok := &oauth2.Token{}
	err = json.NewDecoder(r).Decode(oatok)

	if err != nil {
		e <- err
		return
	}

	client := gauthcnf.Client(context.Background(), oatok)

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
