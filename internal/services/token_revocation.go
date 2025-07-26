package services

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"erp-access-control-go/models"
	"erp-access-control-go/pkg/errors"
)

// TokenRevocationService JWTトークン無効化サービス
type TokenRevocationService struct {
	db *gorm.DB
}

// NewTokenRevocationService 新しいトークン無効化サービスを作成
func NewTokenRevocationService(db *gorm.DB) *TokenRevocationService {
	return &TokenRevocationService{db: db}
}

// RevokeToken JTIをrevoked_tokensテーブルに保存してJWTトークンを無効化
func (s *TokenRevocationService) RevokeToken(jti string, userID uuid.UUID, reason string) error {
	revokedToken := models.RevokedToken{
		TokenJTI:  jti,
		UserID:    userID,
		RevokedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // JWT expiration time
	}

	if err := s.db.Create(&revokedToken).Error; err != nil {
		return errors.NewDatabaseError(err)
	}

	return nil
}

// IsTokenRevoked トークン（JTI）が無効化されているかチェック
func (s *TokenRevocationService) IsTokenRevoked(jti string) (bool, error) {
	var count int64
	if err := s.db.Model(&models.RevokedToken{}).Where("token_jti = ?", jti).Count(&count).Error; err != nil {
		return false, errors.NewDatabaseError(err)
	}

	return count > 0, nil
}

// RevokeAllUserTokens 特定ユーザーの全トークンを無効化
func (s *TokenRevocationService) RevokeAllUserTokens(userID uuid.UUID, reason string) error {
	// Get all active sessions/tokens for the user
	// Since we can't get all JTIs from active tokens, we'll use a different approach
	// We'll store a "revoke_all_before" timestamp for the user

	revokedToken := models.RevokedToken{
		TokenJTI:  "*", // Special marker for "revoke all"
		UserID:    userID,
		RevokedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // JWT expiration time
	}

	if err := s.db.Create(&revokedToken).Error; err != nil {
		return errors.NewDatabaseError(err)
	}

	return nil
}

// IsUserTokensRevoked ユーザーの全トークンが特定時刻以降に無効化されたかチェック
func (s *TokenRevocationService) IsUserTokensRevoked(userID uuid.UUID, issuedAt time.Time) (bool, error) {
	var revokedToken models.RevokedToken
	err := s.db.Where("user_id = ? AND token_jti = ? AND revoked_at > ?", userID, "*", issuedAt).
		First(&revokedToken).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, errors.NewDatabaseError(err)
	}

	return true, nil
}

// CleanupExpiredTokens 期限切れの無効化トークンをデータベースから削除
func (s *TokenRevocationService) CleanupExpiredTokens(olderThan time.Duration) error {
	// TODO: パフォーマンス最適化
	// - バッチ削除処理 (一度に大量削除ではなく分割実行)
	// - インデックス最適化 (revoked_at、expires_atの複合インデックス)
	// - 統計情報収集 (削除件数、実行時間)
	// - 自動スケジューリング (cron job、定期実行)
	
	cutoffTime := time.Now().Add(-olderThan)

	if err := s.db.Where("revoked_at < ?", cutoffTime).Delete(&models.RevokedToken{}).Error; err != nil {
		return errors.NewDatabaseError(err)
	}

	return nil
}

// GetRevokedTokens ページネーション付きで無効化トークンリストを取得
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

// GetUserRevokedTokens 特定ユーザーの無効化トークンを取得
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

// ValidateTokenStatus 包括的なトークン状態検証を実行
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
