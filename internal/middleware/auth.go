package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"erp-access-control-go/internal/services"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/pkg/jwt"
	"erp-access-control-go/pkg/logger"
)

// AuthMiddleware JWT認証ミドルウェア
type AuthMiddleware struct {
	jwtService        *jwt.Service
	revocationService *services.TokenRevocationService
	logger            *logger.Logger
}

// NewAuthMiddleware 新しい認証ミドルウェアを作成
func NewAuthMiddleware(jwtService *jwt.Service, revocationService *services.TokenRevocationService, logger *logger.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService:        jwtService,
		revocationService: revocationService,
		logger:            logger,
	}
}

// ErrorHandler エラーハンドリングミドルウェア
func ErrorHandler(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// パニックリカバリー
		defer func() {
			if err := recover(); err != nil {
				// スタックトレースの取得
				stack := string(debug.Stack())

				// エラーログの出力
				log.Error("Panic recovered", fmt.Errorf("%v", err), map[string]interface{}{
					"stack_trace": stack,
					"path":        c.Request.URL.Path,
					"method":      c.Request.Method,
				})

				// レスポンスが既に送信されていない場合のみレスポンス送信
				if !c.Writer.Written() {
					c.JSON(http.StatusInternalServerError, errors.NewInternalError("An unexpected error occurred"))
				}
				c.Abort()
			}
		}()

		c.Next()

		// エラーハンドリング（レスポンスが既に送信されていない場合のみ）
		if len(c.Errors) > 0 && !c.Writer.Written() {
			err := c.Errors.Last().Err
			var apiErr *errors.APIError

			// エラー種別の判定
			switch {
			case errors.IsAuthenticationError(err):
				apiErr = err.(*errors.APIError)
				log.Warn("Authentication error", map[string]interface{}{
					"error": apiErr.Error(),
					"path":  c.Request.URL.Path,
				})
			case errors.IsAuthorizationError(err):
				apiErr = err.(*errors.APIError)
				log.Warn("Authorization error", map[string]interface{}{
					"error": apiErr.Error(),
					"path":  c.Request.URL.Path,
				})
			case errors.IsValidationError(err):
				apiErr = err.(*errors.APIError)
				log.Info("Validation error", map[string]interface{}{
					"error": apiErr.Error(),
					"path":  c.Request.URL.Path,
				})
			default:
				// 未知のエラーは内部エラーとして処理
				apiErr = errors.NewInternalError(err.Error())
				log.Error("Internal error", err, map[string]interface{}{
					"path": c.Request.URL.Path,
				})
			}

			c.JSON(apiErr.Status, apiErr)
			c.Abort()
		}
	}
}

// Authentication JWTトークンを検証してユーザーコンテキストを設定
func (m *AuthMiddleware) Authentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			m.logger.Warn("Missing authorization header", map[string]interface{}{
				"path": c.Request.URL.Path,
				"ip":   c.ClientIP(),
			})
			c.Error(errors.ErrInvalidToken)
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			m.logger.Warn("Invalid token format", map[string]interface{}{
				"path": c.Request.URL.Path,
				"ip":   c.ClientIP(),
			})
			c.Error(errors.ErrInvalidToken)
			c.Abort()
			return
		}

		claims, err := m.jwtService.ValidateToken(tokenString)
		if err != nil {
			m.logger.Warn("Token validation failed", map[string]interface{}{
				"error": err.Error(),
				"path":  c.Request.URL.Path,
				"ip":    c.ClientIP(),
			})
			c.Error(errors.ErrInvalidToken)
			c.Abort()
			return
		}

		// トークンの失効確認
		if err := m.revocationService.ValidateTokenStatus(claims.ID, claims.UserID, claims.IssuedAt.Time); err != nil {
			m.logger.Warn("Token revoked", map[string]interface{}{
				"token_id": claims.ID,
				"user_id":  claims.UserID,
				"path":     c.Request.URL.Path,
			})
			c.Error(errors.ErrTokenRevoked)
			c.Abort()
			return
		}

		// ユーザー情報をコンテキストに保存
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("permissions", claims.Permissions)
		c.Set("jti", claims.ID)
		c.Set("primary_role_id", claims.PrimaryRoleID)
		c.Set("active_roles", claims.ActiveRoles)
		c.Set("highest_role", claims.HighestRole)

		// アクセスログ
		m.logger.Info("Authenticated request", map[string]interface{}{
			"user_id": claims.UserID,
			"email":   claims.Email,
			"path":    c.Request.URL.Path,
			"method":  c.Request.Method,
		})

		c.Next()
	}
}

// RequirePermissions ユーザーが必要な権限を持っているかチェック
func RequirePermissions(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userPerms, exists := c.Get("permissions")
		if !exists {
			c.Error(errors.ErrPermissionDenied)
			c.Abort()
			return
		}

		userPermissions, ok := userPerms.([]string)
		if !ok {
			c.Error(errors.ErrPermissionDenied)
			c.Abort()
			return
		}

		for _, requiredPerm := range permissions {
			if !hasPermission(userPermissions, requiredPerm) {
				c.Error(errors.NewAuthorizationError(fmt.Sprintf("Missing required permission: %s", requiredPerm)))
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// RequireAnyPermission ユーザーが必要な権限のいずれかを持っているかチェック
func RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userPerms, exists := c.Get("permissions")
		if !exists {
			c.Error(errors.ErrPermissionDenied)
			c.Abort()
			return
		}

		userPermissions, ok := userPerms.([]string)
		if !ok {
			c.Error(errors.ErrPermissionDenied)
			c.Abort()
			return
		}

		for _, requiredPerm := range permissions {
			if hasPermission(userPermissions, requiredPerm) {
				c.Next()
				return
			}
		}

		c.Error(errors.NewAuthorizationError(fmt.Sprintf("Missing any of required permissions: %s", strings.Join(permissions, ", "))))
		c.Abort()
	}
}

// RequireOwnership ユーザーがリソースの所有者かチェック
func RequireOwnership() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.Error(errors.ErrPermissionDenied)
			c.Abort()
			return
		}

		currentUserID, ok := userID.(uuid.UUID)
		if !ok {
			c.Error(errors.ErrPermissionDenied)
			c.Abort()
			return
		}

		resourceUserID := c.Param("user_id")
		if resourceUserID == "" {
			c.Error(errors.NewValidationError("user_id", "missing user ID parameter"))
			c.Abort()
			return
		}

		resourceUserUUID, err := uuid.Parse(resourceUserID)
		if err != nil {
			c.Error(errors.NewValidationError("user_id", "invalid user ID format"))
			c.Abort()
			return
		}

		if currentUserID != resourceUserUUID {
			c.Error(errors.NewAuthorizationError("You do not have permission to access this resource"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetCurrentUserID コンテキストから現在のユーザーIDを取得
func GetCurrentUserID(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.NewAuthenticationError("User not authenticated")
	}

	currentUserID, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.NewAuthenticationError("Invalid user ID in context")
	}

	return currentUserID, nil
}

// GetCurrentUserEmail コンテキストから現在のユーザーメールアドレスを取得
func GetCurrentUserEmail(c *gin.Context) (string, error) {
	email, exists := c.Get("email")
	if !exists {
		return "", errors.NewAuthenticationError("User email not found in context")
	}

	userEmail, ok := email.(string)
	if !ok {
		return "", errors.NewAuthenticationError("Invalid email in context")
	}

	return userEmail, nil
}

// GetCurrentUserPermissions コンテキストから現在のユーザー権限を取得
func GetCurrentUserPermissions(c *gin.Context) ([]string, error) {
	perms, exists := c.Get("permissions")
	if !exists {
		return nil, errors.NewAuthenticationError("Permissions not found in context")
	}

	permissions, ok := perms.([]string)
	if !ok {
		return nil, errors.NewAuthenticationError("Invalid permissions in context")
	}

	return permissions, nil
}

// hasPermission ユーザーの権限リストに指定された権限が存在するかチェック
func hasPermission(userPermissions []string, requiredPermission string) bool {
	for _, perm := range userPermissions {
		if perm == requiredPermission || perm == "*" || perm == "*:*" {
			return true
		}
	}
	return false
}
