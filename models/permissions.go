package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Permission 権限テーブル
type Permission struct {
	BaseModel
	Module string `gorm:"not null;index" json:"module"`
	Action string `gorm:"not null;index" json:"action"`

	// リレーション
	Roles []Role `gorm:"many2many:role_permissions;constraint:OnDelete:CASCADE" json:"roles,omitempty"`
}

// TableName テーブル名を指定
func (Permission) TableName() string {
	return "permissions"
}

// RolePermission ロール-権限の関連テーブル
type RolePermission struct {
	RoleID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"role_id"`
	PermissionID uuid.UUID `gorm:"type:uuid;primaryKey" json:"permission_id"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`

	// リレーション
	Role       Role       `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE" json:"role,omitempty"`
	Permission Permission `gorm:"foreignKey:PermissionID;constraint:OnDelete:CASCADE" json:"permission,omitempty"`
}

// TableName テーブル名を指定
func (RolePermission) TableName() string {
	return "role_permissions"
}

// =============================================================================
// 権限管理のメソッド
// =============================================================================

// GetUniqueKey 権限の一意キーを取得（module + action）
func (p *Permission) GetUniqueKey() string {
	return p.Module + ":" + p.Action
}

// String 権限の文字列表現
func (p *Permission) String() string {
	return p.GetUniqueKey()
}

// IsValidAction アクションが有効かチェック
func (p *Permission) IsValidAction() bool {
	validActions := []string{
		"view", "create", "update", "delete",
		"approve", "reject", "export", "import",
		"manage", "admin",
	}

	for _, action := range validActions {
		if p.Action == action {
			return true
		}
	}
	return false
}

// IsValidModule モジュールが有効かチェック
func (p *Permission) IsValidModule() bool {
	validModules := []string{
		"inventory", "orders", "reports", "users",
		"departments", "roles", "permissions", "audit",
		"dashboard", "settings", "finance", "hr",
	}

	for _, module := range validModules {
		if p.Module == module {
			return true
		}
	}
	return false
}

// BeforeCreate 作成前のバリデーション
func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	if !p.IsValidModule() || !p.IsValidAction() {
		return gorm.ErrInvalidData
	}
	return nil
}

// BeforeUpdate 更新前のバリデーション
func (p *Permission) BeforeUpdate(tx *gorm.DB) error {
	if !p.IsValidModule() || !p.IsValidAction() {
		return gorm.ErrInvalidData
	}
	return nil
}

// =============================================================================
// クエリ用ヘルパー関数
// =============================================================================

// FindPermissionByID IDで権限を検索
func FindPermissionByID(db *gorm.DB, id uuid.UUID) (*Permission, error) {
	var permission Permission
	err := db.Where("id = ?", id).First(&permission).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// FindPermissionByModuleAction モジュール・アクションで権限を検索
func FindPermissionByModuleAction(db *gorm.DB, module, action string) (*Permission, error) {
	var permission Permission
	err := db.Where("module = ? AND action = ?", module, action).First(&permission).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// FindPermissionsByModule モジュールで権限を検索
func FindPermissionsByModule(db *gorm.DB, module string) ([]Permission, error) {
	var permissions []Permission
	err := db.Where("module = ?", module).Find(&permissions).Error
	return permissions, err
}

// FindPermissionsByRole ロールに紐づく権限を検索
func FindPermissionsByRole(db *gorm.DB, roleID uuid.UUID) ([]Permission, error) {
	var permissions []Permission
	err := db.Joins("JOIN role_permissions rp ON permissions.id = rp.permission_id").
		Where("rp.role_id = ?", roleID).
		Find(&permissions).Error
	return permissions, err
}

// FindRolesByPermission 権限に紐づくロールを検索
func FindRolesByPermission(db *gorm.DB, permissionID uuid.UUID) ([]Role, error) {
	var roles []Role
	err := db.Joins("JOIN role_permissions rp ON roles.id = rp.role_id").
		Where("rp.permission_id = ?", permissionID).
		Find(&roles).Error
	return roles, err
}

// GetAllPermissions すべての権限を取得
func GetAllPermissions(db *gorm.DB) ([]Permission, error) {
	var permissions []Permission
	err := db.Order("module, action").Find(&permissions).Error
	return permissions, err
}

// GetPermissionsByModules 複数モジュールの権限を取得
func GetPermissionsByModules(db *gorm.DB, modules []string) ([]Permission, error) {
	var permissions []Permission
	err := db.Where("module IN ?", modules).Order("module, action").Find(&permissions).Error
	return permissions, err
}

// CreatePermissionIfNotExists 権限が存在しない場合は作成
func CreatePermissionIfNotExists(db *gorm.DB, module, action string) (*Permission, error) {
	permission, err := FindPermissionByModuleAction(db, module, action)
	if err == nil {
		return permission, nil // 既に存在
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err // 検索エラー
	}

	// 新規作成
	permission = &Permission{
		Module: module,
		Action: action,
	}

	err = db.Create(permission).Error
	if err != nil {
		return nil, err
	}

	return permission, nil
}

// =============================================================================
// ロール-権限関連のヘルパー関数
// =============================================================================

// AssignRolePermission ロールに権限を付与
func AssignRolePermission(db *gorm.DB, roleID, permissionID uuid.UUID) error {
	rolePermission := &RolePermission{
		RoleID:       roleID,
		PermissionID: permissionID,
	}

	return db.Create(rolePermission).Error
}

// RevokeRolePermission ロールから権限を剥奪
func RevokeRolePermission(db *gorm.DB, roleID, permissionID uuid.UUID) error {
	return db.Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Delete(&RolePermission{}).Error
}

// RevokeAllRolePermissions ロールからすべての権限を剥奪
func RevokeAllRolePermissions(db *gorm.DB, roleID uuid.UUID) error {
	return db.Where("role_id = ?", roleID).Delete(&RolePermission{}).Error
}

// RevokeAllPermissionRoles 権限からすべてのロールを剥奪
func RevokeAllPermissionRoles(db *gorm.DB, permissionID uuid.UUID) error {
	return db.Where("permission_id = ?", permissionID).Delete(&RolePermission{}).Error
}

// GetRolePermissionMatrix ロール-権限マトリクスを取得
func GetRolePermissionMatrix(db *gorm.DB) (map[string]map[string]bool, error) {
	var rolePermissions []struct {
		RoleName string `json:"role_name"`
		Module   string `json:"module"`
		Action   string `json:"action"`
	}

	query := `
		SELECT r.name as role_name, p.module, p.action
		FROM roles r
		JOIN role_permissions rp ON r.id = rp.role_id
		JOIN permissions p ON rp.permission_id = p.id
		ORDER BY r.name, p.module, p.action
	`

	err := db.Raw(query).Scan(&rolePermissions).Error
	if err != nil {
		return nil, err
	}

	matrix := make(map[string]map[string]bool)
	for _, rp := range rolePermissions {
		if matrix[rp.RoleName] == nil {
			matrix[rp.RoleName] = make(map[string]bool)
		}
		matrix[rp.RoleName][rp.Module+":"+rp.Action] = true
	}

	return matrix, nil
}
