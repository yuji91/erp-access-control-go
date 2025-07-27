package jwt

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: ベンチマークテストの追加
// - JWT生成のパフォーマンス測定
// - トークン検証の処理時間評価
// - 大量のクレーム情報を含むトークンの性能測定

// TODO: セキュリティテストの強化
// - トークン改ざん検知テスト（署名部分の変更）
// - 有効期限切れトークンのテスト
// - 異なるアルゴリズムでの検証テスト
// - クレーム情報の境界値テスト

// TODO: エッジケースの追加
// - 空の権限配列を持つトークンの生成・検証
// - 非常に長いユーザーIDやEmailの処理
// - 時間ベースのテスト（現在時刻の前後での検証）

func TestNewService(t *testing.T) {
	secret := "test-secret"
	expiresIn := 24 * time.Hour
	
	service := NewService(secret, expiresIn)
	
	assert.NotNil(t, service)
	assert.Equal(t, []byte(secret), service.secretKey)
	assert.Equal(t, expiresIn, service.expiresIn)
	
	// TODO: 無効な設定でのサービス作成テスト
	// - 空文字列のシークレット
	// - 0以下の有効期限
	// - 極端に長いシークレット
}

func TestGenerateToken(t *testing.T) {
	service := NewService("test-secret", 24*time.Hour)
	
	userID := uuid.New()
	email := "test@example.com"
	permissions := []string{"read", "write"}
	primaryRoleID := uuid.New()
	activeRoles := []RoleInfo{
		{
			ID:       primaryRoleID,
			Name:     "admin",
			Priority: 1,
			ValidTo:  nil,
		},
	}
	highestRole := &RoleInfo{
		ID:       primaryRoleID,
		Name:     "admin", 
		Priority: 1,
		ValidTo:  nil,
	}

	token, err := service.GenerateToken(userID, email, permissions, &primaryRoleID, activeRoles, highestRole)
	
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestGenerateTokenSimple(t *testing.T) {
	service := NewService("test-secret", 24*time.Hour)
	
	userID := uuid.New()
	email := "test@example.com"
	permissions := []string{"read"}
	
	token, err := service.GenerateTokenSimple(userID, email, permissions)
	
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateToken(t *testing.T) {
	service := NewService("test-secret", 24*time.Hour)
	
	userID := uuid.New()
	email := "test@example.com"
	permissions := []string{"read", "write"}
	
	// トークン生成
	token, err := service.GenerateTokenSimple(userID, email, permissions)
	require.NoError(t, err)
	
	// トークン検証
	claims, err := service.ValidateToken(token)
	require.NoError(t, err)
	
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, permissions, claims.Permissions)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	service := NewService("test-secret", 24*time.Hour)
	
	invalidToken := "invalid.token.here"
	
	claims, err := service.ValidateToken(invalidToken)
	
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	service1 := NewService("secret1", 24*time.Hour)
	service2 := NewService("secret2", 24*time.Hour)
	
	userID := uuid.New()
	email := "test@example.com"
	permissions := []string{"read"}
	
	// service1でトークン生成
	token, err := service1.GenerateTokenSimple(userID, email, permissions)
	require.NoError(t, err)
	
	// service2（異なるシークレット）で検証
	claims, err := service2.ValidateToken(token)
	
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestRefreshToken(t *testing.T) {
	service := NewService("test-secret", 24*time.Hour)
	
	userID := uuid.New()
	email := "test@example.com"
	permissions := []string{"read", "write"}
	primaryRoleID := uuid.New()
	activeRoles := []RoleInfo{
		{
			ID:       primaryRoleID,
			Name:     "admin",
			Priority: 1,
			ValidTo:  nil,
		},
	}
	highestRole := &RoleInfo{
		ID:       primaryRoleID,
		Name:     "admin",
		Priority: 1,
		ValidTo:  nil,
	}
	
	// 元のトークン生成
	originalToken, err := service.GenerateToken(userID, email, permissions, &primaryRoleID, activeRoles, highestRole)
	require.NoError(t, err)
	
	// トークン検証
	_, err = service.ValidateToken(originalToken)
	require.NoError(t, err)
	
	// リフレッシュトークン生成
	newToken, err := service.RefreshToken(originalToken)
	require.NoError(t, err)
	
	assert.NotEmpty(t, newToken)
	assert.NotEqual(t, originalToken, newToken)
	
	// 新しいトークンが有効であることを確認
	newClaims, err := service.ValidateToken(newToken)
	require.NoError(t, err)
	assert.Equal(t, userID, newClaims.UserID)
	assert.Equal(t, email, newClaims.Email)
	assert.Equal(t, permissions, newClaims.Permissions)
}

func TestRoleInfo_Structure(t *testing.T) {
	validTo := time.Now().Add(24 * time.Hour)
	roleInfo := RoleInfo{
		ID:       uuid.New(),
		Name:     "test-role",
		Priority: 1,
		ValidTo:  &validTo,
	}
	
	assert.NotEqual(t, uuid.Nil, roleInfo.ID)
	assert.Equal(t, "test-role", roleInfo.Name)
	assert.Equal(t, 1, roleInfo.Priority)
	assert.NotNil(t, roleInfo.ValidTo)
	assert.True(t, roleInfo.ValidTo.After(time.Now()))
}

func TestCustomClaims_MultiRole(t *testing.T) {
	userID := uuid.New()
	primaryRoleID := uuid.New()
	secondaryRoleID := uuid.New()
	
	activeRoles := []RoleInfo{
		{
			ID:       primaryRoleID,
			Name:     "admin",
			Priority: 1,
			ValidTo:  nil,
		},
		{
			ID:       secondaryRoleID,
			Name:     "manager",
			Priority: 2,
			ValidTo:  nil,
		},
	}
	
	claims := CustomClaims{
		UserID:        userID,
		Email:         "multi@example.com",
		Permissions:   []string{"read", "write", "admin"},
		PrimaryRoleID: &primaryRoleID,
		ActiveRoles:   activeRoles,
		HighestRole:   &activeRoles[0], // Priority 1が最高
	}
	
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "multi@example.com", claims.Email)
	assert.Equal(t, &primaryRoleID, claims.PrimaryRoleID)
	assert.Len(t, claims.ActiveRoles, 2)
	assert.Equal(t, "admin", claims.HighestRole.Name)
	assert.Equal(t, 1, claims.HighestRole.Priority)
} 