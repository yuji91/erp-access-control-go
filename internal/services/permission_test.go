package services

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"erp-access-control-go/models"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/pkg/logger"
)

// テスト用ヘルパー関数
func setupTestPermission(t *testing.T) (*PermissionService, *gorm.DB) {
	db := setupTestDB(t)

	// テストデータをクリア
	db.Exec("DELETE FROM role_permissions")
	db.Exec("DELETE FROM user_roles")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM permissions")
	db.Exec("DELETE FROM roles")

	// 権限テーブルを作成
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS permissions (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			module TEXT NOT NULL,
			action TEXT NOT NULL
		)
	`).Error
	require.NoError(t, err)

	// ロールテーブルを作成
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS roles (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			name TEXT NOT NULL,
			parent_id TEXT,
			FOREIGN KEY (parent_id) REFERENCES roles(id)
		)
	`).Error
	require.NoError(t, err)

	// ロール権限中間テーブルを作成
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS role_permissions (
			role_id TEXT NOT NULL,
			permission_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (role_id, permission_id),
			FOREIGN KEY (role_id) REFERENCES roles(id),
			FOREIGN KEY (permission_id) REFERENCES permissions(id)
		)
	`).Error
	require.NoError(t, err)

	// ユーザーロール中間テーブルを作成
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_roles (
			user_id TEXT NOT NULL,
			role_id TEXT NOT NULL,
			is_active BOOLEAN DEFAULT true,
			valid_from DATETIME DEFAULT CURRENT_TIMESTAMP,
			valid_to DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, role_id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (role_id) REFERENCES roles(id)
		)
	`).Error
	require.NoError(t, err)

	appLogger := logger.NewLogger()
	service := NewPermissionService(db, appLogger)

	return service, db
}

// createPermissionForPermissionTest テスト用権限作成ヘルパー
func createPermissionForPermissionTest(t *testing.T, db *gorm.DB, module, action string) *models.Permission {
	permissionID := uuid.New()
	err := db.Exec("INSERT INTO permissions (id, module, action, created_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)",
		permissionID.String(), module, action).Error
	require.NoError(t, err)

	permission := &models.Permission{
		BaseModel: models.BaseModel{ID: permissionID},
		Module:    module,
		Action:    action,
	}
	return permission
}

// createRoleForPermissionTest テスト用ロール作成ヘルパー
func createRoleForPermissionTest(t *testing.T, db *gorm.DB, name string, parentID *uuid.UUID) *models.Role {
	roleID := uuid.New()
	var parentIDStr interface{}
	if parentID != nil {
		parentIDStr = parentID.String()
	}

	err := db.Exec("INSERT INTO roles (id, name, parent_id, created_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)",
		roleID.String(), name, parentIDStr).Error
	require.NoError(t, err)

	role := &models.Role{
		BaseModel: models.BaseModel{ID: roleID},
		Name:      name,
		ParentID:  parentID,
	}
	return role
}

// assignPermissionToRole ロールに権限を割り当てるヘルパー
func assignPermissionToRole(t *testing.T, db *gorm.DB, roleID, permissionID uuid.UUID) {
	err := db.Exec("INSERT INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, CURRENT_TIMESTAMP)",
		roleID.String(), permissionID.String()).Error
	require.NoError(t, err)
}

// TestPermissionService_CreatePermission 権限作成のテスト
func TestPermissionService_CreatePermission(t *testing.T) {
	svc, _ := setupTestPermission(t)

	t.Run("正常系: 基本権限作成", func(t *testing.T) {
		req := CreatePermissionRequest{
			Module:      "inventory",
			Action:      "create",
			Description: "在庫作成権限",
		}

		resp, err := svc.CreatePermission(req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "inventory", resp.Module)
		assert.Equal(t, "create", resp.Action)
		assert.Equal(t, "inventory:create", resp.Code)
		assert.Equal(t, "在庫作成権限", resp.Description)
		assert.False(t, resp.IsSystem)
		assert.NotEqual(t, uuid.Nil, resp.ID)
	})

	t.Run("異常系: 重複権限作成", func(t *testing.T) {
		// 既存権限作成（依存関係のない基本権限を使用）
		req1 := CreatePermissionRequest{
			Module: "orders",
			Action: "create", // createは依存関係なし
		}
		_, err := svc.CreatePermission(req1)
		require.NoError(t, err)

		// 同じ権限を再作成（エラーになる）
		req2 := CreatePermissionRequest{
			Module: "orders",
			Action: "create",
		}
		_, err = svc.CreatePermission(req2)

		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("異常系: システム権限作成", func(t *testing.T) {
		req := CreatePermissionRequest{
			Module: "user",
			Action: "read",
		}

		_, err := svc.CreatePermission(req)

		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
	})

	t.Run("異常系: 無効なモジュール", func(t *testing.T) {
		req := CreatePermissionRequest{
			Module: "invalid_module",
			Action: "create",
		}

		_, err := svc.CreatePermission(req)

		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
	})

	t.Run("異常系: 無効なアクション", func(t *testing.T) {
		req := CreatePermissionRequest{
			Module: "inventory",
			Action: "invalid_action",
		}

		_, err := svc.CreatePermission(req)

		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
	})
}

// TestPermissionService_GetPermission 権限詳細取得のテスト
func TestPermissionService_GetPermission(t *testing.T) {
	svc, db := setupTestPermission(t)

	t.Run("正常系: 存在する権限取得", func(t *testing.T) {
		// テスト権限作成（依存関係のない基本権限を使用）
		permission := createPermissionForPermissionTest(t, db, "inventory", "create")

		resp, err := svc.GetPermission(permission.ID)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, permission.ID, resp.ID)
		assert.Equal(t, "inventory", resp.Module)
		assert.Equal(t, "create", resp.Action)
		assert.Equal(t, "inventory:create", resp.Code)
		assert.NotNil(t, resp.Roles)
		assert.NotNil(t, resp.UsageStats)
	})

	t.Run("異常系: 存在しない権限", func(t *testing.T) {
		nonExistentID := uuid.New()

		_, err := svc.GetPermission(nonExistentID)

		assert.Error(t, err)
		assert.True(t, errors.IsNotFound(err))
	})
}

// TestPermissionService_UpdatePermission 権限更新のテスト
func TestPermissionService_UpdatePermission(t *testing.T) {
	svc, db := setupTestPermission(t)

	t.Run("正常系: 説明更新", func(t *testing.T) {
		// テスト権限作成
		permission := createPermissionForPermissionTest(t, db, "inventory", "create")
		newDescription := "更新された在庫作成権限"

		req := UpdatePermissionRequest{
			Description: &newDescription,
		}

		resp, err := svc.UpdatePermission(permission.ID, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, newDescription, resp.Description)
	})

	t.Run("異常系: システム権限更新", func(t *testing.T) {
		// システム権限作成
		permission := createPermissionForPermissionTest(t, db, "user", "list")
		newDescription := "システム権限変更試行"

		req := UpdatePermissionRequest{
			Description: &newDescription,
		}

		_, err := svc.UpdatePermission(permission.ID, req)

		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
	})

	t.Run("異常系: 存在しない権限更新", func(t *testing.T) {
		nonExistentID := uuid.New()
		newDescription := "存在しない権限"

		req := UpdatePermissionRequest{
			Description: &newDescription,
		}

		_, err := svc.UpdatePermission(nonExistentID, req)

		assert.Error(t, err)
		assert.True(t, errors.IsNotFound(err))
	})
}

// TestPermissionService_DeletePermission 権限削除のテスト
func TestPermissionService_DeletePermission(t *testing.T) {
	svc, db := setupTestPermission(t)

	t.Run("正常系: 未使用権限削除", func(t *testing.T) {
		// テスト権限作成
		permission := createPermissionForPermissionTest(t, db, "inventory", "archive")

		err := svc.DeletePermission(permission.ID)

		assert.NoError(t, err)
	})

	t.Run("異常系: システム権限削除", func(t *testing.T) {
		// システム権限作成
		permission := createPermissionForPermissionTest(t, db, "permission", "read")

		err := svc.DeletePermission(permission.ID)

		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
	})

	t.Run("異常系: ロール割り当て済み権限削除", func(t *testing.T) {
		// テスト権限とロール作成
		permission := createPermissionForPermissionTest(t, db, "inventory", "manage")
		role := createRoleForPermissionTest(t, db, "在庫マネージャー", nil)
		assignPermissionToRole(t, db, role.ID, permission.ID)

		err := svc.DeletePermission(permission.ID)

		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
	})

	t.Run("異常系: 存在しない権限削除", func(t *testing.T) {
		nonExistentID := uuid.New()

		err := svc.DeletePermission(nonExistentID)

		assert.Error(t, err)
		assert.True(t, errors.IsNotFound(err))
	})
}

// TestPermissionService_GetPermissions 権限一覧取得のテスト
func TestPermissionService_GetPermissions(t *testing.T) {
	svc, db := setupTestPermission(t)

	// テストデータ準備
	perm1 := createPermissionForPermissionTest(t, db, "inventory", "create")
	_ = createPermissionForPermissionTest(t, db, "inventory", "read")
	createPermissionForPermissionTest(t, db, "task", "create")

	t.Run("正常系: 全権限取得", func(t *testing.T) {
		req := GetPermissionsRequest{
			Page:  1,
			Limit: 10,
		}

		resp, err := svc.GetPermissions(req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int64(3), resp.Total)
		assert.Len(t, resp.Permissions, 3)
		assert.Equal(t, 1, resp.Page)
		assert.Equal(t, 10, resp.Limit)
	})

	t.Run("正常系: モジュールフィルタ", func(t *testing.T) {
		req := GetPermissionsRequest{
			Module: "inventory",
			Page:   1,
			Limit:  10,
		}

		resp, err := svc.GetPermissions(req)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), resp.Total)
		assert.Len(t, resp.Permissions, 2)
	})

	t.Run("正常系: アクションフィルタ", func(t *testing.T) {
		req := GetPermissionsRequest{
			Action: "create",
			Page:   1,
			Limit:  10,
		}

		resp, err := svc.GetPermissions(req)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), resp.Total)
		assert.Len(t, resp.Permissions, 2)
	})

	t.Run("正常系: 検索フィルタ", func(t *testing.T) {
		req := GetPermissionsRequest{
			Search: "inventory",
			Page:   1,
			Limit:  10,
		}

		resp, err := svc.GetPermissions(req)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), resp.Total)
		assert.Len(t, resp.Permissions, 2)
	})

	t.Run("正常系: ページング", func(t *testing.T) {
		req := GetPermissionsRequest{
			Page:  1,
			Limit: 2,
		}

		resp, err := svc.GetPermissions(req)

		assert.NoError(t, err)
		assert.Equal(t, int64(3), resp.Total)
		assert.Len(t, resp.Permissions, 2)
		assert.Equal(t, 2, resp.TotalPages)
	})

	t.Run("正常系: ロール使用フィルタ", func(t *testing.T) {
		// テストロール作成と権限割り当て
		role := createRoleForPermissionTest(t, db, "テストロール", nil)
		assignPermissionToRole(t, db, role.ID, perm1.ID)

		req := GetPermissionsRequest{
			UsedByRole: role.ID.String(),
			Page:       1,
			Limit:      10,
		}

		resp, err := svc.GetPermissions(req)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), resp.Total)
		assert.Len(t, resp.Permissions, 1)
		assert.Equal(t, perm1.ID, resp.Permissions[0].ID)
	})

	t.Run("異常系: 無効なロールID", func(t *testing.T) {
		req := GetPermissionsRequest{
			UsedByRole: "invalid-uuid",
			Page:       1,
			Limit:      10,
		}

		_, err := svc.GetPermissions(req)

		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
	})
}

// TestPermissionService_GetPermissionMatrix 権限マトリックス取得のテスト
func TestPermissionService_GetPermissionMatrix(t *testing.T) {
	svc, db := setupTestPermission(t)

	// テストデータ準備
	perm1 := createPermissionForPermissionTest(t, db, "inventory", "create")
	perm2 := createPermissionForPermissionTest(t, db, "inventory", "read")
	_ = createPermissionForPermissionTest(t, db, "task", "create") // unused permission for testing

	role1 := createRoleForPermissionTest(t, db, "在庫管理者", nil)
	role2 := createRoleForPermissionTest(t, db, "在庫確認者", nil)

	// 権限割り当て
	assignPermissionToRole(t, db, role1.ID, perm1.ID)
	assignPermissionToRole(t, db, role1.ID, perm2.ID)
	assignPermissionToRole(t, db, role2.ID, perm2.ID)

	t.Run("正常系: 権限マトリックス取得", func(t *testing.T) {
		resp, err := svc.GetPermissionMatrix()

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.Modules)
		assert.Equal(t, 3, resp.Summary.TotalPermissions)
		assert.Equal(t, 2, resp.Summary.TotalModules)
		assert.Equal(t, 2, resp.Summary.TotalActions)
		assert.Equal(t, 1, resp.Summary.UnusedPermissions) // perm3 is unused

		// モジュール構造確認
		var inventoryModule *ModuleInfo
		for _, module := range resp.Modules {
			if module.Name == "inventory" {
				inventoryModule = &module
				break
			}
		}
		assert.NotNil(t, inventoryModule)
		assert.Equal(t, "在庫管理", inventoryModule.DisplayName)
		assert.Len(t, inventoryModule.Actions, 2)
	})
}

// TestPermissionService_GetPermissionsByModule モジュール別権限取得のテスト
func TestPermissionService_GetPermissionsByModule(t *testing.T) {
	svc, db := setupTestPermission(t)

	// テストデータ準備
	createPermissionForPermissionTest(t, db, "inventory", "create")
	createPermissionForPermissionTest(t, db, "inventory", "read")
	createPermissionForPermissionTest(t, db, "task", "create")

	t.Run("正常系: 有効なモジュール", func(t *testing.T) {
		resp, err := svc.GetPermissionsByModule("inventory")

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "inventory", resp.Module)
		assert.Equal(t, "在庫管理", resp.DisplayName)
		assert.Equal(t, 2, resp.Total)
		assert.Len(t, resp.Permissions, 2)
	})

	t.Run("異常系: 無効なモジュール", func(t *testing.T) {
		_, err := svc.GetPermissionsByModule("invalid_module")

		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
	})
}

// TestPermissionService_GetRolesByPermission 権限保有ロール取得のテスト
func TestPermissionService_GetRolesByPermission(t *testing.T) {
	svc, db := setupTestPermission(t)

	// テストデータ準備
	permission := createPermissionForPermissionTest(t, db, "inventory", "manage")
	role1 := createRoleForPermissionTest(t, db, "在庫マネージャー", nil)
	role2 := createRoleForPermissionTest(t, db, "チームリーダー", nil)

	// 権限割り当て
	assignPermissionToRole(t, db, role1.ID, permission.ID)
	assignPermissionToRole(t, db, role2.ID, permission.ID)

	t.Run("正常系: 権限保有ロール取得", func(t *testing.T) {
		roles, err := svc.GetRolesByPermission(permission.ID)

		assert.NoError(t, err)
		assert.Len(t, roles, 2)
		assert.Contains(t, []string{roles[0].Name, roles[1].Name}, "在庫マネージャー")
		assert.Contains(t, []string{roles[0].Name, roles[1].Name}, "チームリーダー")
	})

	t.Run("異常系: 存在しない権限", func(t *testing.T) {
		nonExistentID := uuid.New()

		_, err := svc.GetRolesByPermission(nonExistentID)

		assert.Error(t, err)
		assert.True(t, errors.IsNotFound(err))
	})
}

// TestPermissionService_SystemPermissionProtection システム権限保護のテスト
func TestPermissionService_SystemPermissionProtection(t *testing.T) {
	svc, db := setupTestPermission(t)

	systemPermissions := []struct {
		module string
		action string
	}{
		{"user", "read"},
		{"user", "list"},
		{"department", "read"},
		{"role", "list"},
		{"permission", "read"},
		{"system", "admin"},
		{"audit", "read"},
	}

	for _, sp := range systemPermissions {
		t.Run("システム権限: "+sp.module+":"+sp.action, func(t *testing.T) {
			// システム権限作成試行
			_, err := svc.CreatePermission(CreatePermissionRequest{
				Module: sp.module,
				Action: sp.action,
			})
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))

			// 既存システム権限での更新・削除試行
			permission := createPermissionForPermissionTest(t, db, sp.module, sp.action)

			// 更新試行
			_, err = svc.UpdatePermission(permission.ID, UpdatePermissionRequest{
				Description: stringPtr("更新試行"),
			})
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))

			// 削除試行
			err = svc.DeletePermission(permission.ID)
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))
		})
	}
}

// TestPermissionService_Step43_ValidationEnhancements Step 4.3 バリデーション強化テスト
func TestPermissionService_Step43_ValidationEnhancements(t *testing.T) {
	service, db := setupTestPermission(t)

	t.Run("Module-Action組み合わせバリデーション", func(t *testing.T) {
		tests := []struct {
			name     string
			module   string
			action   string
			expected bool
		}{
			{"基本CRUD - user:create", "user", "create", true},
			{"基本CRUD - user:read", "user", "read", true},
			{"基本CRUD - user:update", "user", "update", true},
			{"基本CRUD - user:delete", "user", "delete", true},
			{"基本CRUD - user:list", "user", "list", true},
			{"その他有効 - user:manage", "user", "manage", true},
			{"その他有効 - user:view", "user", "view", true},
			{"audit:view (許可)", "audit", "view", true},
			{"audit:export (許可)", "audit", "export", true},
			{"audit:create (禁止)", "audit", "create", false},
			{"audit:update (禁止)", "audit", "update", false},
			{"audit:delete (禁止)", "audit", "delete", false},
			{"audit:manage (禁止)", "audit", "manage", false},
			{"system:admin (許可)", "system", "admin", true},
			{"system:create (禁止)", "system", "create", false},
			{"system:read (禁止)", "system", "read", false},
			{"system:update (禁止)", "system", "update", false},
			{"orders:create (許可)", "orders", "create", true},
			{"orders:approve (許可)", "orders", "approve", true},
			{"無効なアクション", "user", "invalid", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := service.isValidModuleActionCombination(tt.module, tt.action)
				assert.Equal(t, tt.expected, result, "Expected %v for %s:%s", tt.expected, tt.module, tt.action)
			})
		}
	})

	t.Run("権限依存関係バリデーション", func(t *testing.T) {
		// 前提条件権限を作成
		createPermissionForPermissionTest(t, db, "user", "read")

		// 依存関係チェック（read権限が存在するのでupdate権限作成可能）
		err := service.validatePermissionDependencies("user", "update")
		assert.NoError(t, err)

		// manage権限もread権限が前提
		err = service.validatePermissionDependencies("user", "manage")
		assert.NoError(t, err)

		// 依存関係チェック（readが存在しないモジュールでupdate権限作成）
		err = service.validatePermissionDependencies("nonexistent", "update")
		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
		assert.Contains(t, err.Error(), "Required permission")

		// view権限を作成してexport権限をテスト
		createPermissionForPermissionTest(t, db, "report", "view")
		err = service.validatePermissionDependencies("report", "export")
		assert.NoError(t, err)

		// view権限がない場合のexport権限
		err = service.validatePermissionDependencies("missing", "export")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Required permission")
	})

	t.Run("権限削除時の依存関係チェック", func(t *testing.T) {
		// read権限とupdate権限を作成
		createPermissionForPermissionTest(t, db, "test", "read")
		createPermissionForPermissionTest(t, db, "test", "update")

		// update権限が存在するので、read権限は削除できない
		err := service.validatePermissionDeletion("test", "read")
		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
		assert.Contains(t, err.Error(), "required by")
		assert.Contains(t, err.Error(), "test:update")

		// update権限は削除可能（依存されていない）
		err = service.validatePermissionDeletion("test", "update")
		assert.NoError(t, err)

		// 複数の依存関係をテスト
		createPermissionForPermissionTest(t, db, "complex", "read")
		createPermissionForPermissionTest(t, db, "complex", "update")
		createPermissionForPermissionTest(t, db, "complex", "delete")
		createPermissionForPermissionTest(t, db, "complex", "manage")

		// read権限は多くの権限に依存されている
		err = service.validatePermissionDeletion("complex", "read")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required by")

		// update権限はdelete権限に依存されている
		err = service.validatePermissionDeletion("complex", "update")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required by")
		assert.Contains(t, err.Error(), "complex:delete")
	})

	t.Run("システム権限保護包括テスト", func(t *testing.T) {
		systemPermissions := []struct {
			module string
			action string
		}{
			{"user", "read"},
			{"user", "list"},
			{"department", "read"},
			{"department", "list"},
			{"role", "read"},
			{"role", "list"},
			{"permission", "read"},
			{"permission", "list"},
			{"system", "admin"},
		}

		for _, perm := range systemPermissions {
			t.Run(fmt.Sprintf("システム権限: %s:%s", perm.module, perm.action), func(t *testing.T) {
				req := CreatePermissionRequest{
					Module: perm.module,
					Action: perm.action,
				}

				_, err := service.CreatePermission(req)
				assert.Error(t, err)
				assert.True(t, errors.IsValidationError(err))
				assert.Contains(t, err.Error(), "system permission")
			})
		}
	})

	t.Run("複合バリデーションテスト", func(t *testing.T) {
		// 組み合わせ無効 + システム権限のテスト
		req := CreatePermissionRequest{
			Module: "audit",
			Action: "read", // auditモジュールでreadは無効 + システム権限
		}

		_, err := service.CreatePermission(req)
		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
		// 組み合わせチェックが先に実行される
		assert.Contains(t, err.Error(), "Invalid combination")
	})

	t.Run("権限階層チェーンテスト", func(t *testing.T) {
		// 段階的に権限を作成して依存関係をテスト（システム権限でないモジュールを使用）
		createPermissionForPermissionTest(t, db, "inventory", "read")

		// read → update
		req1 := CreatePermissionRequest{Module: "inventory", Action: "update"}
		_, err := service.CreatePermission(req1)
		assert.NoError(t, err)

		// read + update → delete
		req2 := CreatePermissionRequest{Module: "inventory", Action: "delete"}
		_, err = service.CreatePermission(req2)
		assert.NoError(t, err)

		// read → manage
		req3 := CreatePermissionRequest{Module: "inventory", Action: "manage"}
		_, err = service.CreatePermission(req3)
		assert.NoError(t, err)

		// 階層削除テスト：read権限は削除できない（多くの権限に依存される）
		err = service.validatePermissionDeletion("inventory", "read")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required by")

		// update権限も削除できない（delete権限に依存される）
		err = service.validatePermissionDeletion("inventory", "update")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required by")

		// delete権限とmanage権限は削除可能
		err = service.validatePermissionDeletion("inventory", "delete")
		assert.NoError(t, err)

		err = service.validatePermissionDeletion("inventory", "manage")
		assert.NoError(t, err)
	})
}

// stringPtr 文字列ポインタ作成ヘルパー
func stringPtr(s string) *string {
	return &s
}
