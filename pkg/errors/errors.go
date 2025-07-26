package errors

import (
	"fmt"
	"net/http"
)

// APIError 構造化されたAPIエラーを表現
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Status  int    `json:"-"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// 共通エラーコード
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

// 定義済みエラー
var (
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

	ErrUnauthorized = &APIError{
		Code:    ErrCodeAuthentication,
		Message: "Unauthorized access",
		Status:  http.StatusUnauthorized,
	}

	ErrInvalidCredentials = &APIError{
		Code:    ErrCodeAuthentication,
		Message: "Invalid credentials",
		Status:  http.StatusUnauthorized,
	}

	ErrInvalidToken = &APIError{
		Code:    ErrCodeAuthentication,
		Message: "Invalid token",
		Status:  http.StatusUnauthorized,
	}

	ErrTokenExpired = &APIError{
		Code:    ErrCodeAuthentication,
		Message: "Token expired",
		Status:  http.StatusUnauthorized,
	}

	ErrTokenRevoked = &APIError{
		Code:    ErrCodeAuthentication,
		Message: "Token revoked",
		Status:  http.StatusUnauthorized,
	}

	ErrUserInactive = &APIError{
		Code:    ErrCodeAuthentication,
		Message: "User account is not active",
		Status:  http.StatusUnauthorized,
	}
)

// NewValidationError 新しいバリデーションエラーを作成（オーバーロード対応）
func NewValidationError(args ...interface{}) *APIError {
	apiErr := &APIError{
		Code:    ErrCodeValidation,
		Message: "Validation failed",
		Status:  http.StatusBadRequest,
	}
	
	switch len(args) {
	case 1:
		if err, ok := args[0].(error); ok {
			apiErr.Details = err.Error()
		} else if str, ok := args[0].(string); ok {
			apiErr.Details = str
		}
	case 2:
		field := fmt.Sprintf("%v", args[0])
		message := fmt.Sprintf("%v", args[1])
		apiErr.Details = fmt.Sprintf("Field '%s': %s", field, message)
	}
	
	return apiErr
}

// NewAuthenticationError 新しい認証エラーを作成
func NewAuthenticationError(message string) *APIError {
	return &APIError{
		Code:    ErrCodeAuthentication,
		Message: "Authentication failed",
		Details: message,
		Status:  http.StatusUnauthorized,
	}
}

// NewAuthorizationError 新しい認可エラーを作成
func NewAuthorizationError(message string) *APIError {
	return &APIError{
		Code:    ErrCodeAuthorization,
		Message: "Authorization failed",
		Details: message,
		Status:  http.StatusForbidden,
	}
}

// NewDatabaseError 新しいデータベースエラーを作成
func NewDatabaseError(err error) *APIError {
	return &APIError{
		Code:    ErrCodeDatabase,
		Message: "Database operation failed",
		Details: err.Error(),
		Status:  http.StatusInternalServerError,
	}
}

// NewInternalError 新しい内部サーバーエラーを作成
func NewInternalError(message string) *APIError {
	return &APIError{
		Code:    ErrCodeInternal,
		Message: "Internal server error",
		Details: message,
		Status:  http.StatusInternalServerError,
	}
}

// NewNotFoundError 新しい404エラーを作成
func NewNotFoundError(message string) *APIError {
	return &APIError{
		Code:    ErrCodeNotFound,
		Message: "Resource not found",
		Details: message,
		Status:  http.StatusNotFound,
	}
}

// NewUnauthorizedError 新しい401エラーを作成
func NewUnauthorizedError(message string) *APIError {
	return &APIError{
		Code:    ErrCodeAuthentication,
		Message: "Unauthorized",
		Details: message,
		Status:  http.StatusUnauthorized,
	}
}

// NewInternalServerError 新しい500エラーを作成
func NewInternalServerError(message string) *APIError {
	return &APIError{
		Code:    ErrCodeInternal,
		Message: "Internal server error",
		Details: message,
		Status:  http.StatusInternalServerError,
	}
}
