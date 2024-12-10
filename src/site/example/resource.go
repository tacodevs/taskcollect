package example

import (
	"fmt"
	"main/site"

	"git.sr.ht/~kvo/go-std/errors"
)

func Resource(user site.User, id string) (site.Resource, error) {
	resources := map[string]site.Resource{
		core[0].Id:    core[0],
		core[1].Id:    core[1],
		french[0].Id:  french[0],
		french[1].Id:  french[1],
		physics[0].Id: physics[0],
		physics[1].Id: physics[1],
	}
	resource, exists := resources[id]
	if !exists {
		return resource, errors.New(fmt.Sprintf("no resource with ID %s exists", id), nil)
	}
	return resource, nil
}
