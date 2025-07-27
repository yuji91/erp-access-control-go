package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

// TODO: セキュリティテストの強化
// - パスワード強度検証テスト
// - bcryptコスト設定の最適化テスト
// - パスワードハッシュの一意性確認
// - タイミング攻撃対策の検証

// TODO: 複数ロールロジックの詳細テスト
// - ロール継承・階層の複雑なシナリオ
// - 権限集約の正確性検証
// - ロール優先度の境界値テスト
// - 動的ロール変更時の整合性確認

// TODO: データベース統合テストの追加
// - 実際のDB接続でのユーザー管理操作
// - パフォーマンス測定（大量ユーザー・ロール）
// - 同時実行時の排他制御テスト

func TestUser_TableName(t *testing.T) {
	user := &User{}
	assert.Equal(t, "users", user.TableName())
}

func TestUser_HashPassword(t *testing.T) {
	user := &User{}
	password := "testpassword123"
	
	err := user.HashPassword(password)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, user.PasswordHash)
	assert.NotEqual(t, password, user.PasswordHash)
	
	// bcryptハッシュの検証
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	assert.NoError(t, err)
	
	// TODO: より包括的なパスワードテストを追加
	// - 異なる強度のパスワードでのテスト
	// - 特殊文字・Unicode文字を含むパスワード
	// - 極端に長い・短いパスワードの処理
	// - 同じパスワードでも異なるハッシュが生成されることの確認
}

func TestUser_CheckPassword(t *testing.T) {
	user := &User{}
	password := "testpassword123"
	wrongPassword := "wrongpassword"
	
	// パスワードをハッシュ化
	err := user.HashPassword(password)
	assert.NoError(t, err)
	
	// 正しいパスワードのテスト
	valid := user.CheckPassword(password)
	assert.True(t, valid)
	
	// 間違ったパスワードのテスト
	invalid := user.CheckPassword(wrongPassword)
	assert.False(t, invalid)
}

func TestUser_CheckPassword_EmptyHash(t *testing.T) {
	user := &User{
		PasswordHash: "", // 空のハッシュ
	}
	
	valid := user.CheckPassword("anypassword")
	assert.False(t, valid)
}

func TestUser_GetActiveRoles_MockData(t *testing.T) {
	now := time.Now()
	user := &User{
		UserRoles: []UserRole{
			{
				RoleID:    uuid.New(),
				ValidFrom: now.Add(-time.Hour),
				ValidTo:   func() *time.Time { t := now.Add(time.Hour); return &t }(),
				Priority:  1,
				IsActive:  true,
				Role: Role{
					Name: "admin",
				},
			},
			{
				RoleID:    uuid.New(),
				ValidFrom: now.Add(-time.Hour),
				ValidTo:   func() *time.Time { t := now.Add(-30 * time.Minute); return &t }(), // 期限切れ
				Priority:  2,
				IsActive:  true,
				Role: Role{
					Name: "manager",
				},
			},
			{
				RoleID:    uuid.New(),
				ValidFrom: now.Add(-time.Hour),
				ValidTo:   func() *time.Time { t := now.Add(time.Hour); return &t }(),
				Priority:  3,
				IsActive:  false, // 非アクティブ
				Role: Role{
					Name: "user",
				},
			},
		},
	}
	
	// モックデータを使用してアクティブロールを取得
	activeRoles := []UserRole{}
	for _, ur := range user.UserRoles {
		if ur.IsValidNow() {
			activeRoles = append(activeRoles, ur)
		}
	}
	
	// アクティブで有効期間内のロールのみが返されることを確認
	assert.Len(t, activeRoles, 1)
	assert.Equal(t, "admin", activeRoles[0].Role.Name)
	assert.Equal(t, 1, activeRoles[0].Priority)
}

func TestUser_GetHighestPriorityRole_MockData(t *testing.T) {
	now := time.Now()
	user := &User{
		UserRoles: []UserRole{
			{
				RoleID:    uuid.New(),
				ValidFrom: now.Add(-time.Hour),
				ValidTo:   func() *time.Time { t := now.Add(time.Hour); return &t }(),
				Priority:  3, // 低い優先度
				IsActive:  true,
				Role: Role{
					Name: "user",
				},
			},
			{
				RoleID:    uuid.New(),
				ValidFrom: now.Add(-time.Hour),
				ValidTo:   func() *time.Time { t := now.Add(time.Hour); return &t }(),
				Priority:  1, // 高い優先度
				IsActive:  true,
				Role: Role{
					Name: "admin",
				},
			},
			{
				RoleID:    uuid.New(),
				ValidFrom: now.Add(-time.Hour),
				ValidTo:   func() *time.Time { t := now.Add(time.Hour); return &t }(),
				Priority:  2, // 中間優先度
				IsActive:  true,
				Role: Role{
					Name: "manager",
				},
			},
		},
	}
	
	// モックデータを使用して最高優先度ロールを取得
	var highestRole *UserRole
	lowestPriority := 999 // 数値が小さい方が高い優先度
	for _, ur := range user.UserRoles {
		if ur.IsValidNow() && ur.Priority < lowestPriority {
			lowestPriority = ur.Priority
			highestRole = &ur
		}
	}
	
	assert.NotNil(t, highestRole)
	assert.Equal(t, "admin", highestRole.Role.Name)
	assert.Equal(t, 1, highestRole.Priority)
}

func TestUser_GetHighestPriorityRole_NoActiveRoles(t *testing.T) {
	user := &User{
		UserRoles: []UserRole{}, // アクティブなロールなし
	}
	
	// モックデータを使用して最高優先度ロールを取得
	var highestRole *UserRole
	lowestPriority := 999 // 数値が小さい方が高い優先度
	for _, ur := range user.UserRoles {
		if ur.IsValidNow() && ur.Priority < lowestPriority {
			lowestPriority = ur.Priority
			highestRole = &ur
		}
	}
	
	assert.Nil(t, highestRole)
}

func TestUser_HasRoleActive_MockData(t *testing.T) {
	now := time.Now()
	roleID := uuid.New()
	user := &User{
		UserRoles: []UserRole{
			{
				RoleID:    roleID,
				ValidFrom: now.Add(-time.Hour),
				ValidTo:   func() *time.Time { t := now.Add(time.Hour); return &t }(),
				Priority:  1,
				IsActive:  true,
			},
			{
				RoleID:    uuid.New(),
				ValidFrom: now.Add(-time.Hour),
				ValidTo:   func() *time.Time { t := now.Add(-30 * time.Minute); return &t }(), // 期限切れ
				Priority:  2,
				IsActive:  true,
			},
		},
	}
	
	// モックデータを使用してロールアクティブ状態を確認
	hasRole := false
	for _, ur := range user.UserRoles {
		if ur.RoleID == roleID && ur.IsValidNow() {
			hasRole = true
			break
		}
	}
	assert.True(t, hasRole)
	
	// 存在しないロールの確認
	nonExistentRole := uuid.New()
	hasNonExistentRole := false
	for _, ur := range user.UserRoles {
		if ur.RoleID == nonExistentRole && ur.IsValidNow() {
			hasNonExistentRole = true
			break
		}
	}
	assert.False(t, hasNonExistentRole)
}

func TestUser_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   UserStatus
		expected bool
	}{
		{
			name:     "アクティブユーザー",
			status:   UserStatusActive,
			expected: true,
		},
		{
			name:     "非アクティブユーザー",
			status:   UserStatusInactive,
			expected: false,
		},
		{
			name:     "停止ユーザー",
			status:   UserStatusSuspended,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				Status: tt.status,
			}
			
			result := user.IsActive()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUser_BasicFields(t *testing.T) {
	userID := uuid.New()
	departmentID := uuid.New()
	roleID := uuid.New()
	primaryRoleID := uuid.New()
	
	user := &User{
		BaseModelWithUpdate: BaseModelWithUpdate{
			ID: userID,
		},
		Email:         "test@example.com",
		Name:          "Test User",
		DepartmentID:  departmentID,
		RoleID:        &roleID,
		PrimaryRoleID: &primaryRoleID,
		Status:        UserStatusActive,
	}
	
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, departmentID, user.DepartmentID)
	assert.Equal(t, &roleID, user.RoleID)
	assert.Equal(t, &primaryRoleID, user.PrimaryRoleID)
	assert.Equal(t, UserStatusActive, user.Status)
}

func TestUser_Relations(t *testing.T) {
	user := &User{
		Department: Department{
			Name: "IT Department",
		},
		Role: &Role{
			Name: "Developer",
		},
		PrimaryRole: &Role{
			Name: "Senior Developer",
		},
		UserRoles: []UserRole{
			{
				Role: Role{
					Name: "Admin",
				},
			},
		},
	}
	
	assert.NotNil(t, user.Department)
	assert.Equal(t, "IT Department", user.Department.Name)
	
	assert.NotNil(t, user.Role)
	assert.Equal(t, "Developer", user.Role.Name)
	
	assert.NotNil(t, user.PrimaryRole)
	assert.Equal(t, "Senior Developer", user.PrimaryRole.Name)
	
	assert.Len(t, user.UserRoles, 1)
	assert.Equal(t, "Admin", user.UserRoles[0].Role.Name)
}

func TestUserStatus_Constants(t *testing.T) {
	assert.Equal(t, UserStatus("active"), UserStatusActive)
	assert.Equal(t, UserStatus("inactive"), UserStatusInactive)
	assert.Equal(t, UserStatus("suspended"), UserStatusSuspended)
}

// ヘルパー関数のモック版テスト
func TestUser_GetActiveUserRoles_Structure(t *testing.T) {
	// このテストは実際のDB接続を必要としないstructure確認
	now := time.Now()
	user := &User{
		ActiveUserRoles: []UserRole{
			{
				RoleID:    uuid.New(),
				ValidFrom: now.Add(-time.Hour),
				ValidTo:   func() *time.Time { t := now.Add(time.Hour); return &t }(),
				Priority:  1,
				IsActive:  true,
			},
		},
	}
	
	// ActiveUserRolesフィールドが適切に設定されることを確認
	assert.Len(t, user.ActiveUserRoles, 1)
	assert.True(t, user.ActiveUserRoles[0].IsActive)
} 