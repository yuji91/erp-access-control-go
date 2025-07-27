package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"erp-access-control-go/internal/middleware"
	"erp-access-control-go/internal/services"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/pkg/logger"
)

// AuthHandler 認証ハンドラー
type AuthHandler struct {
	authService *services.AuthService
	logger      *logger.Logger
}

// NewAuthHandler 認証ハンドラーを新規作成
func NewAuthHandler(authService *services.AuthService, logger *logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// LoginRequest ログインリクエスト
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"password123"`
}

// LoginResponse ログインレスポンス
type LoginResponse struct {
	AccessToken string              `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType   string              `json:"token_type" example:"Bearer"`
	ExpiresIn   int64               `json:"expires_in" example:"900"`
	User        *services.UserInfo  `json:"user"`
	Permissions []string            `json:"permissions" example:"['user:read','user:write']"`
	ActiveRoles []services.RoleInfo `json:"active_roles"`
	PrimaryRole *services.RoleInfo  `json:"primary_role"`
	HighestRole *services.RoleInfo  `json:"highest_role"`
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

// Login ユーザーログイン処理
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid login request format", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewValidationError("request", "Invalid request format"))
		return
	}

	h.logger.Info("Login attempt", map[string]interface{}{
		"email": req.Email,
		"ip":    c.ClientIP(),
	})

	// ログイン処理
	userInfo, accessToken, err := h.authService.LoginWithCredentials(req.Email, req.Password)
	if err != nil {
		h.logger.Warn("Login failed", map[string]interface{}{
			"email": req.Email,
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})

		switch err {
		case errors.ErrInvalidCredentials:
			c.Error(errors.NewAuthenticationError("Invalid email or password"))
		case errors.ErrUserNotFound:
			c.Error(errors.NewAuthenticationError("User not found"))
		case errors.ErrUserInactive:
			c.Error(errors.NewAuthenticationError("User account is inactive"))
		default:
			c.Error(errors.NewInternalError("Failed to process login"))
		}
		return
	}

	// 権限情報を取得
	permissions, err := h.authService.GetUserPermissions(userInfo.ID)
	if err != nil {
		h.logger.Error("Failed to get user permissions", err, map[string]interface{}{
			"user_id": userInfo.ID,
		})
		c.Error(errors.NewInternalError("Failed to get user permissions"))
		return
	}

	h.logger.Info("Login successful", map[string]interface{}{
		"user_id": userInfo.ID,
		"email":   userInfo.Email,
		"ip":      c.ClientIP(),
	})

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

// RefreshToken トークンリフレッシュ処理
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid refresh token request format", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewValidationError("request", "Invalid request format"))
		return
	}

	h.logger.Info("Token refresh attempt", map[string]interface{}{
		"ip": c.ClientIP(),
	})

	// トークンリフレッシュ処理
	loginResp, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		h.logger.Warn("Token refresh failed", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})

		switch err {
		case errors.ErrInvalidToken:
			c.Error(errors.NewAuthenticationError("Invalid token"))
		case errors.ErrTokenExpired:
			c.Error(errors.NewAuthenticationError("Token has expired"))
		case errors.ErrTokenRevoked:
			c.Error(errors.NewAuthenticationError("Token has been revoked"))
		default:
			c.Error(errors.NewInternalError("Failed to refresh token"))
		}
		return
	}

	h.logger.Info("Token refresh successful", map[string]interface{}{
		"ip": c.ClientIP(),
	})

	// レスポンス作成
	response := &RefreshResponse{
		AccessToken: loginResp.Token,
		TokenType:   "Bearer",
		ExpiresIn:   900, // 15分
	}

	c.JSON(http.StatusOK, response)
}

// Logout ログアウト処理
func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid logout request format", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewValidationError("request", "Invalid request format"))
		return
	}

	h.logger.Info("Logout attempt", map[string]interface{}{
		"ip": c.ClientIP(),
	})

	// ログアウト処理（トークン無効化）
	err := h.authService.LogoutWithToken(req.RefreshToken)
	if err != nil {
		h.logger.Warn("Logout failed", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})

		switch err {
		case errors.ErrInvalidToken:
			c.Error(errors.NewAuthenticationError("Invalid token"))
		case errors.ErrTokenRevoked:
			c.JSON(http.StatusOK, gin.H{
				"message":   "Already logged out",
				"status":    "success",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			return
		default:
			c.Error(errors.NewInternalError("Failed to process logout"))
		}
		return
	}

	h.logger.Info("Logout successful", map[string]interface{}{
		"ip": c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message":   "Successfully logged out",
		"status":    "success",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetProfile プロフィール取得処理
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		h.logger.Warn("Failed to get current user ID", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewAuthenticationError("Authentication required"))
		return
	}

	h.logger.Info("Profile request", map[string]interface{}{
		"user_id": userID,
		"ip":      c.ClientIP(),
	})

	// ユーザー情報を取得
	userInfo, err := h.authService.GetUserInfo(userID)
	if err != nil {
		h.logger.Error("Failed to get user info", err, map[string]interface{}{
			"user_id": userID,
			"ip":      c.ClientIP(),
		})

		switch err {
		case errors.ErrUserNotFound:
			c.Error(errors.NewNotFoundError("User", "User not found"))
		default:
			c.Error(errors.NewInternalError("Failed to get user information"))
		}
		return
	}

	c.JSON(http.StatusOK, userInfo)
}

// ChangePasswordRequest パスワード変更リクエスト
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required" example:"oldpassword123"`
	NewPassword     string `json:"new_password" binding:"required,min=6" example:"newpassword123"`
}

// ChangePassword パスワード変更処理
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid password change request format", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewValidationError("request", "Invalid request format"))
		return
	}

	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		h.logger.Warn("Failed to get current user ID", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewAuthenticationError("Authentication required"))
		return
	}

	h.logger.Info("Password change attempt", map[string]interface{}{
		"user_id": userID,
		"ip":      c.ClientIP(),
	})

	// パスワード変更処理
	err = h.authService.ChangePassword(userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		h.logger.Warn("Password change failed", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
			"ip":      c.ClientIP(),
		})

		switch err {
		case errors.ErrInvalidCredentials:
			c.Error(errors.NewAuthenticationError("Current password is incorrect"))
		case errors.ErrUserNotFound:
			c.Error(errors.NewNotFoundError("User", "User not found"))
		default:
			c.Error(errors.NewInternalError("Failed to change password"))
		}
		return
	}

	h.logger.Info("Password change successful", map[string]interface{}{
		"user_id": userID,
		"ip":      c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message":   "Password successfully changed",
		"status":    "success",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
