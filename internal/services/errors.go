package services

import "fmt"

// ErrorCode identifies the type of error.
type ErrorCode string

const (
	ErrCodeValidation    ErrorCode = "VALIDATION"
	ErrCodeNotFound      ErrorCode = "NOT_FOUND"
	ErrCodeConflict      ErrorCode = "CONFLICT"
	ErrCodeUnsupported   ErrorCode = "UNSUPPORTED"
	ErrCodeIncompatible  ErrorCode = "INCOMPATIBLE"
	ErrCodeIntegrity     ErrorCode = "INTEGRITY"
	ErrCodeNetwork       ErrorCode = "NETWORK"
	ErrCodeUpstream      ErrorCode = "UPSTREAM"
	ErrCodeCancelled     ErrorCode = "CANCELLED"
	ErrCodeInternal      ErrorCode = "INTERNAL"
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

// NewUnsupportedError creates an unsupported error.
func NewUnsupportedError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeUnsupported, Message: message}
}

// NewIncompatibleError creates an incompatible error.
func NewIncompatibleError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeIncompatible, Message: message}
}

// NewIntegrityError creates an integrity error.
func NewIntegrityError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeIntegrity, Message: message}
}

// NewNetworkError creates a network error.
func NewNetworkError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeNetwork, Message: message}
}

// NewUpstreamError creates an upstream error.
func NewUpstreamError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeUpstream, Message: message}
}

// NewCancelledError creates a cancelled error.
func NewCancelledError(operationID string) *ServiceError {
	return &ServiceError{Code: ErrCodeCancelled, Message: "operation cancelled", Fields: []string{operationID}}
}

// NewInternalError creates an internal error.
func NewInternalError(message string) *ServiceError {
	return &ServiceError{Code: ErrCodeInternal, Message: message}
}
