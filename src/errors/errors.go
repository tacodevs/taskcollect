package errors

import (
	"errors"
	"fmt"
)

var (
	ErrAuthFailed      = errors.New("authentication failed")
	ErrCorruptMIME     = errors.New("corrupt MIME request")
	ErrIncompleteCreds = errors.New("user has incomplete credentials")
	ErrInvalidAuth     = errors.New("invalid session token")
	ErrNoPlatform      = errors.New("unsupported platform")
	ErrNotFound        = errors.New("cannot find resource")
	ErrNeedsGAuth      = errors.New("Google auth required")

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
	Pkg  string
	Err  error
	Text string
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

	return fmt.Errorf("%v: %v: %w", err.Pkg, err.Text, err.Err).Error()
}

func (err *ErrorWrapper) SetError(e error) {
	err.Err = e
}

func (err ErrorWrapper) AsError() error {
	return ErrorWrapper{
		Pkg:  err.Pkg,
		Err:  err.Err,
		Text: err.Text,
	}
}

// NewError returns an ErrorWrapper which contains information on which package the error originated,
// the error itself, and the error text/message.
func NewError(pkg string, err error, text string) ErrorWrapper { // TODO: custom error type?
	return ErrorWrapper{
		Pkg:  pkg,
		Err:  err,
		Text: text,
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
