package pkg

import (
	"chess-engine/app/constant"
	"errors"
	"fmt"
	"net/http"
)

// AppError is a service-layer error carrying the response status the API should
// report.
//
// It replaces PanicException/PanicHandler, which used panic as ordinary control
// flow: a "user not found" and a nil-pointer dereference arrived at the same
// recover(), were told apart by string-splitting the panic value, and threw away
// the stack either way. Errors are returned now, and a panic means a real bug
// again.
type AppError struct {
	Status constant.ResponseStatus
	Msg    string
	Err    error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Status.GetResponseStatus(), e.Msg, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Status.GetResponseStatus(), e.Msg)
}

func (e *AppError) Unwrap() error { return e.Err }

// NewError builds an AppError. msg is shown to the client, so keep it free of
// internals; wrap the underlying cause in err for the logs.
func NewError(status constant.ResponseStatus, msg string, err error) *AppError {
	return &AppError{Status: status, Msg: msg, Err: err}
}

// HTTPStatus maps a response status onto an HTTP status code.
func HTTPStatus(status constant.ResponseStatus) int {
	switch status {
	case constant.Success:
		return http.StatusOK
	case constant.DataNotFound:
		return http.StatusBadRequest
	case constant.InvalidRequest:
		return http.StatusBadRequest
	case constant.Unauthorized:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// AsAppError extracts an AppError from an error chain, reporting whether one was
// found.
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
