package handlers

import (
	"encoding/json"
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

// setupSimplePermissionTest シンプルなテスト環境セットアップ
func setupSimplePermissionTest(t *testing.T) (*gin.Engine, *services.PermissionService, *gorm.DB) {
	// SQLiteインメモリDBセットアップ
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// permissions テーブル作成（単体テストと同じ構造）
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS permissions (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			module TEXT NOT NULL,
			action TEXT NOT NULL
		)
	`).Error
	require.NoError(t, err)

	// roles テーブル作成
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS roles (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			name TEXT NOT NULL,
			parent_id TEXT
		)
	`).Error
	require.NoError(t, err)

	// role_permissions 中間テーブル作成
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS role_permissions (
			role_id TEXT NOT NULL,
			permission_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (role_id, permission_id)
		)
	`).Error
	require.NoError(t, err)

	// user_roles 中間テーブル作成
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_roles (
			user_id TEXT NOT NULL,
			role_id TEXT NOT NULL,
			is_active BOOLEAN DEFAULT true,
			PRIMARY KEY (user_id, role_id)
		)
	`).Error
	require.NoError(t, err)

	// テストデータをクリア
	db.Exec("DELETE FROM role_permissions")
	db.Exec("DELETE FROM user_roles")
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
		permissions.GET("/:id", permissionHandler.GetPermission)
		permissions.GET("/matrix", permissionHandler.GetPermissionMatrix)
	}

	return router, permissionService, db
}

// TestPermissionHandler_Simple_CreateAndGet 基本的な作成・取得テスト
func TestPermissionHandler_Simple_CreateAndGet(t *testing.T) {
	router, _, _ := setupSimplePermissionTest(t)

	// Step 1: 権限作成
	createReq := `{
		"module": "inventory",
		"action": "create"
	}`
	req, _ := http.NewRequest("POST", "/api/v1/permissions", strings.NewReader(createReq))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var createResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &createResponse)
	require.NoError(t, err)
	assert.Contains(t, createResponse, "permission")

	permission := createResponse["permission"].(map[string]interface{})
	permissionID := permission["id"].(string)
	assert.Equal(t, "inventory", permission["module"])
	assert.Equal(t, "create", permission["action"])

	// Step 2: 権限詳細取得
	req, _ = http.NewRequest("GET", "/api/v1/permissions/"+permissionID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var getResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &getResponse)
	require.NoError(t, err)
	assert.Contains(t, getResponse, "permission")

	// Step 3: 権限一覧取得
	req, _ = http.NewRequest("GET", "/api/v1/permissions", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var listResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(t, err)
	assert.Contains(t, listResponse, "permissions")

	permissions := listResponse["permissions"].([]interface{})
	assert.Equal(t, 1, len(permissions))
}

// TestPermissionHandler_Simple_ValidationErrors バリデーションエラーテスト
func TestPermissionHandler_Simple_ValidationErrors(t *testing.T) {
	router, _, _ := setupSimplePermissionTest(t)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
	}{
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
				"module": "inventory"
			}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "異常系: 無効JSON",
			requestBody: `{
				"module": "inventory",
				"action": "create"
			`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/api/v1/permissions", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestPermissionHandler_Simple_Matrix マトリックス取得テスト
func TestPermissionHandler_Simple_Matrix(t *testing.T) {
	router, _, _ := setupSimplePermissionTest(t)

	req, _ := http.NewRequest("GET", "/api/v1/permissions/matrix", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "modules")
	assert.Contains(t, response, "summary")
}
