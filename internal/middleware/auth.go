package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"erp-access-control-go/internal/services"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/pkg/jwt"
)

// AuthMiddleware provides JWT authentication middleware
type AuthMiddleware struct {
	jwtService          *jwt.Service
	revocationService   *services.TokenRevocationService
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtService *jwt.Service, revocationService *services.TokenRevocationService) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService:        jwtService,
		revocationService: revocationService,
	}
}

// Authentication verifies JWT token and sets user context
func (m *AuthMiddleware) Authentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, errors.ErrInvalidToken)
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, errors.ErrInvalidToken)
			c.Abort()
			return
		}

		claims, err := m.jwtService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, errors.ErrInvalidToken)
			c.Abort()
			return
		}

		// Check if token is revoked
		if err := m.revocationService.ValidateTokenStatus(claims.ID, claims.UserID, claims.IssuedAt.Time); err != nil {
			c.JSON(http.StatusUnauthorized, errors.ErrInvalidToken)
			c.Abort()
			return
		}

		// Store user information in context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("permissions", claims.Permissions)
		c.Set("jti", claims.ID)

		c.Next()
	}
}

// RequirePermissions checks if user has required permissions
func RequirePermissions(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userPerms, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, errors.ErrPermissionDenied)
			c.Abort()
			return
		}

		userPermissions, ok := userPerms.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, errors.ErrPermissionDenied)
			c.Abort()
			return
		}

		for _, requiredPerm := range permissions {
			if !hasPermission(userPermissions, requiredPerm) {
				c.JSON(http.StatusForbidden, errors.ErrPermissionDenied)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// RequireAnyPermission checks if user has any of the required permissions
func RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userPerms, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, errors.ErrPermissionDenied)
			c.Abort()
			return
		}

		userPermissions, ok := userPerms.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, errors.ErrPermissionDenied)
			c.Abort()
			return
		}

		for _, requiredPerm := range permissions {
			if hasPermission(userPermissions, requiredPerm) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, errors.ErrPermissionDenied)
		c.Abort()
	}
}

// RequireOwnership checks if user is the owner of the resource
func RequireOwnership() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusForbidden, errors.ErrPermissionDenied)
			c.Abort()
			return
		}

		currentUserID, ok := userID.(uuid.UUID)
		if !ok {
			c.JSON(http.StatusForbidden, errors.ErrPermissionDenied)
			c.Abort()
			return
		}

		// Get resource owner ID from URL parameter
		resourceUserID := c.Param("user_id")
		if resourceUserID == "" {
			c.JSON(http.StatusBadRequest, errors.NewValidationError("user_id", "missing user ID parameter"))
			c.Abort()
			return
		}

		resourceUserUUID, err := uuid.Parse(resourceUserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, errors.NewValidationError("user_id", "invalid user ID format"))
			c.Abort()
			return
		}

		// Check if current user is the owner
		if currentUserID != resourceUserUUID {
			c.JSON(http.StatusForbidden, errors.ErrPermissionDenied)
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetCurrentUserID extracts the current user ID from context
func GetCurrentUserID(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.NewAuthenticationError("user not authenticated")
	}

	currentUserID, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.NewAuthenticationError("invalid user ID in context")
	}

	return currentUserID, nil
}

// GetCurrentUserEmail extracts the current user email from context
func GetCurrentUserEmail(c *gin.Context) (string, error) {
	email, exists := c.Get("email")
	if !exists {
		return "", errors.NewAuthenticationError("user email not found in context")
	}

	userEmail, ok := email.(string)
	if !ok {
		return "", errors.NewAuthenticationError("invalid email in context")
	}

	return userEmail, nil
}

// GetCurrentUserPermissions extracts the current user permissions from context
func GetCurrentUserPermissions(c *gin.Context) ([]string, error) {
	perms, exists := c.Get("permissions")
	if !exists {
		return nil, errors.NewAuthenticationError("permissions not found in context")
	}

	permissions, ok := perms.([]string)
	if !ok {
		return nil, errors.NewAuthenticationError("invalid permissions in context")
	}

	return permissions, nil
}

// hasPermission checks if a permission exists in user's permissions
func hasPermission(userPermissions []string, requiredPermission string) bool {
	for _, perm := range userPermissions {
		if perm == requiredPermission || perm == "*" {
			return true
		}
	}
	return false
} 