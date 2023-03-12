package errors

import (
	"errors"
	"fmt"
)

var (
	ErrAuthFailed           = errors.New("authentication failed")
	ErrCorruptMIME          = errors.New("corrupt MIME request")
	ErrDaymapUpload         = errors.New("Daymap task file uploads don't work (issue #68)")
	ErrGclassApiRestriction = errors.New("Google Classroom API restriction")
	ErrIncompleteCreds      = errors.New("user has incomplete credentials")
	ErrInvalidAuth          = errors.New("invalid session token")
	ErrNoPlatform           = errors.New("unsupported platform")
	ErrNotFound             = errors.New("cannot find resource")
	ErrNeedsGAuth           = errors.New("Google auth required")

	ErrInitFailed = errors.New("initialization failed")

	// File errors

	ErrFileRead     = errors.New("file could not be read")
	ErrMissingFiles = errors.New("the following files are missing")

	// User errors

	ErrBadCommandUsage = errors.New("taskcollect: Invalid invocation. See the documentation on command usage")

	// Misc

	ErrInvalidInterfaceType = errors.New("an invalid interface type was passed as argument")
)

// Custom error wrapper
type ErrorWrapper struct {
	Origin string
	Text   string
	Err    error
}

// Alias for func (ErrorWrapper).Error()
func (err ErrorWrapper) AsString() string {
	return err.Error()
}

// When ErrorWrapper is treated as an error type, this is used.
// The error type has Error() as one of its methods
func (err ErrorWrapper) Error() string {
	// Guard against panics
	//if err.Err != nil {
	//	return err.Err.Error()
	//}

	if err.Err == nil {
		return fmt.Errorf("%v: %v", err.Origin, err.Text).Error()
	}

	return fmt.Errorf("%v: %v: %w", err.Origin, err.Text, err.Err).Error()
}

func (err *ErrorWrapper) SetError(e error) {
	err.Err = e
}

// Return ErrorWrapper as an explicit error type.
// This is most useful when working with nil error values. Since ErrorWrapper is a struct,
// it is incompatible with nil values.
func (err ErrorWrapper) AsError() error {
	return ErrorWrapper{
		Origin: err.Origin,
		Text:   err.Text,
		Err:    err.Err,
	}
}

// HasOnly checks if all elements of s are elem.
func HasOnly(s []error, elem error) bool {
	for _, v := range s {
		if !errors.Is(v, elem) {
			return false
		}
	}
	return true
}

// NewError returns an ErrorWrapper which contains information on which package and/or function
// the error originated, the error text/message, and the error itself
func NewError(origin string, text string, err error) ErrorWrapper {
	return ErrorWrapper{
		Origin: origin,
		Text:   text,
		Err:    err,
	}
}

//
// Reimplement errors module, so only this module needs to be imported to manage errors

func As(err error, target any) bool {
	return errors.As(err, target)
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func New(text string) error {
	return errors.New(text)
}

func Unwrap(err error) error {
	return errors.Unwrap(err)
}
