package logging

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"erp-access-control-go/internal/handlers"
	"erp-access-control-go/models"
	"erp-access-control-go/pkg/logger"
	"erp-access-control-go/tests/integration"
)

type LoggingTestSuite struct {
	suite.Suite
	testUser *models.User
	logBuf   *bytes.Buffer
}

func TestLoggingSuite(t *testing.T) {
	suite.Run(t, new(LoggingTestSuite))
}

func (s *LoggingTestSuite) SetupSuite() {
	integration.SetupTestEnvironment(s.T())
	s.logBuf = new(bytes.Buffer)
	integration.TestLogger = logger.NewLogger(
		logger.WithOutput(s.logBuf),
		logger.WithMinLevel(logger.DEBUG),
		logger.WithEnvironment("test"),
	)
}

func (s *LoggingTestSuite) TearDownSuite() {
	integration.CleanupTestEnvironment(s.T())
}

func (s *LoggingTestSuite) SetupTest() {
	// テストデータベースをクリーンアップ
	integration.TestDB.Exec("DELETE FROM users")

	// テストユーザーの作成
	s.testUser = &models.User{
		Email:        "test@example.com",
		Password:     "password123",
		Status:       models.UserStatusActive,
		DepartmentID: integration.TestDepartmentID,
		Name:         "Test User",
	}
	err := integration.TestDB.Create(s.testUser).Error
	require.NoError(s.T(), err)

	// パスワードをハッシュ化
	err = s.testUser.HashPassword(s.testUser.Password)
	require.NoError(s.T(), err)
	err = integration.TestDB.Save(s.testUser).Error
	require.NoError(s.T(), err)

	// ログバッファのクリア
	s.logBuf.Reset()
}

func (s *LoggingTestSuite) TearDownTest() {
	// テストデータのクリーンアップ
	integration.TestDB.Unscoped().Delete(&models.User{}, "email = ?", s.testUser.Email)
}

func (s *LoggingTestSuite) getLastLogEntry() map[string]interface{} {
	// ログバッファから最後のエントリを取得
	logs := s.logBuf.String()
	lines := bytes.Split([]byte(logs), []byte("\n"))

	// 最後の有効なJSONエントリを探す
	for i := len(lines) - 1; i >= 0; i-- {
		if len(lines[i]) > 0 {
			var entry map[string]interface{}
			err := json.Unmarshal(lines[i], &entry)
			if err == nil {
				return entry
			}
		}
	}

	return nil
}

func (s *LoggingTestSuite) TestLoginSuccessLogging() {
	// ログイン成功のリクエスト
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", handlers.LoginRequest{
		Email:    s.testUser.Email,
		Password: s.testUser.Password,
	})
	require.NoError(s.T(), err)

	// リクエストの実行
	w := integration.ExecuteRequest(req)
	require.Equal(s.T(), http.StatusOK, w.Code)

	// ログエントリの検証
	entry := s.getLastLogEntry()
	require.NotNil(s.T(), entry)
	require.Equal(s.T(), "INFO", entry["level"])
	require.Equal(s.T(), "Login successful", entry["message"])
	require.Equal(s.T(), "test", entry["environment"])
	require.NotEmpty(s.T(), entry["timestamp"])
	require.NotEmpty(s.T(), entry["caller"])

	fields := entry["fields"].(map[string]interface{})
	require.Equal(s.T(), s.testUser.Email, fields["email"])
	require.NotEmpty(s.T(), fields["user_id"])
	require.NotEmpty(s.T(), fields["ip"])
}

func (s *LoggingTestSuite) TestLoginFailureLogging() {
	// ログイン失敗のリクエスト
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", handlers.LoginRequest{
		Email:    s.testUser.Email,
		Password: "wrongpassword",
	})
	require.NoError(s.T(), err)

	// リクエストの実行
	w := integration.ExecuteRequest(req)
	require.Equal(s.T(), http.StatusUnauthorized, w.Code)

	// ログエントリの検証
	entry := s.getLastLogEntry()
	require.NotNil(s.T(), entry)
	require.Equal(s.T(), "WARN", entry["level"])
	require.Equal(s.T(), "Login failed", entry["message"])
	require.Equal(s.T(), "test", entry["environment"])
	require.NotEmpty(s.T(), entry["timestamp"])
	require.NotEmpty(s.T(), entry["caller"])

	fields := entry["fields"].(map[string]interface{})
	require.Equal(s.T(), s.testUser.Email, fields["email"])
	require.NotEmpty(s.T(), fields["error"])
	require.NotEmpty(s.T(), fields["ip"])
}

func (s *LoggingTestSuite) TestSensitiveDataMasking() {
	// パスワード変更のリクエスト
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/change-password", handlers.ChangePasswordRequest{
		CurrentPassword: "password123",
		NewPassword:     "newpassword123",
	})
	require.NoError(s.T(), err)

	// リクエストの実行
	w := integration.ExecuteRequest(req)
	require.Equal(s.T(), http.StatusUnauthorized, w.Code)

	// ログエントリの検証
	entry := s.getLastLogEntry()
	require.NotNil(s.T(), entry)
	require.Equal(s.T(), "WARN", entry["level"])
	require.Equal(s.T(), "Authentication failed", entry["message"])

	// センシティブ情報がマスクされていることを確認
	logData := string(s.logBuf.Bytes())
	require.NotContains(s.T(), logData, "password123")
	require.NotContains(s.T(), logData, "newpassword123")
}

func (s *LoggingTestSuite) TestErrorLogging() {
	// 不正なJSONリクエスト
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", "invalid json")
	require.NoError(s.T(), err)

	// リクエストの実行
	w := integration.ExecuteRequest(req)
	require.Equal(s.T(), http.StatusBadRequest, w.Code)

	// ログエントリの検証
	entry := s.getLastLogEntry()
	require.NotNil(s.T(), entry)
	require.Equal(s.T(), "WARN", entry["level"])
	require.Equal(s.T(), "Invalid login request format", entry["message"])
	require.Equal(s.T(), "test", entry["environment"])

	fields := entry["fields"].(map[string]interface{})
	require.NotEmpty(s.T(), fields["error"])
	require.NotEmpty(s.T(), fields["ip"])
}

func (s *LoggingTestSuite) TestAuthenticationLogging() {
	// 無効なトークンでリクエスト
	req, err := integration.CreateTestRequest("GET", "/api/v1/auth/profile", nil)
	require.NoError(s.T(), err)
	req.Header.Set("Authorization", "Bearer invalid-token")

	// リクエストの実行
	w := integration.ExecuteRequest(req)
	require.Equal(s.T(), http.StatusUnauthorized, w.Code)

	// ログエントリの検証
	entry := s.getLastLogEntry()
	require.NotNil(s.T(), entry)
	require.Equal(s.T(), "WARN", entry["level"])
	require.Equal(s.T(), "Token validation failed", entry["message"])

	fields := entry["fields"].(map[string]interface{})
	require.NotEmpty(s.T(), fields["error"])
	require.NotEmpty(s.T(), fields["path"])
	require.NotEmpty(s.T(), fields["ip"])
}

func (s *LoggingTestSuite) TestValidationLogging() {
	// 無効なメールアドレス形式
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", map[string]interface{}{
		"email":    "invalid-email",
		"password": "password123",
	})
	require.NoError(s.T(), err)

	// リクエストの実行
	w := integration.ExecuteRequest(req)
	require.Equal(s.T(), http.StatusBadRequest, w.Code)

	// ログエントリの検証
	entry := s.getLastLogEntry()
	require.NotNil(s.T(), entry)
	require.Equal(s.T(), "WARN", entry["level"])
	require.Equal(s.T(), "Invalid login request format", entry["message"])

	fields := entry["fields"].(map[string]interface{})
	require.NotEmpty(s.T(), fields["error"])
	require.NotEmpty(s.T(), fields["ip"])
}

func (s *LoggingTestSuite) TestLogLevelFiltering() {
	// DEBUGレベルのログ
	integration.TestLogger.Debug("Debug message", map[string]interface{}{
		"key": "value",
	})

	// INFOレベルのログ
	integration.TestLogger.Info("Info message", map[string]interface{}{
		"key": "value",
	})

	// WARNレベルのログ
	integration.TestLogger.Warn("Warn message", map[string]interface{}{
		"key": "value",
	})

	// ログバッファの内容を解析
	logs := s.logBuf.String()
	lines := bytes.Split([]byte(logs), []byte("\n"))

	var entries []map[string]interface{}
	for _, line := range lines {
		if len(line) > 0 {
			var entry map[string]interface{}
			err := json.Unmarshal(line, &entry)
			require.NoError(s.T(), err)
			entries = append(entries, entry)
		}
	}

	// すべてのログレベルが出力されていることを確認
	var levels []string
	for _, entry := range entries {
		levels = append(levels, entry["level"].(string))
	}

	require.Contains(s.T(), levels, "DEBUG")
	require.Contains(s.T(), levels, "INFO")
	require.Contains(s.T(), levels, "WARN")
}
