package errors

import (
	"fmt"
	"net/http"
)

// ErrorDetails エラーの詳細情報を表現
type ErrorDetails struct {
	Field  string `json:"field,omitempty"`  // エラーが発生したフィールド
	Reason string `json:"reason,omitempty"` // エラーの具体的な理由
}

// APIError 構造化されたAPIエラーを表現
type APIError struct {
	Code    string       `json:"code"`              // エラーコード
	Message string       `json:"message"`           // 人間が読みやすいメッセージ
	Details ErrorDetails `json:"details,omitempty"` // エラーの詳細情報
	Status  int          `json:"-"`                 // HTTPステータスコード
}

func (e *APIError) Error() string {
	if e.Details.Field != "" {
		return fmt.Sprintf("%s: %s (Field: %s, Reason: %s)", e.Code, e.Message, e.Details.Field, e.Details.Reason)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// エラー種別の定義
const (
	// 認証関連エラー
	ErrCodeAuthentication = "AUTHENTICATION_ERROR" // 認証エラー
	ErrCodeInvalidToken   = "INVALID_TOKEN"        // 無効なトークン
	ErrCodeExpiredToken   = "EXPIRED_TOKEN"        // 期限切れトークン
	ErrCodeRevokedToken   = "REVOKED_TOKEN"        // 無効化されたトークン

	// 認可関連エラー
	ErrCodeAuthorization     = "AUTHORIZATION_ERROR" // 認可エラー
	ErrCodePermissionDenied  = "PERMISSION_DENIED"   // 権限不足
	ErrCodeInsufficientScope = "INSUFFICIENT_SCOPE"  // スコープ不足

	// バリデーション関連エラー
	ErrCodeValidation   = "VALIDATION_ERROR" // バリデーションエラー
	ErrCodeInvalidInput = "INVALID_INPUT"    // 不正な入力
	ErrCodeMissingField = "MISSING_FIELD"    // 必須フィールド欠落

	// ビジネスロジック関連エラー
	ErrCodeNotFound     = "NOT_FOUND"           // リソース未発見
	ErrCodeConflict     = "CONFLICT"            // リソース競合
	ErrCodeBusinessRule = "BUSINESS_RULE_ERROR" // ビジネスルール違反

	// システム関連エラー
	ErrCodeDatabase        = "DATABASE_ERROR"         // データベースエラー
	ErrCodeInternal        = "INTERNAL_ERROR"         // 内部サーバーエラー
	ErrCodeExternalService = "EXTERNAL_SERVICE_ERROR" // 外部サービスエラー
)

// 定義済みエラー
var (
	// 認証関連
	ErrInvalidCredentials = &APIError{
		Code:    ErrCodeAuthentication,
		Message: "Invalid credentials",
		Details: ErrorDetails{
			Reason: "The provided credentials are incorrect",
		},
		Status: http.StatusUnauthorized,
	}

	ErrInvalidToken = &APIError{
		Code:    ErrCodeInvalidToken,
		Message: "Invalid token",
		Details: ErrorDetails{
			Reason: "The provided token is invalid or malformed",
		},
		Status: http.StatusUnauthorized,
	}

	ErrTokenExpired = &APIError{
		Code:    ErrCodeExpiredToken,
		Message: "Token expired",
		Details: ErrorDetails{
			Reason: "The provided token has expired",
		},
		Status: http.StatusUnauthorized,
	}

	ErrTokenRevoked = &APIError{
		Code:    ErrCodeRevokedToken,
		Message: "Token revoked",
		Details: ErrorDetails{
			Reason: "The token has been revoked",
		},
		Status: http.StatusUnauthorized,
	}

	// 認可関連
	ErrPermissionDenied = &APIError{
		Code:    ErrCodePermissionDenied,
		Message: "Insufficient permissions",
		Details: ErrorDetails{
			Reason: "You do not have the required permissions for this operation",
		},
		Status: http.StatusForbidden,
	}

	// リソース関連
	ErrUserNotFound = &APIError{
		Code:    ErrCodeNotFound,
		Message: "User not found",
		Details: ErrorDetails{
			Reason: "The requested user does not exist",
		},
		Status: http.StatusNotFound,
	}

	ErrUserInactive = &APIError{
		Code:    ErrCodeBusinessRule,
		Message: "User account is not active",
		Details: ErrorDetails{
			Reason: "The user account must be activated before use",
		},
		Status: http.StatusForbidden,
	}
)

// NewValidationError バリデーションエラーを作成
func NewValidationError(field, reason string) *APIError {
	return &APIError{
		Code:    ErrCodeValidation,
		Message: "Validation failed",
		Details: ErrorDetails{
			Field:  field,
			Reason: reason,
		},
		Status: http.StatusBadRequest,
	}
}

// NewAuthenticationError 認証エラーを作成
func NewAuthenticationError(reason string) *APIError {
	return &APIError{
		Code:    ErrCodeAuthentication,
		Message: "Authentication failed",
		Details: ErrorDetails{
			Reason: reason,
		},
		Status: http.StatusUnauthorized,
	}
}

// NewAuthorizationError 認可エラーを作成
func NewAuthorizationError(reason string) *APIError {
	return &APIError{
		Code:    ErrCodeAuthorization,
		Message: "Authorization failed",
		Details: ErrorDetails{
			Reason: reason,
		},
		Status: http.StatusForbidden,
	}
}

// NewDatabaseError データベースエラーを作成
func NewDatabaseError(err error) *APIError {
	return &APIError{
		Code:    ErrCodeDatabase,
		Message: "Database operation failed",
		Details: ErrorDetails{
			Reason: err.Error(),
		},
		Status: http.StatusInternalServerError,
	}
}

// NewBusinessError ビジネスロジックエラーを作成
func NewBusinessError(code, message, reason string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Details: ErrorDetails{
			Reason: reason,
		},
		Status: http.StatusUnprocessableEntity,
	}
}

// NewInternalError 内部サーバーエラーを作成
func NewInternalError(reason string) *APIError {
	return &APIError{
		Code:    ErrCodeInternal,
		Message: "Internal server error",
		Details: ErrorDetails{
			Reason: reason,
		},
		Status: http.StatusInternalServerError,
	}
}

// NewNotFoundError リソース未発見エラーを作成
func NewNotFoundError(resource, reason string) *APIError {
	return &APIError{
		Code:    ErrCodeNotFound,
		Message: fmt.Sprintf("%s not found", resource),
		Details: ErrorDetails{
			Reason: reason,
		},
		Status: http.StatusNotFound,
	}
}

// IsNotFound エラーがNotFoundエラーかどうかを判定
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Code == ErrCodeNotFound
	}
	return false
}

// IsValidationError エラーがバリデーションエラーかどうかを判定
func IsValidationError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Code == ErrCodeValidation
	}
	return false
}

// IsAuthenticationError エラーが認証エラーかどうかを判定
func IsAuthenticationError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Code == ErrCodeAuthentication
	}
	return false
}

// IsAuthorizationError エラーが認可エラーかどうかを判定
func IsAuthorizationError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Code == ErrCodeAuthorization
	}
	return false
}
