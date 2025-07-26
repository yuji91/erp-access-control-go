package models

import (
	"net/mail"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User ユーザーテーブル
type User struct {
	BaseModelWithUpdate
	Name         string     `gorm:"not null" json:"name"`
	Email        string     `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"not null" json:"-"` // パスワードハッシュ（JSONレスポンスから除外）
	DepartmentID uuid.UUID  `gorm:"type:uuid;not null;index" json:"department_id"`
	RoleID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"role_id"` // TODO: 多対多関係への拡張検討
	Status       UserStatus `gorm:"not null;default:'active';check:status IN ('active','inactive','suspended')" json:"status"`

	// TODO: アーキテクチャ改善
	// - 複数ロール対応: UserRole中間テーブルの導入
	// - パスワード強度追跡: PasswordSetAt, LastPasswordChange
	// - ログイン履歴: LastLoginAt, LoginAttempts, LockoutUntil
	// - プロファイル拡張: FirstName, LastName, PhoneNumber, Timezone

	// リレーション
	Department       Department        `gorm:"foreignKey:DepartmentID;constraint:OnDelete:CASCADE" json:"department,omitempty"`
	Role             Role              `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE" json:"role,omitempty"`
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

// ChangeRole ロールを変更
func (u *User) ChangeRole(db *gorm.DB, newRoleID uuid.UUID) error {
	u.RoleID = newRoleID
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

// GetAllPermissions ユーザーの全権限を取得（階層ロール考慮）
func (u *User) GetAllPermissions(db *gorm.DB) ([]Permission, error) {
	var permissions []Permission
	query := `
		SELECT DISTINCT p.id, p.module, p.action, p.created_at
		FROM get_user_all_permissions(?) 
		AS permissions(module TEXT, action TEXT)
		JOIN permissions p ON p.module = permissions.module AND p.action = permissions.action
		ORDER BY p.module, p.action
	`

	err := db.Raw(query, u.ID).Scan(&permissions).Error
	return permissions, err
}

// HasPermission 特定の権限を持つかチェック
func (u *User) HasPermission(db *gorm.DB, module, action string) (bool, error) {
	var count int64
	query := `
		SELECT COUNT(*) FROM get_user_all_permissions(?) 
		AS permissions(module TEXT, action TEXT)
		WHERE permissions.module = ? AND permissions.action = ?
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
