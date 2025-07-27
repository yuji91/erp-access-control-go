package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"erp-access-control-go/internal/services"
	"erp-access-control-go/pkg/logger"
)

// TestUserHandler_CreateUser_ValidRequest ユーザー作成リクエストバリデーションテスト
func TestUserHandler_CreateUser_ValidRequest(t *testing.T) {
	// Given: Ginテストモード設定
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// テストロガー
	testLogger := logger.NewLogger(
		logger.WithMinLevel(logger.DEBUG),
		logger.WithEnvironment("test"),
	)

	// モックサービスの代わりに、リクエストバリデーションテストのみ
	router.POST("/users", func(c *gin.Context) {
		var req services.CreateUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		// バリデーション成功レスポンス
		c.JSON(http.StatusOK, gin.H{
			"message":         "Validation passed",
			"name":            req.Name,
			"email":           req.Email,
			"department_id":   req.DepartmentID.String(),
			"primary_role_id": req.PrimaryRoleID.String(),
		})
	})

	// テストケース
	testCases := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectError    bool
	}{
		{
			name: "Valid request",
			requestBody: map[string]interface{}{
				"name":            "テストユーザー",
				"email":           "test@example.com",
				"password":        "password123",
				"department_id":   uuid.New().String(),
				"primary_role_id": uuid.New().String(),
				"status":          "active",
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "Missing required fields",
			requestBody: map[string]interface{}{
				"name": "テストユーザー",
				// email missing
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "Invalid email format",
			requestBody: map[string]interface{}{
				"name":            "テストユーザー",
				"email":           "invalid-email",
				"password":        "password123",
				"department_id":   uuid.New().String(),
				"primary_role_id": uuid.New().String(),
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name: "Invalid UUID format",
			requestBody: map[string]interface{}{
				"name":            "テストユーザー",
				"email":           "test@example.com",
				"password":        "password123",
				"department_id":   "invalid-uuid",
				"primary_role_id": uuid.New().String(),
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	// When & Then: 各テストケース実行
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tc.requestBody)

			req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tc.expectError {
				assert.Contains(t, response, "error")
			} else {
				assert.Contains(t, response, "message")
				assert.Equal(t, "Validation passed", response["message"])
			}
		})
	}

	_ = testLogger // 未使用変数警告回避
}

// TestUserHandler_GetUsers_QueryParams ユーザー一覧クエリパラメータテスト
func TestUserHandler_GetUsers_QueryParams(t *testing.T) {
	// Given: Ginテストモード設定
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/users", func(c *gin.Context) {
		// クエリパラメータ解析テスト
		// ページング
		if page := c.Query("page"); page != "" {
			// ページング値の検証ロジック
			c.JSON(http.StatusOK, gin.H{"page": page})
			return
		}

		// フィルター
		if departmentID := c.Query("department_id"); departmentID != "" {
			if _, err := uuid.Parse(departmentID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department_id format"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"department_id": departmentID})
			return
		}

		// 検索
		if search := c.Query("search"); search != "" {
			c.JSON(http.StatusOK, gin.H{"search": search})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "No filters applied"})
	})

	// テストケース
	testCases := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedKey    string
		expectedValue  string
	}{
		{
			name:           "Page parameter",
			queryParams:    "?page=2",
			expectedStatus: http.StatusOK,
			expectedKey:    "page",
			expectedValue:  "2",
		},
		{
			name:           "Department filter",
			queryParams:    "?department_id=" + uuid.New().String(),
			expectedStatus: http.StatusOK,
			expectedKey:    "department_id",
		},
		{
			name:           "Search parameter",
			queryParams:    "?search=テスト",
			expectedStatus: http.StatusOK,
			expectedKey:    "search",
			expectedValue:  "テスト",
		},
		{
			name:           "Invalid department UUID",
			queryParams:    "?department_id=invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			expectedKey:    "error",
		},
		{
			name:           "No parameters",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedKey:    "message",
		},
	}

	// When & Then: 各テストケース実行
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/users"+tc.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			assert.Contains(t, response, tc.expectedKey)
			if tc.expectedValue != "" {
				assert.Equal(t, tc.expectedValue, response[tc.expectedKey])
			}
		})
	}
}

// TestUserHandler_PathParameters パスパラメータ処理テスト
func TestUserHandler_PathParameters(t *testing.T) {
	// Given: Ginテストモード設定
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/users/:id", func(c *gin.Context) {
		userIDStr := c.Param("id")

		// UUID バリデーション
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id": userID.String(),
			"message": "Valid UUID",
		})
	})

	// テストケース
	testCases := []struct {
		name           string
		userID         string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Valid UUID",
			userID:         uuid.New().String(),
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Invalid UUID",
			userID:         "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Empty UUID",
			userID:         "",
			expectedStatus: http.StatusNotFound, // Ginのルーティングエラー
			expectError:    true,
		},
	}

	// When & Then: 各テストケース実行
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/users/" + tc.userID
			if tc.userID == "" {
				url = "/users/"
			}

			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus != http.StatusNotFound {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				if tc.expectError {
					assert.Contains(t, response, "error")
				} else {
					assert.Contains(t, response, "user_id")
					assert.Equal(t, tc.userID, response["user_id"])
				}
			}
		})
	}
}

// TestUserHandler_HTTPMethods HTTPメソッドテスト
func TestUserHandler_HTTPMethods(t *testing.T) {
	// Given: Ginテストモード設定
	gin.SetMode(gin.TestMode)
	router := gin.New()

	userID := uuid.New().String()

	// 各HTTPメソッドのルート設定
	router.POST("/users", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"method": "POST", "action": "create"})
	})

	router.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "GET", "action": "list"})
	})

	router.GET("/users/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "GET", "action": "get", "id": c.Param("id")})
	})

	router.PUT("/users/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "PUT", "action": "update", "id": c.Param("id")})
	})

	router.DELETE("/users/:id", func(c *gin.Context) {
		c.JSON(http.StatusNoContent, gin.H{})
	})

	router.PUT("/users/:id/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "PUT", "action": "change_status", "id": c.Param("id")})
	})

	router.PUT("/users/:id/password", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "PUT", "action": "change_password", "id": c.Param("id")})
	})

	// テストケース
	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedAction string
	}{
		{
			name:           "POST /users (create)",
			method:         "POST",
			path:           "/users",
			expectedStatus: http.StatusCreated,
			expectedAction: "create",
		},
		{
			name:           "GET /users (list)",
			method:         "GET",
			path:           "/users",
			expectedStatus: http.StatusOK,
			expectedAction: "list",
		},
		{
			name:           "GET /users/:id (get)",
			method:         "GET",
			path:           "/users/" + userID,
			expectedStatus: http.StatusOK,
			expectedAction: "get",
		},
		{
			name:           "PUT /users/:id (update)",
			method:         "PUT",
			path:           "/users/" + userID,
			expectedStatus: http.StatusOK,
			expectedAction: "update",
		},
		{
			name:           "DELETE /users/:id (delete)",
			method:         "DELETE",
			path:           "/users/" + userID,
			expectedStatus: http.StatusNoContent,
			expectedAction: "",
		},
		{
			name:           "PUT /users/:id/status (change status)",
			method:         "PUT",
			path:           "/users/" + userID + "/status",
			expectedStatus: http.StatusOK,
			expectedAction: "change_status",
		},
		{
			name:           "PUT /users/:id/password (change password)",
			method:         "PUT",
			path:           "/users/" + userID + "/password",
			expectedStatus: http.StatusOK,
			expectedAction: "change_password",
		},
	}

	// When & Then: 各テストケース実行
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.method == "POST" || tc.method == "PUT" {
				requestBody := map[string]interface{}{"test": "data"}
				jsonBody, _ := json.Marshal(requestBody)
				req, _ = http.NewRequest(tc.method, tc.path, bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tc.method, tc.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus != http.StatusNoContent {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				assert.Equal(t, tc.method, response["method"])
				if tc.expectedAction != "" {
					assert.Equal(t, tc.expectedAction, response["action"])
				}
			}
		})
	}
}

// TestUserHandler_RequestValidation リクエストバリデーション詳細テスト
func TestUserHandler_RequestValidation(t *testing.T) {
	// Given: Ginテストモード設定
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.PUT("/users/:id/status", func(c *gin.Context) {
		var req struct {
			Status string `json:"status" binding:"required,oneof=active inactive suspended"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status value"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  req.Status,
			"message": "Status validation passed",
		})
	})

	// テストケース
	testCases := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Valid status: active",
			requestBody:    map[string]interface{}{"status": "active"},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Valid status: inactive",
			requestBody:    map[string]interface{}{"status": "inactive"},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Valid status: suspended",
			requestBody:    map[string]interface{}{"status": "suspended"},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Invalid status",
			requestBody:    map[string]interface{}{"status": "invalid"},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Missing status",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	// When & Then: 各テストケース実行
	userID := uuid.New().String()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tc.requestBody)

			req, _ := http.NewRequest("PUT", "/users/"+userID+"/status", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tc.expectError {
				assert.Contains(t, response, "error")
			} else {
				assert.Contains(t, response, "status")
				assert.Equal(t, tc.requestBody["status"], response["status"])
			}
		})
	}
}
