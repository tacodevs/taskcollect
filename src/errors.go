package main

import "errors"

var (
	errAuthFailed      = errors.New("main: authentication failed")
	errCorruptMIME     = errors.New("main: corrupt MIME request")
	errIncompleteCreds = errors.New("main: user has incomplete credentials")
	errInvalidAuth     = errors.New("main: invalid session token")
	errNoPlatform      = errors.New("main: unsupported platform")
	errNotFound        = errors.New("main: cannot find resource")
	errNeedsGAuth      = errors.New("main: Google auth required")
)
