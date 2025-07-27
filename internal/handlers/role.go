package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"erp-access-control-go/internal/middleware"
	"erp-access-control-go/internal/services"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/pkg/logger"
)

// RoleHandler ロール管理ハンドラー
type RoleHandler struct {
	roleService *services.RoleService
	logger      *logger.Logger
}

// NewRoleHandler 新しいロールハンドラーを作成
func NewRoleHandler(roleService *services.RoleService, logger *logger.Logger) *RoleHandler {
	return &RoleHandler{
		roleService: roleService,
		logger:      logger,
	}
}

// CreateRole ロールを作成
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req services.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid create role request format", map[string]interface{}{
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

	h.logger.Info("Create role request", map[string]interface{}{
		"name":           req.Name,
		"parent_id":      req.ParentID,
		"permission_ids": req.PermissionIDs,
		"requested_by":   requestUserID,
		"ip":             c.ClientIP(),
	})

	role, err := h.roleService.CreateRole(req)
	if err != nil {
		h.logger.Error("Failed to create role", err, map[string]interface{}{
			"name":         req.Name,
			"requested_by": requestUserID,
			"ip":           c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("Role created successfully", map[string]interface{}{
		"role_id":      role.ID,
		"name":         role.Name,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	c.JSON(http.StatusCreated, role)
}

// GetRoles ロール一覧を取得
func (h *RoleHandler) GetRoles(c *gin.Context) {
	// クエリパラメータ解析
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	parentIDStr := c.Query("parent_id")
	permissionIDStr := c.Query("permission_id")
	search := c.Query("search")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		h.logger.Warn("Invalid page parameter", map[string]interface{}{
			"page":  pageStr,
			"error": err,
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewValidationError("page", "Invalid page number"))
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		h.logger.Warn("Invalid limit parameter", map[string]interface{}{
			"limit": limitStr,
			"error": err,
			"ip":    c.ClientIP(),
		})
		c.Error(errors.NewValidationError("limit", "Invalid limit number (1-100)"))
		return
	}

	// 親ロールID解析
	var parentID *uuid.UUID
	if parentIDStr != "" {
		parsedParentID, err := uuid.Parse(parentIDStr)
		if err != nil {
			h.logger.Warn("Invalid parent_id parameter", map[string]interface{}{
				"parent_id": parentIDStr,
				"error":     err.Error(),
				"ip":        c.ClientIP(),
			})
			c.Error(errors.NewValidationError("parent_id", "Invalid UUID format"))
			return
		}
		parentID = &parsedParentID
	}

	// 権限ID解析
	var permissionID *uuid.UUID
	if permissionIDStr != "" {
		parsedPermissionID, err := uuid.Parse(permissionIDStr)
		if err != nil {
			h.logger.Warn("Invalid permission_id parameter", map[string]interface{}{
				"permission_id": permissionIDStr,
				"error":         err.Error(),
				"ip":            c.ClientIP(),
			})
			c.Error(errors.NewValidationError("permission_id", "Invalid UUID format"))
			return
		}
		permissionID = &parsedPermissionID
	}

	// リクエストユーザーID取得（監査ログ用）
	requestUserID, _ := middleware.GetCurrentUserID(c)

	h.logger.Info("Get roles request", map[string]interface{}{
		"page":          page,
		"limit":         limit,
		"parent_id":     parentID,
		"permission_id": permissionID,
		"search":        search,
		"requested_by":  requestUserID,
		"ip":            c.ClientIP(),
	})

	roles, err := h.roleService.GetRoles(page, limit, parentID, permissionID, search)
	if err != nil {
		h.logger.Error("Failed to get roles", err, map[string]interface{}{
			"page":         page,
			"limit":        limit,
			"requested_by": requestUserID,
			"ip":           c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("Roles retrieved successfully", map[string]interface{}{
		"count":        len(roles.Roles),
		"total":        roles.Total,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	c.JSON(http.StatusOK, roles)
}

// GetRole ロール詳細を取得
func (h *RoleHandler) GetRole(c *gin.Context) {
	roleIDStr := c.Param("id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		h.logger.Warn("Invalid role ID parameter", map[string]interface{}{
			"role_id": roleIDStr,
			"error":   err.Error(),
			"ip":      c.ClientIP(),
		})
		c.Error(errors.NewValidationError("id", "Invalid UUID format"))
		return
	}

	// リクエストユーザーID取得（監査ログ用）
	requestUserID, _ := middleware.GetCurrentUserID(c)

	h.logger.Info("Get role request", map[string]interface{}{
		"role_id":      roleID,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	role, err := h.roleService.GetRole(roleID)
	if err != nil {
		h.logger.Error("Failed to get role", err, map[string]interface{}{
			"role_id":      roleID,
			"requested_by": requestUserID,
			"ip":           c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("Role retrieved successfully", map[string]interface{}{
		"role_id":      role.ID,
		"name":         role.Name,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	c.JSON(http.StatusOK, role)
}

// UpdateRole ロールを更新
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	roleIDStr := c.Param("id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		h.logger.Warn("Invalid role ID parameter", map[string]interface{}{
			"role_id": roleIDStr,
			"error":   err.Error(),
			"ip":      c.ClientIP(),
		})
		c.Error(errors.NewValidationError("id", "Invalid UUID format"))
		return
	}

	var req services.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid update role request format", map[string]interface{}{
			"role_id": roleID,
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

	h.logger.Info("Update role request", map[string]interface{}{
		"role_id":      roleID,
		"name":         req.Name,
		"parent_id":    req.ParentID,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	role, err := h.roleService.UpdateRole(roleID, req)
	if err != nil {
		h.logger.Error("Failed to update role", err, map[string]interface{}{
			"role_id":      roleID,
			"requested_by": requestUserID,
			"ip":           c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("Role updated successfully", map[string]interface{}{
		"role_id":      role.ID,
		"name":         role.Name,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	c.JSON(http.StatusOK, role)
}

// DeleteRole ロールを削除
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	roleIDStr := c.Param("id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		h.logger.Warn("Invalid role ID parameter", map[string]interface{}{
			"role_id": roleIDStr,
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

	h.logger.Info("Delete role request", map[string]interface{}{
		"role_id":      roleID,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	err = h.roleService.DeleteRole(roleID)
	if err != nil {
		h.logger.Error("Failed to delete role", err, map[string]interface{}{
			"role_id":      roleID,
			"requested_by": requestUserID,
			"ip":           c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("Role deleted successfully", map[string]interface{}{
		"role_id":      roleID,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "Role deleted successfully"})
}

// AssignPermissions ロールに権限を割り当て
func (h *RoleHandler) AssignPermissions(c *gin.Context) {
	roleIDStr := c.Param("id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		h.logger.Warn("Invalid role ID parameter", map[string]interface{}{
			"role_id": roleIDStr,
			"error":   err.Error(),
			"ip":      c.ClientIP(),
		})
		c.Error(errors.NewValidationError("id", "Invalid UUID format"))
		return
	}

	var req services.AssignPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid assign permissions request format", map[string]interface{}{
			"role_id": roleID,
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

	h.logger.Info("Assign permissions request", map[string]interface{}{
		"role_id":        roleID,
		"permission_ids": req.PermissionIDs,
		"replace":        req.Replace,
		"requested_by":   requestUserID,
		"ip":             c.ClientIP(),
	})

	permissions, err := h.roleService.AssignPermissions(roleID, req)
	if err != nil {
		h.logger.Error("Failed to assign permissions", err, map[string]interface{}{
			"role_id":      roleID,
			"requested_by": requestUserID,
			"ip":           c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("Permissions assigned successfully", map[string]interface{}{
		"role_id":      roleID,
		"count":        len(permissions.AllPermissions),
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	c.JSON(http.StatusOK, permissions)
}

// GetRolePermissions ロールの権限一覧を取得
func (h *RoleHandler) GetRolePermissions(c *gin.Context) {
	roleIDStr := c.Param("id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		h.logger.Warn("Invalid role ID parameter", map[string]interface{}{
			"role_id": roleIDStr,
			"error":   err.Error(),
			"ip":      c.ClientIP(),
		})
		c.Error(errors.NewValidationError("id", "Invalid UUID format"))
		return
	}

	// リクエストユーザーID取得（監査ログ用）
	requestUserID, _ := middleware.GetCurrentUserID(c)

	h.logger.Info("Get role permissions request", map[string]interface{}{
		"role_id":      roleID,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	permissions, err := h.roleService.GetRolePermissions(roleID)
	if err != nil {
		h.logger.Error("Failed to get role permissions", err, map[string]interface{}{
			"role_id":      roleID,
			"requested_by": requestUserID,
			"ip":           c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("Role permissions retrieved successfully", map[string]interface{}{
		"role_id":         roleID,
		"direct_count":    len(permissions.DirectPermissions),
		"inherited_count": len(permissions.InheritedPermissions),
		"total_count":     len(permissions.AllPermissions),
		"requested_by":    requestUserID,
		"ip":              c.ClientIP(),
	})

	c.JSON(http.StatusOK, permissions)
}

// GetRoleHierarchy ロール階層ツリーを取得
func (h *RoleHandler) GetRoleHierarchy(c *gin.Context) {
	// リクエストユーザーID取得（監査ログ用）
	requestUserID, _ := middleware.GetCurrentUserID(c)

	h.logger.Info("Get role hierarchy request", map[string]interface{}{
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	hierarchy, err := h.roleService.GetRoleHierarchy()
	if err != nil {
		h.logger.Error("Failed to get role hierarchy", err, map[string]interface{}{
			"requested_by": requestUserID,
			"ip":           c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("Role hierarchy retrieved successfully", map[string]interface{}{
		"root_count":   len(hierarchy.Roles),
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	c.JSON(http.StatusOK, hierarchy)
}
