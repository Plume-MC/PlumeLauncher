package services

import "fmt"

// ErrorCode identifies the type of error.
type ErrorCode string

const (
	ErrCodeValidation   ErrorCode = "VALIDATION"
	ErrCodeNotFound     ErrorCode = "NOT_FOUND"
	ErrCodeConflict     ErrorCode = "CONFLICT"
	ErrCodeIncompatible ErrorCode = "INCOMPATIBLE"
	ErrCodeIntegrity    ErrorCode = "INTEGRITY"
	ErrCodeUpstream     ErrorCode = "UPSTREAM"
	ErrCodeInternal     ErrorCode = "INTERNAL"
	ErrCodeCancelled    ErrorCode = "CANCELLED"
)

// ServiceError is a structured error returned by service methods.
type ServiceError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Fields  []string  `json:"fields,omitempty"`
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewValidationError creates a validation error with field details.
func NewValidationError(message string, fields ...string) *ServiceError {
	return &ServiceError{Code: ErrCodeValidation, Message: message, Fields: fields}
}

// NewNotFoundError creates a not-found error.
func NewNotFoundError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeNotFound, Message: message}
}

// NewConflictError creates a conflict error.
func NewConflictError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeConflict, Message: message}
}

// NewIncompatibleError creates an incompatible error.
func NewIncompatibleError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeIncompatible, Message: message}
}

// NewIntegrityError creates an integrity error.
func NewIntegrityError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeIntegrity, Message: message}
}

// NewUpstreamError creates an upstream error.
func NewUpstreamError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeUpstream, Message: message}
}

// NewInternalError creates an internal error.
func NewInternalError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeInternal, Message: message}
}

// NewCancelledError creates an operation cancellation error.
func NewCancelledError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeCancelled, Message: message}
}
