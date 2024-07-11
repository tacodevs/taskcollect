package gclass

import (
	"main/site"
)

func ListRes(creds User, r chan []site.Resource, e chan []error) {
	r <- nil
	e <- nil
	return
}
