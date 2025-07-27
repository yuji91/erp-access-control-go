package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"erp-access-control-go/internal/middleware"
	"erp-access-control-go/internal/services"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/pkg/logger"
)

// PermissionHandler 権限管理ハンドラー
type PermissionHandler struct {
	permissionService *services.PermissionService
	logger            *logger.Logger
}

// NewPermissionHandler 新しい権限ハンドラーを作成
func NewPermissionHandler(permissionService *services.PermissionService, logger *logger.Logger) *PermissionHandler {
	return &PermissionHandler{
		permissionService: permissionService,
		logger:            logger,
	}
}

// CreatePermission 権限を作成
func (h *PermissionHandler) CreatePermission(c *gin.Context) {
	var req services.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid create permission request format", map[string]interface{}{
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

	h.logger.Info("Creating permission", map[string]interface{}{
		"module":        req.Module,
		"action":        req.Action,
		"requestUserID": requestUserID,
		"ip":            c.ClientIP(),
	})

	permission, err := h.permissionService.CreatePermission(req)
	if err != nil {
		h.logger.Error("Failed to create permission", err, map[string]interface{}{
			"module":        req.Module,
			"action":        req.Action,
			"requestUserID": requestUserID,
			"ip":            c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("Permission created successfully", map[string]interface{}{
		"permissionID":  permission.ID,
		"module":        permission.Module,
		"action":        permission.Action,
		"requestUserID": requestUserID,
		"ip":            c.ClientIP(),
	})

	c.JSON(http.StatusCreated, gin.H{
		"permission": permission,
	})
}

// GetPermissions 権限一覧を取得
func (h *PermissionHandler) GetPermissions(c *gin.Context) {
	var req services.GetPermissionsRequest

	// クエリパラメータをバインド
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Warn("Invalid get permissions query parameters", map[string]interface{}{
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewValidationError("query", "Invalid query parameters"))
		return
	}

	// デフォルト値設定
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}

	// used_by_role パラメータのUUID検証
	if req.UsedByRole != "" {
		if _, err := uuid.Parse(req.UsedByRole); err != nil {
			h.logger.Warn("Invalid used_by_role UUID format", map[string]interface{}{
				"used_by_role": req.UsedByRole,
				"error":        err.Error(),
				"ip":           c.ClientIP(),
			})
			c.Error(errors.NewValidationError("used_by_role", "Invalid UUID format"))
			return
		}
	}

	// リクエストユーザーID取得（監査ログ用）
	requestUserID, _ := middleware.GetCurrentUserID(c)

	h.logger.Info("Getting permissions list", map[string]interface{}{
		"page":          req.Page,
		"limit":         req.Limit,
		"module":        req.Module,
		"action":        req.Action,
		"used_by_role":  req.UsedByRole,
		"search":        req.Search,
		"requestUserID": requestUserID,
		"ip":            c.ClientIP(),
	})

	permissions, err := h.permissionService.GetPermissions(req)
	if err != nil {
		h.logger.Error("Failed to get permissions list", err, map[string]interface{}{
			"requestUserID": requestUserID,
			"ip":            c.ClientIP(),
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, permissions)
}

// GetPermission 権限詳細を取得
func (h *PermissionHandler) GetPermission(c *gin.Context) {
	idStr := c.Param("id")
	permissionID, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Warn("Invalid permission ID format", map[string]interface{}{
			"id":    idStr,
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewValidationError("id", "Invalid UUID format"))
		return
	}

	// リクエストユーザーID取得（監査ログ用）
	requestUserID, _ := middleware.GetCurrentUserID(c)

	h.logger.Info("Getting permission details", map[string]interface{}{
		"permissionID":  permissionID,
		"requestUserID": requestUserID,
		"ip":            c.ClientIP(),
	})

	permission, err := h.permissionService.GetPermission(permissionID)
	if err != nil {
		h.logger.Error("Failed to get permission", err, map[string]interface{}{
			"permissionID":  permissionID,
			"requestUserID": requestUserID,
			"ip":            c.ClientIP(),
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"permission": permission,
	})
}

// UpdatePermission 権限を更新
func (h *PermissionHandler) UpdatePermission(c *gin.Context) {
	idStr := c.Param("id")
	permissionID, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Warn("Invalid permission ID format", map[string]interface{}{
			"id":    idStr,
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewValidationError("id", "Invalid UUID format"))
		return
	}

	var req services.UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid update permission request format", map[string]interface{}{
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

	h.logger.Info("Updating permission", map[string]interface{}{
		"permissionID":  permissionID,
		"requestUserID": requestUserID,
		"ip":            c.ClientIP(),
	})

	permission, err := h.permissionService.UpdatePermission(permissionID, req)
	if err != nil {
		h.logger.Error("Failed to update permission", err, map[string]interface{}{
			"permissionID":  permissionID,
			"requestUserID": requestUserID,
			"ip":            c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("Permission updated successfully", map[string]interface{}{
		"permissionID":  permissionID,
		"requestUserID": requestUserID,
		"ip":            c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{
		"permission": permission,
	})
}

// DeletePermission 権限を削除
func (h *PermissionHandler) DeletePermission(c *gin.Context) {
	idStr := c.Param("id")
	permissionID, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Warn("Invalid permission ID format", map[string]interface{}{
			"id":    idStr,
			"error": err.Error(),
			"ip":    c.ClientIP(),
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

	h.logger.Info("Deleting permission", map[string]interface{}{
		"permissionID":  permissionID,
		"requestUserID": requestUserID,
		"ip":            c.ClientIP(),
	})

	err = h.permissionService.DeletePermission(permissionID)
	if err != nil {
		h.logger.Error("Failed to delete permission", err, map[string]interface{}{
			"permissionID":  permissionID,
			"requestUserID": requestUserID,
			"ip":            c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("Permission deleted successfully", map[string]interface{}{
		"permissionID":  permissionID,
		"requestUserID": requestUserID,
		"ip":            c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Permission deleted successfully",
	})
}

// GetPermissionMatrix 権限マトリックスを取得
func (h *PermissionHandler) GetPermissionMatrix(c *gin.Context) {
	// リクエストユーザーID取得（監査ログ用）
	requestUserID, _ := middleware.GetCurrentUserID(c)

	h.logger.Info("Getting permission matrix", map[string]interface{}{
		"requestUserID": requestUserID,
		"ip":            c.ClientIP(),
	})

	matrix, err := h.permissionService.GetPermissionMatrix()
	if err != nil {
		h.logger.Error("Failed to get permission matrix", err, map[string]interface{}{
			"requestUserID": requestUserID,
			"ip":            c.ClientIP(),
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, matrix)
}

// GetPermissionsByModule モジュール別権限を取得
func (h *PermissionHandler) GetPermissionsByModule(c *gin.Context) {
	module := c.Param("module")
	if module == "" {
		h.logger.Warn("Module parameter is required", map[string]interface{}{
			"ip": c.ClientIP(),
		})
		c.Error(errors.NewValidationError("module", "Module parameter is required"))
		return
	}

	// リクエストユーザーID取得（監査ログ用）
	requestUserID, _ := middleware.GetCurrentUserID(c)

	h.logger.Info("Getting permissions by module", map[string]interface{}{
		"module":        module,
		"requestUserID": requestUserID,
		"ip":            c.ClientIP(),
	})

	permissions, err := h.permissionService.GetPermissionsByModule(module)
	if err != nil {
		h.logger.Error("Failed to get permissions by module", err, map[string]interface{}{
			"module":        module,
			"requestUserID": requestUserID,
			"ip":            c.ClientIP(),
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"module":      module,
		"permissions": permissions,
	})
}

// GetRolesByPermission 権限を持つロール一覧を取得
func (h *PermissionHandler) GetRolesByPermission(c *gin.Context) {
	idStr := c.Param("id")
	permissionID, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Warn("Invalid permission ID format", map[string]interface{}{
			"id":    idStr,
			"error": err.Error(),
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewValidationError("id", "Invalid UUID format"))
		return
	}

	// リクエストユーザーID取得（監査ログ用）
	requestUserID, _ := middleware.GetCurrentUserID(c)

	h.logger.Info("Getting roles by permission", map[string]interface{}{
		"permissionID":  permissionID,
		"requestUserID": requestUserID,
		"ip":            c.ClientIP(),
	})

	roles, err := h.permissionService.GetRolesByPermission(permissionID)
	if err != nil {
		h.logger.Error("Failed to get roles by permission", err, map[string]interface{}{
			"permissionID":  permissionID,
			"requestUserID": requestUserID,
			"ip":            c.ClientIP(),
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"permission_id": permissionID,
		"roles":         roles,
	})
}
