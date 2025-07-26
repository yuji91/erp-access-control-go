package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role ロールテーブル
type Role struct {
	BaseModel
	Name     string     `gorm:"not null" json:"name"`
	ParentID *uuid.UUID `gorm:"type:uuid;index" json:"parent_id,omitempty"`

	// リレーション
	Parent      *Role           `gorm:"foreignKey:ParentID;constraint:OnDelete:SET NULL" json:"parent,omitempty"`
	Children    []Role          `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Users       []User          `gorm:"foreignKey:RoleID" json:"users,omitempty"`
	Permissions []Permission    `gorm:"many2many:role_permissions;constraint:OnDelete:CASCADE" json:"permissions,omitempty"`
	Approvals   []ApprovalState `gorm:"foreignKey:ApproverRoleID" json:"approvals,omitempty"`
}

// TableName テーブル名を指定
func (Role) TableName() string {
	return "roles"
}

// BeforeCreate 作成前のバリデーション
func (r *Role) BeforeCreate(tx *gorm.DB) error {
	// 自己参照チェック
	if r.ParentID != nil && *r.ParentID == r.ID {
		return gorm.ErrInvalidData
	}
	return nil
}

// BeforeUpdate 更新前のバリデーション
func (r *Role) BeforeUpdate(tx *gorm.DB) error {
	// 自己参照チェック
	if r.ParentID != nil && *r.ParentID == r.ID {
		return gorm.ErrInvalidData
	}
	return nil
}

// =============================================================================
// ロール階層関連のメソッド
// =============================================================================

// IsRoot ルートロールかどうかを判定
func (r *Role) IsRoot() bool {
	return r.ParentID == nil
}

// HasParent 親ロールを持つかどうかを判定
func (r *Role) HasParent() bool {
	return r.ParentID != nil
}

// GetAncestors 祖先ロールを取得（階層上位）
func (r *Role) GetAncestors(db *gorm.DB) ([]Role, error) {
	var ancestors []Role
	query := `
		WITH RECURSIVE role_ancestors AS (
			SELECT id, name, parent_id, 1 as level
			FROM roles WHERE id = ?
			UNION ALL
			SELECT r.id, r.name, r.parent_id, ra.level + 1
			FROM roles r
			JOIN role_ancestors ra ON r.id = ra.parent_id
		)
		SELECT id, name, parent_id FROM role_ancestors WHERE level > 1
		ORDER BY level DESC
	`

	err := db.Raw(query, r.ID).Scan(&ancestors).Error
	return ancestors, err
}

// GetDescendants 子孫ロールを取得（階層下位）
func (r *Role) GetDescendants(db *gorm.DB) ([]Role, error) {
	var descendants []Role
	query := `
		WITH RECURSIVE role_descendants AS (
			SELECT id, name, parent_id, 1 as level
			FROM roles WHERE id = ?
			UNION ALL
			SELECT r.id, r.name, r.parent_id, rd.level + 1
			FROM roles r
			JOIN role_descendants rd ON r.parent_id = rd.id
		)
		SELECT id, name, parent_id FROM role_descendants WHERE level > 1
		ORDER BY level ASC
	`

	err := db.Raw(query, r.ID).Scan(&descendants).Error
	return descendants, err
}

// GetAllPermissions 階層考慮で全権限を取得
func (r *Role) GetAllPermissions(db *gorm.DB) ([]Permission, error) {
	var permissions []Permission
	query := `
		WITH role_hierarchy AS (
			SELECT id FROM roles WHERE id = ?
			UNION
			SELECT rh.id
			FROM roles rh
			JOIN role_hierarchy ON rh.parent_id = role_hierarchy.id
		)
		SELECT DISTINCT p.id, p.module, p.action, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN role_hierarchy rh ON rp.role_id = rh.id
		ORDER BY p.module, p.action
	`

	err := db.Raw(query, r.ID).Scan(&permissions).Error
	return permissions, err
}

// HasPermission 特定の権限を持つかチェック（階層考慮）
func (r *Role) HasPermission(db *gorm.DB, module, action string) (bool, error) {
	var count int64
	query := `
		WITH role_hierarchy AS (
			SELECT id FROM roles WHERE id = ?
			UNION
			SELECT rh.id
			FROM roles rh
			JOIN role_hierarchy ON rh.parent_id = role_hierarchy.id
		)
		SELECT COUNT(*)
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN role_hierarchy rh ON rp.role_id = rh.id
		WHERE p.module = ? AND p.action = ?
	`

	err := db.Raw(query, r.ID, module, action).Count(&count).Error
	return count > 0, err
}

// =============================================================================
// クエリ用ヘルパー関数
// =============================================================================

// FindRoleByID IDでロールを検索
func FindRoleByID(db *gorm.DB, id uuid.UUID) (*Role, error) {
	var role Role
	err := db.Where("id = ?", id).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// FindRoleByName 名前でロールを検索
func FindRoleByName(db *gorm.DB, name string) (*Role, error) {
	var role Role
	err := db.Where("name = ?", name).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// FindRolesByParentID 親IDで子ロールを検索
func FindRolesByParentID(db *gorm.DB, parentID *uuid.UUID) ([]Role, error) {
	var roles []Role
	query := db.Model(&Role{})

	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}

	err := query.Find(&roles).Error
	return roles, err
}

// FindRootRoles ルートロールを取得
func FindRootRoles(db *gorm.DB) ([]Role, error) {
	var roles []Role
	err := db.Where("parent_id IS NULL").Find(&roles).Error
	return roles, err
}

// GetRoleHierarchy ロール階層をツリー構造で取得
func GetRoleHierarchy(db *gorm.DB) ([]Role, error) {
	var hierarchy []Role
	query := `
		WITH RECURSIVE role_tree AS (
			SELECT id, name, parent_id, created_at, 1 as level,
				   ARRAY[id] as path, name as full_path
			FROM roles WHERE parent_id IS NULL
			UNION ALL
			SELECT r.id, r.name, r.parent_id, r.created_at, rt.level + 1,
				   rt.path || r.id, rt.full_path || ' > ' || r.name
			FROM roles r
			JOIN role_tree rt ON r.parent_id = rt.id
			WHERE NOT r.id = ANY(rt.path)
		)
		SELECT id, name, parent_id, created_at FROM role_tree
		ORDER BY level, name
	`

	err := db.Raw(query).Scan(&hierarchy).Error
	return hierarchy, err
}

// AssignPermission ロールに権限を付与
func (r *Role) AssignPermission(db *gorm.DB, permission *Permission) error {
	return db.Model(r).Association("Permissions").Append(permission)
}

// RevokePermission ロールから権限を剥奪
func (r *Role) RevokePermission(db *gorm.DB, permission *Permission) error {
	return db.Model(r).Association("Permissions").Delete(permission)
}

// AssignPermissions 複数の権限を一括付与
func (r *Role) AssignPermissions(db *gorm.DB, permissions []Permission) error {
	return db.Model(r).Association("Permissions").Append(permissions)
}

// RevokeAllPermissions すべての権限を剥奪
func (r *Role) RevokeAllPermissions(db *gorm.DB) error {
	return db.Model(r).Association("Permissions").Clear()
}
