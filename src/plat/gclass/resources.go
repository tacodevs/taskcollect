package gclass

import (
	"main/plat"
)

func ListRes(creds User, r chan []plat.Resource, e chan []error) {
	r <- nil
	e <- nil
	return
}
