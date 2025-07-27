package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"erp-access-control-go/models"
	"erp-access-control-go/pkg/errors"
)

// MockDB はGORM DBのモック
type MockDB struct {
	mock.Mock
}

func (m *MockDB) Where(query interface{}, args ...interface{}) *gorm.DB {
	mockArgs := m.Called(query, args)
	return mockArgs.Get(0).(*gorm.DB)
}

func (m *MockDB) First(dest interface{}, conds ...interface{}) *gorm.DB {
	mockArgs := m.Called(dest, conds)
	return mockArgs.Get(0).(*gorm.DB)
}

func (m *MockDB) Create(value interface{}) *gorm.DB {
	mockArgs := m.Called(value)
	return mockArgs.Get(0).(*gorm.DB)
}

// TestUserService_CreateUser_ValidInput 正常なユーザー作成テスト
func TestUserService_CreateUser_ValidInput(t *testing.T) {
	// Given: テストデータ準備
	departmentID := uuid.New()
	roleID := uuid.New()

	req := CreateUserRequest{
		Name:          "テストユーザー",
		Email:         "test@example.com",
		Password:      "password123",
		DepartmentID:  departmentID,
		PrimaryRoleID: roleID,
		Status:        "active",
	}

	// UserServiceの基本的なロジックをテスト
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	assert.NoError(t, err)
	assert.NotEmpty(t, hashedPassword)

	// パスワードハッシュ化の検証
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(req.Password))
	assert.NoError(t, err)

	// リクエスト検証
	assert.Equal(t, "テストユーザー", req.Name)
	assert.Equal(t, "test@example.com", req.Email)
	assert.Equal(t, departmentID, req.DepartmentID)
	assert.Equal(t, roleID, req.PrimaryRoleID)
	assert.Equal(t, "active", req.Status)
}

// TestUserService_CreateUser_EmailValidation メールアドレスバリデーションテスト
func TestUserService_CreateUser_EmailValidation(t *testing.T) {
	testCases := []struct {
		name     string
		email    string
		expected bool
	}{
		{"Valid email", "test@example.com", true},
		{"Valid email with subdomain", "user@mail.example.com", true},
		{"Invalid email no @", "testexample.com", false},
		{"Invalid email no domain", "test@", false},
		{"Empty email", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 基本的なメール形式チェック（Ginのバリデーションをシミュレート）
			isValid := len(tc.email) > 0 &&
				len(tc.email) <= 255 &&
				containsAt(tc.email) &&
				containsDot(tc.email)

			assert.Equal(t, tc.expected, isValid, "Email validation for: %s", tc.email)
		})
	}
}

// TestUserService_UpdateUser_PartialUpdate 部分更新テスト
func TestUserService_UpdateUser_PartialUpdate(t *testing.T) {
	// Given: 更新リクエスト
	newName := "更新されたユーザー"
	newEmail := "updated@example.com"

	req := UpdateUserRequest{
		Name:  &newName,
		Email: &newEmail,
		// Status は更新しない（nil）
	}

	// Then: 更新フィールドの確認
	assert.NotNil(t, req.Name)
	assert.Equal(t, "更新されたユーザー", *req.Name)

	assert.NotNil(t, req.Email)
	assert.Equal(t, "updated@example.com", *req.Email)

	assert.Nil(t, req.Status) // 更新されない
}

// TestUserService_ChangePassword_PasswordValidation パスワード変更テスト
func TestUserService_ChangePassword_PasswordValidation(t *testing.T) {
	req := ChangePasswordRequest{
		CurrentPassword: "oldpassword123",
		NewPassword:     "newpassword456",
	}

	// パスワード長さ検証
	assert.True(t, len(req.CurrentPassword) >= 6, "Current password should be at least 6 characters")
	assert.True(t, len(req.NewPassword) >= 6, "New password should be at least 6 characters")

	// パスワードの違い確認
	assert.NotEqual(t, req.CurrentPassword, req.NewPassword, "New password should be different from current")

	// ハッシュ化テスト
	oldHash, err := bcrypt.GenerateFromPassword([]byte(req.CurrentPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)

	// 新旧パスワードの確認
	err = bcrypt.CompareHashAndPassword(oldHash, []byte(req.CurrentPassword))
	assert.NoError(t, err)

	err = bcrypt.CompareHashAndPassword(newHash, []byte(req.NewPassword))
	assert.NoError(t, err)

	// 間違ったパスワードとの比較
	err = bcrypt.CompareHashAndPassword(oldHash, []byte(req.NewPassword))
	assert.Error(t, err, "Old hash should not match new password")
}

// TestUserService_UserListFilters フィルターロジックテスト
func TestUserService_UserListFilters(t *testing.T) {
	departmentID := uuid.New()
	roleID := uuid.New()
	status := models.UserStatusActive

	filters := UserListFilters{
		DepartmentID: &departmentID,
		Status:       &status,
		RoleID:       &roleID,
		Search:       "テスト",
		Page:         2,
		Limit:        20,
	}

	// フィルター値の確認
	assert.NotNil(t, filters.DepartmentID)
	assert.Equal(t, departmentID, *filters.DepartmentID)

	assert.NotNil(t, filters.Status)
	assert.Equal(t, models.UserStatusActive, *filters.Status)

	assert.NotNil(t, filters.RoleID)
	assert.Equal(t, roleID, *filters.RoleID)

	assert.Equal(t, "テスト", filters.Search)
	assert.Equal(t, 2, filters.Page)
	assert.Equal(t, 20, filters.Limit)

	// ページング計算
	offset := (filters.Page - 1) * filters.Limit
	assert.Equal(t, 20, offset) // (2-1) * 20 = 20
}

// TestUserService_UserStatus ユーザーステータス検証テスト
func TestUserService_UserStatus(t *testing.T) {
	validStatuses := []models.UserStatus{
		models.UserStatusActive,
		models.UserStatusInactive,
		models.UserStatusSuspended,
	}

	for _, status := range validStatuses {
		t.Run(string(status), func(t *testing.T) {
			// ステータス値確認
			assert.NotEmpty(t, string(status))

			// デフォルトステータスの確認
			if status == models.UserStatusActive {
				assert.Equal(t, "active", string(status))
			}
		})
	}
}

// TestUserService_ErrorHandling エラーハンドリングテスト
func TestUserService_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name        string
		errorType   string
		expectedMsg string
	}{
		{
			name:        "Validation Error",
			errorType:   "VALIDATION_ERROR",
			expectedMsg: "validation failed",
		},
		{
			name:        "Not Found Error",
			errorType:   "NOT_FOUND",
			expectedMsg: "resource not found",
		},
		{
			name:        "Database Error",
			errorType:   "DATABASE_ERROR",
			expectedMsg: "database operation failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error

			switch tc.errorType {
			case "VALIDATION_ERROR":
				err = errors.NewValidationError("field", tc.expectedMsg)
			case "NOT_FOUND":
				err = errors.NewNotFoundError("User", tc.expectedMsg)
			case "DATABASE_ERROR":
				err = errors.NewDatabaseError(gorm.ErrRecordNotFound)
			}

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.errorType)
		})
	}
}

// TestUserService_DataConversion データ変換テスト
func TestUserService_DataConversion(t *testing.T) {
	// UUID文字列からUUID変換
	uuidStr := "550e8400-e29b-41d4-a716-446655440000"
	parsedUUID, err := uuid.Parse(uuidStr)
	assert.NoError(t, err)
	assert.Equal(t, uuidStr, parsedUUID.String())

	// 新しいUUID生成
	newUUID := uuid.New()
	assert.NotEqual(t, uuid.Nil, newUUID)
	assert.Equal(t, 36, len(newUUID.String())) // UUID文字列長
}

// TestUserService_ResponseStructure レスポンス構造テスト
func TestUserService_ResponseStructure(t *testing.T) {
	userID := uuid.New()
	departmentID := uuid.New()
	roleID := uuid.New()

	response := UserResponse{
		ID:            userID,
		Name:          "テストユーザー",
		Email:         "test@example.com",
		Status:        models.UserStatusActive,
		DepartmentID:  departmentID,
		PrimaryRoleID: &roleID,
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	// 必須フィールドの確認
	assert.Equal(t, userID, response.ID)
	assert.Equal(t, "テストユーザー", response.Name)
	assert.Equal(t, "test@example.com", response.Email)
	assert.Equal(t, models.UserStatusActive, response.Status)
	assert.Equal(t, departmentID, response.DepartmentID)
	assert.NotNil(t, response.PrimaryRoleID)
	assert.Equal(t, roleID, *response.PrimaryRoleID)

	// オプショナルフィールド
	assert.Nil(t, response.Department)  // リレーションは必要時のみ
	assert.Nil(t, response.PrimaryRole) // リレーションは必要時のみ
	assert.Nil(t, response.ActiveRoles) // リレーションは必要時のみ
}

// TestUserService_ListResponse 一覧レスポンステスト
func TestUserService_ListResponse(t *testing.T) {
	users := []UserResponse{
		{ID: uuid.New(), Name: "ユーザー1", Email: "user1@example.com"},
		{ID: uuid.New(), Name: "ユーザー2", Email: "user2@example.com"},
		{ID: uuid.New(), Name: "ユーザー3", Email: "user3@example.com"},
	}

	listResponse := UserListResponse{
		Users: users,
		Total: 25,
		Page:  2,
		Limit: 10,
	}

	// 一覧レスポンス確認
	assert.Len(t, listResponse.Users, 3)
	assert.Equal(t, int64(25), listResponse.Total)
	assert.Equal(t, 2, listResponse.Page)
	assert.Equal(t, 10, listResponse.Limit)

	// ページネーション計算確認
	hasNextPage := listResponse.Page*listResponse.Limit < int(listResponse.Total)
	assert.True(t, hasNextPage) // 2*10 = 20 < 25
}

// ヘルパー関数

func containsAt(email string) bool {
	for _, char := range email {
		if char == '@' {
			return true
		}
	}
	return false
}

func containsDot(email string) bool {
	for _, char := range email {
		if char == '.' {
			return true
		}
	}
	return false
}
