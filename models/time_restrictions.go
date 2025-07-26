package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TimeRestriction 時間制限テーブル
type TimeRestriction struct {
	ID           int        `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	ResourceType string     `gorm:"not null;index" json:"resource_type"`
	StartTime    *time.Time `gorm:"type:time" json:"start_time,omitempty"`
	EndTime      *time.Time `gorm:"type:time" json:"end_time,omitempty"`
	AllowedDays  IntArray   `gorm:"type:integer[]" json:"allowed_days,omitempty"`
	Timezone     string     `gorm:"default:'UTC'" json:"timezone"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`

	// リレーション
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

// TableName テーブル名を指定
func (TimeRestriction) TableName() string {
	return "time_restrictions"
}

// BeforeCreate 作成前のバリデーション
func (tr *TimeRestriction) BeforeCreate(tx *gorm.DB) error {
	// リソースタイプの妥当性チェック
	if !tr.IsValidResourceType() {
		return gorm.ErrInvalidValue
	}

	// 許可日の妥当性チェック
	if !tr.IsValidAllowedDays() {
		return gorm.ErrInvalidValue
	}

	// タイムゾーンの妥当性チェック
	if !tr.IsValidTimezone() {
		return gorm.ErrInvalidValue
	}

	return nil
}

// BeforeUpdate 更新前のバリデーション
func (tr *TimeRestriction) BeforeUpdate(tx *gorm.DB) error {
	// リソースタイプの妥当性チェック
	if !tr.IsValidResourceType() {
		return gorm.ErrInvalidValue
	}

	// 許可日の妥当性チェック
	if !tr.IsValidAllowedDays() {
		return gorm.ErrInvalidValue
	}

	// タイムゾーンの妥当性チェック
	if !tr.IsValidTimezone() {
		return gorm.ErrInvalidValue
	}

	return nil
}

// =============================================================================
// 時間制限管理のメソッド
// =============================================================================

// IsValidResourceType リソースタイプが有効かチェック
func (tr *TimeRestriction) IsValidResourceType() bool {
	validResourceTypes := []string{
		"inventory", "orders", "reports", "users",
		"departments", "roles", "permissions", "audit",
		"dashboard", "settings", "finance", "hr",
		"projects", "locations", "assets", "contracts",
		"system", "auth",
	}

	for _, resourceType := range validResourceTypes {
		if tr.ResourceType == resourceType {
			return true
		}
	}
	return false
}

// IsValidAllowedDays 許可日が有効かチェック
func (tr *TimeRestriction) IsValidAllowedDays() bool {
	if len(tr.AllowedDays) == 0 {
		return true // 設定がない場合は有効
	}

	// 1-7の範囲内かチェック（1=日曜日, 7=土曜日）
	for _, day := range tr.AllowedDays {
		if day < 1 || day > 7 {
			return false
		}
	}

	return true
}

// IsValidTimezone タイムゾーンが有効かチェック
func (tr *TimeRestriction) IsValidTimezone() bool {
	_, err := time.LoadLocation(tr.Timezone)
	return err == nil
}

// HasTimeRestriction 時間制限が設定されているかチェック
func (tr *TimeRestriction) HasTimeRestriction() bool {
	return tr.StartTime != nil || tr.EndTime != nil
}

// HasDayRestriction 曜日制限が設定されているかチェック
func (tr *TimeRestriction) HasDayRestriction() bool {
	return len(tr.AllowedDays) > 0
}

// IsAllowedTime 指定した時刻がアクセス許可時間内かチェック
func (tr *TimeRestriction) IsAllowedTime(checkTime time.Time) bool {
	if !tr.HasTimeRestriction() {
		return true // 時間制限がない場合は常に許可
	}

	// タイムゾーンを適用
	location, err := time.LoadLocation(tr.Timezone)
	if err != nil {
		return false
	}

	localTime := checkTime.In(location)
	currentTimeOfDay := time.Date(0, 1, 1, localTime.Hour(), localTime.Minute(), localTime.Second(), 0, time.UTC)

	// 開始時間のチェック
	if tr.StartTime != nil {
		startTimeOfDay := time.Date(0, 1, 1, tr.StartTime.Hour(), tr.StartTime.Minute(), tr.StartTime.Second(), 0, time.UTC)
		if currentTimeOfDay.Before(startTimeOfDay) {
			return false
		}
	}

	// 終了時間のチェック
	if tr.EndTime != nil {
		endTimeOfDay := time.Date(0, 1, 1, tr.EndTime.Hour(), tr.EndTime.Minute(), tr.EndTime.Second(), 0, time.UTC)
		if currentTimeOfDay.After(endTimeOfDay) {
			return false
		}
	}

	return true
}

// IsAllowedDay 指定した曜日がアクセス許可日かチェック
func (tr *TimeRestriction) IsAllowedDay(checkTime time.Time) bool {
	if !tr.HasDayRestriction() {
		return true // 曜日制限がない場合は常に許可
	}

	// タイムゾーンを適用
	location, err := time.LoadLocation(tr.Timezone)
	if err != nil {
		return false
	}

	localTime := checkTime.In(location)
	weekday := int(localTime.Weekday()) + 1 // Goの0=日曜日を1=日曜日に変換

	// 許可曜日リストに含まれているかチェック
	for _, allowedDay := range tr.AllowedDays {
		if int64(weekday) == allowedDay {
			return true
		}
	}

	return false
}

// IsAllowed 指定した日時がアクセス許可範囲内かチェック
func (tr *TimeRestriction) IsAllowed(checkTime time.Time) bool {
	return tr.IsAllowedTime(checkTime) && tr.IsAllowedDay(checkTime)
}

// GetAllowedDaysAsStrings 許可曜日を文字列スライスで取得
func (tr *TimeRestriction) GetAllowedDaysAsStrings() []string {
	if !tr.HasDayRestriction() {
		return nil
	}

	dayNames := []string{"", "日", "月", "火", "水", "木", "金", "土"}
	var result []string

	for _, day := range tr.AllowedDays {
		if day >= 1 && day <= 7 {
			result = append(result, dayNames[day])
		}
	}

	return result
}

// SetAllowedDaysFromInts 整数スライスから許可曜日を設定
func (tr *TimeRestriction) SetAllowedDaysFromInts(days []int) {
	if len(days) == 0 {
		tr.AllowedDays = nil
		return
	}

	allowedDays := make(IntArray, len(days))
	for i, day := range days {
		allowedDays[i] = int64(day)
	}
	tr.AllowedDays = allowedDays
}

// =============================================================================
// クエリ用ヘルパー関数
// =============================================================================

// FindTimeRestrictionByID IDで時間制限を検索
func FindTimeRestrictionByID(db *gorm.DB, id int) (*TimeRestriction, error) {
	var timeRestriction TimeRestriction
	err := db.Preload("User").Where("id = ?", id).First(&timeRestriction).Error
	if err != nil {
		return nil, err
	}
	return &timeRestriction, nil
}

// FindTimeRestrictionsByUser ユーザーIDで時間制限を検索
func FindTimeRestrictionsByUser(db *gorm.DB, userID uuid.UUID) ([]TimeRestriction, error) {
	var timeRestrictions []TimeRestriction
	err := db.Where("user_id = ?", userID).Find(&timeRestrictions).Error
	return timeRestrictions, err
}

// FindTimeRestrictionsByUserAndResource ユーザーとリソースで時間制限を検索
func FindTimeRestrictionsByUserAndResource(db *gorm.DB, userID uuid.UUID, resourceType string) ([]TimeRestriction, error) {
	var timeRestrictions []TimeRestriction
	err := db.Where("user_id = ? AND resource_type = ?", userID, resourceType).Find(&timeRestrictions).Error
	return timeRestrictions, err
}

// FindTimeRestrictionsByResourceType リソースタイプで時間制限を検索
func FindTimeRestrictionsByResourceType(db *gorm.DB, resourceType string) ([]TimeRestriction, error) {
	var timeRestrictions []TimeRestriction
	err := db.Preload("User").Where("resource_type = ?", resourceType).Find(&timeRestrictions).Error
	return timeRestrictions, err
}

// =============================================================================
// 時間制限管理用ヘルパー関数
// =============================================================================

// CreateTimeRestriction 時間制限を作成
func CreateTimeRestriction(db *gorm.DB, userID uuid.UUID, resourceType string, startTime, endTime *time.Time, allowedDays []int, timezone string) (*TimeRestriction, error) {
	timeRestriction := &TimeRestriction{
		UserID:       userID,
		ResourceType: resourceType,
		StartTime:    startTime,
		EndTime:      endTime,
		Timezone:     timezone,
	}

	timeRestriction.SetAllowedDaysFromInts(allowedDays)

	err := db.Create(timeRestriction).Error
	if err != nil {
		return nil, err
	}

	return timeRestriction, nil
}

// UpdateTimeRestriction 時間制限を更新
func UpdateTimeRestriction(db *gorm.DB, id int, updates map[string]interface{}) error {
	return db.Model(&TimeRestriction{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteTimeRestriction 時間制限を削除
func DeleteTimeRestriction(db *gorm.DB, id int) error {
	return db.Delete(&TimeRestriction{}, id).Error
}

// DeleteTimeRestrictionsByUser ユーザーのすべての時間制限を削除
func DeleteTimeRestrictionsByUser(db *gorm.DB, userID uuid.UUID) error {
	return db.Where("user_id = ?", userID).Delete(&TimeRestriction{}).Error
}

// CheckTimeAccess ユーザーの時間制限チェック
func CheckTimeAccess(db *gorm.DB, userID uuid.UUID, resourceType string, checkTime time.Time) (bool, error) {
	timeRestrictions, err := FindTimeRestrictionsByUserAndResource(db, userID, resourceType)
	if err != nil {
		return false, err
	}

	// 時間制限が設定されていない場合は許可
	if len(timeRestrictions) == 0 {
		return true, nil
	}

	// いずれかの時間制限に合致すれば許可
	for _, restriction := range timeRestrictions {
		if restriction.IsAllowed(checkTime) {
			return true, nil
		}
	}

	return false, nil
}

// GetCurrentTimeAccess 現在時刻でのアクセス許可をチェック
func GetCurrentTimeAccess(db *gorm.DB, userID uuid.UUID, resourceType string) (bool, error) {
	return CheckTimeAccess(db, userID, resourceType, time.Now())
}

// GetBusinessHoursRestriction 営業時間制限を作成
func GetBusinessHoursRestriction(startHour, endHour int, weekdays []int, timezone string) (TimeRestriction, error) {
	startTime := time.Date(0, 1, 1, startHour, 0, 0, 0, time.UTC)
	endTime := time.Date(0, 1, 1, endHour, 0, 0, 0, time.UTC)

	restriction := TimeRestriction{
		StartTime: &startTime,
		EndTime:   &endTime,
		Timezone:  timezone,
	}

	restriction.SetAllowedDaysFromInts(weekdays)

	return restriction, nil
}

// GetTimeRestrictionStats 時間制限統計を取得
func GetTimeRestrictionStats(db *gorm.DB) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// リソースタイプ別統計
	var resourceStats []struct {
		ResourceType string `json:"resource_type"`
		Count        int64  `json:"count"`
	}

	err := db.Model(&TimeRestriction{}).
		Select("resource_type, COUNT(*) as count").
		Group("resource_type").
		Scan(&resourceStats).Error

	if err != nil {
		return nil, err
	}

	stats["by_resource_type"] = resourceStats

	// 制限タイプ別統計
	var restrictionTypeStats struct {
		TimeOnly   int64 `json:"time_only"`
		DayOnly    int64 `json:"day_only"`
		TimeAndDay int64 `json:"time_and_day"`
		NoLimits   int64 `json:"no_limits"`
	}

	db.Model(&TimeRestriction{}).
		Where("start_time IS NOT NULL OR end_time IS NOT NULL").
		Where("allowed_days IS NULL OR array_length(allowed_days, 1) IS NULL").
		Count(&restrictionTypeStats.TimeOnly)

	db.Model(&TimeRestriction{}).
		Where("start_time IS NULL AND end_time IS NULL").
		Where("allowed_days IS NOT NULL AND array_length(allowed_days, 1) > 0").
		Count(&restrictionTypeStats.DayOnly)

	db.Model(&TimeRestriction{}).
		Where("(start_time IS NOT NULL OR end_time IS NOT NULL)").
		Where("allowed_days IS NOT NULL AND array_length(allowed_days, 1) > 0").
		Count(&restrictionTypeStats.TimeAndDay)

	db.Model(&TimeRestriction{}).
		Where("start_time IS NULL AND end_time IS NULL").
		Where("allowed_days IS NULL OR array_length(allowed_days, 1) IS NULL").
		Count(&restrictionTypeStats.NoLimits)

	stats["by_restriction_type"] = restrictionTypeStats

	// 総数
	var totalCount int64
	db.Model(&TimeRestriction{}).Count(&totalCount)
	stats["total"] = totalCount

	return stats, nil
}
