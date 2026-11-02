package errorx

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// Error codes
const (
	CodeSuccess       = 0
	CodeBadRequest    = 400
	CodeUnauthorized  = 401
	CodeForbidden     = 403
	CodeNotFound      = 404
	CodeInternalError = 500
)

// CodeError represents a business error with code and message
type CodeError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *CodeError) Error() string {
	return e.Message
}

// Data returns the error response data
func (e *CodeError) Data() *CodeError {
	return e
}

// NewCodeError creates a new CodeError with code and message
func NewCodeError(code int, msg string) error {
	return &CodeError{Code: code, Message: msg}
}

// NewBadRequestError creates a 400 error
func NewBadRequestError(msg string) error {
	return NewCodeError(CodeBadRequest, msg)
}

// NewUnauthorizedError creates a 401 error
func NewUnauthorizedError(msg string) error {
	return NewCodeError(CodeUnauthorized, msg)
}

// NewForbiddenError creates a 403 error
func NewForbiddenError(msg string) error {
	return NewCodeError(CodeForbidden, msg)
}

// NewNotFoundError creates a 404 error
func NewNotFoundError(msg string) error {
	return NewCodeError(CodeNotFound, msg)
}

// NewInternalError creates a 500 error
func NewInternalError(msg string) error {
	return NewCodeError(CodeInternalError, msg)
}

// ErrorResponse is a JSON response struct (does NOT implement error interface)
// This is important because go-zero's ErrorCtx checks if the body implements error,
// and if so, uses http.Error() which writes plain text instead of JSON.
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SetupErrorHandler configures go-zero to return JSON errors
func SetupErrorHandler() {
	httpx.SetErrorHandler(func(err error) (int, interface{}) {
		switch e := err.(type) {
		case *CodeError:
			// Return appropriate HTTP status based on code
			httpStatus := http.StatusBadRequest
			switch e.Code {
			case CodeUnauthorized:
				httpStatus = http.StatusUnauthorized
			case CodeForbidden:
				httpStatus = http.StatusForbidden
			case CodeNotFound:
				httpStatus = http.StatusNotFound
			case CodeInternalError:
				httpStatus = http.StatusInternalServerError
			default:
				// Any other HTTP error status is answered as itself
				// (409, 422, 429, 503 ...).
				if e.Code >= 400 && e.Code < 600 {
					httpStatus = e.Code
				}
			}
			// Return ErrorResponse (not CodeError) to avoid error interface check
			return httpStatus, ErrorResponse{Code: e.Code, Message: e.Message}
		default:
			// For any other error, return as 500 with JSON
			return http.StatusInternalServerError, ErrorResponse{
				Code:    CodeInternalError,
				Message: err.Error(),
			}
		}
	})
}
