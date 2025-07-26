package services

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"erp-access-control-go/models"
	"erp-access-control-go/pkg/errors"
)

// TokenRevocationService handles JWT token revocation
type TokenRevocationService struct {
	db *gorm.DB
}

// NewTokenRevocationService creates a new token revocation service
func NewTokenRevocationService(db *gorm.DB) *TokenRevocationService {
	return &TokenRevocationService{db: db}
}

// RevokeToken revokes a JWT token by storing its JTI in the revoked_tokens table
func (s *TokenRevocationService) RevokeToken(jti string, userID uuid.UUID, reason string) error {
	revokedToken := models.RevokedToken{
		JTI:       jti,
		UserID:    userID,
		RevokedAt: time.Now(),
		Reason:    reason,
	}

	if err := s.db.Create(&revokedToken).Error; err != nil {
		return errors.NewDatabaseError(err)
	}

	return nil
}

// IsTokenRevoked checks if a token (by JTI) has been revoked
func (s *TokenRevocationService) IsTokenRevoked(jti string) (bool, error) {
	var count int64
	if err := s.db.Model(&models.RevokedToken{}).Where("jti = ?", jti).Count(&count).Error; err != nil {
		return false, errors.NewDatabaseError(err)
	}

	return count > 0, nil
}

// RevokeAllUserTokens revokes all tokens for a specific user
func (s *TokenRevocationService) RevokeAllUserTokens(userID uuid.UUID, reason string) error {
	// Get all active sessions/tokens for the user
	// Since we can't get all JTIs from active tokens, we'll use a different approach
	// We'll store a "revoke_all_before" timestamp for the user
	
	revokedToken := models.RevokedToken{
		JTI:       "*", // Special marker for "revoke all"
		UserID:    userID,
		RevokedAt: time.Now(),
		Reason:    reason,
	}

	if err := s.db.Create(&revokedToken).Error; err != nil {
		return errors.NewDatabaseError(err)
	}

	return nil
}

// IsUserTokensRevoked checks if all tokens for a user were revoked after a certain time
func (s *TokenRevocationService) IsUserTokensRevoked(userID uuid.UUID, issuedAt time.Time) (bool, error) {
	var revokedToken models.RevokedToken
	err := s.db.Where("user_id = ? AND jti = ? AND revoked_at > ?", userID, "*", issuedAt).
		First(&revokedToken).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, errors.NewDatabaseError(err)
	}

	return true, nil
}

// CleanupExpiredTokens removes expired revoked tokens from the database
func (s *TokenRevocationService) CleanupExpiredTokens(olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)
	
	if err := s.db.Where("revoked_at < ?", cutoffTime).Delete(&models.RevokedToken{}).Error; err != nil {
		return errors.NewDatabaseError(err)
	}

	return nil
}

// GetRevokedTokens retrieves paginated list of revoked tokens
func (s *TokenRevocationService) GetRevokedTokens(page, limit int) ([]models.RevokedToken, int64, error) {
	var tokens []models.RevokedToken
	var total int64

	// Get total count
	if err := s.db.Model(&models.RevokedToken{}).Count(&total).Error; err != nil {
		return nil, 0, errors.NewDatabaseError(err)
	}

	// Get paginated results
	offset := (page - 1) * limit
	if err := s.db.Preload("User").
		Order("revoked_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&tokens).Error; err != nil {
		return nil, 0, errors.NewDatabaseError(err)
	}

	return tokens, total, nil
}

// GetUserRevokedTokens retrieves revoked tokens for a specific user
func (s *TokenRevocationService) GetUserRevokedTokens(userID uuid.UUID, page, limit int) ([]models.RevokedToken, int64, error) {
	var tokens []models.RevokedToken
	var total int64

	// Get total count for user
	if err := s.db.Model(&models.RevokedToken{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, errors.NewDatabaseError(err)
	}

	// Get paginated results for user
	offset := (page - 1) * limit
	if err := s.db.Where("user_id = ?", userID).
		Order("revoked_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&tokens).Error; err != nil {
		return nil, 0, errors.NewDatabaseError(err)
	}

	return tokens, total, nil
}

// ValidateTokenStatus performs comprehensive token validation
func (s *TokenRevocationService) ValidateTokenStatus(jti string, userID uuid.UUID, issuedAt time.Time) error {
	// Check if specific token is revoked
	isRevoked, err := s.IsTokenRevoked(jti)
	if err != nil {
		return err
	}
	if isRevoked {
		return errors.ErrInvalidToken
	}

	// Check if all user tokens were revoked after this token was issued
	allRevoked, err := s.IsUserTokensRevoked(userID, issuedAt)
	if err != nil {
		return err
	}
	if allRevoked {
		return errors.ErrInvalidToken
	}

	return nil
} 