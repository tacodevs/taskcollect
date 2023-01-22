package server

import "main/errors"

var (
	errAuthFailed = errors.NewError("server", errors.ErrAuthFailed.Error(), nil)
	//errCorruptMIME = errors.NewError("server", errors.ErrCorruptMIME.Error(), nil)
	//errIncompleteCreds = errors.NewError("server", errors.ErrIncompleteCreds.Error(), nil)
	errInvalidAuth = errors.NewError("server", errors.ErrInvalidAuth.Error(), nil)
	errNoPlatform  = errors.NewError("server", errors.ErrNoPlatform.Error(), nil)
	errNotFound    = errors.NewError("server", errors.ErrNotFound.Error(), nil)
	errNeedsGAuth  = errors.NewError("server", errors.ErrNeedsGAuth.Error(), nil)
)
