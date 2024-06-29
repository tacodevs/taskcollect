package gclass

import (
	"main/plat"
)

// Retrieve a list of graded tasks from Google Classroom for a user.
func Graded(creds plat.User, c chan plat.Pair[[]plat.Task, error], done *int) {
	defer plat.Mark(done, c)
	var result plat.Pair[[]plat.Task, error]
	c <- result
}
