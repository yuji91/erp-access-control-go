package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ApprovalState 承認状態テーブル
type ApprovalState struct {
	ID             int       `gorm:"primaryKey;autoIncrement" json:"id"`
	StateName      string    `gorm:"not null" json:"state_name"`
	ApproverRoleID uuid.UUID `gorm:"type:uuid;not null;index" json:"approver_role_id"`
	StepOrder      int       `gorm:"not null;default:1;check:step_order > 0" json:"step_order"`
	ResourceType   *string   `gorm:"index" json:"resource_type,omitempty"`
	Scope          JSONB     `gorm:"type:jsonb" json:"scope,omitempty"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`

	// リレーション
	ApproverRole Role `gorm:"foreignKey:ApproverRoleID;constraint:OnDelete:CASCADE" json:"approver_role,omitempty"`
}

// TableName テーブル名を指定
func (ApprovalState) TableName() string {
	return "approval_states"
}

// BeforeCreate 作成前のバリデーション
func (as *ApprovalState) BeforeCreate(tx *gorm.DB) error {
	// ステップ順序の妥当性チェック
	if as.StepOrder <= 0 {
		return gorm.ErrInvalidValue
	}

	// リソースタイプの妥当性チェック
	if as.ResourceType != nil && !as.IsValidResourceType() {
		return gorm.ErrInvalidValue
	}

	return nil
}

// BeforeUpdate 更新前のバリデーション
func (as *ApprovalState) BeforeUpdate(tx *gorm.DB) error {
	// ステップ順序の妥当性チェック
	if as.StepOrder <= 0 {
		return gorm.ErrInvalidValue
	}

	// リソースタイプの妥当性チェック
	if as.ResourceType != nil && !as.IsValidResourceType() {
		return gorm.ErrInvalidValue
	}

	return nil
}

// =============================================================================
// 承認状態管理のメソッド
// =============================================================================

// IsValidResourceType リソースタイプが有効かチェック
func (as *ApprovalState) IsValidResourceType() bool {
	if as.ResourceType == nil {
		return true // リソースタイプがnullの場合は有効
	}

	validResourceTypes := []string{
		"inventory", "orders", "reports", "users",
		"departments", "roles", "permissions", "audit",
		"dashboard", "settings", "finance", "hr",
		"projects", "locations", "assets", "contracts",
		"purchases", "expenses", "budgets", "invoices",
	}

	for _, resourceType := range validResourceTypes {
		if *as.ResourceType == resourceType {
			return true
		}
	}
	return false
}

// HasScope スコープ条件が設定されているかチェック
func (as *ApprovalState) HasScope() bool {
	return len(as.Scope) > 0
}

// MatchesScope 指定したスコープ条件にマッチするかチェック
func (as *ApprovalState) MatchesScope(conditions JSONB) bool {
	if !as.HasScope() {
		return true // スコープ条件がない場合は常にマッチ
	}

	if conditions == nil {
		return false
	}

	// 承認状態のスコープ条件がすべて満たされているかチェック
	for key, expectedValue := range as.Scope {
		actualValue, exists := conditions[key]
		if !exists || actualValue != expectedValue {
			return false
		}
	}

	return true
}

// GetScopeValue スコープから特定キーの値を取得
func (as *ApprovalState) GetScopeValue(key string) (interface{}, bool) {
	if !as.HasScope() {
		return nil, false
	}

	value, exists := as.Scope[key]
	return value, exists
}

// SetScopeValue スコープに値を設定
func (as *ApprovalState) SetScopeValue(key string, value interface{}) {
	if as.Scope == nil {
		as.Scope = make(JSONB)
	}
	as.Scope[key] = value
}

// IsFirstStep 最初の承認ステップかチェック
func (as *ApprovalState) IsFirstStep() bool {
	return as.StepOrder == 1
}

// GetNextStepOrder 次のステップ順序を取得
func (as *ApprovalState) GetNextStepOrder() int {
	return as.StepOrder + 1
}

// =============================================================================
// クエリ用ヘルパー関数
// =============================================================================

// FindApprovalStateByID IDで承認状態を検索
func FindApprovalStateByID(db *gorm.DB, id int) (*ApprovalState, error) {
	var approvalState ApprovalState
	err := db.Preload("ApproverRole").Where("id = ?", id).First(&approvalState).Error
	if err != nil {
		return nil, err
	}
	return &approvalState, nil
}

// FindApprovalStatesByRole ロールIDで承認状態を検索
func FindApprovalStatesByRole(db *gorm.DB, roleID uuid.UUID) ([]ApprovalState, error) {
	var approvalStates []ApprovalState
	err := db.Preload("ApproverRole").Where("approver_role_id = ?", roleID).Find(&approvalStates).Error
	return approvalStates, err
}

// FindApprovalStatesByResourceType リソースタイプで承認状態を検索
func FindApprovalStatesByResourceType(db *gorm.DB, resourceType string) ([]ApprovalState, error) {
	var approvalStates []ApprovalState
	err := db.Preload("ApproverRole").Where("resource_type = ?", resourceType).
		Order("step_order ASC").Find(&approvalStates).Error
	return approvalStates, err
}

// FindApprovalStatesByStep ステップ順序で承認状態を検索
func FindApprovalStatesByStep(db *gorm.DB, stepOrder int) ([]ApprovalState, error) {
	var approvalStates []ApprovalState
	err := db.Preload("ApproverRole").Where("step_order = ?", stepOrder).Find(&approvalStates).Error
	return approvalStates, err
}

// FindApprovalStatesByScope スコープ条件で承認状態を検索
func FindApprovalStatesByScope(db *gorm.DB, scopeConditions JSONB) ([]ApprovalState, error) {
	var approvalStates []ApprovalState
	err := db.Preload("ApproverRole").Where("scope @> ?", scopeConditions).
		Order("step_order ASC").Find(&approvalStates).Error
	return approvalStates, err
}

// =============================================================================
// 承認フロー管理用ヘルパー関数
// =============================================================================

// CreateApprovalState 承認状態を作成
func CreateApprovalState(db *gorm.DB, stateName string, approverRoleID uuid.UUID, stepOrder int, resourceType *string, scope JSONB) (*ApprovalState, error) {
	approvalState := &ApprovalState{
		StateName:      stateName,
		ApproverRoleID: approverRoleID,
		StepOrder:      stepOrder,
		ResourceType:   resourceType,
		Scope:          scope,
	}

	err := db.Create(approvalState).Error
	if err != nil {
		return nil, err
	}

	return approvalState, nil
}

// UpdateApprovalState 承認状態を更新
func UpdateApprovalState(db *gorm.DB, id int, updates map[string]interface{}) error {
	return db.Model(&ApprovalState{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteApprovalState 承認状態を削除
func DeleteApprovalState(db *gorm.DB, id int) error {
	return db.Delete(&ApprovalState{}, id).Error
}

// GetApprovalFlow 特定リソースの承認フローを取得
func GetApprovalFlow(db *gorm.DB, resourceType string, scopeConditions JSONB) ([]ApprovalState, error) {
	var approvalStates []ApprovalState
	query := db.Preload("ApproverRole")

	if resourceType != "" {
		query = query.Where("resource_type = ? OR resource_type IS NULL", resourceType)
	}

	if len(scopeConditions) > 0 {
		query = query.Where("scope @> ? OR scope IS NULL", scopeConditions)
	}

	err := query.Order("step_order ASC").Find(&approvalStates).Error
	return approvalStates, err
}

// GetNextApprovalStep 次の承認ステップを取得
func GetNextApprovalStep(db *gorm.DB, resourceType string, currentStep int, scopeConditions JSONB) (*ApprovalState, error) {
	var approvalState ApprovalState
	query := db.Preload("ApproverRole").Where("step_order > ?", currentStep)

	if resourceType != "" {
		query = query.Where("resource_type = ? OR resource_type IS NULL", resourceType)
	}

	if len(scopeConditions) > 0 {
		query = query.Where("scope @> ? OR scope IS NULL", scopeConditions)
	}

	err := query.Order("step_order ASC").First(&approvalState).Error
	if err != nil {
		return nil, err
	}

	return &approvalState, nil
}

// GetApprovalStepsByRole ロールが関わる承認ステップを取得
func GetApprovalStepsByRole(db *gorm.DB, roleID uuid.UUID) ([]ApprovalState, error) {
	var approvalStates []ApprovalState
	err := db.Preload("ApproverRole").Where("approver_role_id = ?", roleID).
		Order("resource_type, step_order ASC").Find(&approvalStates).Error
	return approvalStates, err
}

// CheckApprovalRequired 承認が必要かチェック
func CheckApprovalRequired(db *gorm.DB, resourceType string, scopeConditions JSONB) (bool, error) {
	var count int64
	query := db.Model(&ApprovalState{})

	if resourceType != "" {
		query = query.Where("resource_type = ? OR resource_type IS NULL", resourceType)
	}

	if len(scopeConditions) > 0 {
		query = query.Where("scope @> ? OR scope IS NULL", scopeConditions)
	}

	err := query.Count(&count).Error
	return count > 0, err
}

// GetApprovalStats 承認統計を取得
func GetApprovalStats(db *gorm.DB) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// リソースタイプ別統計
	var resourceStats []struct {
		ResourceType *string `json:"resource_type"`
		Count        int64   `json:"count"`
	}

	err := db.Model(&ApprovalState{}).
		Select("resource_type, COUNT(*) as count").
		Group("resource_type").
		Scan(&resourceStats).Error

	if err != nil {
		return nil, err
	}

	stats["by_resource_type"] = resourceStats

	// ステップ順序別統計
	var stepStats []struct {
		StepOrder int   `json:"step_order"`
		Count     int64 `json:"count"`
	}

	err = db.Model(&ApprovalState{}).
		Select("step_order, COUNT(*) as count").
		Group("step_order").
		Order("step_order ASC").
		Scan(&stepStats).Error

	if err != nil {
		return nil, err
	}

	stats["by_step_order"] = stepStats

	// 総数
	var totalCount int64
	db.Model(&ApprovalState{}).Count(&totalCount)
	stats["total"] = totalCount

	return stats, nil
}
