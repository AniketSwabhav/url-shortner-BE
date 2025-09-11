package errors

import "net/http"

// ValidationError Represents Validation type error
type ValidationError struct {
	HTTPStatus int    `example:"400" json:"-"`
	Message    string `example:"Invalid data" json:"message"`
}

// Error Implements error interface
func (e ValidationError) Error() string {
	return e.Message
}

// NewValidationError returns new instance of Validation error.
func NewValidationError(msg string) *ValidationError {
	return &ValidationError{
		HTTPStatus: http.StatusBadRequest,
		Message:    msg,
	}
}

func NewInValidPasswordError(msg string) *UnauthorizedError {
	return &UnauthorizedError{
		Message:    msg,
		HTTPStatus: http.StatusUnauthorized,
	}
}
