package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RevokedToken 無効化されたJWTトークンテーブル
type RevokedToken struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	TokenJTI  string    `gorm:"uniqueIndex;not null" json:"token_jti"` // JWT ID
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	RevokedAt time.Time `gorm:"autoCreateTime" json:"revoked_at"`
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`

	// リレーション
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

// TableName テーブル名を指定
func (RevokedToken) TableName() string {
	return "revoked_tokens"
}

// BeforeCreate 作成前のバリデーション
func (rt *RevokedToken) BeforeCreate(tx *gorm.DB) error {
	// JTIが空でないかチェック
	if rt.TokenJTI == "" {
		return gorm.ErrInvalidValue
	}

	// 有効期限が過去でないかチェック
	if rt.ExpiresAt.Before(time.Now()) {
		return gorm.ErrInvalidValue
	}

	return nil
}

// BeforeUpdate 更新前のバリデーション
func (rt *RevokedToken) BeforeUpdate(tx *gorm.DB) error {
	// JTIが空でないかチェック
	if rt.TokenJTI == "" {
		return gorm.ErrInvalidValue
	}

	return nil
}

// =============================================================================
// トークン管理のメソッド
// =============================================================================

// IsExpired トークンが期限切れかチェック
func (rt *RevokedToken) IsExpired() bool {
	return rt.ExpiresAt.Before(time.Now())
}

// TimeUntilExpiry 有効期限までの時間を取得
func (rt *RevokedToken) TimeUntilExpiry() time.Duration {
	if rt.IsExpired() {
		return 0
	}
	return time.Until(rt.ExpiresAt)
}

// IsRecentlyRevoked 最近無効化されたかチェック（1時間以内）
func (rt *RevokedToken) IsRecentlyRevoked() bool {
	return time.Since(rt.RevokedAt) < time.Hour
}

// GetRevocationAge 無効化からの経過時間を取得
func (rt *RevokedToken) GetRevocationAge() time.Duration {
	return time.Since(rt.RevokedAt)
}

// =============================================================================
// クエリ用ヘルパー関数
// =============================================================================

// FindRevokedTokenByID IDで無効化トークンを検索
func FindRevokedTokenByID(db *gorm.DB, id int) (*RevokedToken, error) {
	var revokedToken RevokedToken
	err := db.Preload("User").Where("id = ?", id).First(&revokedToken).Error
	if err != nil {
		return nil, err
	}
	return &revokedToken, nil
}

// FindRevokedTokenByJTI JTIで無効化トークンを検索
func FindRevokedTokenByJTI(db *gorm.DB, jti string) (*RevokedToken, error) {
	var revokedToken RevokedToken
	err := db.Preload("User").Where("token_jti = ?", jti).First(&revokedToken).Error
	if err != nil {
		return nil, err
	}
	return &revokedToken, nil
}

// FindRevokedTokensByUser ユーザーIDで無効化トークンを検索
func FindRevokedTokensByUser(db *gorm.DB, userID uuid.UUID) ([]RevokedToken, error) {
	var revokedTokens []RevokedToken
	err := db.Where("user_id = ?", userID).Order("revoked_at DESC").Find(&revokedTokens).Error
	return revokedTokens, err
}

// FindExpiredRevokedTokens 期限切れの無効化トークンを検索
func FindExpiredRevokedTokens(db *gorm.DB) ([]RevokedToken, error) {
	var revokedTokens []RevokedToken
	err := db.Where("expires_at < ?", time.Now()).Find(&revokedTokens).Error
	return revokedTokens, err
}

// FindRevokedTokensByTimeRange 期間で無効化トークンを検索
func FindRevokedTokensByTimeRange(db *gorm.DB, startTime, endTime time.Time) ([]RevokedToken, error) {
	var revokedTokens []RevokedToken
	err := db.Preload("User").Where("revoked_at BETWEEN ? AND ?", startTime, endTime).
		Order("revoked_at DESC").Find(&revokedTokens).Error
	return revokedTokens, err
}

// =============================================================================
// トークン管理用ヘルパー関数
// =============================================================================

// CreateRevokedToken 無効化トークンを作成
func CreateRevokedToken(db *gorm.DB, jti string, userID uuid.UUID, expiresAt time.Time) (*RevokedToken, error) {
	revokedToken := &RevokedToken{
		TokenJTI:  jti,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}

	err := db.Create(revokedToken).Error
	if err != nil {
		return nil, err
	}

	return revokedToken, nil
}

// RevokeToken トークンを無効化（既存の場合は何もしない）
func RevokeToken(db *gorm.DB, jti string, userID uuid.UUID, expiresAt time.Time) error {
	// 既に無効化されているかチェック
	var existingToken RevokedToken
	err := db.Where("token_jti = ?", jti).First(&existingToken).Error

	if err == nil {
		// 既に存在する場合は何もしない
		return nil
	}

	if err != gorm.ErrRecordNotFound {
		// 検索エラーの場合
		return err
	}

	// 新規作成
	_, err = CreateRevokedToken(db, jti, userID, expiresAt)
	return err
}

// IsTokenRevoked トークンが無効化されているかチェック
func IsTokenRevoked(db *gorm.DB, jti string) (bool, error) {
	var count int64
	err := db.Model(&RevokedToken{}).Where("token_jti = ?", jti).Count(&count).Error
	return count > 0, err
}

// RevokeAllUserTokens ユーザーのすべてのトークンを無効化マーク
// 注意: 実際のJWTトークンのJTIを知るには、アプリケーション側でトークンを管理する必要があります
func RevokeAllUserTokens(db *gorm.DB, userID uuid.UUID) error {
	// この関数は概念的なもので、実際の実装では
	// アプリケーション側でユーザーのアクティブなトークン一覧を管理する必要があります

	// 既存の無効化トークンの有効期限を現在時刻に設定することで、
	// そのユーザーの既存トークンを無効化する効果を持たせることもできます

	return db.Model(&RevokedToken{}).
		Where("user_id = ? AND expires_at > ?", userID, time.Now()).
		Update("expires_at", time.Now()).Error
}

// CleanupExpiredTokens 期限切れの無効化トークンを削除
func CleanupExpiredTokens(db *gorm.DB) (int64, error) {
	result := db.Where("expires_at < ?", time.Now()).Delete(&RevokedToken{})
	return result.RowsAffected, result.Error
}

// ScheduleTokenCleanup 定期的なトークンクリーンアップ（時間間隔指定）
func ScheduleTokenCleanup(db *gorm.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		for range ticker.C {
			_, err := CleanupExpiredTokens(db)
			if err != nil {
				// ログ出力などのエラーハンドリング
				continue
			}

			// TODO: 削除されたトークン数をログ出力する場合は以下を有効化
			// deletedCount, err := CleanupExpiredTokens(db)
			// if deletedCount > 0 {
			//     log.Printf("Cleaned up %d expired revoked tokens", deletedCount)
			// }
		}
	}()
}

// GetTokenStats トークン統計を取得
func GetTokenStats(db *gorm.DB) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 全体統計
	var totalCount int64
	db.Model(&RevokedToken{}).Count(&totalCount)
	stats["total"] = totalCount

	// 期限切れトークン数
	var expiredCount int64
	db.Model(&RevokedToken{}).Where("expires_at < ?", time.Now()).Count(&expiredCount)
	stats["expired"] = expiredCount

	// アクティブな無効化トークン数
	activeCount := totalCount - expiredCount
	stats["active"] = activeCount

	// 最近無効化されたトークン数（24時間以内）
	var recentCount int64
	db.Model(&RevokedToken{}).Where("revoked_at > ?", time.Now().Add(-24*time.Hour)).Count(&recentCount)
	stats["recent_revoked"] = recentCount

	// ユーザー別統計（上位10名）
	var userStats []struct {
		UserID uuid.UUID `json:"user_id"`
		Count  int64     `json:"count"`
	}

	err := db.Model(&RevokedToken{}).
		Select("user_id, COUNT(*) as count").
		Group("user_id").
		Order("count DESC").
		Limit(10).
		Scan(&userStats).Error

	if err != nil {
		return nil, err
	}

	stats["top_users"] = userStats

	// 日別統計（過去7日）
	var dailyStats []struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		var dayCount int64
		db.Model(&RevokedToken{}).
			Where("revoked_at >= ? AND revoked_at < ?", startOfDay, endOfDay).
			Count(&dayCount)

		dailyStats = append(dailyStats, struct {
			Date  string `json:"date"`
			Count int64  `json:"count"`
		}{
			Date:  date.Format("2006-01-02"),
			Count: dayCount,
		})
	}

	stats["daily_revocations"] = dailyStats

	return stats, nil
}

// GetUserTokenHistory ユーザーのトークン無効化履歴を取得
func GetUserTokenHistory(db *gorm.DB, userID uuid.UUID, limit int) ([]RevokedToken, error) {
	var revokedTokens []RevokedToken
	query := db.Where("user_id = ?", userID).Order("revoked_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&revokedTokens).Error
	return revokedTokens, err
}

// BatchRevokeTokens 複数のトークンを一括無効化
func BatchRevokeTokens(db *gorm.DB, tokens []struct {
	JTI       string
	UserID    uuid.UUID
	ExpiresAt time.Time
}) error {
	if len(tokens) == 0 {
		return nil
	}

	revokedTokens := make([]RevokedToken, len(tokens))
	for i, token := range tokens {
		revokedTokens[i] = RevokedToken{
			TokenJTI:  token.JTI,
			UserID:    token.UserID,
			ExpiresAt: token.ExpiresAt,
		}
	}

	// バッチ挿入（重複は無視）
	return db.Create(&revokedTokens).Error
}
