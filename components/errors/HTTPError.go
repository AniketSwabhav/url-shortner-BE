package errors

// HTTPError Represents HTTP type error
type HTTPError struct {
	HTTPStatus int    `example:"400" json:"-"`
	Message    string `example:"unable to fetch/parse data" json:"message"`
}

// Implementing Error interface
func (httpError HTTPError) Error() string {
	return httpError.Message
}

// NewHTTPError returns new instance of HTTPError
func NewHTTPError(key string, statuscode int) *HTTPError {
	return &HTTPError{
		HTTPStatus: statuscode,
		Message:    key,
	}
}
