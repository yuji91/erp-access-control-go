package handlers

import (
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
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/pkg/logger"
)

// setupPermissionIntegrationTest 統合テスト環境セットアップ
func setupPermissionIntegrationTest(t *testing.T) (*gin.Engine, *services.PermissionService, *gorm.DB) {
	// SQLiteインメモリDBセットアップ
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// permissions テーブル作成
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS permissions (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
			module TEXT NOT NULL,
			action TEXT NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	// roles テーブル作成
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

	// role_permissions 中間テーブル作成
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

	// users テーブル作成
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

	// user_roles 中間テーブル作成
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

	// テストデータをクリア
	db.Exec("DELETE FROM role_permissions")
	db.Exec("DELETE FROM user_roles")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM permissions")
	db.Exec("DELETE FROM roles")

	// サービス・ハンドラーセットアップ
	appLogger := logger.NewLogger()
	permissionService := services.NewPermissionService(db, appLogger)
	permissionHandler := NewPermissionHandler(permissionService, appLogger)

	// Ginセットアップ
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// エラーハンドリングミドルウェア
	router.Use(func(c *gin.Context) {
		c.Next()

		// エラーハンドリング
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			if errors.IsValidationError(err) {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			} else if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			} else if errors.IsAuthenticationError(err) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			c.Abort()
		}
	})

	// 認証モックミドルウェア
	router.Use(func(c *gin.Context) {
		// テスト用のユーザーIDを設定（UUID型で）
		testUserID := uuid.New()
		c.Set("user_id", testUserID)
		c.Next()
	})

	// ルート設定
	v1 := router.Group("/api/v1")
	permissions := v1.Group("/permissions")
	{
		permissions.POST("", permissionHandler.CreatePermission)
		permissions.GET("", permissionHandler.GetPermissions)
		permissions.GET("/matrix", permissionHandler.GetPermissionMatrix)
		permissions.GET("/modules/:module", permissionHandler.GetPermissionsByModule)
		permissions.GET("/:id", permissionHandler.GetPermission)
		permissions.PUT("/:id", permissionHandler.UpdatePermission)
		permissions.DELETE("/:id", permissionHandler.DeletePermission)
		permissions.GET("/:id/roles", permissionHandler.GetRolesByPermission)
	}

	return router, permissionService, db
}

// createPermissionForPermissionIntegrationTest テスト用権限作成ヘルパー
func createPermissionForPermissionIntegrationTest(t *testing.T, db *gorm.DB, module, action string) string {
	permissionID := uuid.New().String()
	err := db.Exec("INSERT INTO permissions (id, module, action, description, created_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)",
		permissionID, module, action, fmt.Sprintf("%s %s permission", module, action)).Error
	require.NoError(t, err)
	return permissionID
}

// createRoleForPermissionIntegrationTest テスト用ロール作成ヘルパー
func createRoleForPermissionIntegrationTest(t *testing.T, db *gorm.DB, name string, parentID *string) string {
	roleID := uuid.New().String()
	var query string
	var args []interface{}

	if parentID != nil {
		query = "INSERT INTO roles (id, name, parent_id, created_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)"
		args = []interface{}{roleID, name, *parentID}
	} else {
		query = "INSERT INTO roles (id, name, created_at) VALUES (?, ?, CURRENT_TIMESTAMP)"
		args = []interface{}{roleID, name}
	}

	err := db.Exec(query, args...).Error
	require.NoError(t, err)
	return roleID
}

// assignPermissionToRoleForIntegrationTest 権限をロールに割り当て
func assignPermissionToRoleForIntegrationTest(t *testing.T, db *gorm.DB, roleID, permissionID string) {
	err := db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)",
		roleID, permissionID).Error
	require.NoError(t, err)
}

// TestPermissionHandler_CreatePermission_Validation 権限作成バリデーションテスト
func TestPermissionHandler_CreatePermission_Validation(t *testing.T) {
	router, _, _ := setupPermissionIntegrationTest(t)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "正常系: 基本権限作成",
			requestBody: `{
				"module": "user",
				"action": "create",
				"description": "ユーザー作成権限"
			}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "正常系: 説明なし権限作成",
			requestBody: `{
				"module": "inventory",
				"action": "read"
			}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "異常系: 必須項目不足 - module",
			requestBody: `{
				"action": "create"
			}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "異常系: 必須項目不足 - action",
			requestBody: `{
				"module": "user"
			}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "異常系: 無効JSON",
			requestBody: `{
				"module": "user",
				"action": "create"
			`,
			expectedStatus: http.StatusBadRequest,
		},

	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/api/v1/permissions", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Test case %d: %s", i+1, tt.name)

			if tt.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "permission")
			}
		})
	}

	// 重複権限作成テスト（分離実装）
	t.Run("異常系: 重複権限作成", func(t *testing.T) {
		// 最初に権限を作成
		firstReq := `{
			"module": "order",
			"action": "create"
		}`
		req, _ := http.NewRequest("POST", "/api/v1/permissions", strings.NewReader(firstReq))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// 同じ権限を再度作成して重複エラーを確認
		req, _ = http.NewRequest("POST", "/api/v1/permissions", strings.NewReader(firstReq))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestPermissionHandler_GetPermissions_QueryParams クエリパラメータテスト
func TestPermissionHandler_GetPermissions_QueryParams(t *testing.T) {
	router, _, db := setupPermissionIntegrationTest(t)

	// テストデータ準備
	perm1 := createPermissionForPermissionIntegrationTest(t, db, "user", "create")
	_ = createPermissionForPermissionIntegrationTest(t, db, "user", "read")
	_ = createPermissionForPermissionIntegrationTest(t, db, "department", "read")
	role1 := createRoleForPermissionIntegrationTest(t, db, "管理者", nil)
	assignPermissionToRoleForIntegrationTest(t, db, role1, perm1)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "正常系: 全権限取得",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedCount:  3,
		},
		{
			name:           "正常系: ページング",
			queryParams:    "?page=1&limit=2",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "正常系: モジュールフィルタ",
			queryParams:    "?module=user",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "正常系: アクションフィルタ",
			queryParams:    "?action=read",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "正常系: ロール使用フィルタ",
			queryParams:    fmt.Sprintf("?used_by_role=%s", role1),
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "正常系: 検索フィルタ",
			queryParams:    "?search=user",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "異常系: 無効ページ",
			queryParams:    "?page=-1",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "異常系: 無効リミット",
			queryParams:    "?limit=101",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "異常系: 無効UUID",
			queryParams:    "?used_by_role=invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/permissions"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Test case %d: %s", i+1, tt.name)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				permissions, ok := response["permissions"].([]interface{})
				assert.True(t, ok)
				assert.Equal(t, tt.expectedCount, len(permissions), "Test case %d: %s", i+1, tt.name)
			}
		})
	}
}

// TestPermissionHandler_GetPermission_PathParams パスパラメータテスト
func TestPermissionHandler_GetPermission_PathParams(t *testing.T) {
	router, _, db := setupPermissionIntegrationTest(t)

	// テストデータ準備
	permissionID := createPermissionForPermissionIntegrationTest(t, db, "user", "read")

	tests := []struct {
		name           string
		permissionID   string
		expectedStatus int
	}{
		{
			name:           "正常系: 存在する権限取得",
			permissionID:   permissionID,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "異常系: 存在しない権限",
			permissionID:   uuid.New().String(),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "異常系: 無効UUID",
			permissionID:   "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/permissions/"+tt.permissionID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Test case %d: %s", i+1, tt.name)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "permission")
			}
		})
	}
}

// TestPermissionHandler_CRUD_Flow CRUD操作フローテスト
func TestPermissionHandler_CRUD_Flow(t *testing.T) {
	router, _, _ := setupPermissionIntegrationTest(t)

	// Step 1: 権限作成
	createReq := `{
		"module": "test",
		"action": "create",
		"description": "テスト権限"
	}`
	req, _ := http.NewRequest("POST", "/api/v1/permissions", strings.NewReader(createReq))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var createResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &createResponse)
	require.NoError(t, err)

	permission := createResponse["permission"].(map[string]interface{})
	permissionID := permission["id"].(string)

	// Step 2: 権限詳細取得
	req, _ = http.NewRequest("GET", "/api/v1/permissions/"+permissionID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Step 3: 権限更新
	updateReq := `{
		"description": "更新されたテスト権限"
	}`
	req, _ = http.NewRequest("PUT", "/api/v1/permissions/"+permissionID, strings.NewReader(updateReq))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Step 4: 権限削除
	req, _ = http.NewRequest("DELETE", "/api/v1/permissions/"+permissionID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Step 5: 削除確認
	req, _ = http.NewRequest("GET", "/api/v1/permissions/"+permissionID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestPermissionHandler_GetPermissionMatrix マトリックス取得テスト
func TestPermissionHandler_GetPermissionMatrix(t *testing.T) {
	router, _, db := setupPermissionIntegrationTest(t)

	// テストデータ準備
	createPermissionForPermissionIntegrationTest(t, db, "user", "create")
	createPermissionForPermissionIntegrationTest(t, db, "user", "read")
	createPermissionForPermissionIntegrationTest(t, db, "department", "read")

	req, _ := http.NewRequest("GET", "/api/v1/permissions/matrix", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "modules")
	assert.Contains(t, response, "summary")

	modules := response["modules"].([]interface{})
	assert.GreaterOrEqual(t, len(modules), 2) // user, department モジュール
}

// TestPermissionHandler_GetPermissionsByModule モジュール別権限取得テスト
func TestPermissionHandler_GetPermissionsByModule(t *testing.T) {
	router, _, db := setupPermissionIntegrationTest(t)

	// テストデータ準備
	createPermissionForPermissionIntegrationTest(t, db, "user", "create")
	createPermissionForPermissionIntegrationTest(t, db, "user", "read")
	createPermissionForPermissionIntegrationTest(t, db, "department", "read")

	tests := []struct {
		name           string
		module         string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "正常系: userモジュール権限取得",
			module:         "user",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "正常系: departmentモジュール権限取得",
			module:         "department",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "正常系: 存在しないモジュール",
			module:         "nonexistent",
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/permissions/modules/"+tt.module, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Test case %d: %s", i+1, tt.name)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "permissions")

				permissions := response["permissions"].([]interface{})
				assert.Equal(t, tt.expectedCount, len(permissions), "Test case %d: %s", i+1, tt.name)
			}
		})
	}
}

// TestPermissionHandler_GetRolesByPermission 権限ロール取得テスト
func TestPermissionHandler_GetRolesByPermission(t *testing.T) {
	router, _, db := setupPermissionIntegrationTest(t)

	// テストデータ準備
	permissionID := createPermissionForPermissionIntegrationTest(t, db, "user", "read")
	role1 := createRoleForPermissionIntegrationTest(t, db, "管理者", nil)
	role2 := createRoleForPermissionIntegrationTest(t, db, "一般ユーザー", nil)
	assignPermissionToRoleForIntegrationTest(t, db, role1, permissionID)
	assignPermissionToRoleForIntegrationTest(t, db, role2, permissionID)

	req, _ := http.NewRequest("GET", "/api/v1/permissions/"+permissionID+"/roles", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "roles")

	roles := response["roles"].([]interface{})
	assert.Equal(t, 2, len(roles)) // 2つのロールが権限を持つ
}
