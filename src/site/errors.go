package site

import "git.sr.ht/~kvo/go-std/errors"

var (
	ErrInitFailed = errors.New(nil, "initialization failed")
	ErrNotFound   = errors.New(nil, "cannot find resource")
)
