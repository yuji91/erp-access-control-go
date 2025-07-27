package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		minLevel    LogLevel
		traceID     string
	}{
		{
			name:        "デフォルト設定",
			environment: "",
			minLevel:    INFO,
			traceID:     "",
		},
		{
			name:        "開発環境設定",
			environment: "development",
			minLevel:    DEBUG,
			traceID:     "test-trace-id",
		},
		{
			name:        "本番環境設定",
			environment: "production",
			minLevel:    WARN,
			traceID:     "prod-trace-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(
				WithOutput(&buf),
				WithMinLevel(tt.minLevel),
				WithEnvironment(tt.environment),
				WithTraceID(tt.traceID),
			)

			assert.NotNil(t, logger)
			assert.Equal(t, tt.minLevel, logger.minLevel)
			assert.Equal(t, tt.environment, logger.environment)
			assert.Equal(t, tt.traceID, logger.traceID)
		})
	}
}

func TestLogger_LogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(
		WithOutput(&buf),
		WithMinLevel(DEBUG),
		WithEnvironment("test"),
		WithTraceID("test-trace"),
	)

	tests := []struct {
		name     string
		logFunc  func(string, map[string]interface{})
		level    string
		message  string
		fields   map[string]interface{}
		expected bool // ログが出力されるべきかどうか
	}{
		{
			name:    "デバッグログ",
			logFunc: logger.Debug,
			level:   "DEBUG",
			message: "debug message",
			fields: map[string]interface{}{
				"key": "value",
			},
			expected: true,
		},
		{
			name:    "情報ログ",
			logFunc: logger.Info,
			level:   "INFO",
			message: "info message",
			fields: map[string]interface{}{
				"user_id": float64(123), // JSON数値型はfloat64として扱われる
			},
			expected: true,
		},
		{
			name:    "警告ログ",
			logFunc: logger.Warn,
			level:   "WARN",
			message: "warn message",
			fields: map[string]interface{}{
				"status": "warning",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc(tt.message, tt.fields)

			if tt.expected {
				var logEntry LogEntry
				err := json.Unmarshal(buf.Bytes(), &logEntry)
				assert.NoError(t, err)
				assert.Equal(t, tt.level, logEntry.Level)
				assert.Equal(t, tt.message, logEntry.Message)
				assert.Equal(t, "test", logEntry.Environment)
				assert.Equal(t, "test-trace", logEntry.TraceID)
				assert.NotEmpty(t, logEntry.Timestamp)
				assert.NotEmpty(t, logEntry.Caller)

				// フィールドの比較
				for k, v := range tt.fields {
					assert.Equal(t, v, logEntry.Fields[k])
				}
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(
		WithOutput(&buf),
		WithMinLevel(DEBUG),
		WithEnvironment("test"),
	)

	testErr := errors.New("test error")
	logger.Error("error occurred", testErr, map[string]interface{}{
		"context": "test",
	})

	var logEntry LogEntry
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, "ERROR", logEntry.Level)
	assert.Equal(t, "error occurred", logEntry.Message)
	assert.Equal(t, testErr.Error(), logEntry.Error)
	assert.Equal(t, "test", logEntry.Fields["context"])
}

func TestLogger_MinLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(
		WithOutput(&buf),
		WithMinLevel(WARN),
		WithEnvironment("test"),
	)

	// DEBUGとINFOは出力されないはず
	logger.Debug("debug message", nil)
	assert.Empty(t, buf.String())
	buf.Reset()

	logger.Info("info message", nil)
	assert.Empty(t, buf.String())
	buf.Reset()

	// WARNは出力されるはず
	logger.Warn("warn message", nil)
	assert.NotEmpty(t, buf.String())
}

func TestMaskSensitiveData(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "センシティブフィールドのマスク",
			input: map[string]interface{}{
				"user_id":      123,
				"password":     "secret123",
				"api_key":      "key123",
				"credit_card":  "1234-5678-9012-3456",
				"access_token": "token123",
				"name":         "John Doe",
			},
			expected: map[string]interface{}{
				"user_id":      123,
				"password":     "********",
				"api_key":      "********",
				"credit_card":  "********",
				"access_token": "********",
				"name":         "John Doe",
			},
		},
		{
			name: "ネストされたセンシティブフィールド",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"id":       123,
					"password": "secret123",
				},
				"meta": map[string]interface{}{
					"api_key": "key123",
				},
			},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"id":       123,
					"password": "********",
				},
				"meta": map[string]interface{}{
					"api_key": "********",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSensitiveData(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
