package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// =============================================================================
// 共通型定義
// =============================================================================

// BaseModel すべてのモデルで共通する基本フィールド
type BaseModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// BaseModelWithUpdate 作成日時・更新日時を持つモデル用
type BaseModelWithUpdate struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// =============================================================================
// Enum定義
// =============================================================================

// UserStatus ユーザーステータス
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
)

// AuditResult 監査ログの結果
type AuditResult string

const (
	AuditResultSuccess AuditResult = "SUCCESS"
	AuditResultDenied  AuditResult = "DENIED"
	AuditResultError   AuditResult = "ERROR"
)

// ScopeType スコープタイプ
type ScopeType string

const (
	ScopeTypeDepartment ScopeType = "department"
	ScopeTypeRegion     ScopeType = "region"
	ScopeTypeProject    ScopeType = "project"
	ScopeTypeLocation   ScopeType = "location"
)

// =============================================================================
// カスタム型定義（PostgreSQL特有型対応）
// =============================================================================

// JSONB PostgreSQLのJSONB型に対応
type JSONB map[string]interface{}

// Value JSONBのdriver.Valuer実装
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan JSONBのdatabase/sql.Scanner実装
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		return errors.New("cannot scan into JSONB")
	}
}

// IntArray PostgreSQLのINTEGER[]型に対応
type IntArray pq.Int64Array

// Value IntArrayのdriver.Valuer実装
func (a IntArray) Value() (driver.Value, error) {
	return pq.Int64Array(a).Value()
}

// Scan IntArrayのdatabase/sql.Scanner実装
func (a *IntArray) Scan(value interface{}) error {
	return (*pq.Int64Array)(a).Scan(value)
}

// =============================================================================
// ヘルパー関数
// =============================================================================

// NewUUID 新しいUUIDを生成
func NewUUID() uuid.UUID {
	return uuid.New()
}

// ParseUUID 文字列からUUIDをパース
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// ValidateUserStatus ユーザーステータスの妥当性チェック
func ValidateUserStatus(status UserStatus) bool {
	switch status {
	case UserStatusActive, UserStatusInactive, UserStatusSuspended:
		return true
	default:
		return false
	}
}

// ValidateScopeType スコープタイプの妥当性チェック
func ValidateScopeType(scopeType ScopeType) bool {
	switch scopeType {
	case ScopeTypeDepartment, ScopeTypeRegion, ScopeTypeProject, ScopeTypeLocation:
		return true
	default:
		return false
	}
}

// ValidateAuditResult 監査結果の妥当性チェック
func ValidateAuditResult(result AuditResult) bool {
	switch result {
	case AuditResultSuccess, AuditResultDenied, AuditResultError:
		return true
	default:
		return false
	}
}

// =============================================================================
// GORM Hooks用インターフェース
// =============================================================================

// BeforeCreate 作成前処理のインターフェース
type BeforeCreateHook interface {
	BeforeCreate(tx *gorm.DB) error
}

// BeforeUpdate 更新前処理のインターフェース
type BeforeUpdateHook interface {
	BeforeUpdate(tx *gorm.DB) error
}

// AfterFind 検索後処理のインターフェース
type AfterFindHook interface {
	AfterFind(tx *gorm.DB) error
}
