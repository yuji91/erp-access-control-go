package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// CustomClaims defines the JWT custom claims structure
type CustomClaims struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	Permissions []string  `json:"permissions"`
	jwt.RegisteredClaims
}

// Service handles JWT operations
type Service struct {
	secretKey []byte
	expiresIn time.Duration
}

// NewService creates a new JWT service instance
func NewService(secret string, expiresIn time.Duration) *Service {
	return &Service{
		secretKey: []byte(secret),
		expiresIn: expiresIn,
	}
}

// GenerateToken creates a new JWT token for a user
func (s *Service) GenerateToken(userID uuid.UUID, email string, permissions []string) (string, error) {
	claims := CustomClaims{
		UserID:      userID,
		Email:       email,
		Permissions: permissions,
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

// ValidateToken validates and parses a JWT token
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

// GetTokenID extracts the JTI (JWT ID) from token claims
func (s *Service) GetTokenID(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.ID, nil
}

// RefreshToken generates a new token with the same claims but new expiration
func (s *Service) RefreshToken(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// Generate new token with same user data but new expiration
	return s.GenerateToken(claims.UserID, claims.Email, claims.Permissions)
}
