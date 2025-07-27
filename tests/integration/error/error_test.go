package error

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"erp-access-control-go/internal/handlers"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/tests/integration"
)

type ErrorTestSuite struct {
	suite.Suite
}

func TestErrorSuite(t *testing.T) {
	suite.Run(t, new(ErrorTestSuite))
}

func (s *ErrorTestSuite) SetupSuite() {
	integration.SetupTestEnvironment(s.T())
}

func (s *ErrorTestSuite) TearDownSuite() {
	integration.CleanupTestEnvironment(s.T())
}

func (s *ErrorTestSuite) TestValidationError() {
	// 無効なリクエストボディ
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", map[string]interface{}{
		"invalid_field": "value",
	})
	require.NoError(s.T(), err)

	// リクエストの実行
	w := integration.ExecuteRequest(req)

	// レスポンスの検証
	require.Equal(s.T(), http.StatusBadRequest, w.Code)

	var response errors.APIError
	integration.ParseResponse(s.T(), w, &response)

	require.Equal(s.T(), errors.ErrCodeValidation, response.Code)
	require.Equal(s.T(), "Validation failed", response.Message)
	require.Equal(s.T(), "request", response.Details.Field)
	require.Equal(s.T(), "Invalid request format", response.Details.Reason)
}

func (s *ErrorTestSuite) TestAuthenticationError() {
	// 無効なトークンでリクエスト
	req, err := integration.CreateTestRequest("GET", "/api/v1/auth/profile", nil)
	require.NoError(s.T(), err)
	req.Header.Set("Authorization", "Bearer invalid-token")

	// リクエストの実行
	w := integration.ExecuteRequest(req)

	// レスポンスの検証
	require.Equal(s.T(), http.StatusUnauthorized, w.Code)

	var response errors.APIError
	integration.ParseResponse(s.T(), w, &response)

	require.Equal(s.T(), errors.ErrCodeAuthentication, response.Code)
	require.Equal(s.T(), "Authentication failed", response.Message)
}

func (s *ErrorTestSuite) TestAuthorizationError() {
	// 認証なしでパスワード変更リクエスト
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/change-password", handlers.ChangePasswordRequest{
		CurrentPassword: "password123",
		NewPassword:     "newpassword123",
	})
	require.NoError(s.T(), err)

	// リクエストの実行
	w := integration.ExecuteRequest(req)

	// レスポンスの検証
	require.Equal(s.T(), http.StatusUnauthorized, w.Code)

	var response errors.APIError
	integration.ParseResponse(s.T(), w, &response)

	require.Equal(s.T(), errors.ErrCodeAuthentication, response.Code)
	require.Equal(s.T(), "Authentication failed", response.Message)
}

func (s *ErrorTestSuite) TestNotFoundError() {
	// 存在しないエンドポイントへのリクエスト
	req, err := integration.CreateTestRequest("GET", "/api/v1/non-existent", nil)
	require.NoError(s.T(), err)

	// リクエストの実行
	w := integration.ExecuteRequest(req)

	// レスポンスの検証
	require.Equal(s.T(), http.StatusNotFound, w.Code)
}

func (s *ErrorTestSuite) TestMethodNotAllowedError() {
	// 不正なHTTPメソッド
	req, err := integration.CreateTestRequest("PUT", "/api/v1/auth/login", nil)
	require.NoError(s.T(), err)

	// リクエストの実行
	w := integration.ExecuteRequest(req)

	// レスポンスの検証
	require.Equal(s.T(), http.StatusNotFound, w.Code) // GinはデフォルトでMethodNotAllowedを返さない
}

func (s *ErrorTestSuite) TestInvalidJSONError() {
	// 不正なJSON形式
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", "invalid json")
	require.NoError(s.T(), err)

	// リクエストの実行
	w := integration.ExecuteRequest(req)

	// レスポンスの検証
	require.Equal(s.T(), http.StatusBadRequest, w.Code)

	var response errors.APIError
	integration.ParseResponse(s.T(), w, &response)

	require.Equal(s.T(), errors.ErrCodeValidation, response.Code)
	require.Equal(s.T(), "Validation failed", response.Message)
}

func (s *ErrorTestSuite) TestMissingRequiredFieldError() {
	// 必須フィールドの欠落
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", map[string]interface{}{
		"email": "test@example.com",
		// パスワードフィールドが欠落
	})
	require.NoError(s.T(), err)

	// リクエストの実行
	w := integration.ExecuteRequest(req)

	// レスポンスの検証
	require.Equal(s.T(), http.StatusBadRequest, w.Code)

	var response errors.APIError
	integration.ParseResponse(s.T(), w, &response)

	require.Equal(s.T(), errors.ErrCodeValidation, response.Code)
	require.Equal(s.T(), "Validation failed", response.Message)
}

func (s *ErrorTestSuite) TestInvalidEmailFormatError() {
	// 無効なメールアドレス形式
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", map[string]interface{}{
		"email":    "invalid-email",
		"password": "password123",
	})
	require.NoError(s.T(), err)

	// リクエストの実行
	w := integration.ExecuteRequest(req)

	// レスポンスの検証
	require.Equal(s.T(), http.StatusBadRequest, w.Code)

	var response errors.APIError
	integration.ParseResponse(s.T(), w, &response)

	require.Equal(s.T(), errors.ErrCodeValidation, response.Code)
	require.Equal(s.T(), "Validation failed", response.Message)
}

func (s *ErrorTestSuite) TestInvalidPasswordLengthError() {
	// パスワード長が不足
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", map[string]interface{}{
		"email":    "test@example.com",
		"password": "short",
	})
	require.NoError(s.T(), err)

	// リクエストの実行
	w := integration.ExecuteRequest(req)

	// レスポンスの検証
	require.Equal(s.T(), http.StatusBadRequest, w.Code)

	var response errors.APIError
	integration.ParseResponse(s.T(), w, &response)

	require.Equal(s.T(), errors.ErrCodeValidation, response.Code)
	require.Equal(s.T(), "Validation failed", response.Message)
}
