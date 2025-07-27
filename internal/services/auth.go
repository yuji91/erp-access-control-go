package services

import (
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"erp-access-control-go/models"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/pkg/jwt"
)

// AuthService 認証・認可サービス
type AuthService struct {
	db                *gorm.DB
	jwtService        *jwt.Service
	permissionService *PermissionService
	revocationService *TokenRevocationService
}

// NewAuthService 新しい認証サービスを作成
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

// LoginRequest ログインリクエストデータ
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginResponse ログインレスポンスデータ
type LoginResponse struct {
	Token       string        `json:"token"`
	ExpiresIn   time.Duration `json:"expires_in"`
	User        UserInfo      `json:"user"`
	Permissions []string      `json:"permissions"`
}

// UserInfo レスポンス用ユーザー情報（複数ロール対応）
type UserInfo struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Status      string     `json:"status"`
	PrimaryRole *RoleInfo  `json:"primary_role,omitempty"`
	ActiveRoles []RoleInfo `json:"active_roles,omitempty"`
	HighestRole *RoleInfo  `json:"highest_role,omitempty"`
	Department  DeptInfo   `json:"department"`
}

// RoleInfo ロール情報（複数ロール対応）
type RoleInfo struct {
	ID       uuid.UUID  `json:"id"`
	Name     string     `json:"name"`
	Priority int        `json:"priority,omitempty"`
	ValidTo  *time.Time `json:"valid_to,omitempty"`
}

// DeptInfo 部門情報
type DeptInfo struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// Login ユーザー認証を行いJWTトークンを返す
func (s *AuthService) Login(req LoginRequest) (*LoginResponse, error) {
	// TODO: セキュリティ強化
	// - レート制限 (IP/ユーザー別ログイン試行回数制限)
	// - ブルートフォース攻撃対策 (アカウントロックアウト)
	// - ログイン履歴記録 (IP、User-Agent、成功/失敗)
	// - MFA (多要素認証) 対応

	// Find user by email（複数ロール対応）
	var user models.User
	if err := s.db.Preload("PrimaryRole").Preload("Department").
		Preload("UserRoles.Role").
		Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// TODO: タイミング攻撃対策 - 常に一定時間でレスポンス
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

	// Get user permissions（複数ロール対応）
	permissions, err := s.permissionService.GetUserPermissions(user.ID)
	if err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// アクティブロール情報を取得
	activeRoles, err := user.GetActiveRoles(s.db)
	if err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// 最高優先度ロールを取得
	highestRole, err := user.GetHighestPriorityRole(s.db)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.NewDatabaseError(err)
	}

	// JWT用のロール情報を構築
	jwtActiveRoles := make([]jwt.RoleInfo, len(activeRoles))
	for i, role := range activeRoles {
		// 対応するUserRoleから期限と優先度を取得
		var priority int
		var validTo *time.Time
		for _, ur := range user.UserRoles {
			if ur.RoleID == role.ID && ur.IsValidNow() {
				priority = ur.Priority
				validTo = ur.ValidTo
				break
			}
		}
		jwtActiveRoles[i] = jwt.RoleInfo{
			ID:       role.ID,
			Name:     role.Name,
			Priority: priority,
			ValidTo:  validTo,
		}
	}

	var jwtHighestRole *jwt.RoleInfo
	if highestRole != nil {
		// 最高優先度ロールの詳細情報を取得
		var priority int
		var validTo *time.Time
		for _, ur := range user.UserRoles {
			if ur.RoleID == highestRole.ID && ur.IsValidNow() {
				priority = ur.Priority
				validTo = ur.ValidTo
				break
			}
		}
		jwtHighestRole = &jwt.RoleInfo{
			ID:       highestRole.ID,
			Name:     highestRole.Name,
			Priority: priority,
			ValidTo:  validTo,
		}
	}

	// Generate JWT token（複数ロール対応）
	token, err := s.jwtService.GenerateToken(
		user.ID,
		user.Email,
		permissions,
		user.PrimaryRoleID,
		jwtActiveRoles,
		jwtHighestRole,
	)
	if err != nil {
		return nil, errors.NewInternalError("failed to generate token")
	}

	// Prepare response（複数ロール対応）
	userInfo := UserInfo{
		ID:     user.ID,
		Name:   user.Name,
		Email:  user.Email,
		Status: string(user.Status),
		Department: DeptInfo{
			ID:   user.Department.ID,
			Name: user.Department.Name,
		},
	}

	// プライマリロール情報を設定
	if user.PrimaryRole != nil {
		userInfo.PrimaryRole = &RoleInfo{
			ID:   user.PrimaryRole.ID,
			Name: user.PrimaryRole.Name,
		}
	}

	// アクティブロール情報を設定
	userInfo.ActiveRoles = make([]RoleInfo, len(activeRoles))
	for i, role := range activeRoles {
		var priority int
		var validTo *time.Time
		for _, ur := range user.UserRoles {
			if ur.RoleID == role.ID && ur.IsValidNow() {
				priority = ur.Priority
				validTo = ur.ValidTo
				break
			}
		}
		userInfo.ActiveRoles[i] = RoleInfo{
			ID:       role.ID,
			Name:     role.Name,
			Priority: priority,
			ValidTo:  validTo,
		}
	}

	// 最高優先度ロール情報を設定
	if highestRole != nil {
		var priority int
		var validTo *time.Time
		for _, ur := range user.UserRoles {
			if ur.RoleID == highestRole.ID && ur.IsValidNow() {
				priority = ur.Priority
				validTo = ur.ValidTo
				break
			}
		}
		userInfo.HighestRole = &RoleInfo{
			ID:       highestRole.ID,
			Name:     highestRole.Name,
			Priority: priority,
			ValidTo:  validTo,
		}
	}

	return &LoginResponse{
		Token:       token,
		ExpiresIn:   24 * time.Hour, // Should match JWT config
		User:        userInfo,
		Permissions: permissions,
	}, nil
}

// Logout 現在のJWTトークンを無効化
func (s *AuthService) Logout(jti string, userID uuid.UUID) error {
	return s.revocationService.RevokeToken(jti, userID, "user_logout")
}

// LogoutAllSessions ユーザーの全てのトークンを無効化
func (s *AuthService) LogoutAllSessions(userID uuid.UUID) error {
	return s.revocationService.RevokeAllUserTokens(userID, "logout_all_sessions")
}

// RefreshToken 既存の有効なトークンから新しいトークンを生成
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
	if err := s.db.Preload("PrimaryRole").Preload("Department").
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

	// Generate new token（複数ロール対応）
	newToken, err := s.jwtService.GenerateTokenSimple(user.ID, user.Email, permissions)
	if err != nil {
		return nil, errors.NewInternalError("failed to generate new token")
	}

	// Prepare response
	userInfo := UserInfo{
		ID:     user.ID,
		Name:   user.Name,
		Email:  user.Email,
		Status: string(user.Status),
		Department: DeptInfo{
			ID:   user.Department.ID,
			Name: user.Department.Name,
		},
	}

	// プライマリロール情報を設定
	if user.PrimaryRole != nil {
		userInfo.PrimaryRole = &RoleInfo{
			ID:   user.PrimaryRole.ID,
			Name: user.PrimaryRole.Name,
		}
	}

	return &LoginResponse{
		Token:       newToken,
		ExpiresIn:   24 * time.Hour,
		User:        userInfo,
		Permissions: permissions,
	}, nil
}

// ChangePassword ユーザーパスワードを変更
func (s *AuthService) ChangePassword(userID uuid.UUID, currentPassword, newPassword string) error {
	// TODO: パスワードポリシー強化
	// - 最小長度 (8文字以上)
	// - 複雑性要件 (大文字、小文字、数字、特殊文字)
	// - 過去のパスワード履歴チェック (直近N回と重複禁止)
	// - 辞書攻撃対策 (一般的なパスワードの禁止)
	// - パスワード強度スコア計算

	// Get user
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return errors.NewDatabaseError(err)
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return errors.NewAuthenticationError("current password is incorrect")
	}

	// TODO: パスワード強度バリデーション実装
	// if !isStrongPassword(newPassword) {
	//     return errors.NewValidationError("password", "password does not meet security requirements")
	// }

	// Hash new password
	// TODO: bcrypt cost調整 (現在は10、本番では12-14推奨)
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

// ValidatePermission ユーザーが必要な権限を持っているかチェック
func (s *AuthService) ValidatePermission(userID uuid.UUID, permission string) (bool, error) {
	return s.permissionService.CheckPermission(userID, permission)
}

// ValidatePermissionWithScope スコープ条件付きで権限をチェック
func (s *AuthService) ValidatePermissionWithScope(userID uuid.UUID, permission string, scope map[string]interface{}) (bool, error) {
	return s.permissionService.CheckPermissionWithScope(userID, permission, scope)
}

// GetUserProfile ユーザープロファイル情報を取得
func (s *AuthService) GetUserProfile(userID uuid.UUID) (*UserInfo, error) {
	var user models.User
	if err := s.db.Preload("PrimaryRole").Preload("Department").
		First(&user, userID).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	userInfo := &UserInfo{
		ID:     user.ID,
		Name:   user.Name,
		Email:  user.Email,
		Status: string(user.Status),
		Department: DeptInfo{
			ID:   user.Department.ID,
			Name: user.Department.Name,
		},
	}

	// プライマリロール情報を設定
	if user.PrimaryRole != nil {
		userInfo.PrimaryRole = &RoleInfo{
			ID:   user.PrimaryRole.ID,
			Name: user.PrimaryRole.Name,
		}
	}

	return userInfo, nil
}

// LoginWithCredentials ユーザー認証を行いJWTトークンを返す（ハンドラー用）
func (s *AuthService) LoginWithCredentials(email, password string) (*UserInfo, string, error) {
	req := LoginRequest{
		Email:    email,
		Password: password,
	}

	resp, err := s.Login(req)
	if err != nil {
		return nil, "", err
	}

	return &resp.User, resp.Token, nil
}

// GetUserPermissions ユーザーの権限一覧を取得
func (s *AuthService) GetUserPermissions(userID uuid.UUID) ([]string, error) {
	return s.permissionService.GetUserPermissions(userID)
}

// GetUserInfo ユーザー情報を取得
func (s *AuthService) GetUserInfo(userID uuid.UUID) (*UserInfo, error) {
	var user models.User
	if err := s.db.Preload("PrimaryRole").Preload("Department").
		Preload("UserRoles.Role").
		Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewDatabaseError(err)
	}

	// アクティブロール情報を取得
	activeRoles, err := user.GetActiveRoles(s.db)
	if err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// 最高優先度ロールを取得
	highestRole, err := user.GetHighestPriorityRole(s.db)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.NewDatabaseError(err)
	}

	// UserInfoを構築
	userInfo := &UserInfo{
		ID:     user.ID,
		Name:   user.Name,
		Email:  user.Email,
		Status: string(user.Status),
		Department: DeptInfo{
			ID:   user.Department.ID,
			Name: user.Department.Name,
		},
	}

	// プライマリロール情報を設定
	if user.PrimaryRole != nil {
		userInfo.PrimaryRole = &RoleInfo{
			ID:   user.PrimaryRole.ID,
			Name: user.PrimaryRole.Name,
		}
	}

	// アクティブロール情報を設定
	userInfo.ActiveRoles = make([]RoleInfo, len(activeRoles))
	for i, role := range activeRoles {
		// 対応するUserRoleから期限と優先度を取得
		var priority int
		var validTo *time.Time
		for _, ur := range user.UserRoles {
			if ur.RoleID == role.ID && ur.IsValidNow() {
				priority = ur.Priority
				validTo = ur.ValidTo
				break
			}
		}
		userInfo.ActiveRoles[i] = RoleInfo{
			ID:       role.ID,
			Name:     role.Name,
			Priority: priority,
			ValidTo:  validTo,
		}
	}

	// 最高優先度ロール情報を設定
	if highestRole != nil {
		var priority int
		var validTo *time.Time
		for _, ur := range user.UserRoles {
			if ur.RoleID == highestRole.ID && ur.IsValidNow() {
				priority = ur.Priority
				validTo = ur.ValidTo
				break
			}
		}
		userInfo.HighestRole = &RoleInfo{
			ID:       highestRole.ID,
			Name:     highestRole.Name,
			Priority: priority,
			ValidTo:  validTo,
		}
	}

	return userInfo, nil
}

// LogoutWithToken ログアウト処理（リフレッシュトークン無効化）
func (s *AuthService) LogoutWithToken(refreshToken string) error {
	// JWTトークンを検証してJTIを取得
	claims, err := s.jwtService.ValidateToken(refreshToken)
	if err != nil {
		return errors.ErrInvalidToken
	}

	// トークンを無効化
	return s.revocationService.RevokeToken(claims.ID, claims.UserID, "logout")
}
