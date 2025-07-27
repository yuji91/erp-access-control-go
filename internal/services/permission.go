package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"erp-access-control-go/models"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/pkg/logger"
)

// PermissionService 権限評価・管理サービス
type PermissionService struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewPermissionService 新しい権限サービスを作成
func NewPermissionService(db *gorm.DB, logger *logger.Logger) *PermissionService {
	return &PermissionService{
		db:     db,
		logger: logger,
	}
}

// =============================================================================
// CRUD操作用の新しい構造体
// =============================================================================

// CreatePermissionRequest 権限作成リクエスト
type CreatePermissionRequest struct {
	Module      string `json:"module" binding:"required,min=2,max=50,alphanum"`
	Action      string `json:"action" binding:"required,min=2,max=50,alphanum"`
	Description string `json:"description" binding:"omitempty,max=255"`
}

// UpdatePermissionRequest 権限更新リクエスト
type UpdatePermissionRequest struct {
	Description *string `json:"description" binding:"omitempty,max=255"`
}

// GetPermissionsRequest 権限一覧取得リクエスト
type GetPermissionsRequest struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	Limit      int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Module     string `form:"module" binding:"omitempty,alphanum"`
	Action     string `form:"action" binding:"omitempty,alphanum"`
	UsedByRole string `form:"used_by_role" binding:"omitempty,uuid"`
	Search     string `form:"search" binding:"omitempty,max=100"`
}

// PermissionResponse 権限レスポンス
type PermissionResponse struct {
	ID          uuid.UUID            `json:"id"`
	Module      string               `json:"module"`
	Action      string               `json:"action"`
	Code        string               `json:"code"`
	Description string               `json:"description"`
	IsSystem    bool                 `json:"is_system"`
	CreatedAt   string               `json:"created_at"`
	Roles       []PermissionRoleInfo `json:"roles"`
	UsageStats  PermissionUsageStats `json:"usage_stats"`
}

// PermissionRoleInfo 権限に関連するロール情報
type PermissionRoleInfo struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	UserCount int       `json:"user_count"`
}

// PermissionUsageStats 権限使用統計
type PermissionUsageStats struct {
	RoleCount int    `json:"role_count"`
	UserCount int    `json:"user_count"`
	LastUsed  string `json:"last_used"`
}

// PermissionListResponse 権限一覧レスポンス
type PermissionListResponse struct {
	Permissions []PermissionResponse `json:"permissions"`
	Total       int64                `json:"total"`
	Page        int                  `json:"page"`
	Limit       int                  `json:"limit"`
	TotalPages  int                  `json:"total_pages"`
}

// PermissionMatrixResponse 権限マトリックスレスポンス
type PermissionMatrixResponse struct {
	Modules []ModuleInfo      `json:"modules"`
	Summary MatrixSummaryInfo `json:"summary"`
}

// ModuleInfo モジュール情報
type ModuleInfo struct {
	Name        string       `json:"name"`
	DisplayName string       `json:"display_name"`
	Actions     []ActionInfo `json:"actions"`
}

// ActionInfo アクション情報
type ActionInfo struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	PermissionID string   `json:"permission_id"`
	Roles        []string `json:"roles"`
}

// MatrixSummaryInfo マトリックス概要情報
type MatrixSummaryInfo struct {
	TotalPermissions  int `json:"total_permissions"`
	TotalModules      int `json:"total_modules"`
	TotalActions      int `json:"total_actions"`
	UnusedPermissions int `json:"unused_permissions"`
}

// ModulePermissionsResponse モジュール別権限レスポンス
type ModulePermissionsResponse struct {
	Module      string               `json:"module"`
	DisplayName string               `json:"display_name"`
	Permissions []PermissionResponse `json:"permissions"`
	Total       int                  `json:"total"`
}

// =============================================================================
// CRUD操作
// =============================================================================

// CreatePermission 権限作成
func (s *PermissionService) CreatePermission(req CreatePermissionRequest) (*PermissionResponse, error) {
	// 入力値検証
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// 重複チェック
	_, err := s.findPermissionByModuleAction(req.Module, req.Action)
	if err != nil {
		// NotFoundエラー以外はデータベースエラーとして扱う
		if !errors.IsNotFound(err) {
			return nil, errors.NewDatabaseError(err)
		}
		// NotFoundは正常（重複なし）
	} else {
		// 既存権限が見つかった場合は重複エラー
		return nil, errors.NewValidationError("module_action", "Permission with this module and action already exists")
	}

	// システム権限チェック
	if s.isSystemPermission(req.Module, req.Action) {
		return nil, errors.NewValidationError("system_permission", "Cannot create system permissions")
	}

	// 権限作成
	permission := &models.Permission{
		Module: req.Module,
		Action: req.Action,
	}

	if err := s.db.Create(permission).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// レスポンス作成
	response := s.convertToPermissionResponse(permission)
	response.Description = req.Description
	response.IsSystem = false

	return &response, nil
}

// GetPermission 権限詳細取得
func (s *PermissionService) GetPermission(id uuid.UUID) (*PermissionResponse, error) {
	var permission models.Permission
	if err := s.db.Preload("Roles").First(&permission, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("permission", "Permission not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	response := s.convertToPermissionResponse(&permission)
	response.Roles = s.getRoleInfoForPermission(permission.ID)
	response.UsageStats = s.getUsageStats(permission.ID)

	return &response, nil
}

// UpdatePermission 権限更新
func (s *PermissionService) UpdatePermission(id uuid.UUID, req UpdatePermissionRequest) (*PermissionResponse, error) {
	// 権限取得
	var permission models.Permission
	if err := s.db.First(&permission, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("permission", "Permission not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// システム権限保護
	if s.isSystemPermission(permission.Module, permission.Action) {
		return nil, errors.NewValidationError("system_permission", "Cannot update system permissions")
	}

	// 部分更新（現在はDescriptionのみ更新可能）
	updateData := make(map[string]interface{})
	if req.Description != nil {
		// 注意: 現在のPermissionモデルにDescriptionフィールドがないため、
		// 実際のデータベース更新は行わない
	}

	if len(updateData) > 0 {
		if err := s.db.Model(&permission).Updates(updateData).Error; err != nil {
			return nil, errors.NewDatabaseError(err)
		}
	}

	// 更新後のレスポンス作成
	response := s.convertToPermissionResponse(&permission)
	if req.Description != nil {
		response.Description = *req.Description
	}
	response.Roles = s.getRoleInfoForPermission(permission.ID)
	response.UsageStats = s.getUsageStats(permission.ID)

	return &response, nil
}

// DeletePermission 権限削除
func (s *PermissionService) DeletePermission(id uuid.UUID) error {
	// 権限取得
	var permission models.Permission
	if err := s.db.First(&permission, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewNotFoundError("permission", "Permission not found")
		}
		return errors.NewDatabaseError(err)
	}

	// システム権限保護
	if s.isSystemPermission(permission.Module, permission.Action) {
		return errors.NewValidationError("system_permission", "Cannot delete system permissions")
	}

	// ロール割り当てチェック
	var roleCount int64
	if err := s.db.Model(&models.RolePermission{}).Where("permission_id = ?", id).Count(&roleCount).Error; err != nil {
		return errors.NewDatabaseError(err)
	}

	if roleCount > 0 {
		return errors.NewValidationError("role_assigned", fmt.Sprintf("Cannot delete permission assigned to %d roles", roleCount))
	}

	// 権限削除
	if err := s.db.Delete(&permission).Error; err != nil {
		return errors.NewDatabaseError(err)
	}

	return nil
}

// GetPermissions 権限一覧取得
func (s *PermissionService) GetPermissions(req GetPermissionsRequest) (*PermissionListResponse, error) {
	// デフォルト値設定
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}

	query := s.db.Model(&models.Permission{})

	// フィルタリング
	if req.Module != "" {
		query = query.Where("module = ?", req.Module)
	}
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}
	if req.UsedByRole != "" {
		roleID, err := uuid.Parse(req.UsedByRole)
		if err != nil {
			return nil, errors.NewValidationError("role_id", "Invalid role ID format")
		}
		query = query.Joins("JOIN role_permissions rp ON permissions.id = rp.permission_id").
			Where("rp.role_id = ?", roleID)
	}
	if req.Search != "" {
		searchTerm := "%" + strings.ToLower(req.Search) + "%"
		query = query.Where("LOWER(module) LIKE ? OR LOWER(action) LIKE ?", searchTerm, searchTerm)
	}

	// 総数取得
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// ページング
	offset := (req.Page - 1) * req.Limit
	var permissions []models.Permission
	if err := query.Order("module, action").Offset(offset).Limit(req.Limit).Find(&permissions).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// レスポンス作成
	responses := make([]PermissionResponse, len(permissions))
	for i, permission := range permissions {
		responses[i] = s.convertToPermissionResponse(&permission)
		responses[i].Roles = s.getRoleInfoForPermission(permission.ID)
		responses[i].UsageStats = s.getUsageStats(permission.ID)
	}

	totalPages := int((total + int64(req.Limit) - 1) / int64(req.Limit))

	result := &PermissionListResponse{
		Permissions: responses,
		Total:       total,
		Page:        req.Page,
		Limit:       req.Limit,
		TotalPages:  totalPages,
	}

	return result, nil
}

// GetPermissionMatrix 権限マトリックス取得
func (s *PermissionService) GetPermissionMatrix() (*PermissionMatrixResponse, error) {
	// 全権限取得
	var permissions []models.Permission
	if err := s.db.Order("module, action").Find(&permissions).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// ロール-権限関係取得
	var rolePerms []struct {
		RoleID       uuid.UUID `json:"role_id"`
		RoleName     string    `json:"role_name"`
		PermissionID uuid.UUID `json:"permission_id"`
		Module       string    `json:"module"`
		Action       string    `json:"action"`
	}

	query := `
		SELECT rp.role_id, r.name as role_name, rp.permission_id, p.module, p.action
		FROM role_permissions rp
		JOIN roles r ON rp.role_id = r.id
		JOIN permissions p ON rp.permission_id = p.id
		ORDER BY r.name, p.module, p.action
	`

	if err := s.db.Raw(query).Scan(&rolePerms).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// ロール別権限マップ作成
	permissionRoles := make(map[string][]string)
	for _, rp := range rolePerms {
		permKey := rp.Module + ":" + rp.Action
		permissionRoles[permKey] = append(permissionRoles[permKey], rp.RoleName)
	}

	// モジュール別にグループ化
	moduleMap := make(map[string][]ActionInfo)
	actionSet := make(map[string]bool)

	for _, perm := range permissions {
		permKey := perm.Module + ":" + perm.Action
		actionInfo := ActionInfo{
			Name:         perm.Action,
			DisplayName:  s.getActionDisplayName(perm.Action),
			PermissionID: perm.ID.String(),
			Roles:        permissionRoles[permKey],
		}

		moduleMap[perm.Module] = append(moduleMap[perm.Module], actionInfo)
		actionSet[perm.Action] = true
	}

	// モジュール情報作成
	modules := make([]ModuleInfo, 0, len(moduleMap))
	for moduleName, actions := range moduleMap {
		modules = append(modules, ModuleInfo{
			Name:        moduleName,
			DisplayName: s.getModuleDisplayName(moduleName),
			Actions:     actions,
		})
	}

	// 未使用権限数計算
	unusedCount := 0
	for _, perm := range permissions {
		permKey := perm.Module + ":" + perm.Action
		if len(permissionRoles[permKey]) == 0 {
			unusedCount++
		}
	}

	summary := MatrixSummaryInfo{
		TotalPermissions:  len(permissions),
		TotalModules:      len(moduleMap),
		TotalActions:      len(actionSet),
		UnusedPermissions: unusedCount,
	}

	result := &PermissionMatrixResponse{
		Modules: modules,
		Summary: summary,
	}

	return result, nil
}

// GetPermissionsByModule モジュール別権限取得
func (s *PermissionService) GetPermissionsByModule(module string) (*ModulePermissionsResponse, error) {
	// モジュール検証
	if !s.isValidModule(module) {
		return nil, errors.NewValidationError("module", "Invalid module")
	}

	// 権限取得
	var permissions []models.Permission
	if err := s.db.Where("module = ?", module).Order("action").Find(&permissions).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// レスポンス作成
	responses := make([]PermissionResponse, len(permissions))
	for i, permission := range permissions {
		responses[i] = s.convertToPermissionResponse(&permission)
		responses[i].Roles = s.getRoleInfoForPermission(permission.ID)
		responses[i].UsageStats = s.getUsageStats(permission.ID)
	}

	result := &ModulePermissionsResponse{
		Module:      module,
		DisplayName: s.getModuleDisplayName(module),
		Permissions: responses,
		Total:       len(responses),
	}

	return result, nil
}

// GetRolesByPermission 権限を持つロール一覧取得
func (s *PermissionService) GetRolesByPermission(id uuid.UUID) ([]PermissionRoleInfo, error) {
	// 権限存在確認
	var permission models.Permission
	if err := s.db.First(&permission, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("permission", "Permission not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	roles := s.getRoleInfoForPermission(id)
	return roles, nil
}

// =============================================================================
// ヘルパーメソッド
// =============================================================================

// validateCreateRequest 作成リクエストのバリデーション
func (s *PermissionService) validateCreateRequest(req CreatePermissionRequest) error {
	if !s.isValidModule(req.Module) {
		return errors.NewValidationError("module", "Invalid module: "+req.Module)
	}
	if !s.isValidAction(req.Action) {
		return errors.NewValidationError("action", "Invalid action: "+req.Action)
	}
	return nil
}

// findPermissionByModuleAction モジュール・アクションで権限検索
func (s *PermissionService) findPermissionByModuleAction(module, action string) (*models.Permission, error) {
	var permission models.Permission
	err := s.db.Where("module = ? AND action = ?", module, action).First(&permission).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("permission", "Permission not found")
		}
		return nil, err
	}
	return &permission, nil
}

// isSystemPermission システム権限かチェック
func (s *PermissionService) isSystemPermission(module, action string) bool {
	systemPermissions := []string{
		"user:read", "user:list", "department:read", "department:list",
		"role:read", "role:list", "permission:read", "permission:list",
		"system:admin", "audit:read",
	}

	permKey := module + ":" + action
	for _, sysPerm := range systemPermissions {
		if permKey == sysPerm {
			return true
		}
	}
	return false
}

// convertToPermissionResponse Permissionモデルをレスポンスに変換
func (s *PermissionService) convertToPermissionResponse(permission *models.Permission) PermissionResponse {
	return PermissionResponse{
		ID:          permission.ID,
		Module:      permission.Module,
		Action:      permission.Action,
		Code:        permission.Module + ":" + permission.Action,
		Description: s.getPermissionDescription(permission.Module, permission.Action),
		IsSystem:    s.isSystemPermission(permission.Module, permission.Action),
		CreatedAt:   permission.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// getRoleInfoForPermission 権限に関連するロール情報取得
func (s *PermissionService) getRoleInfoForPermission(permissionID uuid.UUID) []PermissionRoleInfo {
	var roleInfos []PermissionRoleInfo

	query := `
		SELECT r.id, r.name, COUNT(ur.user_id) as user_count
		FROM roles r
		JOIN role_permissions rp ON r.id = rp.role_id
		LEFT JOIN user_roles ur ON r.id = ur.role_id AND ur.is_active = true
		WHERE rp.permission_id = ?
		GROUP BY r.id, r.name
		ORDER BY r.name
	`

	var results []struct {
		ID        uuid.UUID `json:"id"`
		Name      string    `json:"name"`
		UserCount int       `json:"user_count"`
	}

	if err := s.db.Raw(query, permissionID).Scan(&results).Error; err != nil {
		// データベースエラーが発生した場合は空のスライスを返す
		return roleInfos
	}

	for _, result := range results {
		roleInfos = append(roleInfos, PermissionRoleInfo{
			ID:        result.ID,
			Name:      result.Name,
			UserCount: result.UserCount,
		})
	}

	return roleInfos
}

// getUsageStats 権限使用統計取得
func (s *PermissionService) getUsageStats(permissionID uuid.UUID) PermissionUsageStats {
	stats := PermissionUsageStats{}

	// ロール数
	var roleCount int64
	s.db.Model(&models.RolePermission{}).Where("permission_id = ?", permissionID).Count(&roleCount)
	stats.RoleCount = int(roleCount)

	// ユーザー数（アクティブなロール経由）
	query := `
		SELECT COUNT(DISTINCT ur.user_id) as user_count
		FROM user_roles ur
		JOIN role_permissions rp ON ur.role_id = rp.role_id
		WHERE rp.permission_id = ? AND ur.is_active = true
	`
	var userCount int64
	s.db.Raw(query, permissionID).Scan(&userCount)
	stats.UserCount = int(userCount)

	// 最終使用日（簡略化：作成日を使用）
	var permission models.Permission
	if err := s.db.First(&permission, permissionID).Error; err == nil {
		stats.LastUsed = permission.CreatedAt.Format("2006-01-02T15:04:05Z")
	}

	return stats
}

// getModuleDisplayName モジュール表示名取得
func (s *PermissionService) getModuleDisplayName(module string) string {
	displayNames := map[string]string{
		"user":       "ユーザー管理",
		"department": "部署管理",
		"role":       "ロール管理",
		"permission": "権限管理",
		"audit":      "監査ログ",
		"system":     "システム管理",
		"inventory":  "在庫管理",
		"orders":     "注文管理",
		"reports":    "レポート",
		"dashboard":  "ダッシュボード",
		"settings":   "設定",
		"finance":    "財務管理",
		"hr":         "人事管理",
	}

	if displayName, exists := displayNames[module]; exists {
		return displayName
	}
	return module
}

// getActionDisplayName アクション表示名取得
func (s *PermissionService) getActionDisplayName(action string) string {
	displayNames := map[string]string{
		"create":  "作成",
		"read":    "閲覧",
		"update":  "更新",
		"delete":  "削除",
		"list":    "一覧",
		"manage":  "管理",
		"view":    "表示",
		"approve": "承認",
		"export":  "エクスポート",
		"admin":   "管理者",
	}

	if displayName, exists := displayNames[action]; exists {
		return displayName
	}
	return action
}

// getPermissionDescription 権限説明取得
func (s *PermissionService) getPermissionDescription(module, action string) string {
	moduleDisplay := s.getModuleDisplayName(module)
	actionDisplay := s.getActionDisplayName(action)
	return fmt.Sprintf("%s%s権限", moduleDisplay, actionDisplay)
}

// Module システムモジュールを表す
type Module string

const (
	ModuleUser       Module = "user"
	ModuleDepartment Module = "department"
	ModuleRole       Module = "role"
	ModulePermission Module = "permission"
	ModuleAudit      Module = "audit"
	ModuleSystem     Module = "system"
	ModuleInventory  Module = "inventory"
	ModuleOrders     Module = "orders"
	ModuleReports    Module = "reports"
)

// Action 実行可能なアクションを表す
type Action string

const (
	ActionCreate  Action = "create"
	ActionRead    Action = "read"
	ActionUpdate  Action = "update"
	ActionDelete  Action = "delete"
	ActionList    Action = "list"
	ActionManage  Action = "manage"
	ActionView    Action = "view"
	ActionApprove Action = "approve"
	ActionExport  Action = "export"
	ActionAdmin   Action = "admin"
)

// Permission 権限文字列を表す
type Permission string

// NewPermission 標準的な権限フォーマット: module:action (例: "user:create", "role:read")
func NewPermission(module Module, action Action) Permission {
	return Permission(fmt.Sprintf("%s:%s", module, action))
}

// PermissionMatrix システムの権限マトリックスを定義
// TODO: アーキテクチャ改善
// - データベースベース権限管理への移行検討
// - 動的権限追加・削除機能
// - 時間ベース権限 (営業時間のみ有効など)
// - 地理的制限 (特定IPレンジ、地域からのみアクセス)
// - 継承ベース階層権限 (部門長→課長→係長の自動継承)
var PermissionMatrix = map[string][]Permission{
	// システム管理者 - Full access
	"システム管理者": {
		"*:*", // Wildcard permission for everything
	},

	// 部門管理者 - Most permissions except system management
	"部門管理者": {
		NewPermission(ModuleUser, ActionCreate),
		NewPermission(ModuleUser, ActionRead),
		NewPermission(ModuleUser, ActionUpdate),
		NewPermission(ModuleUser, ActionDelete),
		NewPermission(ModuleUser, ActionList),
		NewPermission(ModuleDepartment, ActionCreate),
		NewPermission(ModuleDepartment, ActionRead),
		NewPermission(ModuleDepartment, ActionUpdate),
		NewPermission(ModuleDepartment, ActionDelete),
		NewPermission(ModuleDepartment, ActionList),
		NewPermission(ModuleRole, ActionCreate),
		NewPermission(ModuleRole, ActionRead),
		NewPermission(ModuleRole, ActionUpdate),
		NewPermission(ModuleRole, ActionDelete),
		NewPermission(ModuleRole, ActionList),
		NewPermission(ModulePermission, ActionRead),
		NewPermission(ModulePermission, ActionList),
		NewPermission(ModuleAudit, ActionRead),
		NewPermission(ModuleAudit, ActionList),
	},

	// 開発者 - Development related permissions
	"開発者": {
		NewPermission(ModuleUser, ActionRead),
		NewPermission(ModuleUser, ActionUpdate),
		NewPermission(ModuleUser, ActionList),
		NewPermission(ModuleDepartment, ActionRead),
		NewPermission(ModuleDepartment, ActionList),
		NewPermission(ModuleRole, ActionRead),
		NewPermission(ModuleRole, ActionList),
		NewPermission(ModuleAudit, ActionRead),
	},

	// 一般ユーザー - Basic read access
	"一般ユーザー": {
		NewPermission(ModuleUser, ActionRead),
		NewPermission(ModuleDepartment, ActionRead),
		NewPermission(ModuleRole, ActionRead),
	},

	// ゲストユーザー - Read-only access
	"ゲストユーザー": {
		NewPermission(ModuleUser, ActionRead),
		NewPermission(ModuleUser, ActionList),
		NewPermission(ModuleDepartment, ActionRead),
		NewPermission(ModuleDepartment, ActionList),
		NewPermission(ModuleRole, ActionRead),
		NewPermission(ModuleRole, ActionList),
	},

	// プロジェクトマネージャー - Project management permissions
	"プロジェクトマネージャー": {
		NewPermission(ModuleUser, ActionRead),
		NewPermission(ModuleUser, ActionUpdate),
		NewPermission(ModuleUser, ActionList),
		NewPermission(ModuleDepartment, ActionRead),
		NewPermission(ModuleDepartment, ActionUpdate),
		NewPermission(ModuleDepartment, ActionList),
		NewPermission(ModuleRole, ActionRead),
		NewPermission(ModuleRole, ActionUpdate),
		NewPermission(ModuleRole, ActionList),
		NewPermission(ModuleAudit, ActionRead),
	},

	// テスター - Testing related permissions
	"テスター": {
		NewPermission(ModuleUser, ActionRead),
		NewPermission(ModuleUser, ActionList),
		NewPermission(ModuleDepartment, ActionRead),
		NewPermission(ModuleDepartment, ActionList),
		NewPermission(ModuleRole, ActionRead),
		NewPermission(ModuleRole, ActionList),
		NewPermission(ModuleAudit, ActionRead),
	},
}

// GetUserPermissions ユーザーの全権限を取得（複数ロール対応）
func (s *PermissionService) GetUserPermissions(userID uuid.UUID) ([]string, error) {
	// TODO: パフォーマンス最適化
	// - Redis/Memcachedによる権限キャッシュ (TTL: 5-15分)
	// - 階層的権限の事前計算とキャッシュ
	// - バッチ権限取得機能 (複数ユーザー一括処理)
	// - 権限変更時のキャッシュ無効化戦略

	var user models.User
	if err := s.db.Preload("UserRoles.Role.Permissions").Preload("PrimaryRole.Permissions").First(&user, userID).Error; err != nil {
		return nil, err
	}

	permissionSet := make(map[string]bool)

	// 複数ロールから権限を集約
	for _, userRole := range user.UserRoles {
		// アクティブで有効期間内のロールのみ処理
		if !userRole.IsValidNow() {
			continue
		}

		// Get base permissions from matrix for each role
		if basePerms, exists := PermissionMatrix[userRole.Role.Name]; exists {
			for _, perm := range basePerms {
				permissionSet[string(perm)] = true
			}
		}

		// Add explicit permissions from database for each role
		for _, perm := range userRole.Role.Permissions {
			permissionSet[perm.GetUniqueKey()] = true
		}
	}

	// 後方互換性: PrimaryRoleが設定されている場合は優先
	if user.PrimaryRole != nil {
		if basePerms, exists := PermissionMatrix[user.PrimaryRole.Name]; exists {
			for _, perm := range basePerms {
				permissionSet[string(perm)] = true
			}
		}
	}

	// Convert to slice
	permissions := make([]string, 0, len(permissionSet))
	for perm := range permissionSet {
		permissions = append(permissions, perm)
	}

	return permissions, nil
}

// GetUserRoleHierarchyPermissions 階層ロール権限を含めて取得
func (s *PermissionService) GetUserRoleHierarchyPermissions(userID uuid.UUID) ([]string, error) {
	var permissions []string

	query := `
		WITH RECURSIVE role_hierarchy AS (
			-- アクティブなユーザーロール
			SELECT ur.role_id, ur.priority
			FROM user_roles ur
			WHERE ur.user_id = ? 
				AND ur.is_active = true
				AND ur.valid_from <= NOW()
				AND (ur.valid_to IS NULL OR ur.valid_to > NOW())
			
			UNION
			
			-- 親ロールを辿る
			SELECT r.parent_id, rh.priority
			FROM roles r
			JOIN role_hierarchy rh ON r.id = rh.role_id
			WHERE r.parent_id IS NOT NULL
		)
		SELECT DISTINCT CONCAT(p.module, ':', p.action) as permission
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN role_hierarchy rh ON rp.role_id = rh.role_id
		ORDER BY permission
	`

	err := s.db.Raw(query, userID).Pluck("permission", &permissions).Error
	return permissions, err
}

// CheckPermission ユーザーが特定の権限を持っているかチェック（複数ロール対応）
func (s *PermissionService) CheckPermission(userID uuid.UUID, requiredPermission string) (bool, error) {
	// TODO: 監査ログ強化
	// - 権限チェックの詳細ログ (成功/失敗、リソース、時刻)
	// - セキュリティアラート (権限昇格試行、異常パターン)
	// - パフォーマンスメトリクス (応答時間、呼び出し頻度)

	permissions, err := s.GetUserPermissions(userID)
	if err != nil {
		return false, err
	}

	hasPermission := s.hasPermission(permissions, requiredPermission)

	// TODO: ログ出力実装
	// log.Info("Permission check", zap.String("user_id", userID.String()),
	//     zap.String("permission", requiredPermission), zap.Bool("granted", hasPermission))

	return hasPermission, nil
}

// CheckPermissionWithScope スコープ条件付きで権限をチェック
func (s *PermissionService) CheckPermissionWithScope(userID uuid.UUID, requiredPermission string, resourceScope map[string]interface{}) (bool, error) {
	// First check basic permission
	hasBasicPerm, err := s.CheckPermission(userID, requiredPermission)
	if err != nil {
		return false, err
	}
	if !hasBasicPerm {
		return false, nil
	}

	// Check scope restrictions
	var userScopes []models.UserScope
	if err := s.db.Where("user_id = ?", userID).Find(&userScopes).Error; err != nil {
		return false, err
	}

	// If no scopes defined, allow access
	if len(userScopes) == 0 {
		return true, nil
	}

	// Check if any scope matches
	for _, scope := range userScopes {
		// Convert JSONB to json.RawMessage
		scopeJSON, err := json.Marshal(scope.ScopeValue)
		if err != nil {
			continue
		}
		if s.evaluateScope(json.RawMessage(scopeJSON), resourceScope) {
			return true, nil
		}
	}

	return false, nil
}

// evaluateScope JSONBスコープ条件をリソーススコープと照合評価
func (s *PermissionService) evaluateScope(scopeValue json.RawMessage, resourceScope map[string]interface{}) bool {
	var conditions map[string]interface{}
	if err := json.Unmarshal(scopeValue, &conditions); err != nil {
		return false
	}

	for key, expectedValue := range conditions {
		if actualValue, exists := resourceScope[key]; exists {
			if !s.compareValues(expectedValue, actualValue) {
				return false
			}
		} else {
			// Required scope field missing
			return false
		}
	}

	return true
}

// compareValues 配列とワイルドカードをサポートしてスコープ値を比較
func (s *PermissionService) compareValues(expected, actual interface{}) bool {
	switch exp := expected.(type) {
	case string:
		if exp == "*" {
			return true // Wildcard matches anything
		}
		if act, ok := actual.(string); ok {
			return exp == act
		}
	case []interface{}:
		// Check if actual value is in the expected array
		for _, val := range exp {
			if s.compareValues(val, actual) {
				return true
			}
		}
	case map[string]interface{}:
		// Nested conditions
		if actMap, ok := actual.(map[string]interface{}); ok {
			for k, v := range exp {
				if actVal, exists := actMap[k]; exists {
					if !s.compareValues(v, actVal) {
						return false
					}
				} else {
					return false
				}
			}
			return true
		}
	default:
		return expected == actual
	}
	return false
}

// hasPermission ワイルドカードサポート付きでユーザー権限に指定権限が存在するかチェック
func (s *PermissionService) hasPermission(userPermissions []string, requiredPermission string) bool {
	for _, perm := range userPermissions {
		if perm == "*" || perm == "*:*" {
			return true // Super admin wildcard
		}
		if perm == requiredPermission {
			return true // Exact match
		}
		if s.matchesWildcard(perm, requiredPermission) {
			return true // Wildcard pattern match
		}
	}
	return false
}

// matchesWildcard 権限がワイルドカードパターンにマッチするかチェック
func (s *PermissionService) matchesWildcard(pattern, permission string) bool {
	// Handle module:* patterns (e.g., "user:*" matches "user:read")
	if strings.HasSuffix(pattern, ":*") {
		module := strings.TrimSuffix(pattern, ":*")
		return strings.HasPrefix(permission, module+":")
	}

	// Handle *:action patterns (e.g., "*:read" matches "user:read")
	if strings.HasPrefix(pattern, "*:") {
		action := strings.TrimPrefix(pattern, "*:")
		return strings.HasSuffix(permission, ":"+action)
	}

	return false
}

// GetRolePermissions 特定のロールの権限を取得
func (s *PermissionService) GetRolePermissions(roleName string) []Permission {
	if perms, exists := PermissionMatrix[roleName]; exists {
		return perms
	}
	return []Permission{}
}

// ValidatePermission 権限文字列が有効かバリデーション
func (s *PermissionService) ValidatePermission(permission string) bool {
	if permission == "*" || permission == "*:*" {
		return true
	}

	parts := strings.Split(permission, ":")
	if len(parts) != 2 {
		return false
	}

	module, action := parts[0], parts[1]
	return s.isValidModule(module) && s.isValidAction(action)
}

// isValidModule モジュールが有効かチェック
func (s *PermissionService) isValidModule(module string) bool {
	if module == "*" {
		return true
	}
	validModules := []string{
		string(ModuleUser),
		string(ModuleDepartment),
		string(ModuleRole),
		string(ModulePermission),
		string(ModuleAudit),
		string(ModuleSystem),
		string(ModuleInventory),
		string(ModuleOrders),
		string(ModuleReports),
	}

	for _, valid := range validModules {
		if module == valid {
			return true
		}
	}
	return false
}

// isValidAction アクションが有効かチェック
func (s *PermissionService) isValidAction(action string) bool {
	if action == "*" {
		return true
	}
	validActions := []string{
		string(ActionCreate),
		string(ActionRead),
		string(ActionUpdate),
		string(ActionDelete),
		string(ActionList),
		string(ActionManage),
	}

	for _, valid := range validActions {
		if action == valid {
			return true
		}
	}
	return false
}
