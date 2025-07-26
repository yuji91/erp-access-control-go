package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"erp-access-control-go/internal/services"
	"erp-access-control-go/pkg/errors"
)

// AuthHandler 認証ハンドラー
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler 認証ハンドラーを新規作成
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// LoginRequest ログインリクエスト
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"password123"`
}

// LoginResponse ログインレスポンス
type LoginResponse struct {
	AccessToken  string                 `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType    string                 `json:"token_type" example:"Bearer"`
	ExpiresIn    int64                  `json:"expires_in" example:"900"`
	User         *services.UserInfo     `json:"user"`
	Permissions  []string               `json:"permissions" example:"['user:read','user:write']"`
	ActiveRoles  []services.RoleInfo    `json:"active_roles"`
	PrimaryRole  *services.RoleInfo     `json:"primary_role"`
	HighestRole  *services.RoleInfo     `json:"highest_role"`
}

// RefreshRequest リフレッシュリクエスト
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// RefreshResponse リフレッシュレスポンス
type RefreshResponse struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType   string `json:"token_type" example:"Bearer"`
	ExpiresIn   int64  `json:"expires_in" example:"900"`
}

// LogoutRequest ログアウトリクエスト
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// Login godoc
// @Summary ユーザーログイン
// @Description メールアドレスとパスワードでログインし、JWTトークンを取得
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "ログイン情報"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	fmt.Printf("DEBUG: Login handler called\n")
	
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("DEBUG: JSON binding error: %v\n", err)
		c.JSON(http.StatusBadRequest, errors.NewValidationError("リクエストの形式が正しくありません", err.Error()))
		return
	}
	
	fmt.Printf("DEBUG: Login request received for email: %s\n", req.Email)

	// ログイン処理
	userInfo, accessToken, err := h.authService.LoginWithCredentials(req.Email, req.Password)
	if err != nil {
		fmt.Printf("DEBUG: Login error occurred: %v\n", err)
		fmt.Printf("DEBUG: Error type: %T\n", err)
		
		switch err {
		case errors.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, errors.NewUnauthorizedError("メールアドレスまたはパスワードが正しくありません"))
		case errors.ErrUserNotFound:
			c.JSON(http.StatusUnauthorized, errors.NewUnauthorizedError("ユーザーが見つかりません"))
		case errors.ErrUserInactive:
			c.JSON(http.StatusUnauthorized, errors.NewUnauthorizedError("ユーザーアカウントが無効です"))
		default:
			fmt.Printf("DEBUG: Unhandled error in login: %v\n", err)
			c.JSON(http.StatusInternalServerError, errors.NewInternalServerError("ログイン処理中にエラーが発生しました"))
		}
		return
	}
	
	fmt.Printf("DEBUG: Login successful for user: %s\n", userInfo.Email)

	// 権限情報を取得
	permissions, err := h.authService.GetUserPermissions(userInfo.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.NewInternalServerError("権限情報の取得に失敗しました"))
		return
	}

	// レスポンス作成
	response := &LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   900, // 15分
		User:        userInfo,
		Permissions: permissions,
		ActiveRoles: userInfo.ActiveRoles,
		PrimaryRole: userInfo.PrimaryRole,
		HighestRole: userInfo.HighestRole,
	}

	c.JSON(http.StatusOK, response)
}

// RefreshToken godoc
// @Summary トークンリフレッシュ
// @Description リフレッシュトークンを使用して新しいアクセストークンを取得
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "リフレッシュトークン"
// @Success 200 {object} RefreshResponse
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.NewValidationError("リクエストの形式が正しくありません", err.Error()))
		return
	}

	// トークンリフレッシュ処理
	loginResp, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		switch err {
		case errors.ErrInvalidToken:
			c.JSON(http.StatusUnauthorized, errors.NewUnauthorizedError("無効なトークンです"))
		case errors.ErrTokenExpired:
			c.JSON(http.StatusUnauthorized, errors.NewUnauthorizedError("トークンの有効期限が切れています"))
		case errors.ErrTokenRevoked:
			c.JSON(http.StatusUnauthorized, errors.NewUnauthorizedError("トークンが取り消されています"))
		default:
			c.JSON(http.StatusInternalServerError, errors.NewInternalServerError("トークンリフレッシュ中にエラーが発生しました"))
		}
		return
	}

	// レスポンス作成
	response := &RefreshResponse{
		AccessToken: loginResp.Token,
		TokenType:   "Bearer",
		ExpiresIn:   900, // 15分
	}

	c.JSON(http.StatusOK, response)
}

// Logout godoc
// @Summary ユーザーログアウト
// @Description リフレッシュトークンを無効化してログアウト
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LogoutRequest true "リフレッシュトークン"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.NewValidationError("リクエストの形式が正しくありません", err.Error()))
		return
	}

	// ログアウト処理（トークン無効化）
	err := h.authService.LogoutWithToken(req.RefreshToken)
	if err != nil {
		switch err {
		case errors.ErrInvalidToken:
			c.JSON(http.StatusUnauthorized, errors.NewUnauthorizedError("無効なトークンです"))
		case errors.ErrTokenRevoked:
			c.JSON(http.StatusOK, gin.H{
				"message": "既にログアウト済みです",
				"status":  "success",
			})
			return
		default:
			c.JSON(http.StatusInternalServerError, errors.NewInternalServerError("ログアウト処理中にエラーが発生しました"))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ログアウトが完了しました",
		"status":  "success",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetProfile godoc
// @Summary ユーザープロフィール取得
// @Description 現在のユーザー情報と権限を取得
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} services.UserInfo
// @Failure 401 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /api/v1/auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	// コンテキストからユーザーIDを取得
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, errors.NewUnauthorizedError("認証情報が見つかりません"))
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewValidationError("無効なユーザーIDです"))
		return
	}

	// ユーザー情報を取得
	userInfo, err := h.authService.GetUserInfo(userID)
	if err != nil {
		switch err {
		case errors.ErrUserNotFound:
			c.JSON(http.StatusNotFound, errors.NewNotFoundError("ユーザーが見つかりません"))
		default:
			c.JSON(http.StatusInternalServerError, errors.NewInternalServerError("ユーザー情報の取得に失敗しました"))
		}
		return
	}

	c.JSON(http.StatusOK, userInfo)
}

// ChangePassword godoc
// @Summary パスワード変更
// @Description 現在のパスワードを確認して新しいパスワードに変更
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "パスワード変更情報"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /api/v1/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.NewValidationError("リクエストの形式が正しくありません", err.Error()))
		return
	}

	// コンテキストからユーザーIDを取得
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, errors.NewUnauthorizedError("認証情報が見つかりません"))
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewValidationError("無効なユーザーIDです"))
		return
	}

	// パスワード変更処理
	err = h.authService.ChangePassword(userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch err {
		case errors.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, errors.NewUnauthorizedError("現在のパスワードが正しくありません"))
		case errors.ErrUserNotFound:
			c.JSON(http.StatusNotFound, errors.NewNotFoundError("ユーザーが見つかりません"))
		default:
			c.JSON(http.StatusInternalServerError, errors.NewInternalServerError("パスワード変更中にエラーが発生しました"))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "パスワードが正常に変更されました",
		"status":  "success",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ChangePasswordRequest パスワード変更リクエスト
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required" example:"oldpassword123"`
	NewPassword     string `json:"new_password" binding:"required,min=6" example:"newpassword123"`
} 