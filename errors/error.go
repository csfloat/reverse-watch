package errors

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"
)

type Error struct {
	Code    uint64 `json:"code"`
	status  int
	Message string `json:"message"`
	Details string `json:"detail,omitempty"`
	wrapped error
}

// Render satisfies the render.Renderer interface
func (e *Error) Render(w http.ResponseWriter, r *http.Request) error {
	var err error
	if e.Code == 0 {
		err = New(Unknown, "unknown server status code")
	}

	render.Status(r, e.status)
	return err
}

func (e *Error) Error() string {
	msg := fmt.Sprintf("%d -- %s", e.Code, e.Message)
	if e.Details != "" {
		msg += fmt.Sprintf(" - %s", e.Details)
	}
	if e.wrapped != nil {
		msg += fmt.Sprintf(" | Cause: %v", e.wrapped)
	}
	return msg
}

func (e *Error) Unwrap() error {
	return e.wrapped
}

func New(base Error, details string, errs ...error) *Error {
	base.Details = details
	if len(errs) > 0 {
		base.wrapped = errs[0]
	}
	return &base
}

func Newf(base Error, cause error, format string, a ...interface{}) *Error {
	return New(base, fmt.Sprintf(format, a...), cause)
}

var (
	Unknown             = Error{Code: 0, status: http.StatusInternalServerError, Message: "unknown error"}
	BadRequest          = Error{Code: 1, status: http.StatusBadRequest, Message: "malformed request"}
	Forbidden           = Error{Code: 2, status: http.StatusForbidden, Message: "forbidden"}
	NotFound            = Error{Code: 3, status: http.StatusNotFound, Message: "resource not found"}
	InternalServerError = Error{Code: 4, status: http.StatusInternalServerError, Message: "internal server error"}
	UnknownResource     = Error{Code: 5, status: http.StatusBadRequest, Message: "unknown resource"}
	InvalidPermission   = Error{Code: 6, status: http.StatusUnauthorized, Message: "invalid permission"}
	InvalidApiKey       = Error{Code: 7, status: http.StatusUnauthorized, Message: "invalid api key"}
	DBCreate            = Error{Code: 8, status: http.StatusInternalServerError, Message: "failed to create resource"}
	DBRead              = Error{Code: 9, status: http.StatusInternalServerError, Message: "failed to read resource"}
	DBUpdate            = Error{Code: 10, status: http.StatusInternalServerError, Message: "failed to update resource"}
	DBDelete            = Error{Code: 11, status: http.StatusInternalServerError, Message: "failed to delete resource"}
	JSONDecode          = Error{Code: 12, status: http.StatusBadRequest, Message: "failed to decode JSON"}
	Limiter             = Error{Code: 13, status: http.StatusInternalServerError, Message: "unable to process request"}
	RateLimited         = Error{Code: 14, status: http.StatusTooManyRequests, Message: "rate limited"}
)
