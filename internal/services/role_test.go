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

func createRoleForRoleTest(t *testing.T, db *gorm.DB, name string, parentID *uuid.UUID) *models.Role {
	role := &models.Role{
		Name:     name,
		ParentID: parentID,
	}
	require.NoError(t, db.Create(role).Error)
	return role
}

func createPermissionForRoleTest(t *testing.T, db *gorm.DB, module, action string) *models.Permission {
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

func createUserForRoleTest(t *testing.T, db *gorm.DB, name, email string, primaryRoleID *uuid.UUID) *models.User {
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
			existingRole := createRoleForRoleTest(t, db, "既存ロール", nil)

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
			parentRole := createRoleForRoleTest(t, db, "親ロール", nil)

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
			level1 := createRoleForRoleTest(t, db, "レベル1", nil)
			level2 := createRoleForRoleTest(t, db, "レベル2", &level1.ID)
			level3 := createRoleForRoleTest(t, db, "レベル3", &level2.ID)
			level4 := createRoleForRoleTest(t, db, "レベル4", &level3.ID)
			level5 := createRoleForRoleTest(t, db, "レベル5", &level4.ID)

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
		existingRole := createRoleForRoleTest(t, db, "更新対象ロール", nil)

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
			otherRole := createRoleForRoleTest(t, db, "他のロール", nil)

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
			parentRole := createRoleForRoleTest(t, db, "新しい親ロール", nil)

			req := UpdateRoleRequest{
				ParentID: &parentRole.ID,
			}

			resp, err := svc.UpdateRole(existingRole.ID, req)
			require.NoError(t, err)
			assert.Equal(t, parentRole.ID, *resp.ParentID)
		})
	})

	t.Run("基本的なAssignPermissions検証", func(t *testing.T) {
		testRole := createRoleForRoleTest(t, db, "権限テストロール", nil)

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
		roleA := createRoleForRoleTest(t, db, "ロールA", nil)
		roleB := createRoleForRoleTest(t, db, "ロールB", &roleA.ID)
		roleC := createRoleForRoleTest(t, db, "ロールC", &roleB.ID)

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
			newParent := createRoleForRoleTest(t, db, "新しい親", nil)

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
		level1 := createRoleForRoleTest(t, db, "階層1", nil)
		level2 := createRoleForRoleTest(t, db, "階層2", &level1.ID)
		level3 := createRoleForRoleTest(t, db, "階層3", &level2.ID)
		level4 := createRoleForRoleTest(t, db, "階層4", &level3.ID)

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
			level5 := createRoleForRoleTest(t, db, "階層5-2", &level4.ID)

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
		parentRole := createRoleForRoleTest(t, db, "親ロール", nil)
		childRole := createRoleForRoleTest(t, db, "子ロール", &parentRole.ID)

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
			role := createRoleForRoleTest(t, db, "プライマリロール", nil)
			_ = createUserForRoleTest(t, db, "テストユーザー", "test@example.com", &role.ID)

			err := svc.DeleteRole(role.ID)
			assert.Error(t, err)
			assert.True(t, errors.IsValidationError(err))
			assert.Contains(t, err.Error(), "primary role")
		})

		t.Run("異常系: ユーザーロールとして割り当て済み", func(t *testing.T) {
			role := createRoleForRoleTest(t, db, "ユーザーロール", nil)
			user := createUserForRoleTest(t, db, "テストユーザー2", "test2@example.com", nil)

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

// TestRoleService_PermissionInheritance 権限継承のテスト
func TestRoleService_PermissionInheritance(t *testing.T) {
	svc, db := setupTestRole(t)

	t.Run("権限継承システム（基本構造）", func(t *testing.T) {
		// 3階層のロール構造を作成
		// 親ロール（レベル1）
		parentRole := createRoleForRoleTest(t, db, "管理者", nil)

		// 子ロール（レベル2）
		childRole := createRoleForRoleTest(t, db, "部署管理者", &parentRole.ID)

		// 孫ロール（レベル3）
		grandChildRole := createRoleForRoleTest(t, db, "一般ユーザー", &childRole.ID)

		t.Run("階層構造確認", func(t *testing.T) {
			// 親ロールの詳細取得
			parentResp, err := svc.GetRole(parentRole.ID)
			require.NoError(t, err)
			assert.Equal(t, "管理者", parentResp.Name)
			assert.Nil(t, parentResp.ParentID)

			// 子ロールの詳細取得
			childResp, err := svc.GetRole(childRole.ID)
			require.NoError(t, err)
			assert.Equal(t, "部署管理者", childResp.Name)
			assert.NotNil(t, childResp.ParentID)
			assert.Equal(t, parentRole.ID, *childResp.ParentID)

			// 孫ロールの詳細取得
			grandChildResp, err := svc.GetRole(grandChildRole.ID)
			require.NoError(t, err)
			assert.Equal(t, "一般ユーザー", grandChildResp.Name)
			assert.NotNil(t, grandChildResp.ParentID)
			assert.Equal(t, childRole.ID, *grandChildResp.ParentID)
		})

		t.Run("権限継承メソッドの動作確認", func(t *testing.T) {
			// 継承権限取得機能の確認（実際の権限データがなくても動作）
			inherited, err := svc.getInheritedPermissions(childRole.ID)
			require.NoError(t, err)

			// 権限データがないため継承権限は0件
			assert.Len(t, inherited, 0)

			// 権限マージ機能の確認
			direct := []PermissionInfo{}
			merged := svc.mergePermissions(direct, inherited)
			assert.Len(t, merged, 0)
		})

		t.Run("基本的な権限レスポンス構造", func(t *testing.T) {
			// 権限なしの状態での権限レスポンス確認
			perms, err := svc.GetRolePermissions(parentRole.ID)
			require.NoError(t, err)

			assert.Equal(t, parentRole.ID, perms.RoleID)
			assert.Equal(t, "管理者", perms.RoleName)
			assert.Len(t, perms.DirectPermissions, 0)
			assert.Len(t, perms.InheritedPermissions, 0)
			assert.Len(t, perms.AllPermissions, 0)
		})
	})

	t.Run("権限マージアルゴリズム", func(t *testing.T) {
		t.Run("空の権限リスト", func(t *testing.T) {
			direct := []PermissionInfo{}
			inherited := []InheritedPermissionInfo{}

			result := svc.mergePermissions(direct, inherited)
			assert.Len(t, result, 0)
		})

		t.Run("直接権限のみ", func(t *testing.T) {
			perm := createPermissionForRoleTest(t, db, "test", "action")

			direct := []PermissionInfo{
				{ID: perm.ID, Module: "test", Action: "action", Inherited: false},
			}
			inherited := []InheritedPermissionInfo{}

			result := svc.mergePermissions(direct, inherited)
			assert.Len(t, result, 1)
			assert.False(t, result[0].Inherited)
		})

		t.Run("継承権限のみ", func(t *testing.T) {
			perm := createPermissionForRoleTest(t, db, "test2", "action2")

			direct := []PermissionInfo{}
			inherited := []InheritedPermissionInfo{
				{ID: perm.ID, Module: "test2", Action: "action2"},
			}

			result := svc.mergePermissions(direct, inherited)
			assert.Len(t, result, 1)
			assert.True(t, result[0].Inherited)
		})

		t.Run("重複排除", func(t *testing.T) {
			perm := createPermissionForRoleTest(t, db, "test3", "action3")

			direct := []PermissionInfo{
				{ID: perm.ID, Module: "test3", Action: "action3", Inherited: false},
			}
			inherited := []InheritedPermissionInfo{
				{ID: perm.ID, Module: "test3", Action: "action3"},
			}

			result := svc.mergePermissions(direct, inherited)
			assert.Len(t, result, 1)
			assert.False(t, result[0].Inherited) // 直接権限が優先
		})
	})
}

// TestRoleService_HierarchyManagement 階層管理システムのテスト
func TestRoleService_HierarchyManagement(t *testing.T) {
	svc, db := setupTestRole(t)

	t.Run("階層ツリー構築", func(t *testing.T) {
		// 複雑な階層構造を作成
		/*
			ルート1
			├── 子1-1
			│   ├── 孫1-1-1
			│   └── 孫1-1-2
			└── 子1-2
			ルート2
			└── 子2-1
		*/
		root1 := createRoleForRoleTest(t, db, "ルート1", nil)
		root2 := createRoleForRoleTest(t, db, "ルート2", nil)

		child1_1 := createRoleForRoleTest(t, db, "子1-1", &root1.ID)
		child1_2 := createRoleForRoleTest(t, db, "子1-2", &root1.ID)
		_ = createRoleForRoleTest(t, db, "子2-1", &root2.ID) // root2の子ロール

		grandchild1_1_1 := createRoleForRoleTest(t, db, "孫1-1-1", &child1_1.ID)
		grandchild1_1_2 := createRoleForRoleTest(t, db, "孫1-1-2", &child1_1.ID)

		t.Run("階層ツリー取得", func(t *testing.T) {
			hierarchy, err := svc.GetRoleHierarchy()
			require.NoError(t, err)

			// ルートロールは2つ
			assert.Len(t, hierarchy.Roles, 2)

			// ルート1の構造確認
			var root1Node *RoleHierarchyNode
			for i := range hierarchy.Roles {
				if hierarchy.Roles[i].Name == "ルート1" {
					root1Node = &hierarchy.Roles[i]
					break
				}
			}
			require.NotNil(t, root1Node)
			assert.Equal(t, 0, root1Node.Level)
			assert.Len(t, root1Node.Children, 2)

			// 子1-1の構造確認
			var child1_1Node *RoleHierarchyNode
			for i := range root1Node.Children {
				if root1Node.Children[i].Name == "子1-1" {
					child1_1Node = &root1Node.Children[i]
					break
				}
			}
			require.NotNil(t, child1_1Node)
			assert.Equal(t, 1, child1_1Node.Level)
			assert.Len(t, child1_1Node.Children, 2)
		})

		t.Run("深度計算", func(t *testing.T) {
			depth, err := svc.calculateDepth(grandchild1_1_1.ID)
			require.NoError(t, err)
			assert.Equal(t, 3, depth) // 孫は3階層目

			depth, err = svc.calculateDepth(root1.ID)
			require.NoError(t, err)
			assert.Equal(t, 1, depth) // ルートは1階層目
		})

		t.Run("子孫取得", func(t *testing.T) {
			descendants, err := svc.getDescendants(root1.ID)
			require.NoError(t, err)

			// root1の子孫は4つ（子2つ + 孫2つ）
			assert.Len(t, descendants, 4)

			// 全ての子孫IDが含まれることを確認
			descendantSet := make(map[uuid.UUID]bool)
			for _, id := range descendants {
				descendantSet[id] = true
			}
			assert.True(t, descendantSet[child1_1.ID])
			assert.True(t, descendantSet[child1_2.ID])
			assert.True(t, descendantSet[grandchild1_1_1.ID])
			assert.True(t, descendantSet[grandchild1_1_2.ID])
		})
	})

	t.Run("レベル計算", func(t *testing.T) {
		parent := createRoleForRoleTest(t, db, "レベルテスト親", nil)
		child := createRoleForRoleTest(t, db, "レベルテスト子", &parent.ID)
		grandchild := createRoleForRoleTest(t, db, "レベルテスト孫", &child.ID)

		level, err := svc.calculateLevel(parent.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, level) // 親（ルート）はレベル0

		level, err = svc.calculateLevel(child.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, level) // 子はレベル1

		level, err = svc.calculateLevel(grandchild.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, level) // 孫はレベル2
	})
}

// TestRoleService_CRUDOperations CRUD操作のテスト
func TestRoleService_CRUDOperations(t *testing.T) {
	svc, db := setupTestRole(t)

	t.Run("Create操作", func(t *testing.T) {
		t.Run("正常系: 基本的なロール作成", func(t *testing.T) {
			req := CreateRoleRequest{
				Name: "テストロール",
			}

			resp, err := svc.CreateRole(req)
			require.NoError(t, err)
			assert.Equal(t, "テストロール", resp.Name)
			assert.NotEqual(t, uuid.Nil, resp.ID)
			assert.Nil(t, resp.ParentID)
			assert.NotZero(t, resp.CreatedAt)
		})

		t.Run("正常系: 親ロール付きロール作成", func(t *testing.T) {
			// 親ロール作成
			parent := createRoleForRoleTest(t, db, "親ロール", nil)

			req := CreateRoleRequest{
				Name:     "子ロール",
				ParentID: &parent.ID,
			}

			resp, err := svc.CreateRole(req)
			require.NoError(t, err)
			assert.Equal(t, "子ロール", resp.Name)
			assert.NotNil(t, resp.ParentID)
			assert.Equal(t, parent.ID, *resp.ParentID)
		})
	})

	t.Run("Read操作", func(t *testing.T) {
		role := createRoleForRoleTest(t, db, "読み取りテストロール", nil)

		t.Run("正常系: 存在するロール取得", func(t *testing.T) {
			resp, err := svc.GetRole(role.ID)
			require.NoError(t, err)
			assert.Equal(t, role.ID, resp.ID)
			assert.Equal(t, "読み取りテストロール", resp.Name)
		})

		t.Run("異常系: 存在しないロール取得", func(t *testing.T) {
			nonExistentID := uuid.New()
			_, err := svc.GetRole(nonExistentID)
			assert.Error(t, err)
			assert.True(t, errors.IsNotFound(err))
		})
	})

	t.Run("Update操作", func(t *testing.T) {
		role := createRoleForRoleTest(t, db, "更新テストロール", nil)

		t.Run("正常系: ロール名更新", func(t *testing.T) {
			newName := "更新後ロール名"
			req := UpdateRoleRequest{
				Name: &newName,
			}

			resp, err := svc.UpdateRole(role.ID, req)
			require.NoError(t, err)
			assert.Equal(t, "更新後ロール名", resp.Name)
			assert.Equal(t, role.ID, resp.ID)
		})

		t.Run("正常系: 親ロール設定", func(t *testing.T) {
			parent := createRoleForRoleTest(t, db, "新しい親ロール", nil)
			req := UpdateRoleRequest{
				ParentID: &parent.ID,
			}

			resp, err := svc.UpdateRole(role.ID, req)
			require.NoError(t, err)
			assert.NotNil(t, resp.ParentID)
			assert.Equal(t, parent.ID, *resp.ParentID)
		})

		t.Run("異常系: 存在しないロール更新", func(t *testing.T) {
			nonExistentID := uuid.New()
			newName := "存在しないロール"
			req := UpdateRoleRequest{
				Name: &newName,
			}

			_, err := svc.UpdateRole(nonExistentID, req)
			assert.Error(t, err)
			assert.True(t, errors.IsNotFound(err))
		})
	})

	t.Run("Delete操作", func(t *testing.T) {
		t.Run("正常系: 基本的なロール削除", func(t *testing.T) {
			role := createRoleForRoleTest(t, db, "削除テストロール", nil)

			err := svc.DeleteRole(role.ID)
			require.NoError(t, err)

			// 削除確認
			_, err = svc.GetRole(role.ID)
			assert.Error(t, err)
			assert.True(t, errors.IsNotFound(err))
		})

		t.Run("異常系: 存在しないロール削除", func(t *testing.T) {
			nonExistentID := uuid.New()
			err := svc.DeleteRole(nonExistentID)
			assert.Error(t, err)
			assert.True(t, errors.IsNotFound(err))
		})
	})

	t.Run("List操作", func(t *testing.T) {
		// テストロールを複数作成
		role1 := createRoleForRoleTest(t, db, "リストテスト1", nil)
		_ = createRoleForRoleTest(t, db, "リストテスト2", nil)
		child1 := createRoleForRoleTest(t, db, "子ロール1", &role1.ID)

		t.Run("正常系: 全ロール取得", func(t *testing.T) {
			resp, err := svc.GetRoles(1, 10, nil, nil, "")
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(resp.Roles), 3) // 最低3つのロールが存在
			assert.GreaterOrEqual(t, resp.Total, int64(3))
		})

		t.Run("正常系: 親ロールフィルタ", func(t *testing.T) {
			resp, err := svc.GetRoles(1, 10, &role1.ID, nil, "")
			require.NoError(t, err)
			assert.Len(t, resp.Roles, 1)
			assert.Equal(t, child1.ID, resp.Roles[0].ID)
		})

		t.Run("正常系: 検索フィルタ", func(t *testing.T) {
			resp, err := svc.GetRoles(1, 10, nil, nil, "リストテスト")
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(resp.Roles), 2) // "リストテスト1", "リストテスト2"
		})

		t.Run("正常系: ページング", func(t *testing.T) {
			resp, err := svc.GetRoles(1, 1, nil, nil, "")
			require.NoError(t, err)
			assert.Len(t, resp.Roles, 1)
			assert.Equal(t, 1, resp.Page)
			assert.Equal(t, 1, resp.Limit)
		})
	})
}

// TestRoleService_PermissionManagement 権限管理のテスト
func TestRoleService_PermissionManagement(t *testing.T) {
	svc, db := setupTestRole(t)

	t.Run("権限割り当て操作", func(t *testing.T) {
		role := createRoleForRoleTest(t, db, "権限テストロール", nil)

		t.Run("正常系: 空の権限割り当て", func(t *testing.T) {
			req := AssignPermissionsRequest{
				PermissionIDs: []uuid.UUID{},
				Replace:       true,
			}

			resp, err := svc.AssignPermissions(role.ID, req)
			require.NoError(t, err)
			assert.Equal(t, role.ID, resp.RoleID)
			assert.Len(t, resp.DirectPermissions, 0)
		})

		t.Run("異常系: 存在しないロールに権限割り当て", func(t *testing.T) {
			nonExistentID := uuid.New()
			req := AssignPermissionsRequest{
				PermissionIDs: []uuid.UUID{},
				Replace:       true,
			}

			_, err := svc.AssignPermissions(nonExistentID, req)
			assert.Error(t, err)
			assert.True(t, errors.IsNotFound(err))
		})
	})

	t.Run("権限取得操作", func(t *testing.T) {
		role := createRoleForRoleTest(t, db, "権限取得テストロール", nil)

		t.Run("正常系: ロール権限取得", func(t *testing.T) {
			resp, err := svc.GetRolePermissions(role.ID)
			require.NoError(t, err)
			assert.Equal(t, role.ID, resp.RoleID)
			assert.Equal(t, "権限取得テストロール", resp.RoleName)
			assert.Len(t, resp.DirectPermissions, 0)
			assert.Len(t, resp.InheritedPermissions, 0)
			assert.Len(t, resp.AllPermissions, 0)
		})

		t.Run("異常系: 存在しないロールの権限取得", func(t *testing.T) {
			nonExistentID := uuid.New()
			_, err := svc.GetRolePermissions(nonExistentID)
			assert.Error(t, err)
			assert.True(t, errors.IsNotFound(err))
		})
	})

	t.Run("階層ツリー取得", func(t *testing.T) {
		// 階層構造作成
		root := createRoleForRoleTest(t, db, "ツリールート", nil)
		child := createRoleForRoleTest(t, db, "ツリー子", &root.ID)
		_ = createRoleForRoleTest(t, db, "ツリー孫", &child.ID)

		t.Run("正常系: 階層ツリー取得", func(t *testing.T) {
			resp, err := svc.GetRoleHierarchy()
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(resp.Roles), 1)

			// ルートロールを検索
			var rootNode *RoleHierarchyNode
			for i := range resp.Roles {
				if resp.Roles[i].Name == "ツリールート" {
					rootNode = &resp.Roles[i]
					break
				}
			}
			require.NotNil(t, rootNode)
			assert.Equal(t, 0, rootNode.Level)
			assert.GreaterOrEqual(t, len(rootNode.Children), 1)
		})
	})
}
