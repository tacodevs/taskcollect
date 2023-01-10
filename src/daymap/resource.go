package daymap

import (
	"time"
)

type Resource struct {
	Name     string
	Class    string
	Link     string
	Desc     string
	Posted   time.Time
	ResLinks [][2]string
	Platform string
	Id       string
}

// Get a resource from DayMap for a user.
func GetResource(creds User, id string) (Resource, error) {
	// TODO: Complete this function.
	return Resource{}, nil
}
