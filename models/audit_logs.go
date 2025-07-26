package models

import (
	"net"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditLog 監査ログテーブル
type AuditLog struct {
	ID           int         `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uuid.UUID   `gorm:"type:uuid;not null;index" json:"user_id"`
	Action       string      `gorm:"not null" json:"action"`
	ResourceType string      `gorm:"not null;index" json:"resource_type"`
	ResourceID   string      `gorm:"not null;index" json:"resource_id"`
	Result       AuditResult `gorm:"not null;check:result IN ('SUCCESS','DENIED','ERROR')" json:"result"`
	Reason       *string     `json:"reason,omitempty"`
	ReasonCode   *string     `gorm:"index" json:"reason_code,omitempty"`
	IPAddress    *net.IP     `gorm:"type:inet" json:"ip_address,omitempty"`
	UserAgent    *string     `json:"user_agent,omitempty"`
	Timestamp    time.Time   `gorm:"not null;default:now();index" json:"timestamp"`

	// リレーション
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

// TableName テーブル名を指定
func (AuditLog) TableName() string {
	return "audit_logs"
}

// BeforeCreate 作成前のバリデーション
func (al *AuditLog) BeforeCreate(tx *gorm.DB) error {
	// 結果の妥当性チェック
	if !ValidateAuditResult(al.Result) {
		return gorm.ErrInvalidValue
	}

	// アクションの妥当性チェック
	if !al.IsValidAction() {
		return gorm.ErrInvalidValue
	}

	// リソースタイプの妥当性チェック
	if !al.IsValidResourceType() {
		return gorm.ErrInvalidValue
	}

	return nil
}

// BeforeUpdate 更新前のバリデーション
func (al *AuditLog) BeforeUpdate(tx *gorm.DB) error {
	// 結果の妥当性チェック
	if !ValidateAuditResult(al.Result) {
		return gorm.ErrInvalidValue
	}

	// アクションの妥当性チェック
	if !al.IsValidAction() {
		return gorm.ErrInvalidValue
	}

	// リソースタイプの妥当性チェック
	if !al.IsValidResourceType() {
		return gorm.ErrInvalidValue
	}

	return nil
}

// =============================================================================
// 監査ログ管理のメソッド
// =============================================================================

// IsValidAction アクションが有効かチェック
func (al *AuditLog) IsValidAction() bool {
	validActions := []string{
		"view", "create", "update", "delete",
		"approve", "reject", "export", "import",
		"login", "logout", "access_denied",
		"permission_check", "role_change", "status_change",
		"password_reset", "email_change", "profile_update",
	}

	for _, action := range validActions {
		if al.Action == action {
			return true
		}
	}
	return false
}

// IsValidResourceType リソースタイプが有効かチェック
func (al *AuditLog) IsValidResourceType() bool {
	validResourceTypes := []string{
		"inventory", "orders", "reports", "users",
		"departments", "roles", "permissions", "audit",
		"dashboard", "settings", "finance", "hr",
		"projects", "locations", "assets", "contracts",
		"auth", "session", "system",
	}

	for _, resourceType := range validResourceTypes {
		if al.ResourceType == resourceType {
			return true
		}
	}
	return false
}

// IsSuccess 成功したアクションかチェック
func (al *AuditLog) IsSuccess() bool {
	return al.Result == AuditResultSuccess
}

// IsDenied 拒否されたアクションかチェック
func (al *AuditLog) IsDenied() bool {
	return al.Result == AuditResultDenied
}

// IsError エラーが発生したアクションかチェック
func (al *AuditLog) IsError() bool {
	return al.Result == AuditResultError
}

// HasIPAddress IPアドレスが記録されているかチェック
func (al *AuditLog) HasIPAddress() bool {
	return al.IPAddress != nil
}

// GetIPAddressString IPアドレスを文字列として取得
func (al *AuditLog) GetIPAddressString() string {
	if al.IPAddress == nil {
		return ""
	}
	return al.IPAddress.String()
}

// SetIPAddressFromString 文字列からIPアドレスを設定
func (al *AuditLog) SetIPAddressFromString(ipStr string) error {
	if ipStr == "" {
		al.IPAddress = nil
		return nil
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return gorm.ErrInvalidValue
	}

	al.IPAddress = &ip
	return nil
}

// GetReasonCodeCategory 理由コードのカテゴリを取得
func (al *AuditLog) GetReasonCodeCategory() string {
	if al.ReasonCode == nil {
		return "UNKNOWN"
	}

	reasonCode := *al.ReasonCode

	// 理由コードのプレフィックスでカテゴリ分け
	switch {
	case len(reasonCode) >= 4 && reasonCode[:4] == "AUTH":
		return "AUTHENTICATION"
	case len(reasonCode) >= 4 && reasonCode[:4] == "PERM":
		return "PERMISSION"
	case len(reasonCode) >= 4 && reasonCode[:4] == "VALR":
		return "VALIDATION"
	case len(reasonCode) >= 4 && reasonCode[:4] == "SYSR":
		return "SYSTEM"
	default:
		return "OTHER"
	}
}

// =============================================================================
// クエリ用ヘルパー関数
// =============================================================================

// FindAuditLogByID IDで監査ログを検索
func FindAuditLogByID(db *gorm.DB, id int) (*AuditLog, error) {
	var auditLog AuditLog
	err := db.Preload("User").Where("id = ?", id).First(&auditLog).Error
	if err != nil {
		return nil, err
	}
	return &auditLog, nil
}

// FindAuditLogsByUser ユーザーIDで監査ログを検索
func FindAuditLogsByUser(db *gorm.DB, userID uuid.UUID, limit int) ([]AuditLog, error) {
	var auditLogs []AuditLog
	query := db.Preload("User").Where("user_id = ?", userID).Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&auditLogs).Error
	return auditLogs, err
}

// FindAuditLogsByResource リソースで監査ログを検索
func FindAuditLogsByResource(db *gorm.DB, resourceType, resourceID string, limit int) ([]AuditLog, error) {
	var auditLogs []AuditLog
	query := db.Preload("User").Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&auditLogs).Error
	return auditLogs, err
}

// FindAuditLogsByAction アクションで監査ログを検索
func FindAuditLogsByAction(db *gorm.DB, action string, limit int) ([]AuditLog, error) {
	var auditLogs []AuditLog
	query := db.Preload("User").Where("action = ?", action).Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&auditLogs).Error
	return auditLogs, err
}

// FindAuditLogsByResult 結果で監査ログを検索
func FindAuditLogsByResult(db *gorm.DB, result AuditResult, limit int) ([]AuditLog, error) {
	var auditLogs []AuditLog
	query := db.Preload("User").Where("result = ?", result).Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&auditLogs).Error
	return auditLogs, err
}

// FindAuditLogsByTimeRange 期間で監査ログを検索
func FindAuditLogsByTimeRange(db *gorm.DB, startTime, endTime time.Time, limit int) ([]AuditLog, error) {
	var auditLogs []AuditLog
	query := db.Preload("User").Where("timestamp BETWEEN ? AND ?", startTime, endTime).
		Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&auditLogs).Error
	return auditLogs, err
}

// FindAuditLogsByIPAddress IPアドレスで監査ログを検索
func FindAuditLogsByIPAddress(db *gorm.DB, ipAddress string, limit int) ([]AuditLog, error) {
	var auditLogs []AuditLog
	query := db.Preload("User").Where("ip_address = ?", ipAddress).Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&auditLogs).Error
	return auditLogs, err
}

// =============================================================================
// 監査ログ作成用ヘルパー関数
// =============================================================================

// CreateAuditLog 監査ログを作成
func CreateAuditLog(db *gorm.DB, userID uuid.UUID, action, resourceType, resourceID string, result AuditResult, reason, reasonCode *string, ipAddress *string, userAgent *string) (*AuditLog, error) {
	auditLog := &AuditLog{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Result:       result,
		Reason:       reason,
		ReasonCode:   reasonCode,
		UserAgent:    userAgent,
		Timestamp:    time.Now(),
	}

	// IPアドレスを設定
	if ipAddress != nil {
		err := auditLog.SetIPAddressFromString(*ipAddress)
		if err != nil {
			return nil, err
		}
	}

	err := db.Create(auditLog).Error
	if err != nil {
		return nil, err
	}

	return auditLog, nil
}

// CreateSuccessAuditLog 成功の監査ログを作成
func CreateSuccessAuditLog(db *gorm.DB, userID uuid.UUID, action, resourceType, resourceID string, ipAddress *string, userAgent *string) (*AuditLog, error) {
	return CreateAuditLog(db, userID, action, resourceType, resourceID, AuditResultSuccess, nil, nil, ipAddress, userAgent)
}

// CreateDeniedAuditLog 拒否の監査ログを作成
func CreateDeniedAuditLog(db *gorm.DB, userID uuid.UUID, action, resourceType, resourceID string, reason, reasonCode *string, ipAddress *string, userAgent *string) (*AuditLog, error) {
	return CreateAuditLog(db, userID, action, resourceType, resourceID, AuditResultDenied, reason, reasonCode, ipAddress, userAgent)
}

// CreateErrorAuditLog エラーの監査ログを作成
func CreateErrorAuditLog(db *gorm.DB, userID uuid.UUID, action, resourceType, resourceID string, reason, reasonCode *string, ipAddress *string, userAgent *string) (*AuditLog, error) {
	return CreateAuditLog(db, userID, action, resourceType, resourceID, AuditResultError, reason, reasonCode, ipAddress, userAgent)
}

// =============================================================================
// 監査統計用ヘルパー関数
// =============================================================================

// GetAuditStats 監査統計を取得
func GetAuditStats(db *gorm.DB, startTime, endTime *time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	query := db.Model(&AuditLog{})

	// 期間指定がある場合
	if startTime != nil && endTime != nil {
		query = query.Where("timestamp BETWEEN ? AND ?", *startTime, *endTime)
	}

	// 結果別統計
	var resultStats []struct {
		Result AuditResult `json:"result"`
		Count  int64       `json:"count"`
	}

	err := query.Select("result, COUNT(*) as count").Group("result").Scan(&resultStats).Error
	if err != nil {
		return nil, err
	}
	stats["by_result"] = resultStats

	// アクション別統計
	var actionStats []struct {
		Action string `json:"action"`
		Count  int64  `json:"count"`
	}

	err = query.Select("action, COUNT(*) as count").Group("action").
		Order("count DESC").Limit(10).Scan(&actionStats).Error
	if err != nil {
		return nil, err
	}
	stats["top_actions"] = actionStats

	// リソースタイプ別統計
	var resourceStats []struct {
		ResourceType string `json:"resource_type"`
		Count        int64  `json:"count"`
	}

	err = query.Select("resource_type, COUNT(*) as count").Group("resource_type").
		Order("count DESC").Scan(&resourceStats).Error
	if err != nil {
		return nil, err
	}
	stats["by_resource_type"] = resourceStats

	// 総数
	var totalCount int64
	query.Count(&totalCount)
	stats["total"] = totalCount

	return stats, nil
}

// GetUserActivityStats ユーザーアクティビティ統計を取得
func GetUserActivityStats(db *gorm.DB, userID uuid.UUID, startTime, endTime *time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	query := db.Model(&AuditLog{}).Where("user_id = ?", userID)

	if startTime != nil && endTime != nil {
		query = query.Where("timestamp BETWEEN ? AND ?", *startTime, *endTime)
	}

	// アクション別統計
	var actionStats []struct {
		Action string `json:"action"`
		Count  int64  `json:"count"`
	}

	err := query.Select("action, COUNT(*) as count").Group("action").
		Order("count DESC").Scan(&actionStats).Error
	if err != nil {
		return nil, err
	}
	stats["actions"] = actionStats

	// 結果別統計
	var resultStats []struct {
		Result AuditResult `json:"result"`
		Count  int64       `json:"count"`
	}

	err = query.Select("result, COUNT(*) as count").Group("result").Scan(&resultStats).Error
	if err != nil {
		return nil, err
	}
	stats["results"] = resultStats

	// 総数
	var totalCount int64
	query.Count(&totalCount)
	stats["total"] = totalCount

	return stats, nil
}
