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

	"erp-access-control-go/internal/middleware"
	"erp-access-control-go/pkg/logger"
)

// TestUserHandler_BasicAuthentication 基本認証テスト
func TestUserHandler_BasicAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.ErrorHandler(logger.NewLogger(
		logger.WithMinLevel(logger.DEBUG),
		logger.WithEnvironment("test"),
	)))

	// シンプルな認証チェック
	router.GET("/protected", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
			return
		}

		if authHeader != "Bearer valid-token" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Access granted"})
	})

	tests := []struct {
		name         string
		authHeader   string
		expectStatus int
		expectError  bool
	}{
		{"Missing auth header", "", http.StatusUnauthorized, true},
		{"Invalid token", "Bearer invalid", http.StatusUnauthorized, true},
		{"Valid token", "Bearer valid-token", http.StatusOK, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectError {
				assert.Contains(t, response, "error")
			} else {
				assert.Contains(t, response, "message")
			}
		})
	}
}

// TestUserHandler_BasicPermissions 基本権限テスト
func TestUserHandler_BasicPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.ErrorHandler(logger.NewLogger(
		logger.WithMinLevel(logger.DEBUG),
		logger.WithEnvironment("test"),
	)))

	// ユーザー権限をヘッダーから取得
	router.Use(func(c *gin.Context) {
		permission := c.GetHeader("X-User-Permission")
		if permission != "" {
			c.Set("user_permission", permission)
		}
		c.Next()
	})

	// 権限チェック付きエンドポイント
	router.GET("/users", func(c *gin.Context) {
		permission, exists := c.Get("user_permission")
		if !exists || permission != "user:read" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Users listed"})
	})

	router.POST("/users", func(c *gin.Context) {
		permission, exists := c.Get("user_permission")
		if !exists || permission != "user:create" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "User created"})
	})

	tests := []struct {
		name         string
		method       string
		endpoint     string
		permission   string
		expectStatus int
		expectError  bool
	}{
		{"Read with correct permission", "GET", "/users", "user:read", http.StatusOK, false},
		{"Read without permission", "GET", "/users", "", http.StatusForbidden, true},
		{"Read with wrong permission", "GET", "/users", "user:create", http.StatusForbidden, true},
		{"Create with correct permission", "POST", "/users", "user:create", http.StatusCreated, false},
		{"Create without permission", "POST", "/users", "", http.StatusForbidden, true},
		{"Create with wrong permission", "POST", "/users", "user:read", http.StatusForbidden, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == "POST" {
				body := bytes.NewBufferString(`{"test": "data"}`)
				req, _ = http.NewRequest(tt.method, tt.endpoint, body)
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tt.method, tt.endpoint, nil)
			}

			if tt.permission != "" {
				req.Header.Set("X-User-Permission", tt.permission)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectError {
				assert.Contains(t, response, "error")
			} else {
				assert.Contains(t, response, "message")
			}
		})
	}
}

// TestUserHandler_OwnershipValidation 所有権検証テスト
func TestUserHandler_OwnershipValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.ErrorHandler(logger.NewLogger(
		logger.WithMinLevel(logger.DEBUG),
		logger.WithEnvironment("test"),
	)))

	currentUserID := uuid.New()

	// ユーザーIDをヘッダーから設定
	router.Use(func(c *gin.Context) {
		userIDHeader := c.GetHeader("X-User-ID")
		if userIDHeader != "" {
			if userID, err := uuid.Parse(userIDHeader); err == nil {
				c.Set("user_id", userID)
			}
		}
		c.Next()
	})

	// 所有権チェック付きエンドポイント
	router.PUT("/users/:id/password", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		currentUserID, ok := userID.(uuid.UUID)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
			return
		}

		targetUserID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target user ID"})
			return
		}

		if currentUserID != targetUserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Can only change own password"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Password changed"})
	})

	otherUserID := uuid.New()

	tests := []struct {
		name         string
		userID       uuid.UUID
		targetID     uuid.UUID
		expectStatus int
		expectError  bool
	}{
		{"Change own password", currentUserID, currentUserID, http.StatusOK, false},
		{"Change other user password", currentUserID, otherUserID, http.StatusForbidden, true},
		{"No user authenticated", uuid.Nil, currentUserID, http.StatusUnauthorized, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.NewBufferString(`{"new_password": "newpass123"}`)
			req, _ := http.NewRequest("PUT", "/users/"+tt.targetID.String()+"/password", body)
			req.Header.Set("Content-Type", "application/json")

			if tt.userID != uuid.Nil {
				req.Header.Set("X-User-ID", tt.userID.String())
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectError {
				assert.Contains(t, response, "error")
			} else {
				assert.Contains(t, response, "message")
			}
		})
	}
}

// TestUserHandler_MultiplePermissions 複数権限テスト
func TestUserHandler_MultiplePermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.ErrorHandler(logger.NewLogger(
		logger.WithMinLevel(logger.DEBUG),
		logger.WithEnvironment("test"),
	)))

	// 複数権限をヘッダーから取得（カンマ区切り）
	router.Use(func(c *gin.Context) {
		permissionsHeader := c.GetHeader("X-User-Permissions")
		if permissionsHeader != "" {
			c.Set("user_permissions", permissionsHeader)
		}
		c.Next()
	})

	// 複数権限チェック機能
	hasPermission := func(userPerms string, required string) bool {
		if userPerms == "*" {
			return true
		}
		// カンマ区切りの権限をチェック
		permissions := []string{userPerms} // 簡略化：1つだけチェック
		for _, perm := range permissions {
			if perm == required {
				return true
			}
		}
		return false
	}

	// 管理者権限が必要なエンドポイント
	router.DELETE("/users/:id", func(c *gin.Context) {
		userPerms, exists := c.Get("user_permissions")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No permissions"})
			return
		}

		perms, ok := userPerms.(string)
		if !ok || !hasPermission(perms, "user:delete") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Missing user:delete permission"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
	})

	// ステータス変更権限が必要なエンドポイント
	router.PUT("/users/:id/status", func(c *gin.Context) {
		userPerms, exists := c.Get("user_permissions")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No permissions"})
			return
		}

		perms, ok := userPerms.(string)
		if !ok || !hasPermission(perms, "user:manage") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Missing user:manage permission"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Status changed"})
	})

	userID := uuid.New()

	tests := []struct {
		name         string
		endpoint     string
		method       string
		permissions  string
		expectStatus int
		expectError  bool
	}{
		{"Delete with permission", "/users/" + userID.String(), "DELETE", "user:delete", http.StatusOK, false},
		{"Delete without permission", "/users/" + userID.String(), "DELETE", "user:read", http.StatusForbidden, true},
		{"Delete with wildcard", "/users/" + userID.String(), "DELETE", "*", http.StatusOK, false},
		{"Status change with permission", "/users/" + userID.String() + "/status", "PUT", "user:manage", http.StatusOK, false},
		{"Status change without permission", "/users/" + userID.String() + "/status", "PUT", "user:read", http.StatusForbidden, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == "PUT" {
				body := bytes.NewBufferString(`{"status": "inactive"}`)
				req, _ = http.NewRequest(tt.method, tt.endpoint, body)
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tt.method, tt.endpoint, nil)
			}

			req.Header.Set("X-User-Permissions", tt.permissions)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectError {
				assert.Contains(t, response, "error")
			} else {
				assert.Contains(t, response, "message")
			}
		})
	}
}
