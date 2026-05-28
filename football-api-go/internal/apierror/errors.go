package apierror

import "fmt"

// APIError is a JSON-serialisable error with an HTTP status code.
type APIError struct {
	Code   int    `json:"-"`
	Detail string `json:"detail"`
}

func (e *APIError) Error() string { return e.Detail }

func BadRequest(msg string) error    { return &APIError{Code: 400, Detail: msg} }
func NotFound(msg string) error      { return &APIError{Code: 404, Detail: msg} }
func Unauthorized() error            { return &APIError{Code: 401, Detail: "not authenticated"} }
func Forbidden(msg string) error     { return &APIError{Code: 403, Detail: msg} }
func Conflict(msg string) error      { return &APIError{Code: 409, Detail: msg} }
func Unprocessable(msg string) error { return &APIError{Code: 422, Detail: msg} }
func TooManyRequests() error         { return &APIError{Code: 429, Detail: "too many requests"} }
func PlanLimitExceeded() error       { return &APIError{Code: 403, Detail: "PLAN_LIMIT_EXCEEDED"} }
func Internal(msg string) error      { return &APIError{Code: 500, Detail: msg} }

func Unprocessablef(format string, args ...any) error {
	return Unprocessable(fmt.Sprintf(format, args...))
}
