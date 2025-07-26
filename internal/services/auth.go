package services

import (
	"crypto/bcrypt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"erp-access-control-go/models"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/pkg/jwt"
)

// AuthService provides authentication and authorization services
type AuthService struct {
	db                *gorm.DB
	jwtService        *jwt.Service
	permissionService *PermissionService
	revocationService *TokenRevocationService
}

// NewAuthService creates a new authentication service
func NewAuthService(
	db *gorm.DB,
	jwtService *jwt.Service,
	permissionService *PermissionService,
	revocationService *TokenRevocationService,
) *AuthService {
	return &AuthService{
		db:                db,
		jwtService:        jwtService,
		permissionService: permissionService,
		revocationService: revocationService,
	}
}

// LoginRequest represents login request data
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginResponse represents login response data
type LoginResponse struct {
	Token       string        `json:"token"`
	ExpiresIn   time.Duration `json:"expires_in"`
	User        UserInfo      `json:"user"`
	Permissions []string      `json:"permissions"`
}

// UserInfo represents user information in responses
type UserInfo struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Status      string    `json:"status"`
	Roles       []string  `json:"roles"`
	Departments []string  `json:"departments"`
}

// Login authenticates a user and returns JWT token
func (s *AuthService) Login(req LoginRequest) (*LoginResponse, error) {
	// Find user by email
	var user models.User
	if err := s.db.Preload("Roles").Preload("Departments").
		Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewAuthenticationError("invalid email or password")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// Check if user is active
	if user.Status != models.UserStatusActive {
		return nil, errors.NewAuthenticationError("user account is not active")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.NewAuthenticationError("invalid email or password")
	}

	// Get user permissions
	permissions, err := s.permissionService.GetUserPermissions(user.ID)
	if err != nil {
		return nil, err
	}

	// Generate JWT token
	token, err := s.jwtService.GenerateToken(user.ID, user.Email, permissions)
	if err != nil {
		return nil, errors.NewInternalError("failed to generate token")
	}

	// Prepare response
	userInfo := UserInfo{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Status:    string(user.Status),
		Roles:     make([]string, len(user.Roles)),
		Departments: make([]string, len(user.Departments)),
	}

	for i, role := range user.Roles {
		userInfo.Roles[i] = role.Name
	}

	for i, dept := range user.Departments {
		userInfo.Departments[i] = dept.Name
	}

	return &LoginResponse{
		Token:       token,
		ExpiresIn:   24 * time.Hour, // Should match JWT config
		User:        userInfo,
		Permissions: permissions,
	}, nil
}

// Logout revokes the current JWT token
func (s *AuthService) Logout(jti string, userID uuid.UUID) error {
	return s.revocationService.RevokeToken(jti, userID, "user_logout")
}

// LogoutAllSessions revokes all tokens for a user
func (s *AuthService) LogoutAllSessions(userID uuid.UUID) error {
	return s.revocationService.RevokeAllUserTokens(userID, "logout_all_sessions")
}

// RefreshToken generates a new token from an existing valid token
func (s *AuthService) RefreshToken(currentToken string) (*LoginResponse, error) {
	// Validate current token
	claims, err := s.jwtService.ValidateToken(currentToken)
	if err != nil {
		return nil, errors.NewAuthenticationError("invalid token")
	}

	// Check if token is revoked
	if err := s.revocationService.ValidateTokenStatus(claims.ID, claims.UserID, claims.IssuedAt.Time); err != nil {
		return nil, errors.NewAuthenticationError("token is revoked")
	}

	// Get updated user info and permissions
	var user models.User
	if err := s.db.Preload("Roles").Preload("Departments").
		First(&user, claims.UserID).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// Check if user is still active
	if user.Status != models.UserStatusActive {
		return nil, errors.NewAuthenticationError("user account is no longer active")
	}

	// Get current permissions
	permissions, err := s.permissionService.GetUserPermissions(user.ID)
	if err != nil {
		return nil, err
	}

	// Revoke old token
	if err := s.revocationService.RevokeToken(claims.ID, claims.UserID, "token_refresh"); err != nil {
		return nil, err
	}

	// Generate new token
	newToken, err := s.jwtService.GenerateToken(user.ID, user.Email, permissions)
	if err != nil {
		return nil, errors.NewInternalError("failed to generate new token")
	}

	// Prepare response
	userInfo := UserInfo{
		ID:          user.ID,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Status:      string(user.Status),
		Roles:       make([]string, len(user.Roles)),
		Departments: make([]string, len(user.Departments)),
	}

	for i, role := range user.Roles {
		userInfo.Roles[i] = role.Name
	}

	for i, dept := range user.Departments {
		userInfo.Departments[i] = dept.Name
	}

	return &LoginResponse{
		Token:       newToken,
		ExpiresIn:   24 * time.Hour,
		User:        userInfo,
		Permissions: permissions,
	}, nil
}

// ChangePassword changes user password
func (s *AuthService) ChangePassword(userID uuid.UUID, currentPassword, newPassword string) error {
	// Get user
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return errors.NewDatabaseError(err)
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return errors.NewAuthenticationError("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.NewInternalError("failed to hash password")
	}

	// Update password
	if err := s.db.Model(&user).Update("password_hash", string(hashedPassword)).Error; err != nil {
		return errors.NewDatabaseError(err)
	}

	// Revoke all existing tokens for security
	return s.revocationService.RevokeAllUserTokens(userID, "password_change")
}

// ValidatePermission checks if user has required permission
func (s *AuthService) ValidatePermission(userID uuid.UUID, permission string) (bool, error) {
	return s.permissionService.CheckPermission(userID, permission)
}

// ValidatePermissionWithScope checks permission with scope conditions
func (s *AuthService) ValidatePermissionWithScope(userID uuid.UUID, permission string, scope map[string]interface{}) (bool, error) {
	return s.permissionService.CheckPermissionWithScope(userID, permission, scope)
}

// GetUserProfile returns user profile information
func (s *AuthService) GetUserProfile(userID uuid.UUID) (*UserInfo, error) {
	var user models.User
	if err := s.db.Preload("Roles").Preload("Departments").
		First(&user, userID).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	userInfo := &UserInfo{
		ID:          user.ID,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Status:      string(user.Status),
		Roles:       make([]string, len(user.Roles)),
		Departments: make([]string, len(user.Departments)),
	}

	for i, role := range user.Roles {
		userInfo.Roles[i] = role.Name
	}

	for i, dept := range user.Departments {
		userInfo.Departments[i] = dept.Name
	}

	return userInfo, nil
} 