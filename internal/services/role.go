package services

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"erp-access-control-go/models"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/pkg/logger"
)

// RoleService ロール管理サービス
type RoleService struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewRoleService 新しいロールサービスを作成
func NewRoleService(db *gorm.DB, logger *logger.Logger) *RoleService {
	return &RoleService{
		db:     db,
		logger: logger,
	}
}

// CreateRoleRequest ロール作成リクエスト
type CreateRoleRequest struct {
	Name          string      `json:"name" binding:"required,min=2,max=100"`
	ParentID      *uuid.UUID  `json:"parent_id" binding:"omitempty"`
	PermissionIDs []uuid.UUID `json:"permission_ids" binding:"omitempty,dive,uuid"`
}

// UpdateRoleRequest ロール更新リクエスト
type UpdateRoleRequest struct {
	Name     *string    `json:"name" binding:"omitempty,min=2,max=100"`
	ParentID *uuid.UUID `json:"parent_id"`
}

// AssignPermissionsRequest 権限割り当てリクエスト
type AssignPermissionsRequest struct {
	PermissionIDs []uuid.UUID `json:"permission_ids" binding:"required,dive,uuid"`
	Replace       bool        `json:"replace"` // trueの場合、既存権限を置き換え
}

// RoleResponse ロールレスポンス
type RoleResponse struct {
	ID                   uuid.UUID                 `json:"id"`
	Name                 string                    `json:"name"`
	ParentID             *uuid.UUID                `json:"parent_id,omitempty"`
	Level                int                       `json:"level"`
	CreatedAt            string                    `json:"created_at"`
	Parent               *RoleBasicInfo            `json:"parent,omitempty"`
	Children             []RoleBasicInfo           `json:"children,omitempty"`
	Permissions          []PermissionInfo          `json:"permissions,omitempty"`
	InheritedPermissions []InheritedPermissionInfo `json:"inherited_permissions,omitempty"`
	UserCount            int64                     `json:"user_count"`
}

// RoleBasicInfo ロール基本情報
type RoleBasicInfo struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Level int       `json:"level"`
}

// PermissionInfo 権限情報
type PermissionInfo struct {
	ID        uuid.UUID `json:"id"`
	Module    string    `json:"module"`
	Action    string    `json:"action"`
	Inherited bool      `json:"inherited"`
}

// InheritedPermissionInfo 継承権限情報
type InheritedPermissionInfo struct {
	ID            uuid.UUID `json:"id"`
	Module        string    `json:"module"`
	Action        string    `json:"action"`
	InheritedFrom uuid.UUID `json:"inherited_from"`
	FromRoleName  string    `json:"from_role_name"`
}

// RoleListResponse ロール一覧レスポンス
type RoleListResponse struct {
	Roles []RoleResponse `json:"roles"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

// RoleHierarchyNode 階層ツリーノード
type RoleHierarchyNode struct {
	ID              uuid.UUID           `json:"id"`
	Name            string              `json:"name"`
	Level           int                 `json:"level"`
	PermissionCount int                 `json:"permission_count"`
	UserCount       int64               `json:"user_count"`
	Children        []RoleHierarchyNode `json:"children,omitempty"`
}

// RoleHierarchyResponse 階層ツリーレスポンス
type RoleHierarchyResponse struct {
	Roles []RoleHierarchyNode `json:"roles"`
}

// RolePermissionsResponse ロール権限一覧レスポンス
type RolePermissionsResponse struct {
	RoleID               uuid.UUID                 `json:"role_id"`
	RoleName             string                    `json:"role_name"`
	DirectPermissions    []PermissionInfo          `json:"direct_permissions"`
	InheritedPermissions []InheritedPermissionInfo `json:"inherited_permissions"`
	AllPermissions       []PermissionInfo          `json:"all_permissions"`
}

// CreateRole ロールを作成
func (s *RoleService) CreateRole(req CreateRoleRequest) (*RoleResponse, error) {
	s.logger.Info("Creating new role", map[string]interface{}{
		"name":           req.Name,
		"parent_id":      req.ParentID,
		"permission_ids": req.PermissionIDs,
	})

	// 名前の重複チェック
	var existingRole models.Role
	if err := s.db.Where("name = ?", req.Name).First(&existingRole).Error; err == nil {
		return nil, errors.NewValidationError("name", "Role name already exists")
	} else if err != gorm.ErrRecordNotFound {
		return nil, errors.NewDatabaseError(err)
	}

	// 親ロール存在確認（指定された場合）
	if req.ParentID != nil {
		var parent models.Role
		if err := s.db.First(&parent, *req.ParentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.NewNotFoundError("Role", "Parent role does not exist")
			}
			return nil, errors.NewDatabaseError(err)
		}

		// 階層深度チェック
		depth, err := s.calculateDepth(*req.ParentID)
		if err != nil {
			return nil, err
		}
		if depth >= 5 {
			return nil, errors.NewValidationError("parent_id", "Maximum hierarchy depth (5 levels) exceeded")
		}
	}

	// 権限存在確認（指定された場合）
	if len(req.PermissionIDs) > 0 {
		var permissionCount int64
		if err := s.db.Model(&models.Permission{}).Where("id IN ?", req.PermissionIDs).Count(&permissionCount).Error; err != nil {
			return nil, errors.NewDatabaseError(err)
		}
		if int(permissionCount) != len(req.PermissionIDs) {
			return nil, errors.NewValidationError("permission_ids", "One or more permissions do not exist")
		}
	}

	// トランザクション内でロール作成と権限割り当て
	var role models.Role
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// ロール作成
		role = models.Role{
			Name:     req.Name,
			ParentID: req.ParentID,
		}

		if err := tx.Create(&role).Error; err != nil {
			return err
		}

		// 権限割り当て（指定された場合）
		if len(req.PermissionIDs) > 0 {
			var permissions []models.Permission
			if err := tx.Where("id IN ?", req.PermissionIDs).Find(&permissions).Error; err != nil {
				return err
			}

			if err := tx.Model(&role).Association("Permissions").Append(permissions); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		s.logger.Error("Failed to create role", err, map[string]interface{}{
			"name": req.Name,
		})
		return nil, errors.NewDatabaseError(err)
	}

	s.logger.Info("Role created successfully", map[string]interface{}{
		"role_id": role.ID,
		"name":    role.Name,
	})

	// 作成されたロールを詳細付きで取得
	return s.GetRole(role.ID)
}

// GetRole ロール詳細を取得
func (s *RoleService) GetRole(roleID uuid.UUID) (*RoleResponse, error) {
	var role models.Role
	if err := s.db.
		Preload("Parent").
		Preload("Children").
		Preload("Permissions").
		First(&role, roleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Role", "Role not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	return s.convertToRoleResponse(&role)
}

// UpdateRole ロール情報を更新
func (s *RoleService) UpdateRole(roleID uuid.UUID, req UpdateRoleRequest) (*RoleResponse, error) {
	s.logger.Info("Updating role", map[string]interface{}{
		"role_id": roleID,
	})

	// ロール存在確認
	var role models.Role
	if err := s.db.First(&role, roleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Role", "Role not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// 名前重複チェック（自分以外）
	if req.Name != nil {
		var existingRole models.Role
		if err := s.db.Where("name = ? AND id != ?", *req.Name, roleID).First(&existingRole).Error; err == nil {
			return nil, errors.NewValidationError("name", "Role name already exists")
		} else if err != gorm.ErrRecordNotFound {
			return nil, errors.NewDatabaseError(err)
		}
	}

	// 親ロール変更時の検証
	if req.ParentID != nil {
		// 自己参照チェック
		if *req.ParentID == roleID {
			return nil, errors.NewValidationError("parent_id", "Role cannot be its own parent")
		}

		// 循環参照チェック
		if err := s.checkCircularReference(roleID, *req.ParentID); err != nil {
			return nil, err
		}

		// 親ロール存在確認
		var parent models.Role
		if err := s.db.First(&parent, *req.ParentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.NewNotFoundError("Role", "Parent role does not exist")
			}
			return nil, errors.NewDatabaseError(err)
		}

		// 階層深度チェック
		depth, err := s.calculateDepth(*req.ParentID)
		if err != nil {
			return nil, err
		}
		if depth >= 5 {
			return nil, errors.NewValidationError("parent_id", "Maximum hierarchy depth (5 levels) exceeded")
		}
	}

	// 更新用のマップを作成
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.ParentID != nil {
		updates["parent_id"] = *req.ParentID
	}

	// 更新実行
	if len(updates) > 0 {
		if err := s.db.Model(&role).Updates(updates).Error; err != nil {
			s.logger.Error("Failed to update role", err, map[string]interface{}{
				"role_id": roleID,
			})
			return nil, errors.NewDatabaseError(err)
		}
	}

	s.logger.Info("Role updated successfully", map[string]interface{}{
		"role_id": roleID,
	})

	// 更新されたロールを取得
	return s.GetRole(roleID)
}

// DeleteRole ロールを削除
func (s *RoleService) DeleteRole(roleID uuid.UUID) error {
	s.logger.Info("Deleting role", map[string]interface{}{
		"role_id": roleID,
	})

	// ロール存在確認
	var role models.Role
	if err := s.db.First(&role, roleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewNotFoundError("Role", "Role not found")
		}
		return errors.NewDatabaseError(err)
	}

	// 子ロール存在チェック
	var childCount int64
	if err := s.db.Model(&models.Role{}).Where("parent_id = ?", roleID).Count(&childCount).Error; err != nil {
		return errors.NewDatabaseError(err)
	}
	if childCount > 0 {
		return errors.NewValidationError("parent_id", "Cannot delete role with child roles")
	}

	// ユーザー割り当てチェック（プライマリロール）
	var primaryUserCount int64
	if err := s.db.Model(&models.User{}).Where("primary_role_id = ?", roleID).Count(&primaryUserCount).Error; err != nil {
		return errors.NewDatabaseError(err)
	}
	if primaryUserCount > 0 {
		return errors.NewValidationError("role_id", "Cannot delete role assigned as primary role to users")
	}

	// ユーザー割り当てチェック（追加ロール）
	var userRoleCount int64
	if err := s.db.Model(&models.UserRole{}).Where("role_id = ?", roleID).Count(&userRoleCount).Error; err != nil {
		return errors.NewDatabaseError(err)
	}
	if userRoleCount > 0 {
		return errors.NewValidationError("role_id", "Cannot delete role assigned to users")
	}

	// ロール削除（権限の関連も自動削除される）
	if err := s.db.Delete(&role).Error; err != nil {
		s.logger.Error("Failed to delete role", err, map[string]interface{}{
			"role_id": roleID,
		})
		return errors.NewDatabaseError(err)
	}

	s.logger.Info("Role deleted successfully", map[string]interface{}{
		"role_id": roleID,
	})

	return nil
}

// GetRoles ロール一覧を取得
func (s *RoleService) GetRoles(page, limit int, parentID *uuid.UUID, permissionID *uuid.UUID, search string) (*RoleListResponse, error) {
	offset := (page - 1) * limit

	query := s.db.Model(&models.Role{}).
		Preload("Parent").
		Preload("Children").
		Preload("Permissions")

	// 親ロールフィルタ
	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
	}

	// 権限フィルタ
	if permissionID != nil {
		query = query.Joins("JOIN role_permissions ON roles.id = role_permissions.role_id").
			Where("role_permissions.permission_id = ?", *permissionID)
	}

	// 検索フィルタ
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(name) LIKE ?", searchTerm)
	}

	// 総件数取得
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// データ取得
	var roles []models.Role
	if err := query.
		Offset(offset).
		Limit(limit).
		Order("name ASC").
		Find(&roles).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// レスポンス変換
	roleResponses := make([]RoleResponse, len(roles))
	for i, role := range roles {
		resp, err := s.convertToRoleResponse(&role)
		if err != nil {
			return nil, err
		}
		roleResponses[i] = *resp
	}

	return &RoleListResponse{
		Roles: roleResponses,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

// AssignPermissions ロールに権限を割り当て
func (s *RoleService) AssignPermissions(roleID uuid.UUID, req AssignPermissionsRequest) (*RolePermissionsResponse, error) {
	s.logger.Info("Assigning permissions to role", map[string]interface{}{
		"role_id":        roleID,
		"permission_ids": req.PermissionIDs,
		"replace":        req.Replace,
	})

	// ロール存在確認
	var role models.Role
	if err := s.db.First(&role, roleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Role", "Role not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// 権限存在確認
	var permissions []models.Permission
	if err := s.db.Where("id IN ?", req.PermissionIDs).Find(&permissions).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}
	if len(permissions) != len(req.PermissionIDs) {
		return nil, errors.NewValidationError("permission_ids", "One or more permissions do not exist")
	}

	// 権限割り当て
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if req.Replace {
			// 既存権限をクリア
			if err := tx.Model(&role).Association("Permissions").Clear(); err != nil {
				return err
			}
		}

		// 新しい権限を追加
		if err := tx.Model(&role).Association("Permissions").Append(permissions); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		s.logger.Error("Failed to assign permissions", err, map[string]interface{}{
			"role_id": roleID,
		})
		return nil, errors.NewDatabaseError(err)
	}

	s.logger.Info("Permissions assigned successfully", map[string]interface{}{
		"role_id": roleID,
	})

	// 権限一覧を取得して返却
	return s.GetRolePermissions(roleID)
}

// GetRolePermissions ロールの権限一覧を取得
func (s *RoleService) GetRolePermissions(roleID uuid.UUID) (*RolePermissionsResponse, error) {
	// ロール存在確認
	var role models.Role
	if err := s.db.Preload("Permissions").First(&role, roleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Role", "Role not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// 直接権限
	directPermissions := make([]PermissionInfo, len(role.Permissions))
	for i, perm := range role.Permissions {
		directPermissions[i] = PermissionInfo{
			ID:        perm.ID,
			Module:    perm.Module,
			Action:    perm.Action,
			Inherited: false,
		}
	}

	// 継承権限
	inheritedPermissions, err := s.getInheritedPermissions(roleID)
	if err != nil {
		return nil, err
	}

	// 全権限（重複除去）
	allPermissions := s.mergePermissions(directPermissions, inheritedPermissions)

	return &RolePermissionsResponse{
		RoleID:               roleID,
		RoleName:             role.Name,
		DirectPermissions:    directPermissions,
		InheritedPermissions: inheritedPermissions,
		AllPermissions:       allPermissions,
	}, nil
}

// GetRoleHierarchy ロール階層ツリーを取得
func (s *RoleService) GetRoleHierarchy() (*RoleHierarchyResponse, error) {
	// ルートロールを取得
	var rootRoles []models.Role
	if err := s.db.Where("parent_id IS NULL").Order("name ASC").Find(&rootRoles).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// 階層ツリーを構築
	hierarchy := make([]RoleHierarchyNode, len(rootRoles))
	for i, role := range rootRoles {
		node, err := s.buildHierarchyNode(&role, 0)
		if err != nil {
			return nil, err
		}
		hierarchy[i] = *node
	}

	return &RoleHierarchyResponse{
		Roles: hierarchy,
	}, nil
}

// calculateDepth 指定されたロールの階層深度を計算
func (s *RoleService) calculateDepth(roleID uuid.UUID) (int, error) {
	depth := 0
	currentID := &roleID

	for currentID != nil && depth < 10 { // 無限ループ防止
		var role models.Role
		if err := s.db.Select("parent_id").First(&role, *currentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				break
			}
			return 0, errors.NewDatabaseError(err)
		}

		if role.ParentID == nil {
			break
		}

		currentID = role.ParentID
		depth++
	}

	return depth + 1, nil
}

// checkCircularReference 循環参照をチェック
func (s *RoleService) checkCircularReference(roleID, newParentID uuid.UUID) error {
	// 新しい親が自分の子孫でないかチェック
	descendants, err := s.getDescendants(roleID)
	if err != nil {
		return err
	}

	for _, descendant := range descendants {
		if descendant == newParentID {
			return errors.NewValidationError("parent_id", "Circular reference detected")
		}
	}

	return nil
}

// getDescendants 子孫ロールIDを取得
func (s *RoleService) getDescendants(roleID uuid.UUID) ([]uuid.UUID, error) {
	var descendants []uuid.UUID
	query := `
		WITH RECURSIVE role_descendants AS (
			SELECT id FROM roles WHERE parent_id = ?
			UNION ALL
			SELECT r.id FROM roles r
			JOIN role_descendants rd ON r.parent_id = rd.id
		)
		SELECT id FROM role_descendants
	`

	rows, err := s.db.Raw(query, roleID).Rows()
	if err != nil {
		return nil, errors.NewDatabaseError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, errors.NewDatabaseError(err)
		}
		descendants = append(descendants, id)
	}

	return descendants, nil
}

// buildHierarchyNode 階層ツリーノードを構築
func (s *RoleService) buildHierarchyNode(role *models.Role, level int) (*RoleHierarchyNode, error) {
	// 権限数を取得
	permissionCount := s.db.Model(role).Association("Permissions").Count()

	// ユーザー数を取得（プライマリロール + 追加ロール）
	userCount, err := s.getUserCount(role.ID)
	if err != nil {
		return nil, err
	}

	node := &RoleHierarchyNode{
		ID:              role.ID,
		Name:            role.Name,
		Level:           level,
		PermissionCount: int(permissionCount),
		UserCount:       userCount,
	}

	// 子ロールを取得
	var children []models.Role
	if err := s.db.Where("parent_id = ?", role.ID).Order("name ASC").Find(&children).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// 子ノードを再帰的に構築
	node.Children = make([]RoleHierarchyNode, len(children))
	for i, child := range children {
		childNode, err := s.buildHierarchyNode(&child, level+1)
		if err != nil {
			return nil, err
		}
		node.Children[i] = *childNode
	}

	return node, nil
}

// convertToRoleResponse ロールモデルをレスポンス形式に変換
func (s *RoleService) convertToRoleResponse(role *models.Role) (*RoleResponse, error) {
	// 階層レベル計算
	level, err := s.calculateLevel(role.ID)
	if err != nil {
		return nil, err
	}

	// ユーザー数取得
	userCount, err := s.getUserCount(role.ID)
	if err != nil {
		return nil, err
	}

	// 親ロール情報
	var parent *RoleBasicInfo
	if role.Parent != nil {
		parentLevel, err := s.calculateLevel(role.Parent.ID)
		if err != nil {
			return nil, err
		}
		parent = &RoleBasicInfo{
			ID:    role.Parent.ID,
			Name:  role.Parent.Name,
			Level: parentLevel,
		}
	}

	// 子ロール情報
	children := make([]RoleBasicInfo, len(role.Children))
	for i, child := range role.Children {
		childLevel, err := s.calculateLevel(child.ID)
		if err != nil {
			return nil, err
		}
		children[i] = RoleBasicInfo{
			ID:    child.ID,
			Name:  child.Name,
			Level: childLevel,
		}
	}

	// 直接権限
	permissions := make([]PermissionInfo, len(role.Permissions))
	for i, perm := range role.Permissions {
		permissions[i] = PermissionInfo{
			ID:        perm.ID,
			Module:    perm.Module,
			Action:    perm.Action,
			Inherited: false,
		}
	}

	// 継承権限
	inheritedPermissions, err := s.getInheritedPermissions(role.ID)
	if err != nil {
		return nil, err
	}

	return &RoleResponse{
		ID:                   role.ID,
		Name:                 role.Name,
		ParentID:             role.ParentID,
		Level:                level,
		CreatedAt:            role.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Parent:               parent,
		Children:             children,
		Permissions:          permissions,
		InheritedPermissions: inheritedPermissions,
		UserCount:            userCount,
	}, nil
}

// calculateLevel ロールの階層レベルを計算（ルートが0）
func (s *RoleService) calculateLevel(roleID uuid.UUID) (int, error) {
	level := 0
	currentID := &roleID

	for currentID != nil && level < 10 { // 無限ループ防止
		var role models.Role
		if err := s.db.Select("parent_id").First(&role, *currentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				break
			}
			return 0, errors.NewDatabaseError(err)
		}

		if role.ParentID == nil {
			break
		}

		currentID = role.ParentID
		level++
	}

	return level, nil
}

// getUserCount ロールに割り当てられているユーザー数を取得
func (s *RoleService) getUserCount(roleID uuid.UUID) (int64, error) {
	// プライマリロールとしてのユーザー数
	var primaryCount int64
	if err := s.db.Model(&models.User{}).Where("primary_role_id = ?", roleID).Count(&primaryCount).Error; err != nil {
		return 0, errors.NewDatabaseError(err)
	}

	// 追加ロールとしてのユーザー数（アクティブのみ）
	var additionalCount int64
	if err := s.db.Model(&models.UserRole{}).Where("role_id = ? AND is_active = ?", roleID, true).Count(&additionalCount).Error; err != nil {
		return 0, errors.NewDatabaseError(err)
	}

	return primaryCount + additionalCount, nil
}

// getInheritedPermissions 継承権限を取得
func (s *RoleService) getInheritedPermissions(roleID uuid.UUID) ([]InheritedPermissionInfo, error) {
	var inheritedPermissions []InheritedPermissionInfo

	// 親ロールからの権限継承を再帰的に取得
	query := `
		WITH RECURSIVE role_ancestors AS (
			SELECT id, name, parent_id, 1 as level
			FROM roles WHERE id = ?
			UNION ALL
			SELECT r.id, r.name, r.parent_id, ra.level + 1
			FROM roles r
			JOIN role_ancestors ra ON r.id = ra.parent_id
		),
		inherited_permissions AS (
			SELECT DISTINCT p.id, p.module, p.action, 
				   ra.id as from_role_id, ra.name as from_role_name
			FROM permissions p
			JOIN role_permissions rp ON p.id = rp.permission_id
			JOIN role_ancestors ra ON rp.role_id = ra.id
			WHERE ra.level > 1
		)
		SELECT id, module, action, from_role_id, from_role_name
		FROM inherited_permissions
		ORDER BY module, action
	`

	rows, err := s.db.Raw(query, roleID).Rows()
	if err != nil {
		return nil, errors.NewDatabaseError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var permission InheritedPermissionInfo
		if err := rows.Scan(
			&permission.ID,
			&permission.Module,
			&permission.Action,
			&permission.InheritedFrom,
			&permission.FromRoleName,
		); err != nil {
			return nil, errors.NewDatabaseError(err)
		}
		inheritedPermissions = append(inheritedPermissions, permission)
	}

	return inheritedPermissions, nil
}

// mergePermissions 直接権限と継承権限をマージし、重複を除去
func (s *RoleService) mergePermissions(direct []PermissionInfo, inherited []InheritedPermissionInfo) []PermissionInfo {
	permissionMap := make(map[uuid.UUID]PermissionInfo)

	// 直接権限を追加
	for _, perm := range direct {
		permissionMap[perm.ID] = perm
	}

	// 継承権限を追加（直接権限が優先）
	for _, inherited := range inherited {
		if _, exists := permissionMap[inherited.ID]; !exists {
			permissionMap[inherited.ID] = PermissionInfo{
				ID:        inherited.ID,
				Module:    inherited.Module,
				Action:    inherited.Action,
				Inherited: true,
			}
		}
	}

	// マップからスライスに変換
	allPermissions := make([]PermissionInfo, 0, len(permissionMap))
	for _, perm := range permissionMap {
		allPermissions = append(allPermissions, perm)
	}

	return allPermissions
}
