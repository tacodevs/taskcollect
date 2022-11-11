package daymap

import "main/errors"

var (
	errAuthFailed                  = errors.NewError("daymap", errors.ErrAuthFailed.Error(), nil)
	errInvalidDmJson               = errors.NewError("daymap", "invalid lessons JSON", nil)
	errInvalidHtmlForm             = errors.NewError("daymap", "invalid HTML form", nil)
	errInvalidResp                 = errors.NewError("daymap", "invalid HTML response", nil)
	errInvalidTaskResp             = errors.NewError("daymap", "invalid task HTML response", nil)
	errNoActionAttrib              = errors.NewError("daymap", `cannot find "action=" in SAML form`, nil)
	errNoClientRequestID           = errors.NewError("daymap", "could not find client request ID", nil)
	errUnterminatedClientRequestID = errors.NewError("daymap", "client request ID has no end", nil)
)
