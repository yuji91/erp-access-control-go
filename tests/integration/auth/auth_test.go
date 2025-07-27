package auth

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"erp-access-control-go/internal/handlers"
	"erp-access-control-go/models"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/tests/integration"
)

type AuthTestSuite struct {
	suite.Suite
	testUser *models.User
}

func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}

func (s *AuthTestSuite) SetupSuite() {
	integration.SetupTestEnvironment(s.T())
}

func (s *AuthTestSuite) TearDownSuite() {
	integration.CleanupTestEnvironment(s.T())
}

func (s *AuthTestSuite) SetupTest() {
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
}

func (s *AuthTestSuite) TearDownTest() {
	// テストデータのクリーンアップ
	integration.TestDB.Unscoped().Delete(&models.User{}, "email = ?", s.testUser.Email)
}

func (s *AuthTestSuite) TestLoginSuccess() {
	// リクエストの作成
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", handlers.LoginRequest{
		Email:    s.testUser.Email,
		Password: s.testUser.Password,
	})
	require.NoError(s.T(), err)

	// リクエストの実行
	w := integration.ExecuteRequest(req)

	// レスポンスの検証
	require.Equal(s.T(), http.StatusOK, w.Code)

	var response handlers.LoginResponse
	integration.ParseResponse(s.T(), w, &response)

	require.NotEmpty(s.T(), response.AccessToken)
	require.Equal(s.T(), "Bearer", response.TokenType)
	require.Equal(s.T(), int64(900), response.ExpiresIn)
	require.Equal(s.T(), s.testUser.Email, response.User.Email)
}

func (s *AuthTestSuite) TestLoginInvalidCredentials() {
	// リクエストの作成
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", handlers.LoginRequest{
		Email:    s.testUser.Email,
		Password: "wrongpassword",
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

func (s *AuthTestSuite) TestLoginValidationError() {
	// リクエストの作成（無効なメールアドレス）
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", handlers.LoginRequest{
		Email:    "invalid-email",
		Password: "password123",
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

func (s *AuthTestSuite) TestLoginInactiveUser() {
	// ユーザーを非アクティブに設定
	s.testUser.Status = models.UserStatusInactive
	err := integration.TestDB.Save(s.testUser).Error
	require.NoError(s.T(), err)

	// リクエストの作成
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", handlers.LoginRequest{
		Email:    s.testUser.Email,
		Password: s.testUser.Password,
	})
	require.NoError(s.T(), err)

	// リクエストの実行
	w := integration.ExecuteRequest(req)

	// レスポンスの検証
	require.Equal(s.T(), http.StatusForbidden, w.Code)

	var response errors.APIError
	integration.ParseResponse(s.T(), w, &response)

	require.Equal(s.T(), errors.ErrCodeBusinessRule, response.Code)
	require.Equal(s.T(), "User account is not active", response.Message)
}

func (s *AuthTestSuite) TestChangePasswordSuccess() {
	// ログイントークンの取得
	loginReq, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", handlers.LoginRequest{
		Email:    s.testUser.Email,
		Password: s.testUser.Password,
	})
	require.NoError(s.T(), err)

	loginResp := integration.ExecuteRequest(loginReq)
	require.Equal(s.T(), http.StatusOK, loginResp.Code)

	var loginData handlers.LoginResponse
	integration.ParseResponse(s.T(), loginResp, &loginData)

	// パスワード変更リクエストの作成
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/change-password", handlers.ChangePasswordRequest{
		CurrentPassword: s.testUser.Password,
		NewPassword:     "newpassword123",
	})
	require.NoError(s.T(), err)
	req.Header.Set("Authorization", "Bearer "+loginData.AccessToken)

	// リクエストの実行
	w := integration.ExecuteRequest(req)

	// レスポンスの検証
	require.Equal(s.T(), http.StatusOK, w.Code)

	// 新しいパスワードでログインできることを確認
	newLoginReq, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", handlers.LoginRequest{
		Email:    s.testUser.Email,
		Password: "newpassword123",
	})
	require.NoError(s.T(), err)

	newLoginResp := integration.ExecuteRequest(newLoginReq)
	require.Equal(s.T(), http.StatusOK, newLoginResp.Code)
}

func (s *AuthTestSuite) TestChangePasswordUnauthorized() {
	// 認証トークンなしでリクエスト
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/change-password", handlers.ChangePasswordRequest{
		CurrentPassword: s.testUser.Password,
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

func (s *AuthTestSuite) TestChangePasswordIncorrectCurrent() {
	// ログイントークンの取得
	loginReq, err := integration.CreateTestRequest("POST", "/api/v1/auth/login", handlers.LoginRequest{
		Email:    s.testUser.Email,
		Password: s.testUser.Password,
	})
	require.NoError(s.T(), err)

	loginResp := integration.ExecuteRequest(loginReq)
	require.Equal(s.T(), http.StatusOK, loginResp.Code)

	var loginData handlers.LoginResponse
	integration.ParseResponse(s.T(), loginResp, &loginData)

	// 誤った現在のパスワードでリクエスト
	req, err := integration.CreateTestRequest("POST", "/api/v1/auth/change-password", handlers.ChangePasswordRequest{
		CurrentPassword: "wrongpassword",
		NewPassword:     "newpassword123",
	})
	require.NoError(s.T(), err)
	req.Header.Set("Authorization", "Bearer "+loginData.AccessToken)

	// リクエストの実行
	w := integration.ExecuteRequest(req)

	// レスポンスの検証
	require.Equal(s.T(), http.StatusUnauthorized, w.Code)

	var response errors.APIError
	integration.ParseResponse(s.T(), w, &response)

	require.Equal(s.T(), errors.ErrCodeAuthentication, response.Code)
	require.Equal(s.T(), "Current password is incorrect", response.Message)
}
