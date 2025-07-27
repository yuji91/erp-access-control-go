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

// DepartmentHandler 部署管理ハンドラー
type DepartmentHandler struct {
	departmentService *services.DepartmentService
	logger            *logger.Logger
}

// NewDepartmentHandler 新しい部署ハンドラーを作成
func NewDepartmentHandler(departmentService *services.DepartmentService, logger *logger.Logger) *DepartmentHandler {
	return &DepartmentHandler{
		departmentService: departmentService,
		logger:            logger,
	}
}

// CreateDepartment 部署を作成
func (h *DepartmentHandler) CreateDepartment(c *gin.Context) {
	var req services.CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid create department request format", map[string]interface{}{
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

	h.logger.Info("Create department request", map[string]interface{}{
		"name":         req.Name,
		"parent_id":    req.ParentID,
		"requested_by": requestUserID,
		"ip":           c.ClientIP(),
	})

	department, err := h.departmentService.CreateDepartment(req)
	if err != nil {
		h.logger.Error("Failed to create department", err, map[string]interface{}{
			"name":         req.Name,
			"requested_by": requestUserID,
			"ip":           c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("Department created successfully", map[string]interface{}{
		"department_id": department.ID,
		"name":          department.Name,
		"requested_by":  requestUserID,
		"ip":            c.ClientIP(),
	})

	c.JSON(http.StatusCreated, department)
}

// GetDepartments 部署一覧を取得
func (h *DepartmentHandler) GetDepartments(c *gin.Context) {
	// クエリパラメータ解析
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	parentIDStr := c.Query("parent_id")
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
		c.Error(errors.NewValidationError("limit", "Invalid limit (1-100)"))
		return
	}

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

	h.logger.Info("Get departments request", map[string]interface{}{
		"page":      page,
		"limit":     limit,
		"parent_id": parentID,
		"search":    search,
		"ip":        c.ClientIP(),
	})

	departments, err := h.departmentService.GetDepartments(page, limit, parentID, search)
	if err != nil {
		h.logger.Error("Failed to get departments", err, map[string]interface{}{
			"page":      page,
			"limit":     limit,
			"parent_id": parentID,
			"search":    search,
			"ip":        c.ClientIP(),
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, departments)
}

// GetDepartment 部署詳細を取得
func (h *DepartmentHandler) GetDepartment(c *gin.Context) {
	departmentIDStr := c.Param("id")
	departmentID, err := uuid.Parse(departmentIDStr)
	if err != nil {
		h.logger.Warn("Invalid department ID format", map[string]interface{}{
			"department_id": departmentIDStr,
			"error":         err.Error(),
			"ip":            c.ClientIP(),
		})
		c.Error(errors.NewValidationError("id", "Invalid UUID format"))
		return
	}

	h.logger.Info("Get department request", map[string]interface{}{
		"department_id": departmentID,
		"ip":            c.ClientIP(),
	})

	department, err := h.departmentService.GetDepartment(departmentID)
	if err != nil {
		h.logger.Warn("Failed to get department", map[string]interface{}{
			"department_id": departmentID,
			"error":         err.Error(),
			"ip":            c.ClientIP(),
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, department)
}

// UpdateDepartment 部署情報を更新
func (h *DepartmentHandler) UpdateDepartment(c *gin.Context) {
	departmentIDStr := c.Param("id")
	departmentID, err := uuid.Parse(departmentIDStr)
	if err != nil {
		h.logger.Warn("Invalid department ID format", map[string]interface{}{
			"department_id": departmentIDStr,
			"error":         err.Error(),
			"ip":            c.ClientIP(),
		})
		c.Error(errors.NewValidationError("id", "Invalid UUID format"))
		return
	}

	var req services.UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid update department request format", map[string]interface{}{
			"department_id": departmentID,
			"error":         err.Error(),
			"ip":            c.ClientIP(),
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

	h.logger.Info("Update department request", map[string]interface{}{
		"department_id": departmentID,
		"requested_by":  requestUserID,
		"ip":            c.ClientIP(),
	})

	department, err := h.departmentService.UpdateDepartment(departmentID, req)
	if err != nil {
		h.logger.Error("Failed to update department", err, map[string]interface{}{
			"department_id": departmentID,
			"requested_by":  requestUserID,
			"ip":            c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("Department updated successfully", map[string]interface{}{
		"department_id": departmentID,
		"requested_by":  requestUserID,
		"ip":            c.ClientIP(),
	})

	c.JSON(http.StatusOK, department)
}

// DeleteDepartment 部署を削除
func (h *DepartmentHandler) DeleteDepartment(c *gin.Context) {
	departmentIDStr := c.Param("id")
	departmentID, err := uuid.Parse(departmentIDStr)
	if err != nil {
		h.logger.Warn("Invalid department ID format", map[string]interface{}{
			"department_id": departmentIDStr,
			"error":         err.Error(),
			"ip":            c.ClientIP(),
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

	h.logger.Info("Delete department request", map[string]interface{}{
		"department_id": departmentID,
		"requested_by":  requestUserID,
		"ip":            c.ClientIP(),
	})

	err = h.departmentService.DeleteDepartment(departmentID)
	if err != nil {
		h.logger.Error("Failed to delete department", err, map[string]interface{}{
			"department_id": departmentID,
			"requested_by":  requestUserID,
			"ip":            c.ClientIP(),
		})
		c.Error(err)
		return
	}

	h.logger.Info("Department deleted successfully", map[string]interface{}{
		"department_id": departmentID,
		"requested_by":  requestUserID,
		"ip":            c.ClientIP(),
	})

	c.JSON(http.StatusNoContent, nil)
}

// GetDepartmentHierarchy 部署階層構造を取得
func (h *DepartmentHandler) GetDepartmentHierarchy(c *gin.Context) {
	h.logger.Info("Get department hierarchy request", map[string]interface{}{
		"ip": c.ClientIP(),
	})

	hierarchy, err := h.departmentService.GetDepartmentHierarchy()
	if err != nil {
		h.logger.Error("Failed to get department hierarchy", err, map[string]interface{}{
			"ip": c.ClientIP(),
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, hierarchy)
}
