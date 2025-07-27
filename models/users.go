package models

import (
	"net/mail"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User ユーザーテーブル（複数ロール対応版）
type User struct {
	BaseModelWithUpdate
	Name           string     `gorm:"not null" json:"name"`
	Email          string     `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash   string     `gorm:"not null" json:"-"` // パスワードハッシュ（JSONレスポンスから除外）
	DepartmentID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"department_id"`
	RoleID         *uuid.UUID `gorm:"type:uuid;index" json:"role_id,omitempty"` // 旧: 後方互換性のため保持（段階的削除予定）
	PrimaryRoleID  *uuid.UUID `gorm:"type:uuid;index" json:"primary_role_id,omitempty"` // メインロール
	Status         UserStatus `gorm:"not null;default:'active';check:status IN ('active','inactive','suspended')" json:"status"`

	// TODO: アーキテクチャ改善
	// - パスワード強度追跡: PasswordSetAt, LastPasswordChange
	// - ログイン履歴: LastLoginAt, LoginAttempts, LockoutUntil
	// - プロファイル拡張: FirstName, LastName, PhoneNumber, Timezone

	// リレーション
	Department       Department        `gorm:"foreignKey:DepartmentID;constraint:OnDelete:CASCADE" json:"department,omitempty"`
	Role             *Role             `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE" json:"role,omitempty"` // 旧: 後方互換性
	PrimaryRole      *Role             `gorm:"foreignKey:PrimaryRoleID;constraint:OnDelete:SET NULL" json:"primary_role,omitempty"`
	UserRoles        []UserRole        `gorm:"foreignKey:UserID" json:"user_roles,omitempty"`
	ActiveUserRoles  []UserRole        `gorm:"foreignKey:UserID;->:false" json:"active_user_roles,omitempty"` // カスタムクエリ用
	UserScopes       []UserScope       `gorm:"foreignKey:UserID" json:"user_scopes,omitempty"`
	AuditLogs        []AuditLog        `gorm:"foreignKey:UserID" json:"audit_logs,omitempty"`
	TimeRestrictions []TimeRestriction `gorm:"foreignKey:UserID" json:"time_restrictions,omitempty"`
	RevokedTokens    []RevokedToken    `gorm:"foreignKey:UserID" json:"revoked_tokens,omitempty"`
}

// TableName テーブル名を指定
func (User) TableName() string {
	return "users"
}

// BeforeCreate 作成前のバリデーション
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// メールアドレスの妥当性チェック
	if !u.IsValidEmail() {
		return gorm.ErrInvalidValue
	}

	// ステータスの妥当性チェック
	if !ValidateUserStatus(u.Status) {
		return gorm.ErrInvalidValue
	}

	return nil
}

// BeforeUpdate 更新前のバリデーション
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	// メールアドレスの妥当性チェック
	if !u.IsValidEmail() {
		return gorm.ErrInvalidValue
	}

	// ステータスの妥当性チェック
	if !ValidateUserStatus(u.Status) {
		return gorm.ErrInvalidValue
	}

	return nil
}

// =============================================================================
// ユーザー管理のメソッド
// =============================================================================

// IsValidEmail メールアドレスの妥当性チェック
func (u *User) IsValidEmail() bool {
	_, err := mail.ParseAddress(u.Email)
	return err == nil
}

// HashPassword パスワードをハッシュ化
func (u *User) HashPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword パスワードを検証
func (u *User) CheckPassword(password string) bool {
	if u.PasswordHash == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// IsActive アクティブなユーザーかどうかを判定
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsInactive 非アクティブなユーザーかどうかを判定
func (u *User) IsInactive() bool {
	return u.Status == UserStatusInactive
}

// IsSuspended 停止中のユーザーかどうかを判定
func (u *User) IsSuspended() bool {
	return u.Status == UserStatusSuspended
}

// Activate ユーザーをアクティブ化
func (u *User) Activate(db *gorm.DB) error {
	u.Status = UserStatusActive
	return db.Save(u).Error
}

// Deactivate ユーザーを非アクティブ化
func (u *User) Deactivate(db *gorm.DB) error {
	u.Status = UserStatusInactive
	return db.Save(u).Error
}

// Suspend ユーザーを停止
func (u *User) Suspend(db *gorm.DB) error {
	u.Status = UserStatusSuspended
	return db.Save(u).Error
}

// ChangeRole ロールを変更（後方互換性）
func (u *User) ChangeRole(db *gorm.DB, newRoleID uuid.UUID) error {
	u.RoleID = &newRoleID
	return db.Save(u).Error
}

// ChangePrimaryRole プライマリロールを変更
func (u *User) ChangePrimaryRole(db *gorm.DB, newRoleID uuid.UUID) error {
	u.PrimaryRoleID = &newRoleID
	return db.Save(u).Error
}

// ChangeDepartment 部門を変更
func (u *User) ChangeDepartment(db *gorm.DB, newDepartmentID uuid.UUID) error {
	u.DepartmentID = newDepartmentID
	return db.Save(u).Error
}

// =============================================================================
// 権限関連のメソッド
// =============================================================================

// GetAllPermissions ユーザーの全権限を取得（複数ロール・階層考慮）
func (u *User) GetAllPermissions(db *gorm.DB) ([]Permission, error) {
	var permissions []Permission
	
	// 複数ロールからの権限集約クエリ
	query := `
		SELECT DISTINCT p.id, p.module, p.action, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = ? 
			AND ur.is_active = true
			AND ur.valid_from <= NOW()
			AND (ur.valid_to IS NULL OR ur.valid_to > NOW())
		ORDER BY p.module, p.action
	`

	err := db.Raw(query, u.ID).Scan(&permissions).Error
	return permissions, err
}

// HasPermission 特定の権限を持つかチェック（複数ロール対応）
func (u *User) HasPermission(db *gorm.DB, module, action string) (bool, error) {
	var count int64
	
	query := `
		SELECT COUNT(DISTINCT p.id)
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = ? 
			AND ur.is_active = true
			AND ur.valid_from <= NOW()
			AND (ur.valid_to IS NULL OR ur.valid_to > NOW())
			AND p.module = ? AND p.action = ?
	`

	err := db.Raw(query, u.ID, module, action).Count(&count).Error
	return count > 0, err
}

// GetDepartmentUsers 同じ部門のユーザーを取得
func (u *User) GetDepartmentUsers(db *gorm.DB) ([]User, error) {
	var users []User
	err := db.Where("department_id = ? AND id != ?", u.DepartmentID, u.ID).Find(&users).Error
	return users, err
}

// GetRoleUsers 同じロールのユーザーを取得
func (u *User) GetRoleUsers(db *gorm.DB) ([]User, error) {
	var users []User
	err := db.Where("role_id = ? AND id != ?", u.RoleID, u.ID).Find(&users).Error
	return users, err
}

// =============================================================================
// クエリ用ヘルパー関数
// =============================================================================

// FindUserByID IDでユーザーを検索
func FindUserByID(db *gorm.DB, id uuid.UUID) (*User, error) {
	var user User
	err := db.Preload("Department").Preload("Role").Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindUserByEmail メールアドレスでユーザーを検索
func FindUserByEmail(db *gorm.DB, email string) (*User, error) {
	var user User
	err := db.Preload("Department").Preload("Role").Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindUsersByDepartment 部門でユーザーを検索
func FindUsersByDepartment(db *gorm.DB, departmentID uuid.UUID) ([]User, error) {
	var users []User
	err := db.Preload("Department").Preload("Role").
		Where("department_id = ?", departmentID).Find(&users).Error
	return users, err
}

// FindUsersByRole ロールでユーザーを検索
func FindUsersByRole(db *gorm.DB, roleID uuid.UUID) ([]User, error) {
	var users []User
	err := db.Preload("Department").Preload("Role").
		Where("role_id = ?", roleID).Find(&users).Error
	return users, err
}

// FindUsersByStatus ステータスでユーザーを検索
func FindUsersByStatus(db *gorm.DB, status UserStatus) ([]User, error) {
	var users []User
	err := db.Preload("Department").Preload("Role").
		Where("status = ?", status).Find(&users).Error
	return users, err
}

// FindActiveUsers アクティブなユーザーを検索
func FindActiveUsers(db *gorm.DB) ([]User, error) {
	return FindUsersByStatus(db, UserStatusActive)
}

// FindInactiveUsers 非アクティブなユーザーを検索
func FindInactiveUsers(db *gorm.DB) ([]User, error) {
	return FindUsersByStatus(db, UserStatusInactive)
}

// FindSuspendedUsers 停止中のユーザーを検索
func FindSuspendedUsers(db *gorm.DB) ([]User, error) {
	return FindUsersByStatus(db, UserStatusSuspended)
}

// SearchUsers ユーザーを検索（名前・メールアドレス部分一致）
func SearchUsers(db *gorm.DB, keyword string) ([]User, error) {
	var users []User
	searchPattern := "%" + keyword + "%"
	err := db.Preload("Department").Preload("Role").
		Where("name ILIKE ? OR email ILIKE ?", searchPattern, searchPattern).
		Find(&users).Error
	return users, err
}

// GetUserStats ユーザー統計を取得
func GetUserStats(db *gorm.DB) (map[string]int64, error) {
	stats := make(map[string]int64)

	// ステータス別統計
	var activeCount, inactiveCount, suspendedCount int64

	db.Model(&User{}).Where("status = ?", UserStatusActive).Count(&activeCount)
	db.Model(&User{}).Where("status = ?", UserStatusInactive).Count(&inactiveCount)
	db.Model(&User{}).Where("status = ?", UserStatusSuspended).Count(&suspendedCount)

	stats["active"] = activeCount
	stats["inactive"] = inactiveCount
	stats["suspended"] = suspendedCount
	stats["total"] = activeCount + inactiveCount + suspendedCount

	return stats, nil
}

// GetUsersWithPermissionView 権限統合ビューからユーザー情報を取得
func GetUsersWithPermissionView(db *gorm.DB, module, action string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	query := `
		SELECT user_id, user_name, email, department_name, role_name, module, action, user_status
		FROM user_permissions_view
		WHERE module = ? AND action = ?
		ORDER BY user_name
	`

	err := db.Raw(query, module, action).Scan(&results).Error
	return results, err
}

// =============================================================================
// 複数ロール管理のメソッド
// =============================================================================

// GetActiveRoles アクティブなロールを取得
func (u *User) GetActiveRoles(db *gorm.DB) ([]Role, error) {
	var roles []Role
	
	err := db.Joins("JOIN user_roles ur ON roles.id = ur.role_id").
		Where("ur.user_id = ? AND ur.is_active = ? AND ur.valid_from <= ? AND (ur.valid_to IS NULL OR ur.valid_to > ?)", 
			  u.ID, true, time.Now(), time.Now()).
		Order("ur.priority DESC, ur.created_at ASC").
		Find(&roles).Error
	
	return roles, err
}

// GetHighestPriorityRole 最高優先度のロールを取得
func (u *User) GetHighestPriorityRole(db *gorm.DB) (*Role, error) {
	var role Role
	
	err := db.Joins("JOIN user_roles ur ON roles.id = ur.role_id").
		Where("ur.user_id = ? AND ur.is_active = ? AND ur.valid_from <= ? AND (ur.valid_to IS NULL OR ur.valid_to > ?)", 
			  u.ID, true, time.Now(), time.Now()).
		Order("ur.priority DESC, ur.created_at ASC").
		First(&role).Error
	
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// AssignRole ロールを割り当て
func (u *User) AssignRole(db *gorm.DB, roleID uuid.UUID, validFrom time.Time, validTo *time.Time, priority int, assignedBy uuid.UUID, reason string) error {
	userRole := UserRole{
		UserID:         u.ID,
		RoleID:         roleID,
		ValidFrom:      validFrom,
		ValidTo:        validTo,
		Priority:       priority,
		IsActive:       true,
		AssignedBy:     &assignedBy,
		AssignedReason: reason,
	}
	
	return db.Create(&userRole).Error
}

// RevokeRole ロールを取り消し
func (u *User) RevokeRole(db *gorm.DB, roleID uuid.UUID, revokedBy uuid.UUID, reason string) error {
	return db.Model(&UserRole{}).
		Where("user_id = ? AND role_id = ? AND is_active = ?", u.ID, roleID, true).
		Updates(map[string]interface{}{
			"is_active":       false,
			"valid_to":        time.Now(),
			"assigned_by":     revokedBy,
			"assigned_reason": reason,
			"updated_at":      time.Now(),
		}).Error
}

// UpdateRolePriority ロール優先度を更新
func (u *User) UpdateRolePriority(db *gorm.DB, roleID uuid.UUID, newPriority int, updatedBy uuid.UUID, reason string) error {
	return db.Model(&UserRole{}).
		Where("user_id = ? AND role_id = ? AND is_active = ?", u.ID, roleID, true).
		Updates(map[string]interface{}{
			"priority":        newPriority,
			"assigned_by":     updatedBy,
			"assigned_reason": reason,
			"updated_at":      time.Now(),
		}).Error
}

// ExtendRole ロール期限を延長
func (u *User) ExtendRole(db *gorm.DB, roleID uuid.UUID, newValidTo *time.Time, extendedBy uuid.UUID, reason string) error {
	return db.Model(&UserRole{}).
		Where("user_id = ? AND role_id = ? AND is_active = ?", u.ID, roleID, true).
		Updates(map[string]interface{}{
			"valid_to":        newValidTo,
			"assigned_by":     extendedBy,
			"assigned_reason": reason,
			"updated_at":      time.Now(),
		}).Error
}

// HasRoleActive 特定のロールがアクティブかチェック
func (u *User) HasRoleActive(db *gorm.DB, roleID uuid.UUID) (bool, error) {
	var count int64
	now := time.Now()
	
	err := db.Model(&UserRole{}).
		Where("user_id = ? AND role_id = ? AND is_active = ? AND valid_from <= ? AND (valid_to IS NULL OR valid_to > ?)", 
			  u.ID, roleID, true, now, now).
		Count(&count).Error
	
	return count > 0, err
}

// GetUserRoles ユーザーの全ロール（アクティブ・非アクティブ含む）を取得
func (u *User) GetUserRoles(db *gorm.DB) ([]UserRole, error) {
	var userRoles []UserRole
	
	err := db.Preload("Role").Preload("AssignedByUser").
		Where("user_id = ?", u.ID).
		Order("priority DESC, created_at ASC").
		Find(&userRoles).Error
	
	return userRoles, err
}

// GetActiveUserRoles アクティブなUserRoleを取得
func (u *User) GetActiveUserRoles(db *gorm.DB) ([]UserRole, error) {
	return FindActiveUserRolesByUserID(db, u.ID)
}
