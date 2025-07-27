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
func setupTestRole(t *testing.T) (*RoleService, *gorm.DB) {
	db := setupTestDB(t)

	// テストデータをクリア
	db.Exec("DELETE FROM role_permissions")
	db.Exec("DELETE FROM user_roles")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM permissions")
	db.Exec("DELETE FROM roles")

	// ロールテーブルを作成
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS roles (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			name TEXT NOT NULL,
			parent_id TEXT,
			FOREIGN KEY (parent_id) REFERENCES roles(id)
		)
	`).Error
	require.NoError(t, err)

	// 権限テーブルを作成
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS permissions (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			module TEXT NOT NULL,
			action TEXT NOT NULL
		)
	`).Error
	require.NoError(t, err)

	// ロール権限中間テーブルを作成
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS role_permissions (
			role_id TEXT,
			permission_id TEXT,
			PRIMARY KEY (role_id, permission_id),
			FOREIGN KEY (role_id) REFERENCES roles(id),
			FOREIGN KEY (permission_id) REFERENCES permissions(id)
		)
	`).Error
	require.NoError(t, err)

	// ユーザーロール中間テーブルを作成
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_roles (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			user_id TEXT NOT NULL,
			role_id TEXT NOT NULL,
			is_active BOOLEAN DEFAULT true,
			priority INTEGER DEFAULT 0,
			valid_from DATETIME DEFAULT CURRENT_TIMESTAMP,
			valid_to DATETIME,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (role_id) REFERENCES roles(id)
		)
	`).Error
	require.NoError(t, err)

	log := logger.NewLogger()
	return NewRoleService(db, log), db
}

func createTestRole(t *testing.T, db *gorm.DB, name string, parentID *uuid.UUID) *models.Role {
	role := &models.Role{
		Name:     name,
		ParentID: parentID,
	}
	require.NoError(t, db.Create(role).Error)
	return role
}

func createTestPermission(t *testing.T, db *gorm.DB, module, action string) *models.Permission {
	// 直接SQLで挿入（GORM/SQLiteのUUID問題回避）
	permID := uuid.New()
	err := db.Exec(`
		INSERT INTO permissions (id, module, action)
		VALUES (?, ?, ?)
	`, permID.String(), module, action).Error
	require.NoError(t, err)

	permission := &models.Permission{
		Module: module,
		Action: action,
	}
	permission.ID = permID
	return permission
}

func createTestUser(t *testing.T, db *gorm.DB, name, email string, primaryRoleID *uuid.UUID) *models.User {
	// 直接SQLで挿入（GORM/SQLiteのUUID問題回避）
	userID := uuid.New()
	err := db.Exec(`
		INSERT INTO users (id, name, email, password_hash, status, primary_role_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID.String(), name, email, "dummy_hash", "active", primaryRoleID).Error
	require.NoError(t, err)

	user := &models.User{
		Name:          name,
		Email:         email,
		PrimaryRoleID: primaryRoleID,
	}
	return user
}

// TestRoleService_InputValidation 入力値検証のテスト
func TestRoleService_InputValidation(t *testing.T) {
	svc, db := setupTestRole(t)

	t.Run("CreateRole入力値検証", func(t *testing.T) {
		t.Run("正常系: 有効なロール名", func(t *testing.T) {
			req := CreateRoleRequest{
				Name: "管理者",
			}

			resp, err := svc.CreateRole(req)
			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, req.Name, resp.Name)
			assert.Nil(t, resp.ParentID)
		})

		t.Run("正常系: 空のロール名（サービス層では許可）", func(t *testing.T) {
			req := CreateRoleRequest{
				Name: "",
			}

			resp, err := svc.CreateRole(req)
			require.NoError(t, err)
			// 注意: Gin bindingのバリデーションはハンドラー層で行われるため、
			// サービス層では空文字も通る
			assert.Equal(t, "", resp.Name)
		})

		t.Run("正常系: 最小長ロール名", func(t *testing.T) {
			req := CreateRoleRequest{
				Name: "AB", // 2文字（最小長）
			}

			resp, err := svc.CreateRole(req)
			require.NoError(t, err)
			assert.Equal(t, req.Name, resp.Name)
		})

		t.Run("正常系: 最大長ロール名", func(t *testing.T) {
			// 100文字のロール名
			longName := "管理者ロール" + string(make([]byte, 85)) // 100文字に調整
			for i := range longName[15:] {
				longName = longName[:15+i] + "A" + longName[15+i+1:]
			}
			req := CreateRoleRequest{
				Name: longName[:100], // 確実に100文字
			}

			resp, err := svc.CreateRole(req)
			require.NoError(t, err)
			assert.Equal(t, req.Name, resp.Name)
		})

		t.Run("異常系: 重複ロール名", func(t *testing.T) {
			// 最初のロール作成
			existingRole := createTestRole(t, db, "既存ロール", nil)

			req := CreateRoleRequest{
				Name: existingRole.Name,
			}

			_, err := svc.CreateRole(req)
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))
			assert.Contains(t, err.Error(), "already exists")
		})

		t.Run("異常系: 存在しない親ロールID", func(t *testing.T) {
			nonExistentID := uuid.New()
			req := CreateRoleRequest{
				Name:     "子ロール",
				ParentID: &nonExistentID,
			}

			_, err := svc.CreateRole(req)
			assert.Error(t, err)
			assert.True(t, errors.IsNotFound(err))
			assert.Contains(t, err.Error(), "Role not found")
		})

		t.Run("正常系: 有効な親ロールID", func(t *testing.T) {
			parentRole := createTestRole(t, db, "親ロール", nil)

			req := CreateRoleRequest{
				Name:     "子ロール",
				ParentID: &parentRole.ID,
			}

			resp, err := svc.CreateRole(req)
			require.NoError(t, err)
			assert.Equal(t, req.Name, resp.Name)
			assert.Equal(t, parentRole.ID, *resp.ParentID)
		})

		t.Run("異常系: 存在しない権限ID", func(t *testing.T) {
			nonExistentPermID := uuid.New()
			req := CreateRoleRequest{
				Name:          "テストロール",
				PermissionIDs: []uuid.UUID{nonExistentPermID},
			}

			_, err := svc.CreateRole(req)
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))
			assert.Contains(t, err.Error(), "permissions do not exist")
		})

		t.Run("正常系: 権限なしでロール作成", func(t *testing.T) {
			req := CreateRoleRequest{
				Name:          "権限なしロール",
				PermissionIDs: []uuid.UUID{}, // 空の権限配列
			}

			resp, err := svc.CreateRole(req)
			require.NoError(t, err)
			assert.Equal(t, req.Name, resp.Name)
		})

		t.Run("異常系: 最大階層深度超過", func(t *testing.T) {
			// 5階層のロール構造を作成
			level1 := createTestRole(t, db, "レベル1", nil)
			level2 := createTestRole(t, db, "レベル2", &level1.ID)
			level3 := createTestRole(t, db, "レベル3", &level2.ID)
			level4 := createTestRole(t, db, "レベル4", &level3.ID)
			level5 := createTestRole(t, db, "レベル5", &level4.ID)

			// 6階層目を作成しようとする
			req := CreateRoleRequest{
				Name:     "レベル6",
				ParentID: &level5.ID,
			}

			_, err := svc.CreateRole(req)
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))
			assert.Contains(t, err.Error(), "Maximum hierarchy depth")
		})
	})

	t.Run("UpdateRole入力値検証", func(t *testing.T) {
		existingRole := createTestRole(t, db, "更新対象ロール", nil)

		t.Run("正常系: 有効なロール名更新", func(t *testing.T) {
			newName := "更新後ロール名"
			req := UpdateRoleRequest{
				Name: &newName,
			}

			resp, err := svc.UpdateRole(existingRole.ID, req)
			require.NoError(t, err)
			assert.Equal(t, newName, resp.Name)
		})

		t.Run("異常系: 更新時の重複ロール名", func(t *testing.T) {
			otherRole := createTestRole(t, db, "他のロール", nil)

			req := UpdateRoleRequest{
				Name: &otherRole.Name,
			}

			_, err := svc.UpdateRole(existingRole.ID, req)
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))
			assert.Contains(t, err.Error(), "already exists")
		})

		t.Run("異常系: 自己参照", func(t *testing.T) {
			req := UpdateRoleRequest{
				ParentID: &existingRole.ID,
			}

			_, err := svc.UpdateRole(existingRole.ID, req)
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))
			assert.Contains(t, err.Error(), "cannot be its own parent")
		})

		t.Run("正常系: 親ロール変更", func(t *testing.T) {
			parentRole := createTestRole(t, db, "新しい親ロール", nil)

			req := UpdateRoleRequest{
				ParentID: &parentRole.ID,
			}

			resp, err := svc.UpdateRole(existingRole.ID, req)
			require.NoError(t, err)
			assert.Equal(t, parentRole.ID, *resp.ParentID)
		})
	})

	t.Run("基本的なAssignPermissions検証", func(t *testing.T) {
		testRole := createTestRole(t, db, "権限テストロール", nil)

		t.Run("異常系: 存在しない権限ID", func(t *testing.T) {
			nonExistentID := uuid.New()
			req := AssignPermissionsRequest{
				PermissionIDs: []uuid.UUID{nonExistentID},
				Replace:       true,
			}

			_, err := svc.AssignPermissions(testRole.ID, req)
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))
			assert.Contains(t, err.Error(), "permissions do not exist")
		})

		t.Run("異常系: 存在しないロールID", func(t *testing.T) {
			nonExistentRoleID := uuid.New()

			req := AssignPermissionsRequest{
				PermissionIDs: []uuid.UUID{}, // 空配列でもロール存在チェックは実行される
				Replace:       true,
			}

			_, err := svc.AssignPermissions(nonExistentRoleID, req)
			assert.Error(t, err)
			assert.True(t, errors.IsNotFound(err))
			assert.Contains(t, err.Error(), "Role not found")
		})

		t.Run("正常系: 空の権限ID配列（権限なし設定）", func(t *testing.T) {
			req := AssignPermissionsRequest{
				PermissionIDs: []uuid.UUID{},
				Replace:       true,
			}

			resp, err := svc.AssignPermissions(testRole.ID, req)
			require.NoError(t, err)
			assert.Len(t, resp.DirectPermissions, 0)
		})
	})
}

// TestRoleService_HierarchyValidation 階層構造検証のテスト
func TestRoleService_HierarchyValidation(t *testing.T) {
	svc, db := setupTestRole(t)

	t.Run("循環参照検証", func(t *testing.T) {
		// A -> B -> C の階層を作成
		roleA := createTestRole(t, db, "ロールA", nil)
		roleB := createTestRole(t, db, "ロールB", &roleA.ID)
		roleC := createTestRole(t, db, "ロールC", &roleB.ID)

		t.Run("異常系: 直接循環参照", func(t *testing.T) {
			// A の親を C にしようとする（A -> B -> C -> A の循環）
			req := UpdateRoleRequest{
				ParentID: &roleC.ID,
			}

			_, err := svc.UpdateRole(roleA.ID, req)
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))
			assert.Contains(t, err.Error(), "Circular reference")
		})

		t.Run("異常系: 間接循環参照", func(t *testing.T) {
			// B の親を C にしようとする（B -> C -> B の循環）
			req := UpdateRoleRequest{
				ParentID: &roleC.ID,
			}

			_, err := svc.UpdateRole(roleB.ID, req)
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))
			assert.Contains(t, err.Error(), "Circular reference")
		})

		t.Run("正常系: 循環しない移動", func(t *testing.T) {
			// 新しい独立したロールを作成し、C の親にする
			newParent := createTestRole(t, db, "新しい親", nil)

			req := UpdateRoleRequest{
				ParentID: &newParent.ID,
			}

			resp, err := svc.UpdateRole(roleC.ID, req)
			require.NoError(t, err)
			assert.Equal(t, newParent.ID, *resp.ParentID)
		})
	})

	t.Run("階層深度検証", func(t *testing.T) {
		// 4階層の構造を作成
		level1 := createTestRole(t, db, "階層1", nil)
		level2 := createTestRole(t, db, "階層2", &level1.ID)
		level3 := createTestRole(t, db, "階層3", &level2.ID)
		level4 := createTestRole(t, db, "階層4", &level3.ID)

		t.Run("正常系: 5階層目まで作成可能", func(t *testing.T) {
			req := CreateRoleRequest{
				Name:     "階層5",
				ParentID: &level4.ID,
			}

			resp, err := svc.CreateRole(req)
			require.NoError(t, err)
			assert.Equal(t, req.Name, resp.Name)
		})

		t.Run("異常系: 6階層目は作成不可", func(t *testing.T) {
			level5 := createTestRole(t, db, "階層5-2", &level4.ID)

			req := CreateRoleRequest{
				Name:     "階層6",
				ParentID: &level5.ID,
			}

			_, err := svc.CreateRole(req)
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))
			assert.Contains(t, err.Error(), "Maximum hierarchy depth")
		})
	})
}

// TestRoleService_DeleteValidation 削除検証のテスト
func TestRoleService_DeleteValidation(t *testing.T) {
	svc, db := setupTestRole(t)

	t.Run("削除制限検証", func(t *testing.T) {
		parentRole := createTestRole(t, db, "親ロール", nil)
		childRole := createTestRole(t, db, "子ロール", &parentRole.ID)

		t.Run("異常系: 子ロールが存在する場合", func(t *testing.T) {
			err := svc.DeleteRole(parentRole.ID)
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))
			assert.Contains(t, err.Error(), "child roles")
		})

		t.Run("正常系: 子ロール削除後は親も削除可能", func(t *testing.T) {
			// 先に子ロールを削除
			err := svc.DeleteRole(childRole.ID)
			require.NoError(t, err)

			// 親ロールを削除
			err = svc.DeleteRole(parentRole.ID)
			require.NoError(t, err)
		})

		t.Run("異常系: プライマリロールとして割り当て済み", func(t *testing.T) {
			role := createTestRole(t, db, "プライマリロール", nil)
			_ = createTestUser(t, db, "テストユーザー", "test@example.com", &role.ID)

			err := svc.DeleteRole(role.ID)
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))
			assert.Contains(t, err.Error(), "primary role")
		})

		t.Run("異常系: ユーザーロールとして割り当て済み", func(t *testing.T) {
			role := createTestRole(t, db, "ユーザーロール", nil)
			user := createTestUser(t, db, "テストユーザー2", "test2@example.com", nil)

			// ユーザーロール関係を作成
			// UserIDを再取得してuser_rolesに挿入
			var userUUID string
			err := db.Raw("SELECT id FROM users WHERE name = ?", user.Name).Scan(&userUUID).Error
			require.NoError(t, err)

			err = db.Exec(`
				INSERT INTO user_roles (user_id, role_id, is_active)
				VALUES (?, ?, ?)
			`, userUUID, role.ID.String(), true).Error
			require.NoError(t, err)

			err = svc.DeleteRole(role.ID)
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))
			assert.Contains(t, err.Error(), "assigned to users")
		})

		t.Run("異常系: 存在しないロール削除", func(t *testing.T) {
			nonExistentID := uuid.New()

			err := svc.DeleteRole(nonExistentID)
			assert.Error(t, err)
			assert.True(t, errors.IsNotFound(err))
			assert.Contains(t, err.Error(), "Role not found")
		})
	})
}
