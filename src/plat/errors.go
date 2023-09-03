package plat

import (
	"git.sr.ht/~kvo/libgo/errors"
)

// Predefined errors for use in TaskCollect.
var (
	// Daymap errors
	ErrGetGradesFailed             = errors.New("failed to get task grades", nil)
	ErrInvalidDmJson               = errors.New("invalid lessons JSON", nil)
	ErrInvalidHtmlForm             = errors.New("invalid HTML form", nil)
	ErrInvalidResp                 = errors.New("invalid HTML response", nil)
	ErrInvalidTaskResp             = errors.New("invalid task HTML response", nil)
	ErrNoActionAttrib              = errors.New(`cannot find "action=" in SAML form`, nil)
	ErrNoDateFound                 = errors.New(`resource has no post date`, nil)
	ErrNoClientRequestID           = errors.New("could not find client request ID", nil)
	ErrUnterminatedClientRequestID = errors.New("client request ID has no end", nil)

	// Google Classroom errors
	ErrInvalidTaskID        = errors.New("invalid task ID", nil)

	// File errors
	ErrFileRead     = errors.New("file could not be read", nil)
	ErrMissingFiles = errors.New("the following files are missing", nil)

	// User errors
	ErrBadCommandUsage = errors.New("invalid invocation", nil)

	// Miscellaneous errors
	ErrInitFailed           = errors.New("initialization failed", nil)
	ErrInvalidInterfaceType = errors.New("invalid interface type passed as argument", nil)
	ErrAuthFailed           = errors.New("authentication failed", nil)
	ErrCorruptMIME          = errors.New("corrupt MIME request", nil)
	ErrIncompleteCreds      = errors.New("user has incomplete credentials", nil)
	ErrInvalidAuth          = errors.New("invalid session token", nil)
	ErrNoPlatform           = errors.New("unsupported platform", nil)
	ErrNotFound             = errors.New("cannot find resource", nil)
	ErrNeedsGAuth           = errors.New("Google auth required", nil)
)
