package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"erp-access-control-go/internal/middleware"
	"erp-access-control-go/internal/services"
	"erp-access-control-go/models"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/pkg/logger"
)

// UserHandler ユーザー管理ハンドラー
type UserHandler struct {
	userService *services.UserService
	logger      *logger.Logger
}

// NewUserHandler 新しいユーザーハンドラーを作成
func NewUserHandler(userService *services.UserService, logger *logger.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// CreateUser ユーザーを作成
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req services.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid create user request format", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewValidationError("request", "Invalid request format"))
		return
	}

	// リクエストユーザーID取得（監査ログ用）
	requestUserID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		h.logger.Warn("Failed to get current user ID", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewAuthenticationError("Authentication required"))
		return
	}

	h.logger.Info("Create user request", map[string]interface{}{
		"email":           req.Email,
		"department_id":   req.DepartmentID,
		"primary_role_id": req.PrimaryRoleID,
		"requested_by":    requestUserID,
		"ip":              c.ClientIP(),
	})

	user, err := h.userService.CreateUser(req)
	if err != nil {
		h.logger.Error("Failed to create user", err, map[string]interface{}{
			"email":        req.Email,
			"requested_by": requestUserID,
			"ip":           c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("User created successfully", map[string]interface{}{
		"user_id":      user.ID,
		"email":        user.Email,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	c.JSON(http.StatusCreated, user)
}

// GetUsers ユーザー一覧を取得
func (h *UserHandler) GetUsers(c *gin.Context) {
	var filters services.UserListFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		h.logger.Warn("Invalid get users query parameters", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewValidationError("query", "Invalid query parameters"))
		return
	}

	h.logger.Info("Get users request", map[string]interface{}{
		"filters": filters,
		"ip":      c.ClientIP(),
	})

	users, err := h.userService.GetUsers(filters)
	if err != nil {
		h.logger.Error("Failed to get users", err, map[string]interface{}{
			"filters": filters,
			"ip":      c.ClientIP(),
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUser ユーザー詳細を取得
func (h *UserHandler) GetUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Warn("Invalid user ID format", map[string]interface{}{
			"user_id": userIDStr,
			"error":   err.Error(),
			"ip":      c.ClientIP(),
		})
		c.Error(errors.NewValidationError("id", "Invalid UUID format"))
		return
	}

	h.logger.Info("Get user request", map[string]interface{}{
		"user_id": userID,
		"ip":      c.ClientIP(),
	})

	user, err := h.userService.GetUser(userID)
	if err != nil {
		h.logger.Warn("Failed to get user", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
			"ip":      c.ClientIP(),
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser ユーザー情報を更新
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Warn("Invalid user ID format", map[string]interface{}{
			"user_id": userIDStr,
			"error":   err.Error(),
			"ip":      c.ClientIP(),
		})
		c.Error(errors.NewValidationError("id", "Invalid UUID format"))
		return
	}

	var req services.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid update user request format", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
			"ip":      c.ClientIP(),
		})
		c.Error(errors.NewValidationError("request", "Invalid request format"))
		return
	}

	// リクエストユーザーID取得（監査ログ用）
	requestUserID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		h.logger.Warn("Failed to get current user ID", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewAuthenticationError("Authentication required"))
		return
	}

	h.logger.Info("Update user request", map[string]interface{}{
		"user_id":      userID,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	user, err := h.userService.UpdateUser(userID, req)
	if err != nil {
		h.logger.Error("Failed to update user", err, map[string]interface{}{
			"user_id":      userID,
			"requested_by": requestUserID,
			"ip":           c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("User updated successfully", map[string]interface{}{
		"user_id":      userID,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	c.JSON(http.StatusOK, user)
}

// DeleteUser ユーザーを削除
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Warn("Invalid user ID format", map[string]interface{}{
			"user_id": userIDStr,
			"error":   err.Error(),
			"ip":      c.ClientIP(),
		})
		c.Error(errors.NewValidationError("id", "Invalid UUID format"))
		return
	}

	// リクエストユーザーID取得（監査ログ用）
	requestUserID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		h.logger.Warn("Failed to get current user ID", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewAuthenticationError("Authentication required"))
		return
	}

	h.logger.Info("Delete user request", map[string]interface{}{
		"user_id":      userID,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	err = h.userService.DeleteUser(userID)
	if err != nil {
		h.logger.Error("Failed to delete user", err, map[string]interface{}{
			"user_id":      userID,
			"requested_by": requestUserID,
			"ip":           c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("User deleted successfully", map[string]interface{}{
		"user_id":      userID,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	c.JSON(http.StatusNoContent, nil)
}

// ChangeUserStatus ユーザーステータスを変更
func (h *UserHandler) ChangeUserStatus(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Warn("Invalid user ID format", map[string]interface{}{
			"user_id": userIDStr,
			"error":   err.Error(),
			"ip":      c.ClientIP(),
		})
		c.Error(errors.NewValidationError("id", "Invalid UUID format"))
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=active inactive suspended"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid change status request format", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
			"ip":      c.ClientIP(),
		})
		c.Error(errors.NewValidationError("request", "Invalid request format"))
		return
	}

	// リクエストユーザーID取得（監査ログ用）
	requestUserID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		h.logger.Warn("Failed to get current user ID", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewAuthenticationError("Authentication required"))
		return
	}

	h.logger.Info("Change user status request", map[string]interface{}{
		"user_id":      userID,
		"new_status":   req.Status,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	user, err := h.userService.ChangeUserStatus(userID, models.UserStatus(req.Status))
	if err != nil {
		h.logger.Error("Failed to change user status", err, map[string]interface{}{
			"user_id":      userID,
			"new_status":   req.Status,
			"requested_by": requestUserID,
			"ip":           c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("User status changed successfully", map[string]interface{}{
		"user_id":      userID,
		"new_status":   req.Status,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	c.JSON(http.StatusOK, user)
}

// ChangePassword ユーザーのパスワードを変更（自分自身のみ）
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Warn("Invalid user ID format", map[string]interface{}{
			"user_id": userIDStr,
			"error":   err.Error(),
			"ip":      c.ClientIP(),
		})
		c.Error(errors.NewValidationError("id", "Invalid UUID format"))
		return
	}

	var req services.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid change password request format", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
			"ip":      c.ClientIP(),
		})
		c.Error(errors.NewValidationError("request", "Invalid request format"))
		return
	}

	// リクエストユーザーID取得
	requestUserID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		h.logger.Warn("Failed to get current user ID", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewAuthenticationError("Authentication required"))
		return
	}

	// 自分自身のパスワードのみ変更可能（管理者権限は別途実装予定）
	if requestUserID != userID {
		h.logger.Warn("Unauthorized password change attempt", map[string]interface{}{
			"user_id":      userID,
			"requested_by": requestUserID,
			"ip":           c.ClientIP(),
		})
		c.Error(errors.NewAuthorizationError("You can only change your own password"))
		return
	}

	h.logger.Info("Change password request", map[string]interface{}{
		"user_id": userID,
		"ip":      c.ClientIP(),
	})

	err = h.userService.ChangePassword(userID, req)
	if err != nil {
		h.logger.Error("Failed to change password", err, map[string]interface{}{
			"user_id": userID,
			"ip":      c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("Password changed successfully", map[string]interface{}{
		"user_id": userID,
		"ip":      c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}
