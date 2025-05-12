package errors

import (
	"fmt"
	"time"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	// AuthError represents authentication/authorization errors
	AuthError ErrorType = "auth"

	// ConfigError represents configuration errors
	ConfigError ErrorType = "config"

	// ClientError represents client initialization errors
	ClientError ErrorType = "client"

	// OperationError represents errors during operations
	OperationError ErrorType = "operation"

	// ValidationError represents validation errors
	ValidationError ErrorType = "validation"

	// IOError represents input/output errors
	IOError ErrorType = "io"

	// ContextError represents context-related errors
	ContextError ErrorType = "context"

	// SystemError represents system-level errors
	SystemError ErrorType = "system"

	// UnknownError represents unclassified errors
	UnknownError ErrorType = "unknown"
)

// DomainError is the base error type for domain-specific errors
type DomainError struct {
	Type      ErrorType
	Domain    string
	Message   string
	Cause     error
	Timestamp time.Time
	Details   map[string]interface{}
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Domain, e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Domain, e.Type, e.Message)
}

// Unwrap returns the cause of the error
func (e *DomainError) Unwrap() error {
	return e.Cause
}

// WithDetail adds a detail to the error
func (e *DomainError) WithDetail(key string, value interface{}) *DomainError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// NewAuthError creates a new authentication error
func NewAuthError(domain, message string, cause error) *DomainError {
	return &DomainError{
		Type:      AuthError,
		Domain:    domain,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// NewConfigError creates a new configuration error
func NewConfigError(domain, message string, cause error) *DomainError {
	return &DomainError{
		Type:      ConfigError,
		Domain:    domain,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// NewClientError creates a new client initialization error
func NewClientError(domain, message string, cause error) *DomainError {
	return &DomainError{
		Type:      ClientError,
		Domain:    domain,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// NewOperationError creates a new operation error
func NewOperationError(domain, message string, cause error) *DomainError {
	return &DomainError{
		Type:      OperationError,
		Domain:    domain,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// NewValidationError creates a new validation error
func NewValidationError(domain, message string, cause error) *DomainError {
	return &DomainError{
		Type:      ValidationError,
		Domain:    domain,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// NewIOError creates a new input/output error
func NewIOError(domain, message string, cause error) *DomainError {
	return &DomainError{
		Type:      IOError,
		Domain:    domain,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// NewContextError creates a new context error
func NewContextError(domain, message string, cause error) *DomainError {
	return &DomainError{
		Type:      ContextError,
		Domain:    domain,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// NewSystemError creates a new system-level error
func NewSystemError(domain, message string, cause error) *DomainError {
	return &DomainError{
		Type:      SystemError,
		Domain:    domain,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// IsErrorOfType checks if an error is of a specific type
func IsErrorOfType(err error, errorType ErrorType) bool {
	if domainErr, ok := err.(*DomainError); ok {
		return domainErr.Type == errorType
	}
	return false
}

// IsAuthError checks if an error is an authentication error
func IsAuthError(err error) bool {
	return IsErrorOfType(err, AuthError)
}

// IsConfigError checks if an error is a configuration error
func IsConfigError(err error) bool {
	return IsErrorOfType(err, ConfigError)
}

// IsClientError checks if an error is a client error
func IsClientError(err error) bool {
	return IsErrorOfType(err, ClientError)
}

// IsOperationError checks if an error is an operation error
func IsOperationError(err error) bool {
	return IsErrorOfType(err, OperationError)
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	return IsErrorOfType(err, ValidationError)
}

// IsIOError checks if an error is an I/O error
func IsIOError(err error) bool {
	return IsErrorOfType(err, IOError)
}

// IsContextError checks if an error is a context error
func IsContextError(err error) bool {
	return IsErrorOfType(err, ContextError)
}

// IsSystemError checks if an error is a system error
func IsSystemError(err error) bool {
	return IsErrorOfType(err, SystemError)
}
