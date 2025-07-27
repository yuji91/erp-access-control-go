package services

import (
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

// createTestPermissionForPermissionService テスト用権限作成ヘルパー
func createTestPermissionForPermissionService(t *testing.T, db *gorm.DB, module, action string) *models.Permission {
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

// createTestRoleForPermissionService テスト用ロール作成ヘルパー
func createTestRoleForPermissionService(t *testing.T, db *gorm.DB, name string, parentID *uuid.UUID) *models.Role {
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
		// 既存権限作成
		req1 := CreatePermissionRequest{
			Module: "orders",
			Action: "read",
		}
		_, err := svc.CreatePermission(req1)
		require.NoError(t, err)

		// 同じ権限を再作成（エラーになる）
		req2 := CreatePermissionRequest{
			Module: "orders",
			Action: "read",
		}
		_, err = svc.CreatePermission(req2)

		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
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
		// テスト権限作成
		permission := createTestPermissionForPermissionService(t, db, "inventory", "update")

		resp, err := svc.GetPermission(permission.ID)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, permission.ID, resp.ID)
		assert.Equal(t, "inventory", resp.Module)
		assert.Equal(t, "update", resp.Action)
		assert.Equal(t, "inventory:update", resp.Code)
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
		permission := createTestPermissionForPermissionService(t, db, "inventory", "delete")
		newDescription := "更新された在庫削除権限"

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
		permission := createTestPermissionForPermissionService(t, db, "user", "list")
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
		permission := createTestPermissionForPermissionService(t, db, "inventory", "archive")

		err := svc.DeletePermission(permission.ID)

		assert.NoError(t, err)
	})

	t.Run("異常系: システム権限削除", func(t *testing.T) {
		// システム権限作成
		permission := createTestPermissionForPermissionService(t, db, "permission", "read")

		err := svc.DeletePermission(permission.ID)

		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
	})

	t.Run("異常系: ロール割り当て済み権限削除", func(t *testing.T) {
		// テスト権限とロール作成
		permission := createTestPermissionForPermissionService(t, db, "inventory", "manage")
		role := createTestRoleForPermissionService(t, db, "在庫マネージャー", nil)
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
	perm1 := createTestPermissionForPermissionService(t, db, "inventory", "create")
	_ = createTestPermissionForPermissionService(t, db, "inventory", "read")
	createTestPermissionForPermissionService(t, db, "task", "create")

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
		role := createTestRoleForPermissionService(t, db, "テストロール", nil)
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
	perm1 := createTestPermissionForPermissionService(t, db, "inventory", "create")
	perm2 := createTestPermissionForPermissionService(t, db, "inventory", "read")
	_ = createTestPermissionForPermissionService(t, db, "task", "create") // unused permission for testing

	role1 := createTestRoleForPermissionService(t, db, "在庫管理者", nil)
	role2 := createTestRoleForPermissionService(t, db, "在庫確認者", nil)

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
	createTestPermissionForPermissionService(t, db, "inventory", "create")
	createTestPermissionForPermissionService(t, db, "inventory", "read")
	createTestPermissionForPermissionService(t, db, "task", "create")

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
	permission := createTestPermissionForPermissionService(t, db, "inventory", "manage")
	role1 := createTestRoleForPermissionService(t, db, "在庫マネージャー", nil)
	role2 := createTestRoleForPermissionService(t, db, "チームリーダー", nil)

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
			permission := createTestPermissionForPermissionService(t, db, sp.module, sp.action)

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

// stringPtr 文字列ポインタ作成ヘルパー
func stringPtr(s string) *string {
	return &s
}
