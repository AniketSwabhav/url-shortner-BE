package errors

import "net/http"

type DatabaseError struct {
	HTTPStatus int    `example:"400" json:"-"`
	Message    string `example:"Invalid data" json:"message"`
}

// Implementing Error interface
func (dataBaseError DatabaseError) Error() string {
	return dataBaseError.Message
}

func NewDatabaseError(msg string) *DatabaseError {
	return &DatabaseError{
		Message:    msg,
		HTTPStatus: http.StatusInternalServerError,
	}
}

func NewNotFoundError(msg string) *DatabaseError {
	return &DatabaseError{
		Message:    msg,
		HTTPStatus: http.StatusNotFound,
	}
}
