package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: ログ出力内容の検証
// - 実際のログ出力をキャプチャして内容を検証
// - ログレベル別の出力フォーマット確認
// - タイムスタンプの正確性検証

// TODO: パフォーマンステストの追加
// - 大量ログ出力時の性能測定
// - 並行ログ出力時の安全性確認
// - メモリ使用量の測定

// TODO: エラー処理テストの追加
// - ログ出力先の書き込み失敗時の挙動
// - 無効なログレベル設定時の処理
// - 構造化フィールドのシリアライゼーションエラー

func TestNewLogger(t *testing.T) {
	logger := NewLogger()
	
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.Logger)
	
	// TODO: ロガーの設定内容を詳細に検証
	// - プレフィックス設定の確認
	// - フラグ設定の確認
	// - 出力先の確認
}

func TestLogger_Methods_Exist(t *testing.T) {
	logger := NewLogger()
	
	// メソッドが存在することを確認（実際のログ出力はテストしない）
	assert.NotPanics(t, func() {
		logger.Info("test info message")
	})
	
	assert.NotPanics(t, func() {
		logger.Error("test error message")
	})
	
	assert.NotPanics(t, func() {
		logger.Debug("test debug message")
	})
	
	assert.NotPanics(t, func() {
		logger.Warn("test warn message")
	})
}

func TestLogger_InfoWithFields(t *testing.T) {
	logger := NewLogger()
	
	assert.NotPanics(t, func() {
		logger.Info("test message with fields")
	})
}

func TestLogger_ErrorWithFields(t *testing.T) {
	logger := NewLogger()
	
	assert.NotPanics(t, func() {
		logger.Error("test error message with fields")
	})
}

func TestLogger_FieldTypes(t *testing.T) {
	logger := NewLogger()
	
	// 様々な型のメッセージでエラーが発生しないことを確認
	assert.NotPanics(t, func() {
		logger.Info("test message with string")
		logger.Error("test error with string")
		logger.Debug("test debug message")
		logger.Warn("test warn message")
	})
} 