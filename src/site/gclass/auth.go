package gclass

import (
	"time"
)

type User struct {
	Timezone *time.Location
	Token    string
}
