package services

type APIError struct {
	Status  int
	Message string
	Code    string
	Err     error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func NewAPIError(status int, code, message string, err error) *APIError {
	return &APIError{Status: status, Code: code, Message: message, Err: err}
}

func BadRequest(message string, err error) *APIError {
	return NewAPIError(400, "bad_request", message, err)
}

func Internal(message string, err error) *APIError {
	return NewAPIError(500, "internal_error", message, err)
}

func NotFound(message string, err error) *APIError {
	return NewAPIError(404, "not_found", message, err)
}
