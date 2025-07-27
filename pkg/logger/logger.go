package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel ログレベルを表す型
type LogLevel int

const (
	// ログレベル定義
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String ログレベルを文字列に変換
func (l LogLevel) String() string {
	return [...]string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}[l]
}

// LogEntry 構造化ログエントリ
type LogEntry struct {
	Timestamp   string                 `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Caller      string                 `json:"caller,omitempty"`
	TraceID     string                 `json:"trace_id,omitempty"`
	Environment string                 `json:"environment,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// Logger 改善されたロガー実装
type Logger struct {
	output      io.Writer
	minLevel    LogLevel
	environment string
	traceID     string
}

// LoggerOption ロガー設定オプション
type LoggerOption func(*Logger)

// WithOutput 出力先を設定
func WithOutput(output io.Writer) LoggerOption {
	return func(l *Logger) {
		l.output = output
	}
}

// WithMinLevel 最小ログレベルを設定
func WithMinLevel(level LogLevel) LoggerOption {
	return func(l *Logger) {
		l.minLevel = level
	}
}

// WithEnvironment 環境を設定
func WithEnvironment(env string) LoggerOption {
	return func(l *Logger) {
		l.environment = env
	}
}

// WithTraceID トレースIDを設定
func WithTraceID(traceID string) LoggerOption {
	return func(l *Logger) {
		l.traceID = traceID
	}
}

// NewLogger 新しいロガーを作成
func NewLogger(opts ...LoggerOption) *Logger {
	l := &Logger{
		output:      os.Stdout,
		minLevel:    INFO, // デフォルトはINFO
		environment: os.Getenv("APP_ENV"),
	}

	// オプションの適用
	for _, opt := range opts {
		opt(l)
	}

	return l
}

// log 共通ログ出力処理
func (l *Logger) log(level LogLevel, msg string, fields map[string]interface{}, err error) {
	if level < l.minLevel {
		return
	}

	// 呼び出し元の情報を取得
	_, file, line, ok := runtime.Caller(2)
	var caller string
	if ok {
		caller = fmt.Sprintf("%s:%d", file[strings.LastIndex(file, "/")+1:], line)
	}

	// ログエントリの作成
	entry := LogEntry{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Level:       level.String(),
		Message:     msg,
		Caller:      caller,
		TraceID:     l.traceID,
		Environment: l.environment,
		Fields:      fields,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	// JSON形式でログ出力
	if jsonData, err := json.Marshal(entry); err == nil {
		fmt.Fprintln(l.output, string(jsonData))
	}
}

// WithFields フィールド付きのログ出力
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	return &Logger{
		output:      l.output,
		minLevel:    l.minLevel,
		environment: l.environment,
		traceID:     l.traceID,
	}
}

// Debug デバッグレベルのログを出力
func (l *Logger) Debug(msg string, fields map[string]interface{}) {
	l.log(DEBUG, msg, fields, nil)
}

// Info 情報レベルのログを出力
func (l *Logger) Info(msg string, fields map[string]interface{}) {
	l.log(INFO, msg, fields, nil)
}

// Warn 警告レベルのログを出力
func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	l.log(WARN, msg, fields, nil)
}

// Error エラーレベルのログを出力
func (l *Logger) Error(msg string, err error, fields map[string]interface{}) {
	l.log(ERROR, msg, fields, err)
}

// Fatal 致命的エラーレベルのログを出力
func (l *Logger) Fatal(msg string, err error, fields map[string]interface{}) {
	l.log(FATAL, msg, fields, err)
	os.Exit(1)
}

// SensitiveFields センシティブ情報をマスクするフィールド
var SensitiveFields = map[string]bool{
	"password":     true,
	"token":        true,
	"api_key":      true,
	"credit_card":  true,
	"access_token": true,
}

// MaskSensitiveData センシティブ情報をマスク
func MaskSensitiveData(data map[string]interface{}) map[string]interface{} {
	masked := make(map[string]interface{})
	for k, v := range data {
		if SensitiveFields[strings.ToLower(k)] {
			masked[k] = "********"
		} else if nestedMap, ok := v.(map[string]interface{}); ok {
			masked[k] = MaskSensitiveData(nestedMap)
		} else {
			masked[k] = v
		}
	}
	return masked
}

// Example usage:
/*
logger := NewLogger(
	WithMinLevel(DEBUG),
	WithEnvironment("development"),
	WithTraceID("trace-123"),
)

logger.Info("User logged in", map[string]interface{}{
	"user_id": "123",
	"ip": "192.168.1.1",
})

logger.Error("Failed to process payment",
	fmt.Errorf("invalid card"),
	map[string]interface{}{
		"user_id": "123",
		"amount": 100,
		"credit_card": "1234-5678-9012-3456", // Will be masked
	},
)
*/
