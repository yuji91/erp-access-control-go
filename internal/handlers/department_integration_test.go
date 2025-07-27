package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// setupTestDepartmentHandler テスト用ハンドラーセットアップ
func setupTestDepartmentHandler(t *testing.T) (*DepartmentHandler, *gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)

	// テスト用SQLiteデータベース
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// テーブル作成
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS departments (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-' || '4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
			name TEXT NOT NULL,
			parent_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (parent_id) REFERENCES departments(id)
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-' || '4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
			name TEXT NOT NULL,
			department_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (department_id) REFERENCES departments(id)
		)
	`).Error
	require.NoError(t, err)

	// テストロガー
	testLogger := logger.NewLogger(
		logger.WithMinLevel(logger.DEBUG),
		logger.WithEnvironment("test"),
	)

	// サービス・ハンドラー作成
	departmentService := services.NewDepartmentService(db, testLogger)
	handler := NewDepartmentHandler(departmentService, testLogger)

	// ルーター設定
	router := gin.New()
	router.Use(gin.Recovery())

	// エラーハンドリングミドルウェア
	router.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			var statusCode int
			var message string

			switch {
			case errors.IsValidationError(err):
				statusCode = http.StatusBadRequest
				message = err.Error()
			case errors.IsNotFound(err):
				statusCode = http.StatusNotFound
				message = err.Error()
			case errors.IsAuthenticationError(err):
				statusCode = http.StatusUnauthorized
				message = err.Error()
			default:
				statusCode = http.StatusInternalServerError
				message = "Internal Server Error"
			}

			c.JSON(statusCode, gin.H{"error": message})
		}
	})

	// 認証ミドルウェアのモック（テスト用ユーザーID設定）
	router.Use(func(c *gin.Context) {
		testUserID := uuid.New()
		c.Set("user_id", testUserID)
		c.Next()
	})

	return handler, router, db
}

// TestDepartmentHandler_CreateDepartment_Validation 部署作成バリデーションテスト
func TestDepartmentHandler_CreateDepartment_Validation(t *testing.T) {
	handler, router, db := setupTestDepartmentHandler(t)

	// ルート設定
	router.POST("/departments", handler.CreateDepartment)

	// テストデータクリア
	db.Exec("DELETE FROM departments")
	
	// 親部署作成
	parentID := uuid.New()
	db.Exec("INSERT INTO departments (id, name) VALUES (?, ?)", parentID.String(), "親部署")

	testCases := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectError    bool
	}{
		{
			name: "正常系: 親部署なしで作成",
			requestBody: map[string]interface{}{
				"name": "営業部",
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name: "正常系: 親部署ありで作成",
			requestBody: map[string]interface{}{
				"name":      "東京営業所",
				"parent_id": parentID.String(),
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name: "異常系: 名前未指定",
			requestBody: map[string]interface{}{
				"parent_id": uuid.New().String(),
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "異常系: 無効なUUID形式",
			requestBody: map[string]interface{}{
				"name":      "無効部署",
				"parent_id": "invalid-uuid",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "異常系: 空リクエスト",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// リクエスト作成
			reqBody, _ := json.Marshal(tc.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/departments", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			// レスポンス記録
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// ステータスコード検証
			assert.Equal(t, tc.expectedStatus, w.Code)

			// エラーレスポンス検証
			if tc.expectError {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			} else {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "id")
				assert.Contains(t, response, "name")
			}
		})
	}
}

// TestDepartmentHandler_GetDepartments_QueryParams 部署一覧取得クエリパラメータテスト
func TestDepartmentHandler_GetDepartments_QueryParams(t *testing.T) {
	handler, router, db := setupTestDepartmentHandler(t)

	// ルート設定
	router.GET("/departments", handler.GetDepartments)

	// テストデータ作成
	db.Exec("DELETE FROM departments")
	rootID := uuid.New()
	childID := uuid.New()
	db.Exec("INSERT INTO departments (id, name) VALUES (?, ?)", rootID.String(), "本社")
	db.Exec("INSERT INTO departments (id, name, parent_id) VALUES (?, ?, ?)", childID.String(), "営業部", rootID.String())

	testCases := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "正常系: デフォルトパラメータ",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "正常系: ページング指定",
			queryParams:    "?page=1&limit=5",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "正常系: 親部署フィルタ",
			queryParams:    fmt.Sprintf("?parent_id=%s", rootID.String()),
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "正常系: 名前検索",
			queryParams:    "?search=営業",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "異常系: 無効なページ番号",
			queryParams:    "?page=0",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "異常系: 無効なリミット",
			queryParams:    "?limit=0",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "異常系: リミット超過",
			queryParams:    "?limit=101",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "異常系: 無効なUUID",
			queryParams:    "?parent_id=invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// リクエスト作成
			req := httptest.NewRequest(http.MethodGet, "/departments"+tc.queryParams, nil)

			// レスポンス記録
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// ステータスコード検証
			assert.Equal(t, tc.expectedStatus, w.Code)

			// レスポンス内容検証
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tc.expectError {
				assert.Contains(t, response, "error")
			} else {
				assert.Contains(t, response, "departments")
				assert.Contains(t, response, "total")
				assert.Contains(t, response, "page")
				assert.Contains(t, response, "limit")
			}
		})
	}
}

// TestDepartmentHandler_GetDepartment_PathParams 部署詳細取得パスパラメータテスト
func TestDepartmentHandler_GetDepartment_PathParams(t *testing.T) {
	handler, router, db := setupTestDepartmentHandler(t)

	// ルート設定
	router.GET("/departments/:id", handler.GetDepartment)

	// テストデータ作成
	db.Exec("DELETE FROM departments")
	testID := uuid.New()
	db.Exec("INSERT INTO departments (id, name) VALUES (?, ?)", testID.String(), "テスト部署")

	testCases := []struct {
		name           string
		departmentID   string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "正常系: 存在する部署ID",
			departmentID:   testID.String(),
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "異常系: 存在しない部署ID",
			departmentID:   uuid.New().String(),
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "異常系: 無効なUUID形式",
			departmentID:   "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// リクエスト作成
			req := httptest.NewRequest(http.MethodGet, "/departments/"+tc.departmentID, nil)

			// レスポンス記録
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// ステータスコード検証
			assert.Equal(t, tc.expectedStatus, w.Code)

			// レスポンス内容検証
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tc.expectError {
				assert.Contains(t, response, "error")
			} else {
				assert.Contains(t, response, "id")
				assert.Contains(t, response, "name")
				assert.Equal(t, testID.String(), response["id"])
				assert.Equal(t, "テスト部署", response["name"])
			}
		})
	}
}

// TestDepartmentHandler_CRUD_Flow CRUD操作フローテスト
func TestDepartmentHandler_CRUD_Flow(t *testing.T) {
	handler, router, db := setupTestDepartmentHandler(t)

	// ルート設定
	router.POST("/departments", handler.CreateDepartment)
	router.GET("/departments/:id", handler.GetDepartment)
	router.PUT("/departments/:id", handler.UpdateDepartment)
	router.DELETE("/departments/:id", handler.DeleteDepartment)

	// データクリア
	db.Exec("DELETE FROM departments")

	var createdID string

	// 1. 部署作成
	t.Run("Create Department", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "新規部署",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/departments", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "id")
		assert.Equal(t, "新規部署", response["name"])

		createdID = response["id"].(string)
	})

	// 2. 部署取得
	t.Run("Get Department", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/departments/"+createdID, nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, createdID, response["id"])
		assert.Equal(t, "新規部署", response["name"])
	})

	// 3. 部署更新
	t.Run("Update Department", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "更新された部署",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/departments/"+createdID, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, createdID, response["id"])
		assert.Equal(t, "更新された部署", response["name"])
	})

	// 4. 部署削除
	t.Run("Delete Department", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/departments/"+createdID, nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	// 5. 削除後取得確認
	t.Run("Get Deleted Department", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/departments/"+createdID, nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestDepartmentHandler_GetDepartmentHierarchy 階層構造取得テスト
func TestDepartmentHandler_GetDepartmentHierarchy(t *testing.T) {
	handler, router, db := setupTestDepartmentHandler(t)

	// ルート設定
	router.GET("/departments/hierarchy", handler.GetDepartmentHierarchy)

	// テストデータ作成
	db.Exec("DELETE FROM departments")
	rootID := uuid.New()
	childID := uuid.New()
	db.Exec("INSERT INTO departments (id, name) VALUES (?, ?)", rootID.String(), "本社")
	db.Exec("INSERT INTO departments (id, name, parent_id) VALUES (?, ?, ?)", childID.String(), "営業部", rootID.String())

	t.Run("正常系: 階層構造取得", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/departments/hierarchy", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "departments")

		departments := response["departments"].([]interface{})
		assert.GreaterOrEqual(t, len(departments), 1)
	})
}
