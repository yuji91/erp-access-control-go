package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRole ユーザー・ロール関連テーブル
type UserRole struct {
	BaseModelWithUpdate
	UserID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	RoleID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"role_id"`
	ValidFrom      time.Time  `gorm:"default:NOW()" json:"valid_from"`
	ValidTo        *time.Time `gorm:"default:null" json:"valid_to,omitempty"`
	Priority       int        `gorm:"default:1;check:priority > 0" json:"priority"`
	IsActive       bool       `gorm:"default:true" json:"is_active"`
	AssignedBy     *uuid.UUID `gorm:"type:uuid" json:"assigned_by,omitempty"`
	AssignedReason string     `gorm:"type:text" json:"assigned_reason,omitempty"`

	// リレーション
	User           User  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Role           Role  `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE" json:"role,omitempty"`
	AssignedByUser *User `gorm:"foreignKey:AssignedBy" json:"assigned_by_user,omitempty"`
}

// TableName テーブル名を指定
func (UserRole) TableName() string {
	return "user_roles"
}

// BeforeCreate 作成前のバリデーション
func (ur *UserRole) BeforeCreate(tx *gorm.DB) error {
	// 期限の妥当性チェック
	if ur.ValidTo != nil && ur.ValidFrom.After(*ur.ValidTo) {
		return gorm.ErrInvalidValue
	}

	// 重複チェック（同じユーザー・ロールの組み合わせでアクティブなレコード）
	var count int64
	err := tx.Model(&UserRole{}).
		Where("user_id = ? AND role_id = ? AND is_active = ?", ur.UserID, ur.RoleID, true).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return gorm.ErrDuplicatedKey
	}

	return nil
}

// BeforeUpdate 更新前のバリデーション
func (ur *UserRole) BeforeUpdate(tx *gorm.DB) error {
	// 期限の妥当性チェック
	if ur.ValidTo != nil && ur.ValidFrom.After(*ur.ValidTo) {
		return gorm.ErrInvalidValue
	}

	return nil
}

// =============================================================================
// UserRole管理のメソッド
// =============================================================================

// IsValidNow 現在時刻で有効かどうかを判定
func (ur *UserRole) IsValidNow() bool {
	now := time.Now()
	return ur.IsActive && 
		   ur.ValidFrom.Before(now) && 
		   (ur.ValidTo == nil || ur.ValidTo.After(now))
}

// IsExpired 期限切れかどうかを判定
func (ur *UserRole) IsExpired() bool {
	if ur.ValidTo == nil {
		return false
	}
	return ur.ValidTo.Before(time.Now())
}

// Deactivate ロールを無効化
func (ur *UserRole) Deactivate(db *gorm.DB, deactivatedBy uuid.UUID, reason string) error {
	ur.IsActive = false
	ur.ValidTo = &time.Time{}
	*ur.ValidTo = time.Now()
	ur.AssignedBy = &deactivatedBy
	ur.AssignedReason = reason

	return db.Save(ur).Error
}

// Extend ロール期限を延長
func (ur *UserRole) Extend(db *gorm.DB, newValidTo *time.Time, extendedBy uuid.UUID, reason string) error {
	ur.ValidTo = newValidTo
	ur.AssignedBy = &extendedBy
	ur.AssignedReason = reason

	return db.Save(ur).Error
}

// UpdatePriority 優先度を更新
func (ur *UserRole) UpdatePriority(db *gorm.DB, newPriority int, updatedBy uuid.UUID, reason string) error {
	ur.Priority = newPriority
	ur.AssignedBy = &updatedBy
	ur.AssignedReason = reason

	return db.Save(ur).Error
}

// =============================================================================
// クエリ用ヘルパー関数
// =============================================================================

// FindUserRolesByUserID ユーザーIDで UserRole を検索
func FindUserRolesByUserID(db *gorm.DB, userID uuid.UUID) ([]UserRole, error) {
	var userRoles []UserRole
	err := db.Preload("Role").Preload("AssignedByUser").
		Where("user_id = ?", userID).
		Order("priority DESC, created_at ASC").
		Find(&userRoles).Error
	return userRoles, err
}

// FindActiveUserRolesByUserID アクティブな UserRole を検索
func FindActiveUserRolesByUserID(db *gorm.DB, userID uuid.UUID) ([]UserRole, error) {
	var userRoles []UserRole
	now := time.Now()
	err := db.Preload("Role").Preload("AssignedByUser").
		Where("user_id = ? AND is_active = ? AND valid_from <= ? AND (valid_to IS NULL OR valid_to > ?)", 
			  userID, true, now, now).
		Order("priority DESC, created_at ASC").
		Find(&userRoles).Error
	return userRoles, err
}

// FindUserRolesByRoleID ロールIDで UserRole を検索
func FindUserRolesByRoleID(db *gorm.DB, roleID uuid.UUID) ([]UserRole, error) {
	var userRoles []UserRole
	err := db.Preload("User").Preload("AssignedByUser").
		Where("role_id = ?", roleID).
		Order("priority DESC, created_at ASC").
		Find(&userRoles).Error
	return userRoles, err
}

// FindExpiredUserRoles 期限切れの UserRole を検索
func FindExpiredUserRoles(db *gorm.DB) ([]UserRole, error) {
	var userRoles []UserRole
	now := time.Now()
	err := db.Preload("User").Preload("Role").
		Where("is_active = ? AND valid_to IS NOT NULL AND valid_to <= ?", true, now).
		Find(&userRoles).Error
	return userRoles, err
}

// CleanupExpiredUserRoles 期限切れのUserRoleを自動無効化
func CleanupExpiredUserRoles(db *gorm.DB) (int64, error) {
	now := time.Now()
	result := db.Model(&UserRole{}).
		Where("is_active = ? AND valid_to IS NOT NULL AND valid_to <= ?", true, now).
		Updates(map[string]interface{}{
			"is_active":       false,
			"assigned_reason": "Automatically deactivated due to expiration",
			"updated_at":      now,
		})
	
	return result.RowsAffected, result.Error
}

// GetUserRoleStats UserRole統計を取得
func GetUserRoleStats(db *gorm.DB) (map[string]int64, error) {
	stats := make(map[string]int64)

	// アクティブな UserRole 数
	var activeCount, inactiveCount, expiredCount int64

	now := time.Now()

	// アクティブ
	db.Model(&UserRole{}).
		Where("is_active = ? AND valid_from <= ? AND (valid_to IS NULL OR valid_to > ?)", 
			  true, now, now).
		Count(&activeCount)

	// 非アクティブ
	db.Model(&UserRole{}).
		Where("is_active = ?", false).
		Count(&inactiveCount)

	// 期限切れ（アクティブだが期限切れ）
	db.Model(&UserRole{}).
		Where("is_active = ? AND valid_to IS NOT NULL AND valid_to <= ?", 
			  true, now).
		Count(&expiredCount)

	stats["active"] = activeCount
	stats["inactive"] = inactiveCount  
	stats["expired"] = expiredCount
	stats["total"] = activeCount + inactiveCount + expiredCount

	return stats, nil
} 