package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"erp-access-control-go/internal/services"
	"erp-access-control-go/pkg/logger"
)

// テスト用ヘルパー関数
func setupRoleIntegrationTest(t *testing.T) (*gin.Engine, *services.RoleService, *gorm.DB) {
	// SQLiteインメモリDBセットアップ
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// テーブル作成
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS roles (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
			name TEXT NOT NULL,
			parent_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (parent_id) REFERENCES roles(id)
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
			name TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			status TEXT DEFAULT 'active',
			primary_role_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (primary_role_id) REFERENCES roles(id)
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_roles (
			user_id TEXT NOT NULL,
			role_id TEXT NOT NULL,
			is_active BOOLEAN DEFAULT true,
			PRIMARY KEY (user_id, role_id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (role_id) REFERENCES roles(id)
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS permissions (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
			module TEXT NOT NULL,
			action TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS role_permissions (
			role_id TEXT NOT NULL,
			permission_id TEXT NOT NULL,
			PRIMARY KEY (role_id, permission_id),
			FOREIGN KEY (role_id) REFERENCES roles(id),
			FOREIGN KEY (permission_id) REFERENCES permissions(id)
		)
	`).Error
	require.NoError(t, err)

	// テストデータをクリア
	db.Exec("DELETE FROM role_permissions")
	db.Exec("DELETE FROM user_roles")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM permissions")
	db.Exec("DELETE FROM roles")

	// サービス・ハンドラーセットアップ
	appLogger := logger.NewLogger()
	roleService := services.NewRoleService(db, appLogger)
	roleHandler := NewRoleHandler(roleService, appLogger)

	// Ginセットアップ
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 認証モックミドルウェア
	router.Use(func(c *gin.Context) {
		// テスト用のユーザーIDを設定
		c.Set("user_id", "test-user-id")
		c.Next()
	})

	// ルート設定
	v1 := router.Group("/api/v1")
	roles := v1.Group("/roles")
	{
		roles.POST("", roleHandler.CreateRole)
		roles.GET("", roleHandler.GetRoles)
		roles.GET("/hierarchy", roleHandler.GetRoleHierarchy)
		roles.GET("/:id", roleHandler.GetRole)
		roles.PUT("/:id", roleHandler.UpdateRole)
		roles.DELETE("/:id", roleHandler.DeleteRole)
		roles.PUT("/:id/permissions", roleHandler.AssignPermissions)
		roles.GET("/:id/permissions", roleHandler.GetRolePermissions)
	}

	return router, roleService, db
}

func createTestRoleViaDB(t *testing.T, db *gorm.DB, name string, parentID *string) string {
	roleID := uuid.New().String()
	var query string
	var args []interface{}

	if parentID != nil {
		query = "INSERT INTO roles (id, name, parent_id) VALUES (?, ?, ?)"
		args = []interface{}{roleID, name, *parentID}
	} else {
		query = "INSERT INTO roles (id, name) VALUES (?, ?)"
		args = []interface{}{roleID, name}
	}

	err := db.Exec(query, args...).Error
	require.NoError(t, err)
	return roleID
}

// TestRoleHandler_CreateRole_Validation ロール作成バリデーションテスト
func TestRoleHandler_CreateRole_Validation(t *testing.T) {
	router, _, db := setupRoleIntegrationTest(t)

	t.Run("正常系: 基本的なロール作成", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "テストロール",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)

		var result map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "テストロール", result["name"])
		assert.NotEmpty(t, result["id"])
	})

	t.Run("正常系: 親ロール付きロール作成", func(t *testing.T) {
		// 親ロール作成
		parentID := createTestRoleViaDB(t, db, "親ロール", nil)

		reqBody := map[string]interface{}{
			"name":      "子ロール",
			"parent_id": parentID,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)

		var result map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "子ロール", result["name"])
		assert.Equal(t, parentID, result["parent_id"])
	})

	t.Run("異常系: 必須項目不足", func(t *testing.T) {
		reqBody := map[string]interface{}{} // nameが不足

		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("異常系: 存在しない親ロールID", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		reqBody := map[string]interface{}{
			"name":      "無効な子ロール",
			"parent_id": nonExistentID,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})

	t.Run("異常系: 重複ロール名", func(t *testing.T) {
		// 既存ロール作成
		createTestRoleViaDB(t, db, "重複テストロール", nil)

		reqBody := map[string]interface{}{
			"name": "重複テストロール",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("異常系: 無効なJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/roles", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

// TestRoleHandler_GetRoles_QueryParams クエリパラメータテスト
func TestRoleHandler_GetRoles_QueryParams(t *testing.T) {
	router, _, db := setupRoleIntegrationTest(t)

	// テストデータ作成
	role1ID := createTestRoleViaDB(t, db, "親ロール1", nil)
	role2ID := createTestRoleViaDB(t, db, "親ロール2", nil)
	child1ID := createTestRoleViaDB(t, db, "子ロール1", &role1ID)
	createTestRoleViaDB(t, db, "子ロール2", &role2ID)

	t.Run("正常系: 全ロール取得", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/roles", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var result map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, int(result["total"].(float64)), 4)
	})

	t.Run("正常系: ページングパラメータ", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/roles?page=1&limit=2", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var result map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, float64(1), result["page"])
		assert.Equal(t, float64(2), result["limit"])
		roles := result["roles"].([]interface{})
		assert.LessOrEqual(t, len(roles), 2)
	})

	t.Run("正常系: 親ロールフィルタ", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/roles?parent_id=%s", role1ID), nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var result map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		roles := result["roles"].([]interface{})
		assert.Len(t, roles, 1)

		role := roles[0].(map[string]interface{})
		assert.Equal(t, child1ID, role["id"])
	})

	t.Run("正常系: 検索パラメータ", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/roles?search=親ロール", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var result map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		roles := result["roles"].([]interface{})
		assert.GreaterOrEqual(t, len(roles), 2) // "親ロール1", "親ロール2"
	})

	t.Run("異常系: 無効なページ番号", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/roles?page=0", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("異常系: 無効なリミット", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/roles?limit=0", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("異常系: 無効なUUID形式", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/roles?parent_id=invalid-uuid", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

// TestRoleHandler_GetRole_PathParams パスパラメータテスト
func TestRoleHandler_GetRole_PathParams(t *testing.T) {
	router, _, db := setupRoleIntegrationTest(t)

	// テストデータ作成
	roleID := createTestRoleViaDB(t, db, "取得テストロール", nil)

	t.Run("正常系: 存在するロール取得", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/roles/%s", roleID), nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var result map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, roleID, result["id"])
		assert.Equal(t, "取得テストロール", result["name"])
	})

	t.Run("異常系: 存在しないロール", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/roles/%s", nonExistentID), nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})

	t.Run("異常系: 無効なUUID形式", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/roles/invalid-uuid", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

// TestRoleHandler_CRUD_Flow CRUD操作フロー
func TestRoleHandler_CRUD_Flow(t *testing.T) {
	router, _, _ := setupRoleIntegrationTest(t)

	var createdRoleID string

	t.Run("Step1: ロール作成", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "CRUD テストロール",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/roles", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)

		var result map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		createdRoleID = result["id"].(string)
		assert.Equal(t, "CRUD テストロール", result["name"])
	})

	t.Run("Step2: ロール取得", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/roles/%s", createdRoleID), nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var result map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, createdRoleID, result["id"])
		assert.Equal(t, "CRUD テストロール", result["name"])
	})

	t.Run("Step3: ロール更新", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "CRUD 更新後ロール",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/roles/%s", createdRoleID), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var result map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "CRUD 更新後ロール", result["name"])
	})

	t.Run("Step4: 権限管理", func(t *testing.T) {
		// 空の権限割り当て
		reqBody := map[string]interface{}{
			"permission_ids": []string{},
			"replace":        true,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/roles/%s/permissions", createdRoleID), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Step5: ロール削除", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/roles/%s", createdRoleID), nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		// 削除確認
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/roles/%s", createdRoleID), nil)
		resp = httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

// TestRoleHandler_GetRoleHierarchy 階層構造取得
func TestRoleHandler_GetRoleHierarchy(t *testing.T) {
	router, _, db := setupRoleIntegrationTest(t)

	// 階層構造作成
	rootID := createTestRoleViaDB(t, db, "ルートロール", nil)
	childID := createTestRoleViaDB(t, db, "子ロール", &rootID)
	createTestRoleViaDB(t, db, "孫ロール", &childID)

	t.Run("正常系: 階層ツリー取得", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/roles/hierarchy", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var result map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		roles := result["roles"].([]interface{})
		assert.GreaterOrEqual(t, len(roles), 1)

		// ルートロールを検索
		found := false
		for _, roleInterface := range roles {
			role := roleInterface.(map[string]interface{})
			if role["name"] == "ルートロール" {
				found = true
				assert.Equal(t, float64(0), role["level"])
				children := role["children"].([]interface{})
				assert.GreaterOrEqual(t, len(children), 1)
				break
			}
		}
		assert.True(t, found, "ルートロールが見つかりませんでした")
	})
}
