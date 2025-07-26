package services

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"erp-access-control-go/models"
	"erp-access-control-go/pkg/errors"
)

// UserRoleService 複数ロール管理サービス
type UserRoleService struct {
	db *gorm.DB
}

// NewUserRoleService 新しいユーザーロールサービスを作成
func NewUserRoleService(db *gorm.DB) *UserRoleService {
	return &UserRoleService{
		db: db,
	}
}

// AssignRole ユーザーにロールを割り当て
func (s *UserRoleService) AssignRole(
	userID, roleID uuid.UUID,
	validFrom time.Time,
	validTo *time.Time,
	priority int,
	assignedBy uuid.UUID,
	reason string,
) (*models.UserRole, error) {
	// ユーザー存在確認
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("user not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// ロール存在確認
	var role models.Role
	if err := s.db.First(&role, roleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("role not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// 重複チェック（アクティブなロール）
	var count int64
	err := s.db.Model(&models.UserRole{}).
		Where("user_id = ? AND role_id = ? AND is_active = ?", userID, roleID, true).
		Count(&count).Error
	if err != nil {
		return nil, errors.NewDatabaseError(err)
	}
	if count > 0 {
		return nil, errors.NewValidationError("user already has this role assigned")
	}

	// UserRoleを作成
	userRole := &models.UserRole{
		UserID:         userID,
		RoleID:         roleID,
		ValidFrom:      validFrom,
		ValidTo:        validTo,
		Priority:       priority,
		IsActive:       true,
		AssignedBy:     &assignedBy,
		AssignedReason: reason,
	}

	if err := s.db.Create(userRole).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// ロール情報をPreload
	if err := s.db.Preload("Role").First(userRole, userRole.ID).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	return userRole, nil
}

// RevokeRole ユーザーのロールを取り消し
func (s *UserRoleService) RevokeRole(
	userID, roleID uuid.UUID,
	revokedBy uuid.UUID,
	reason string,
) (*models.UserRole, error) {
	var userRole models.UserRole
	
	// アクティブなUserRoleを検索
	err := s.db.Preload("Role").
		Where("user_id = ? AND role_id = ? AND is_active = ?", userID, roleID, true).
		First(&userRole).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("active user role not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// 取り消し処理
	now := time.Now()
	userRole.IsActive = false
	userRole.ValidTo = &now
	userRole.AssignedBy = &revokedBy
	userRole.AssignedReason = reason

	if err := s.db.Save(&userRole).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	return &userRole, nil
}

// UpdateRole ユーザーロールを更新
func (s *UserRoleService) UpdateRole(
	userID, roleID uuid.UUID,
	priority *int,
	validTo *time.Time,
	updatedBy uuid.UUID,
	reason string,
) (*models.UserRole, error) {
	var userRole models.UserRole
	
	// アクティブなUserRoleを検索
	err := s.db.Preload("Role").
		Where("user_id = ? AND role_id = ? AND is_active = ?", userID, roleID, true).
		First(&userRole).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("active user role not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// 更新処理
	updates := make(map[string]interface{})
	if priority != nil {
		updates["priority"] = *priority
	}
	if validTo != nil {
		updates["valid_to"] = *validTo
	}
	if reason != "" {
		updates["assigned_by"] = updatedBy
		updates["assigned_reason"] = reason
	}

	if len(updates) > 0 {
		if err := s.db.Model(&userRole).Updates(updates).Error; err != nil {
			return nil, errors.NewDatabaseError(err)
		}
	}

	// 更新後のデータを再取得
	if err := s.db.Preload("Role").First(&userRole, userRole.ID).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	return &userRole, nil
}

// GetUserRoles ユーザーのロール一覧を取得（全て）
func (s *UserRoleService) GetUserRoles(userID uuid.UUID) ([]models.UserRole, error) {
	var userRoles []models.UserRole
	
	err := s.db.Preload("Role").
		Where("user_id = ?", userID).
		Order("priority DESC, created_at ASC").
		Find(&userRoles).Error
	if err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	return userRoles, nil
}

// GetActiveUserRoles ユーザーのアクティブロール一覧を取得
func (s *UserRoleService) GetActiveUserRoles(userID uuid.UUID) ([]models.UserRole, error) {
	var userRoles []models.UserRole
	
	err := s.db.Preload("Role").
		Where("user_id = ? AND is_active = ? AND valid_from <= ? AND (valid_to IS NULL OR valid_to > ?)", 
			userID, true, time.Now(), time.Now()).
		Order("priority DESC, created_at ASC").
		Find(&userRoles).Error
	if err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	return userRoles, nil
}

// GetUserRole 特定のUserRoleを取得
func (s *UserRoleService) GetUserRole(userID, roleID uuid.UUID) (*models.UserRole, error) {
	var userRole models.UserRole
	
	err := s.db.Preload("Role").
		Where("user_id = ? AND role_id = ?", userID, roleID).
		First(&userRole).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("user role not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	return &userRole, nil
}

// ExtendRole ロール期限を延長
func (s *UserRoleService) ExtendRole(
	userID, roleID uuid.UUID,
	newValidTo *time.Time,
	extendedBy uuid.UUID,
	reason string,
) (*models.UserRole, error) {
	var userRole models.UserRole
	
	// アクティブなUserRoleを検索
	err := s.db.Preload("Role").
		Where("user_id = ? AND role_id = ? AND is_active = ?", userID, roleID, true).
		First(&userRole).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("active user role not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// 期限延長処理
	userRole.ValidTo = newValidTo
	userRole.AssignedBy = &extendedBy
	userRole.AssignedReason = reason

	if err := s.db.Save(&userRole).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	return &userRole, nil
}

// UpdatePriority ロール優先度を更新
func (s *UserRoleService) UpdatePriority(
	userID, roleID uuid.UUID,
	newPriority int,
	updatedBy uuid.UUID,
	reason string,
) (*models.UserRole, error) {
	var userRole models.UserRole
	
	// アクティブなUserRoleを検索
	err := s.db.Preload("Role").
		Where("user_id = ? AND role_id = ? AND is_active = ?", userID, roleID, true).
		First(&userRole).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("active user role not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// 優先度更新処理
	userRole.Priority = newPriority
	userRole.AssignedBy = &updatedBy
	userRole.AssignedReason = reason

	if err := s.db.Save(&userRole).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	return &userRole, nil
}

// CleanupExpiredRoles 期限切れロールの自動無効化
func (s *UserRoleService) CleanupExpiredRoles() error {
	now := time.Now()
	
	err := s.db.Model(&models.UserRole{}).
		Where("is_active = ? AND valid_to IS NOT NULL AND valid_to < ?", true, now).
		Updates(map[string]interface{}{
			"is_active":       false,
			"assigned_reason": "auto_expired",
			"updated_at":      now,
		}).Error
	
	if err != nil {
		return errors.NewDatabaseError(err)
	}

	return nil
}

// GetUserRoleStats ユーザーロール統計情報を取得
func (s *UserRoleService) GetUserRoleStats(userID uuid.UUID) (map[string]int, error) {
	stats := make(map[string]int)

	// 総ロール数
	var totalCount int64
	if err := s.db.Model(&models.UserRole{}).Where("user_id = ?", userID).Count(&totalCount).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}
	stats["total"] = int(totalCount)

	// アクティブロール数
	var activeCount int64
	if err := s.db.Model(&models.UserRole{}).
		Where("user_id = ? AND is_active = ? AND valid_from <= ? AND (valid_to IS NULL OR valid_to > ?)", 
			userID, true, time.Now(), time.Now()).
		Count(&activeCount).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}
	stats["active"] = int(activeCount)

	// 期限切れロール数
	var expiredCount int64
	if err := s.db.Model(&models.UserRole{}).
		Where("user_id = ? AND is_active = ? AND valid_to IS NOT NULL AND valid_to < ?", 
			userID, true, time.Now()).
		Count(&expiredCount).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}
	stats["expired"] = int(expiredCount)

	return stats, nil
} 