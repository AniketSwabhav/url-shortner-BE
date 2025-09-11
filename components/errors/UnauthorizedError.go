package errors

import "net/http"

// UnauthorizedError represents errors due to insufficient authorization when accessing a resource.
type UnauthorizedError struct {
	HTTPStatus int    `example:"401" json:"-"`
	Message    string `example:"Token must be specified" json:"message"`
}

// Error Implements error interface
func (e UnauthorizedError) Error() string {
	return e.Message
}

// NewUnauthorizedError returns new instance of Validation error.
func NewUnauthorizedError(msg string) *UnauthorizedError {
	return &UnauthorizedError{
		HTTPStatus: http.StatusUnauthorized,
		Message:    msg,
	}
}

func NewInActiveUserError(msg string) *UnauthorizedError {
	return &UnauthorizedError{
		HTTPStatus: http.StatusUnauthorized,
		Message:    msg,
	}
}
