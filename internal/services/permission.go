package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"erp-access-control-go/models"
)

// PermissionService 権限評価・管理サービス
type PermissionService struct {
	db *gorm.DB
}

// NewPermissionService 新しい権限サービスを作成
func NewPermissionService(db *gorm.DB) *PermissionService {
	return &PermissionService{db: db}
}

// Module システムモジュールを表す
type Module string

const (
	ModuleUser       Module = "user"
	ModuleDepartment Module = "department"
	ModuleRole       Module = "role"
	ModulePermission Module = "permission"
	ModuleAudit      Module = "audit"
	ModuleSystem     Module = "system"
)

// Action 実行可能なアクションを表す
type Action string

const (
	ActionCreate Action = "create"
	ActionRead   Action = "read"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionList   Action = "list"
	ActionManage Action = "manage"
)

// Permission 権限文字列を表す
type Permission string

// NewPermission 標準的な権限フォーマット: module:action (例: "user:create", "role:read")
func NewPermission(module Module, action Action) Permission {
	return Permission(fmt.Sprintf("%s:%s", module, action))
}

// PermissionMatrix システムの権限マトリックスを定義
// TODO: アーキテクチャ改善
// - データベースベース権限管理への移行検討
// - 動的権限追加・削除機能
// - 時間ベース権限 (営業時間のみ有効など)
// - 地理的制限 (特定IPレンジ、地域からのみアクセス)
// - 継承ベース階層権限 (部門長→課長→係長の自動継承)
var PermissionMatrix = map[string][]Permission{
	// Super Admin - Full access
	"super_admin": {
		"*:*", // Wildcard permission for everything
	},

	// Admin - Most permissions except system management
	"admin": {
		NewPermission(ModuleUser, ActionCreate),
		NewPermission(ModuleUser, ActionRead),
		NewPermission(ModuleUser, ActionUpdate),
		NewPermission(ModuleUser, ActionDelete),
		NewPermission(ModuleUser, ActionList),
		NewPermission(ModuleDepartment, ActionCreate),
		NewPermission(ModuleDepartment, ActionRead),
		NewPermission(ModuleDepartment, ActionUpdate),
		NewPermission(ModuleDepartment, ActionDelete),
		NewPermission(ModuleDepartment, ActionList),
		NewPermission(ModuleRole, ActionCreate),
		NewPermission(ModuleRole, ActionRead),
		NewPermission(ModuleRole, ActionUpdate),
		NewPermission(ModuleRole, ActionDelete),
		NewPermission(ModuleRole, ActionList),
		NewPermission(ModulePermission, ActionRead),
		NewPermission(ModulePermission, ActionList),
		NewPermission(ModuleAudit, ActionRead),
		NewPermission(ModuleAudit, ActionList),
	},

	// Manager - Department and user management within scope
	"manager": {
		NewPermission(ModuleUser, ActionRead),
		NewPermission(ModuleUser, ActionUpdate),
		NewPermission(ModuleUser, ActionList),
		NewPermission(ModuleDepartment, ActionRead),
		NewPermission(ModuleDepartment, ActionUpdate),
		NewPermission(ModuleDepartment, ActionList),
		NewPermission(ModuleRole, ActionRead),
		NewPermission(ModuleRole, ActionList),
		NewPermission(ModuleAudit, ActionRead),
	},

	// Employee - Basic read access
	"employee": {
		NewPermission(ModuleUser, ActionRead),
		NewPermission(ModuleDepartment, ActionRead),
		NewPermission(ModuleRole, ActionRead),
	},

	// Viewer - Read-only access
	"viewer": {
		NewPermission(ModuleUser, ActionRead),
		NewPermission(ModuleUser, ActionList),
		NewPermission(ModuleDepartment, ActionRead),
		NewPermission(ModuleDepartment, ActionList),
		NewPermission(ModuleRole, ActionRead),
		NewPermission(ModuleRole, ActionList),
	},
}

// GetUserPermissions ユーザーの全権限を取得
func (s *PermissionService) GetUserPermissions(userID uuid.UUID) ([]string, error) {
	// TODO: パフォーマンス最適化
	// - Redis/Memcachedによる権限キャッシュ (TTL: 5-15分)
	// - 階層的権限の事前計算とキャッシュ
	// - バッチ権限取得機能 (複数ユーザー一括処理)
	// - 権限変更時のキャッシュ無効化戦略

	var user models.User
	if err := s.db.Preload("Role.Permissions").First(&user, userID).Error; err != nil {
		return nil, err
	}

	permissionSet := make(map[string]bool)

	// Get base permissions from matrix for user's role
	if basePerms, exists := PermissionMatrix[user.Role.Name]; exists {
		for _, perm := range basePerms {
			permissionSet[string(perm)] = true
		}
	}

	// Add explicit permissions from database for the role
	for _, perm := range user.Role.Permissions {
		permissionSet[perm.GetUniqueKey()] = true
	}

	// Convert to slice
	permissions := make([]string, 0, len(permissionSet))
	for perm := range permissionSet {
		permissions = append(permissions, perm)
	}

	return permissions, nil
}

// CheckPermission ユーザーが特定の権限を持っているかチェック
func (s *PermissionService) CheckPermission(userID uuid.UUID, requiredPermission string) (bool, error) {
	// TODO: 監査ログ強化
	// - 権限チェックの詳細ログ (成功/失敗、リソース、時刻)
	// - セキュリティアラート (権限昇格試行、異常パターン)
	// - パフォーマンスメトリクス (応答時間、呼び出し頻度)
	
	permissions, err := s.GetUserPermissions(userID)
	if err != nil {
		return false, err
	}

	hasPermission := s.hasPermission(permissions, requiredPermission)
	
	// TODO: ログ出力実装
	// log.Info("Permission check", zap.String("user_id", userID.String()), 
	//     zap.String("permission", requiredPermission), zap.Bool("granted", hasPermission))

	return hasPermission, nil
}

// CheckPermissionWithScope スコープ条件付きで権限をチェック
func (s *PermissionService) CheckPermissionWithScope(userID uuid.UUID, requiredPermission string, resourceScope map[string]interface{}) (bool, error) {
	// First check basic permission
	hasBasicPerm, err := s.CheckPermission(userID, requiredPermission)
	if err != nil {
		return false, err
	}
	if !hasBasicPerm {
		return false, nil
	}

	// Check scope restrictions
	var userScopes []models.UserScope
	if err := s.db.Where("user_id = ?", userID).Find(&userScopes).Error; err != nil {
		return false, err
	}

	// If no scopes defined, allow access
	if len(userScopes) == 0 {
		return true, nil
	}

	// Check if any scope matches
	for _, scope := range userScopes {
		// Convert JSONB to json.RawMessage
		scopeJSON, err := json.Marshal(scope.ScopeValue)
		if err != nil {
			continue
		}
		if s.evaluateScope(json.RawMessage(scopeJSON), resourceScope) {
			return true, nil
		}
	}

	return false, nil
}

// evaluateScope JSONBスコープ条件をリソーススコープと照合評価
func (s *PermissionService) evaluateScope(scopeValue json.RawMessage, resourceScope map[string]interface{}) bool {
	var conditions map[string]interface{}
	if err := json.Unmarshal(scopeValue, &conditions); err != nil {
		return false
	}

	for key, expectedValue := range conditions {
		if actualValue, exists := resourceScope[key]; exists {
			if !s.compareValues(expectedValue, actualValue) {
				return false
			}
		} else {
			// Required scope field missing
			return false
		}
	}

	return true
}

// compareValues 配列とワイルドカードをサポートしてスコープ値を比較
func (s *PermissionService) compareValues(expected, actual interface{}) bool {
	switch exp := expected.(type) {
	case string:
		if exp == "*" {
			return true // Wildcard matches anything
		}
		if act, ok := actual.(string); ok {
			return exp == act
		}
	case []interface{}:
		// Check if actual value is in the expected array
		for _, val := range exp {
			if s.compareValues(val, actual) {
				return true
			}
		}
	case map[string]interface{}:
		// Nested conditions
		if actMap, ok := actual.(map[string]interface{}); ok {
			for k, v := range exp {
				if actVal, exists := actMap[k]; exists {
					if !s.compareValues(v, actVal) {
						return false
					}
				} else {
					return false
				}
			}
			return true
		}
	default:
		return expected == actual
	}
	return false
}

// hasPermission ワイルドカードサポート付きでユーザー権限に指定権限が存在するかチェック
func (s *PermissionService) hasPermission(userPermissions []string, requiredPermission string) bool {
	for _, perm := range userPermissions {
		if perm == "*" || perm == "*:*" {
			return true // Super admin wildcard
		}
		if perm == requiredPermission {
			return true // Exact match
		}
		if s.matchesWildcard(perm, requiredPermission) {
			return true // Wildcard pattern match
		}
	}
	return false
}

// matchesWildcard 権限がワイルドカードパターンにマッチするかチェック
func (s *PermissionService) matchesWildcard(pattern, permission string) bool {
	// Handle module:* patterns (e.g., "user:*" matches "user:read")
	if strings.HasSuffix(pattern, ":*") {
		module := strings.TrimSuffix(pattern, ":*")
		return strings.HasPrefix(permission, module+":")
	}

	// Handle *:action patterns (e.g., "*:read" matches "user:read")
	if strings.HasPrefix(pattern, "*:") {
		action := strings.TrimPrefix(pattern, "*:")
		return strings.HasSuffix(permission, ":"+action)
	}

	return false
}

// GetRolePermissions 特定のロールの権限を取得
func (s *PermissionService) GetRolePermissions(roleName string) []Permission {
	if perms, exists := PermissionMatrix[roleName]; exists {
		return perms
	}
	return []Permission{}
}

// ValidatePermission 権限文字列が有効かバリデーション
func (s *PermissionService) ValidatePermission(permission string) bool {
	if permission == "*" || permission == "*:*" {
		return true
	}

	parts := strings.Split(permission, ":")
	if len(parts) != 2 {
		return false
	}

	module, action := parts[0], parts[1]
	return s.isValidModule(module) && s.isValidAction(action)
}

// isValidModule モジュールが有効かチェック
func (s *PermissionService) isValidModule(module string) bool {
	if module == "*" {
		return true
	}
	validModules := []string{
		string(ModuleUser),
		string(ModuleDepartment),
		string(ModuleRole),
		string(ModulePermission),
		string(ModuleAudit),
		string(ModuleSystem),
	}

	for _, valid := range validModules {
		if module == valid {
			return true
		}
	}
	return false
}

// isValidAction アクションが有効かチェック
func (s *PermissionService) isValidAction(action string) bool {
	if action == "*" {
		return true
	}
	validActions := []string{
		string(ActionCreate),
		string(ActionRead),
		string(ActionUpdate),
		string(ActionDelete),
		string(ActionList),
		string(ActionManage),
	}

	for _, valid := range validActions {
		if action == valid {
			return true
		}
	}
	return false
}
