package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// RoleInfo ロール情報
type RoleInfo struct {
	ID       uuid.UUID  `json:"id"`
	Name     string     `json:"name"`
	Priority int        `json:"priority"`
	ValidTo  *time.Time `json:"valid_to,omitempty"`
}

// CustomClaims JWTカスタムクレーム構造体を定義（複数ロール対応）
type CustomClaims struct {
	UserID         uuid.UUID  `json:"user_id"`
	Email          string     `json:"email"`
	Permissions    []string   `json:"permissions"`
	PrimaryRoleID  *uuid.UUID `json:"primary_role_id,omitempty"`
	ActiveRoles    []RoleInfo `json:"active_roles,omitempty"`
	HighestRole    *RoleInfo  `json:"highest_role,omitempty"`
	jwt.RegisteredClaims
}

// Service JWT操作を担当するサービス
type Service struct {
	secretKey []byte
	expiresIn time.Duration
}

// NewService 新しいJWTサービスインスタンスを作成
func NewService(secret string, expiresIn time.Duration) *Service {
	return &Service{
		secretKey: []byte(secret),
		expiresIn: expiresIn,
	}
}

// GenerateToken ユーザー用の新しいJWTトークンを作成（複数ロール対応）
func (s *Service) GenerateToken(userID uuid.UUID, email string, permissions []string, primaryRoleID *uuid.UUID, activeRoles []RoleInfo, highestRole *RoleInfo) (string, error) {
	claims := CustomClaims{
		UserID:        userID,
		Email:         email,
		Permissions:   permissions,
		PrimaryRoleID: primaryRoleID,
		ActiveRoles:   activeRoles,
		HighestRole:   highestRole,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
			Issuer:    "erp-access-control-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// GenerateTokenSimple 簡易版トークン生成（後方互換性）
func (s *Service) GenerateTokenSimple(userID uuid.UUID, email string, permissions []string) (string, error) {
	return s.GenerateToken(userID, email, permissions, nil, nil, nil)
}

// GenerateTokenLegacy 従来版トークン生成（廃止予定・後方互換性）
func (s *Service) GenerateTokenLegacy(userID uuid.UUID, email string, permissions []string) (string, error) {
	return s.GenerateToken(userID, email, permissions, nil, nil, nil)
}

// ValidateToken JWTトークンを検証・解析
func (s *Service) ValidateToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// GetTokenID トークンクレームからJTI（JWT ID）を抽出
func (s *Service) GetTokenID(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.ID, nil
}

// RefreshToken 同じクレームで新しい有効期限を持つトークンを生成
func (s *Service) RefreshToken(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// 同じユーザーデータで新しい有効期限のトークンを生成（複数ロール情報含む）
	return s.GenerateToken(
		claims.UserID, 
		claims.Email, 
		claims.Permissions,
		claims.PrimaryRoleID,
		claims.ActiveRoles,
		claims.HighestRole,
	)
}
