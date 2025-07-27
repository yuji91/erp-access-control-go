package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"erp-access-control-go/internal/middleware"
	"erp-access-control-go/internal/services"
	"erp-access-control-go/models"
	"erp-access-control-go/pkg/errors"
)

// UserRoleHandler 複数ロール管理ハンドラー
type UserRoleHandler struct {
	userRoleService *services.UserRoleService
}

// NewUserRoleHandler 新しいユーザーロールハンドラーを作成
func NewUserRoleHandler(userRoleService *services.UserRoleService) *UserRoleHandler {
	return &UserRoleHandler{
		userRoleService: userRoleService,
	}
}

// AssignRoleRequest ロール割り当てリクエスト
type AssignRoleRequest struct {
	UserID    uuid.UUID  `json:"user_id" binding:"required"`
	RoleID    uuid.UUID  `json:"role_id" binding:"required"`
	ValidFrom *time.Time `json:"valid_from,omitempty"`
	ValidTo   *time.Time `json:"valid_to,omitempty"`
	Priority  int        `json:"priority" binding:"min=1"`
	Reason    string     `json:"reason,omitempty"`
}

// UpdateRoleRequest ロール更新リクエスト
type UpdateRoleRequest struct {
	Priority  *int       `json:"priority,omitempty" binding:"omitempty,min=1"`
	ValidTo   *time.Time `json:"valid_to,omitempty"`
	Reason    string     `json:"reason,omitempty"`
}

// UserRoleResponse ユーザーロールレスポンス
type UserRoleResponse struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	RoleID         uuid.UUID  `json:"role_id"`
	RoleName       string     `json:"role_name"`
	ValidFrom      time.Time  `json:"valid_from"`
	ValidTo        *time.Time `json:"valid_to,omitempty"`
	Priority       int        `json:"priority"`
	IsActive       bool       `json:"is_active"`
	AssignedBy     *uuid.UUID `json:"assigned_by,omitempty"`
	AssignedReason string     `json:"assigned_reason,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// AssignRole ユーザーにロールを割り当て
// @Summary ユーザーロール割り当て
// @Description ユーザーに新しいロールを割り当てます（期限・優先度指定可能）
// @Tags UserRoles
// @Accept json
// @Produce json
// @Param request body AssignRoleRequest true "ロール割り当て情報"
// @Success 201 {object} UserRoleResponse
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 403 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /api/v1/users/roles [post]
func (h *UserRoleHandler) AssignRole(c *gin.Context) {
	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.NewValidationError(err))
		return
	}

	// リクエストユーザーID取得
	assignedBy, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, errors.ErrUnauthorized)
		return
	}

	// デフォルト値設定
	validFrom := time.Now()
	if req.ValidFrom != nil {
		validFrom = *req.ValidFrom
	}
	if req.Priority == 0 {
		req.Priority = 1
	}

	userRole, err := h.userRoleService.AssignRole(
		req.UserID,
		req.RoleID,
		validFrom,
		req.ValidTo,
		req.Priority,
		assignedBy.(uuid.UUID),
		req.Reason,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	response := h.convertToResponse(userRole)
	c.JSON(http.StatusCreated, response)
}

// RevokeRole ユーザーのロールを取り消し
// @Summary ユーザーロール取り消し
// @Description ユーザーの指定ロールを取り消します
// @Tags UserRoles
// @Accept json
// @Produce json
// @Param user_id path string true "ユーザーID"
// @Param role_id path string true "ロールID"
// @Param reason body string false "取り消し理由"
// @Success 200 {object} UserRoleResponse
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 403 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /api/v1/users/{user_id}/roles/{role_id} [delete]
func (h *UserRoleHandler) RevokeRole(c *gin.Context) {
	userIDStr := c.Param("user_id")
	roleIDStr := c.Param("role_id")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewValidationError(err))
		return
	}

	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewValidationError(err))
		return
	}

	// リクエストユーザーID取得
	revokedBy, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errors.ErrUnauthorized)
		return
	}

	// 理由を取得
	var reason string
	if body := c.Request.Body; body != nil {
		var reqBody map[string]string
		if err := c.ShouldBindJSON(&reqBody); err == nil {
			reason = reqBody["reason"]
		}
	}

	userRole, err := h.userRoleService.RevokeRole(
		userID,
		roleID,
		revokedBy,
		reason,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	response := h.convertToResponse(userRole)
	c.JSON(http.StatusOK, response)
}

// UpdateRole ユーザーロールを更新
// @Summary ユーザーロール更新
// @Description ユーザーロールの優先度や期限を更新します
// @Tags UserRoles
// @Accept json
// @Produce json
// @Param user_id path string true "ユーザーID"
// @Param role_id path string true "ロールID"
// @Param request body UpdateRoleRequest true "更新情報"
// @Success 200 {object} UserRoleResponse
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 403 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /api/v1/users/{user_id}/roles/{role_id} [patch]
func (h *UserRoleHandler) UpdateRole(c *gin.Context) {
	userIDStr := c.Param("user_id")
	roleIDStr := c.Param("role_id")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewValidationError(err))
		return
	}

	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewValidationError(err))
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.NewValidationError(err))
		return
	}

	// リクエストユーザーID取得
	updatedBy, err := middleware.GetCurrentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errors.ErrUnauthorized)
		return
	}

	userRole, err := h.userRoleService.UpdateRole(
		userID,
		roleID,
		req.Priority,
		req.ValidTo,
		updatedBy,
		req.Reason,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	response := h.convertToResponse(userRole)
	c.JSON(http.StatusOK, response)
}

// GetUserRoles ユーザーのロール一覧を取得
// @Summary ユーザーロール一覧取得
// @Description 指定ユーザーのロール一覧を取得します
// @Tags UserRoles
// @Produce json
// @Param user_id path string true "ユーザーID"
// @Param active query bool false "アクティブなロールのみ取得"
// @Success 200 {array} UserRoleResponse
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 403 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /api/v1/users/{user_id}/roles [get]
func (h *UserRoleHandler) GetUserRoles(c *gin.Context) {
	userIDStr := c.Param("user_id")
	activeOnly := c.Query("active") == "true"

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewValidationError(err))
		return
	}

	var userRoles []models.UserRole
	if activeOnly {
		userRoles, err = h.userRoleService.GetActiveUserRoles(userID)
	} else {
		userRoles, err = h.userRoleService.GetUserRoles(userID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	responses := make([]UserRoleResponse, len(userRoles))
	for i, ur := range userRoles {
		responses[i] = h.convertToResponse(&ur)
	}

	c.JSON(http.StatusOK, responses)
}

// convertToResponse UserRoleをレスポンス形式に変換
func (h *UserRoleHandler) convertToResponse(userRole *models.UserRole) UserRoleResponse {
	roleName := ""
	if userRole.Role.Name != "" {
		roleName = userRole.Role.Name
	}

	return UserRoleResponse{
		ID:             userRole.ID,
		UserID:         userRole.UserID,
		RoleID:         userRole.RoleID,
		RoleName:       roleName,
		ValidFrom:      userRole.ValidFrom,
		ValidTo:        userRole.ValidTo,
		Priority:       userRole.Priority,
		IsActive:       userRole.IsActive,
		AssignedBy:     userRole.AssignedBy,
		AssignedReason: userRole.AssignedReason,
		CreatedAt:      userRole.CreatedAt,
		UpdatedAt:      userRole.UpdatedAt,
	}
} 