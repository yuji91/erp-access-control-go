package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserScope ユーザースコープテーブル
type UserScope struct {
	ID           int       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	ResourceType string    `gorm:"not null;index" json:"resource_type"`
	ResourceID   *string   `gorm:"index" json:"resource_id,omitempty"`
	ScopeType    ScopeType `gorm:"not null;check:scope_type IN ('department','region','project','location')" json:"scope_type"`
	ScopeValue   JSONB     `gorm:"type:jsonb;not null" json:"scope_value"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`

	// リレーション
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

// TableName テーブル名を指定
func (UserScope) TableName() string {
	return "user_scopes"
}

// BeforeCreate 作成前のバリデーション
func (us *UserScope) BeforeCreate(tx *gorm.DB) error {
	// スコープタイプの妥当性チェック
	if !ValidateScopeType(us.ScopeType) {
		return gorm.ErrInvalidValue
	}

	// ScopeValueがJSONオブジェクトかチェック
	if us.ScopeValue == nil {
		return gorm.ErrInvalidValue
	}

	return nil
}

// BeforeUpdate 更新前のバリデーション
func (us *UserScope) BeforeUpdate(tx *gorm.DB) error {
	// スコープタイプの妥当性チェック
	if !ValidateScopeType(us.ScopeType) {
		return gorm.ErrInvalidValue
	}

	// ScopeValueがJSONオブジェクトかチェック
	if us.ScopeValue == nil {
		return gorm.ErrInvalidValue
	}

	return nil
}

// =============================================================================
// スコープ管理のメソッド
// =============================================================================

// GetScopeValueAsString ScopeValueから特定キーの文字列値を取得
func (us *UserScope) GetScopeValueAsString(key string) (string, bool) {
	if us.ScopeValue == nil {
		return "", false
	}

	value, exists := us.ScopeValue[key]
	if !exists {
		return "", false
	}

	str, ok := value.(string)
	return str, ok
}

// GetScopeValueAsStringSlice ScopeValueから特定キーの文字列配列を取得
func (us *UserScope) GetScopeValueAsStringSlice(key string) ([]string, bool) {
	if us.ScopeValue == nil {
		return nil, false
	}

	value, exists := us.ScopeValue[key]
	if !exists {
		return nil, false
	}

	// interface{}のスライスから文字列スライスに変換
	if arr, ok := value.([]interface{}); ok {
		strSlice := make([]string, len(arr))
		for i, v := range arr {
			if str, ok := v.(string); ok {
				strSlice[i] = str
			} else {
				return nil, false
			}
		}
		return strSlice, true
	}

	return nil, false
}

// SetScopeValue ScopeValueに値を設定
func (us *UserScope) SetScopeValue(key string, value interface{}) {
	if us.ScopeValue == nil {
		us.ScopeValue = make(JSONB)
	}
	us.ScopeValue[key] = value
}

// HasScopeValue 特定のキーが存在するかチェック
func (us *UserScope) HasScopeValue(key string) bool {
	if us.ScopeValue == nil {
		return false
	}
	_, exists := us.ScopeValue[key]
	return exists
}

// MatchesScope 指定したスコープ条件にマッチするかチェック
func (us *UserScope) MatchesScope(conditions JSONB) bool {
	if us.ScopeValue == nil || conditions == nil {
		return false
	}

	for key, expectedValue := range conditions {
		actualValue, exists := us.ScopeValue[key]
		if !exists || actualValue != expectedValue {
			return false
		}
	}

	return true
}

// IsValidResourceType リソースタイプが有効かチェック
func (us *UserScope) IsValidResourceType() bool {
	validResourceTypes := []string{
		"inventory", "orders", "reports", "users",
		"departments", "roles", "permissions", "audit",
		"dashboard", "settings", "finance", "hr",
		"projects", "locations", "assets", "contracts",
	}

	for _, resourceType := range validResourceTypes {
		if us.ResourceType == resourceType {
			return true
		}
	}
	return false
}

// =============================================================================
// クエリ用ヘルパー関数
// =============================================================================

// FindUserScopeByID IDでユーザースコープを検索
func FindUserScopeByID(db *gorm.DB, id int) (*UserScope, error) {
	var userScope UserScope
	err := db.Preload("User").Where("id = ?", id).First(&userScope).Error
	if err != nil {
		return nil, err
	}
	return &userScope, nil
}

// FindUserScopesByUserID ユーザーIDでスコープを検索
func FindUserScopesByUserID(db *gorm.DB, userID uuid.UUID) ([]UserScope, error) {
	var userScopes []UserScope
	err := db.Where("user_id = ?", userID).Find(&userScopes).Error
	return userScopes, err
}

// FindUserScopesByResourceType リソースタイプでスコープを検索
func FindUserScopesByResourceType(db *gorm.DB, resourceType string) ([]UserScope, error) {
	var userScopes []UserScope
	err := db.Preload("User").Where("resource_type = ?", resourceType).Find(&userScopes).Error
	return userScopes, err
}

// FindUserScopesByUserAndResource ユーザーとリソースでスコープを検索
func FindUserScopesByUserAndResource(db *gorm.DB, userID uuid.UUID, resourceType string) ([]UserScope, error) {
	var userScopes []UserScope
	err := db.Where("user_id = ? AND resource_type = ?", userID, resourceType).Find(&userScopes).Error
	return userScopes, err
}

// FindUserScopesByScopeType スコープタイプでスコープを検索
func FindUserScopesByScopeType(db *gorm.DB, scopeType ScopeType) ([]UserScope, error) {
	var userScopes []UserScope
	err := db.Preload("User").Where("scope_type = ?", scopeType).Find(&userScopes).Error
	return userScopes, err
}

// FindUserScopesByJSONBContains JSONBの内容でスコープを検索
func FindUserScopesByJSONBContains(db *gorm.DB, jsonQuery JSONB) ([]UserScope, error) {
	var userScopes []UserScope
	err := db.Preload("User").Where("scope_value @> ?", jsonQuery).Find(&userScopes).Error
	return userScopes, err
}

// FindUserScopesByJSONBKey JSONBの特定キーでスコープを検索
func FindUserScopesByJSONBKey(db *gorm.DB, key string) ([]UserScope, error) {
	var userScopes []UserScope
	err := db.Preload("User").Where("scope_value ? ?", key).Find(&userScopes).Error
	return userScopes, err
}

// =============================================================================
// スコープ管理用ヘルパー関数
// =============================================================================

// CreateUserScope ユーザースコープを作成
func CreateUserScope(db *gorm.DB, userID uuid.UUID, resourceType string, scopeType ScopeType, scopeValue JSONB) (*UserScope, error) {
	userScope := &UserScope{
		UserID:       userID,
		ResourceType: resourceType,
		ScopeType:    scopeType,
		ScopeValue:   scopeValue,
	}

	err := db.Create(userScope).Error
	if err != nil {
		return nil, err
	}

	return userScope, nil
}

// UpdateUserScope ユーザースコープを更新
func UpdateUserScope(db *gorm.DB, id int, scopeValue JSONB) error {
	return db.Model(&UserScope{}).Where("id = ?", id).Update("scope_value", scopeValue).Error
}

// DeleteUserScope ユーザースコープを削除
func DeleteUserScope(db *gorm.DB, id int) error {
	return db.Delete(&UserScope{}, id).Error
}

// DeleteUserScopesByUser ユーザーのすべてのスコープを削除
func DeleteUserScopesByUser(db *gorm.DB, userID uuid.UUID) error {
	return db.Where("user_id = ?", userID).Delete(&UserScope{}).Error
}

// DeleteUserScopesByUserAndResource ユーザーの特定リソースのスコープを削除
func DeleteUserScopesByUserAndResource(db *gorm.DB, userID uuid.UUID, resourceType string) error {
	return db.Where("user_id = ? AND resource_type = ?", userID, resourceType).Delete(&UserScope{}).Error
}

// GetUserResourceAccess ユーザーのリソースアクセス権を取得
func GetUserResourceAccess(db *gorm.DB, userID uuid.UUID, resourceType string) ([]UserScope, error) {
	var scopes []UserScope
	err := db.Where("user_id = ? AND resource_type = ?", userID, resourceType).Find(&scopes).Error
	return scopes, err
}

// CheckUserScopeAccess ユーザーが特定のスコープ条件でアクセス可能かチェック
func CheckUserScopeAccess(db *gorm.DB, userID uuid.UUID, resourceType string, conditions JSONB) (bool, error) {
	var count int64
	err := db.Model(&UserScope{}).
		Where("user_id = ? AND resource_type = ? AND scope_value @> ?", userID, resourceType, conditions).
		Count(&count).Error

	return count > 0, err
}

// GetUsersByScope 特定のスコープ条件を持つユーザーを取得
func GetUsersByScope(db *gorm.DB, resourceType string, scopeConditions JSONB) ([]User, error) {
	var users []User
	err := db.Joins("JOIN user_scopes us ON users.id = us.user_id").
		Where("us.resource_type = ? AND us.scope_value @> ?", resourceType, scopeConditions).
		Distinct().
		Find(&users).Error

	return users, err
}

// GetScopeStats スコープ統計を取得
func GetScopeStats(db *gorm.DB) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// リソースタイプ別統計
	var resourceStats []struct {
		ResourceType string `json:"resource_type"`
		Count        int64  `json:"count"`
	}

	err := db.Model(&UserScope{}).
		Select("resource_type, COUNT(*) as count").
		Group("resource_type").
		Scan(&resourceStats).Error

	if err != nil {
		return nil, err
	}

	stats["by_resource_type"] = resourceStats

	// スコープタイプ別統計
	var scopeTypeStats []struct {
		ScopeType ScopeType `json:"scope_type"`
		Count     int64     `json:"count"`
	}

	err = db.Model(&UserScope{}).
		Select("scope_type, COUNT(*) as count").
		Group("scope_type").
		Scan(&scopeTypeStats).Error

	if err != nil {
		return nil, err
	}

	stats["by_scope_type"] = scopeTypeStats

	// 総数
	var totalCount int64
	db.Model(&UserScope{}).Count(&totalCount)
	stats["total"] = totalCount

	return stats, nil
}
