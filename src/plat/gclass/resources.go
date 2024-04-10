package gclass

import (
	"main/plat"
)

// Get a list of resources from Google Classroom for a user.
func ListRes(creds User, r chan []plat.Resource, e chan []error) {
	r <- nil
	e <- nil
	return
}
