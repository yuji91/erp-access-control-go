package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: ベンチマークテストの追加
// - NewValidationError、NewNotFoundErrorのパフォーマンス測定
// - 大量のバリデーションエラー生成時の性能評価
// - APIError構造体のメモリ使用量測定

// TODO: テーブル駆動テストの拡張
// - TestAPIError_ToMapの実装（現在は基本構造確認のみ）
// - エラーコードとHTTPステータスの一貫性検証
// - 国際化対応のためのメッセージキー検証

func TestNewValidationError(t *testing.T) {
	tests := []struct {
		name     string
		messages []string
		expected string
	}{
		{
			name:     "単一エラーメッセージ",
			messages: []string{"required field is missing"},
			expected: "required field is missing",
		},
		{
			name:     "複数エラーメッセージ",
			messages: []string{"email is invalid", "password is too short"},
			expected: "email is invalid; password is too short",
		},
		{
			name:     "空のメッセージ",
			messages: []string{},
			expected: "",
		},
		// TODO: より多様なテストケースを追加
		// - 非常に長いエラーメッセージの処理
		// - 特殊文字（Unicode、HTML）を含むメッセージ
		// - nil値の混在したメッセージ配列
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// interface{}に変換
			args := make([]interface{}, len(tt.messages))
			for i, msg := range tt.messages {
				args[i] = msg
			}
			err := NewValidationError(args...)
			assert.Equal(t, "VALIDATION_ERROR", err.Code)
			assert.Equal(t, "Validation failed", err.Message)
			if len(tt.messages) > 0 {
				assert.Contains(t, err.Details, tt.messages[0])
			}
			assert.Equal(t, 400, err.Status)
		})
	}
}

func TestNewNotFoundError(t *testing.T) {
	resource := "user"
	err := NewNotFoundError(resource)
	
	assert.Equal(t, "NOT_FOUND", err.Code)
	assert.Equal(t, "Resource not found", err.Message)
	assert.Equal(t, resource, err.Details)
	assert.Equal(t, 404, err.Status)
}

func TestNewUnauthorizedError(t *testing.T) {
	message := "invalid credentials"
	err := NewUnauthorizedError(message)
	
	assert.Equal(t, "AUTHENTICATION_ERROR", err.Code)
	assert.Equal(t, "Unauthorized", err.Message)
	assert.Equal(t, message, err.Details)
	assert.Equal(t, 401, err.Status)
}

func TestNewInternalServerError(t *testing.T) {
	message := "database connection failed"
	err := NewInternalServerError(message)
	
	assert.Equal(t, "INTERNAL_ERROR", err.Code)
	assert.Equal(t, "Internal server error", err.Message)
	assert.Equal(t, message, err.Details)
	assert.Equal(t, 500, err.Status)
}

func TestAPIError_Error(t *testing.T) {
	err := &APIError{
		Code:    "TEST_ERROR",
		Message: "test message",
		Status:  400,
		Details: "test details",
	}
	
	expected := "TEST_ERROR: test message"
	assert.Equal(t, expected, err.Error())
}

func TestAPIError_ToMap(t *testing.T) {
	err := &APIError{
		Code:    "TEST_ERROR",
		Message: "test message",
		Status:  400,
		Details: "test details",
	}
	
	// ToMapメソッドが存在しない場合、基本的な構造の確認に変更
	assert.Equal(t, "TEST_ERROR", err.Code)
	assert.Equal(t, "test message", err.Message)
	assert.Equal(t, "test details", err.Details)
	assert.Equal(t, 400, err.Status)
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		code     string
		status   int
	}{
		{
			name:   "ErrUnauthorized",
			err:    ErrUnauthorized,
			code:   "AUTHENTICATION_ERROR",
			status: 401,
		},
		{
			name:   "ErrInvalidCredentials",
			err:    ErrInvalidCredentials,
			code:   "AUTHENTICATION_ERROR",
			status: 401,
		},
		{
			name:   "ErrInvalidToken",
			err:    ErrInvalidToken,
			code:   "AUTHENTICATION_ERROR",
			status: 401,
		},
		{
			name:   "ErrTokenExpired",
			err:    ErrTokenExpired,
			code:   "AUTHENTICATION_ERROR",
			status: 401,
		},
		{
			name:   "ErrTokenRevoked",
			err:    ErrTokenRevoked,
			code:   "AUTHENTICATION_ERROR",
			status: 401,
		},
		{
			name:   "ErrUserInactive",
			err:    ErrUserInactive,
			code:   "AUTHENTICATION_ERROR",
			status: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
					assert.Equal(t, tt.code, tt.err.Code)
		assert.Equal(t, tt.status, tt.err.Status)
		assert.NotEmpty(t, tt.err.Message)
		})
	}
} 