package daymap

import "main/errors"

var (
	errAuthFailed                  = errors.NewError("daymap", nil, errors.ErrAuthFailed.Error())
	errInvalidDmJson               = errors.NewError("daymap", nil, "invalid lessons JSON")
	errInvalidHtmlForm             = errors.NewError("daymap", nil, "invalid HTML form")
	errInvalidResp                 = errors.NewError("daymap", nil, "invalid HTML response")
	errInvalidTaskResp             = errors.NewError("daymap", nil, "invalid task HTML response")
	errNoActionAttrib              = errors.NewError("daymap", nil, `cannot find "action=" in SAML form`)
	errNoClientRequestID           = errors.NewError("daymap", nil, "could not find client request ID")
	errUnterminatedClientRequestID = errors.NewError("daymap", nil, "client request ID has no end")
)
