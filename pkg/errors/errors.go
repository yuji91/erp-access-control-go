package errors

import (
	"fmt"
	"net/http"
)

// APIError represents a structured API error
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Status  int    `json:"-"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Common error codes
const (
	ErrCodeValidation       = "VALIDATION_ERROR"
	ErrCodeAuthentication   = "AUTHENTICATION_ERROR"
	ErrCodeAuthorization    = "AUTHORIZATION_ERROR"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeDatabase         = "DATABASE_ERROR"
	ErrCodeInternal         = "INTERNAL_ERROR"
	ErrCodeInvalidToken     = "INVALID_TOKEN"
	ErrCodeExpiredToken     = "EXPIRED_TOKEN"
	ErrCodePermissionDenied = "PERMISSION_DENIED"
)

// Predefined errors
var (
	ErrInvalidToken = &APIError{
		Code:    ErrCodeInvalidToken,
		Message: "Invalid or malformed token",
		Status:  http.StatusUnauthorized,
	}

	ErrExpiredToken = &APIError{
		Code:    ErrCodeExpiredToken,
		Message: "Token has expired",
		Status:  http.StatusUnauthorized,
	}

	ErrPermissionDenied = &APIError{
		Code:    ErrCodePermissionDenied,
		Message: "Insufficient permissions",
		Status:  http.StatusForbidden,
	}

	ErrUserNotFound = &APIError{
		Code:    ErrCodeNotFound,
		Message: "User not found",
		Status:  http.StatusNotFound,
	}
)

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *APIError {
	return &APIError{
		Code:    ErrCodeValidation,
		Message: "Validation failed",
		Details: fmt.Sprintf("Field '%s': %s", field, message),
		Status:  http.StatusBadRequest,
	}
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(message string) *APIError {
	return &APIError{
		Code:    ErrCodeAuthentication,
		Message: "Authentication failed",
		Details: message,
		Status:  http.StatusUnauthorized,
	}
}

// NewAuthorizationError creates a new authorization error
func NewAuthorizationError(message string) *APIError {
	return &APIError{
		Code:    ErrCodeAuthorization,
		Message: "Authorization failed",
		Details: message,
		Status:  http.StatusForbidden,
	}
}

// NewDatabaseError creates a new database error
func NewDatabaseError(err error) *APIError {
	return &APIError{
		Code:    ErrCodeDatabase,
		Message: "Database operation failed",
		Details: err.Error(),
		Status:  http.StatusInternalServerError,
	}
}

// NewInternalError creates a new internal server error
func NewInternalError(message string) *APIError {
	return &APIError{
		Code:    ErrCodeInternal,
		Message: "Internal server error",
		Details: message,
		Status:  http.StatusInternalServerError,
	}
}
