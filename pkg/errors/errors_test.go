package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewValidationError(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		reason   string
		expected string
	}{
		{
			name:     "フィールドエラー",
			field:    "email",
			reason:   "Invalid email format",
			expected: "VALIDATION_ERROR: Validation failed (Field: email, Reason: Invalid email format)",
		},
		{
			name:     "必須フィールド",
			field:    "password",
			reason:   "Required field is missing",
			expected: "VALIDATION_ERROR: Validation failed (Field: password, Reason: Required field is missing)",
		},
		{
			name:     "空のフィールド",
			field:    "",
			reason:   "General validation error",
			expected: "VALIDATION_ERROR: Validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidationError(tt.field, tt.reason)
			assert.Equal(t, "VALIDATION_ERROR", err.Code)
			assert.Equal(t, "Validation failed", err.Message)
			assert.Equal(t, tt.field, err.Details.Field)
			assert.Equal(t, tt.reason, err.Details.Reason)
			assert.Equal(t, 400, err.Status)
			assert.Equal(t, tt.expected, err.Error())
		})
	}
}

func TestNewNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		reason   string
		expected string
	}{
		{
			name:     "ユーザー未発見",
			resource: "User",
			reason:   "User with specified ID does not exist",
			expected: "NOT_FOUND: User not found",
		},
		{
			name:     "ロール未発見",
			resource: "Role",
			reason:   "Role has been deleted",
			expected: "NOT_FOUND: Role not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewNotFoundError(tt.resource, tt.reason)
			assert.Equal(t, "NOT_FOUND", err.Code)
			assert.Equal(t, tt.resource+" not found", err.Message)
			assert.Equal(t, tt.reason, err.Details.Reason)
			assert.Equal(t, 404, err.Status)
			assert.Equal(t, tt.expected, err.Error())
		})
	}
}

func TestNewAuthenticationError(t *testing.T) {
	tests := []struct {
		name     string
		reason   string
		expected string
	}{
		{
			name:     "無効な認証情報",
			reason:   "Invalid credentials provided",
			expected: "AUTHENTICATION_ERROR: Authentication failed",
		},
		{
			name:     "トークン期限切れ",
			reason:   "Token has expired",
			expected: "AUTHENTICATION_ERROR: Authentication failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewAuthenticationError(tt.reason)
			assert.Equal(t, "AUTHENTICATION_ERROR", err.Code)
			assert.Equal(t, "Authentication failed", err.Message)
			assert.Equal(t, tt.reason, err.Details.Reason)
			assert.Equal(t, 401, err.Status)
			assert.Equal(t, tt.expected, err.Error())
		})
	}
}

func TestNewInternalError(t *testing.T) {
	tests := []struct {
		name     string
		reason   string
		expected string
	}{
		{
			name:     "データベースエラー",
			reason:   "Failed to connect to database",
			expected: "INTERNAL_ERROR: Internal server error",
		},
		{
			name:     "予期せぬエラー",
			reason:   "Unexpected error occurred",
			expected: "INTERNAL_ERROR: Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewInternalError(tt.reason)
			assert.Equal(t, "INTERNAL_ERROR", err.Code)
			assert.Equal(t, "Internal server error", err.Message)
			assert.Equal(t, tt.reason, err.Details.Reason)
			assert.Equal(t, 500, err.Status)
			assert.Equal(t, tt.expected, err.Error())
		})
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    *APIError
		code   string
		status int
	}{
		{
			name:   "ErrInvalidCredentials",
			err:    ErrInvalidCredentials,
			code:   "AUTHENTICATION_ERROR",
			status: 401,
		},
		{
			name:   "ErrInvalidToken",
			err:    ErrInvalidToken,
			code:   "INVALID_TOKEN",
			status: 401,
		},
		{
			name:   "ErrTokenExpired",
			err:    ErrTokenExpired,
			code:   "EXPIRED_TOKEN",
			status: 401,
		},
		{
			name:   "ErrTokenRevoked",
			err:    ErrTokenRevoked,
			code:   "REVOKED_TOKEN",
			status: 401,
		},
		{
			name:   "ErrPermissionDenied",
			err:    ErrPermissionDenied,
			code:   "PERMISSION_DENIED",
			status: 403,
		},
		{
			name:   "ErrUserNotFound",
			err:    ErrUserNotFound,
			code:   "NOT_FOUND",
			status: 404,
		},
		{
			name:   "ErrUserInactive",
			err:    ErrUserInactive,
			code:   "BUSINESS_RULE_ERROR",
			status: 403,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
			assert.Equal(t, tt.status, tt.err.Status)
			assert.NotEmpty(t, tt.err.Message)
			assert.NotEmpty(t, tt.err.Details.Reason)
		})
	}
}

func TestErrorTypeChecks(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		checks   []func(error) bool
		expected []bool
	}{
		{
			name: "認証エラー",
			err:  NewAuthenticationError("Invalid credentials"),
			checks: []func(error) bool{
				IsAuthenticationError,
				IsAuthorizationError,
				IsValidationError,
				IsNotFound,
			},
			expected: []bool{true, false, false, false},
		},
		{
			name: "認可エラー",
			err:  NewAuthorizationError("Insufficient permissions"),
			checks: []func(error) bool{
				IsAuthenticationError,
				IsAuthorizationError,
				IsValidationError,
				IsNotFound,
			},
			expected: []bool{false, true, false, false},
		},
		{
			name: "バリデーションエラー",
			err:  NewValidationError("email", "Invalid format"),
			checks: []func(error) bool{
				IsAuthenticationError,
				IsAuthorizationError,
				IsValidationError,
				IsNotFound,
			},
			expected: []bool{false, false, true, false},
		},
		{
			name: "未発見エラー",
			err:  NewNotFoundError("User", "User does not exist"),
			checks: []func(error) bool{
				IsAuthenticationError,
				IsAuthorizationError,
				IsValidationError,
				IsNotFound,
			},
			expected: []bool{false, false, false, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, check := range tt.checks {
				assert.Equal(t, tt.expected[i], check(tt.err))
			}
		})
	}
}
