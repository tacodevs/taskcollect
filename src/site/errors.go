package site

import "git.sr.ht/~kvo/go-std/errors"

var (
	ErrInitFailed = errors.New("initialization failed", nil)
	ErrNotFound   = errors.New("cannot find resource", nil)
)
