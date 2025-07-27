package services

import (
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"erp-access-control-go/models"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/pkg/logger"
)

// UserService ユーザー管理サービス
type UserService struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewUserService 新しいユーザーサービスを作成
func NewUserService(db *gorm.DB, logger *logger.Logger) *UserService {
	return &UserService{
		db:     db,
		logger: logger,
	}
}

// CreateUserRequest ユーザー作成リクエスト
type CreateUserRequest struct {
	Name          string    `json:"name" binding:"required,min=1,max=100"`
	Email         string    `json:"email" binding:"required,email,max=255"`
	Password      string    `json:"password" binding:"required,min=6,max=255"`
	DepartmentID  uuid.UUID `json:"department_id" binding:"required"`
	PrimaryRoleID uuid.UUID `json:"primary_role_id" binding:"required"`
	Status        string    `json:"status" binding:"omitempty,oneof=active inactive suspended"`
}

// UpdateUserRequest ユーザー更新リクエスト
type UpdateUserRequest struct {
	Name         *string    `json:"name" binding:"omitempty,min=1,max=100"`
	Email        *string    `json:"email" binding:"omitempty,email,max=255"`
	DepartmentID *uuid.UUID `json:"department_id"`
	Status       *string    `json:"status" binding:"omitempty,oneof=active inactive suspended"`
}

// ChangePasswordRequest パスワード変更リクエスト
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6,max=255"`
}

// UserListFilters ユーザー一覧フィルター
type UserListFilters struct {
	DepartmentID *uuid.UUID         `form:"department_id"`
	Status       *models.UserStatus `form:"status"`
	RoleID       *uuid.UUID         `form:"role_id"`
	Search       string             `form:"search"`
	Page         int                `form:"page,default=1"`
	Limit        int                `form:"limit,default=20"`
}

// UserResponse ユーザーレスポンス
type UserResponse struct {
	ID            uuid.UUID         `json:"id"`
	Name          string            `json:"name"`
	Email         string            `json:"email"`
	Status        models.UserStatus `json:"status"`
	DepartmentID  uuid.UUID         `json:"department_id"`
	PrimaryRoleID *uuid.UUID        `json:"primary_role_id"`
	CreatedAt     string            `json:"created_at"`
	UpdatedAt     string            `json:"updated_at"`
	Department    *DeptInfo         `json:"department,omitempty"`
	PrimaryRole   *RoleInfo         `json:"primary_role,omitempty"`
	ActiveRoles   []RoleInfo        `json:"active_roles,omitempty"`
}

// UserListResponse ユーザー一覧レスポンス
type UserListResponse struct {
	Users []UserResponse `json:"users"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

// CreateUser ユーザーを作成
func (s *UserService) CreateUser(req CreateUserRequest) (*UserResponse, error) {
	s.logger.Info("Creating new user", map[string]interface{}{
		"email":        req.Email,
		"department":   req.DepartmentID,
		"primary_role": req.PrimaryRoleID,
	})

	// メールアドレスの重複チェック
	var existingUser models.User
	if err := s.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return nil, errors.NewValidationError("email", "Email address already exists")
	} else if err != gorm.ErrRecordNotFound {
		return nil, errors.NewDatabaseError(err)
	}

	// 部署存在確認
	var department models.Department
	if err := s.db.First(&department, req.DepartmentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Department", "Department does not exist")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// ロール存在確認
	var role models.Role
	if err := s.db.First(&role, req.PrimaryRoleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Role", "Role does not exist")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// パスワードハッシュ化
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.NewInternalError("Failed to hash password")
	}

	// ユーザー作成
	user := models.User{
		Name:          req.Name,
		Email:         req.Email,
		PasswordHash:  string(hashedPassword),
		DepartmentID:  req.DepartmentID,
		PrimaryRoleID: &req.PrimaryRoleID,
		Status:        models.UserStatus(req.Status),
	}

	// デフォルトステータス設定
	if user.Status == "" {
		user.Status = models.UserStatusActive
	}

	if err := s.db.Create(&user).Error; err != nil {
		s.logger.Error("Failed to create user", err, map[string]interface{}{
			"email": req.Email,
		})
		return nil, errors.NewDatabaseError(err)
	}

	s.logger.Info("User created successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	// 作成されたユーザーを詳細付きで取得
	return s.GetUser(user.ID)
}

// GetUser ユーザー詳細を取得
func (s *UserService) GetUser(userID uuid.UUID) (*UserResponse, error) {
	var user models.User
	if err := s.db.
		Preload("Department").
		Preload("PrimaryRole").
		Preload("UserRoles.Role").
		First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("User", "User not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	return s.convertToUserResponse(&user), nil
}

// UpdateUser ユーザー情報を更新
func (s *UserService) UpdateUser(userID uuid.UUID, req UpdateUserRequest) (*UserResponse, error) {
	s.logger.Info("Updating user", map[string]interface{}{
		"user_id": userID,
	})

	// ユーザー存在確認
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("User", "User not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// メールアドレス重複チェック（自分以外）
	if req.Email != nil {
		var existingUser models.User
		if err := s.db.Where("email = ? AND id != ?", *req.Email, userID).First(&existingUser).Error; err == nil {
			return nil, errors.NewValidationError("email", "Email address already exists")
		} else if err != gorm.ErrRecordNotFound {
			return nil, errors.NewDatabaseError(err)
		}
	}

	// 部署存在確認
	if req.DepartmentID != nil {
		var department models.Department
		if err := s.db.First(&department, *req.DepartmentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.NewNotFoundError("Department", "Department does not exist")
			}
			return nil, errors.NewDatabaseError(err)
		}
	}

	// 更新用のマップを作成
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.DepartmentID != nil {
		updates["department_id"] = *req.DepartmentID
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	// 更新実行
	if err := s.db.Model(&user).Updates(updates).Error; err != nil {
		s.logger.Error("Failed to update user", err, map[string]interface{}{
			"user_id": userID,
		})
		return nil, errors.NewDatabaseError(err)
	}

	s.logger.Info("User updated successfully", map[string]interface{}{
		"user_id": userID,
	})

	return s.GetUser(userID)
}

// DeleteUser ユーザーを削除（ソフトデリート）
func (s *UserService) DeleteUser(userID uuid.UUID) error {
	s.logger.Info("Deleting user", map[string]interface{}{
		"user_id": userID,
	})

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewNotFoundError("User", "User not found")
		}
		return errors.NewDatabaseError(err)
	}

	// ソフトデリート実行
	if err := s.db.Delete(&user).Error; err != nil {
		s.logger.Error("Failed to delete user", err, map[string]interface{}{
			"user_id": userID,
		})
		return errors.NewDatabaseError(err)
	}

	s.logger.Info("User deleted successfully", map[string]interface{}{
		"user_id": userID,
	})

	return nil
}

// GetUsers ユーザー一覧を取得（フィルタリング・ページング対応）
func (s *UserService) GetUsers(filters UserListFilters) (*UserListResponse, error) {
	query := s.db.Model(&models.User{}).
		Preload("Department").
		Preload("PrimaryRole")

	// フィルタ適用
	if filters.DepartmentID != nil {
		query = query.Where("department_id = ?", *filters.DepartmentID)
	}

	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	if filters.RoleID != nil {
		query = query.Where("primary_role_id = ?", *filters.RoleID)
	}

	if filters.Search != "" {
		searchTerm := "%" + strings.ToLower(filters.Search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(email) LIKE ?", searchTerm, searchTerm)
	}

	// 総数取得
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// ページング設定
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Limit < 1 || filters.Limit > 100 {
		filters.Limit = 20
	}

	offset := (filters.Page - 1) * filters.Limit

	// ユーザー取得
	var users []models.User
	if err := query.
		Offset(offset).
		Limit(filters.Limit).
		Order("created_at DESC").
		Find(&users).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// レスポンス変換
	userResponses := make([]UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = *s.convertToUserResponse(&user)
	}

	return &UserListResponse{
		Users: userResponses,
		Total: total,
		Page:  filters.Page,
		Limit: filters.Limit,
	}, nil
}

// ChangeUserStatus ユーザーステータスを変更
func (s *UserService) ChangeUserStatus(userID uuid.UUID, status models.UserStatus) (*UserResponse, error) {
	s.logger.Info("Changing user status", map[string]interface{}{
		"user_id": userID,
		"status":  status,
	})

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("User", "User not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// ステータス更新
	if err := s.db.Model(&user).Update("status", status).Error; err != nil {
		s.logger.Error("Failed to change user status", err, map[string]interface{}{
			"user_id": userID,
			"status":  status,
		})
		return nil, errors.NewDatabaseError(err)
	}

	s.logger.Info("User status changed successfully", map[string]interface{}{
		"user_id": userID,
		"status":  status,
	})

	return s.GetUser(userID)
}

// ChangePassword ユーザーのパスワードを変更
func (s *UserService) ChangePassword(userID uuid.UUID, req ChangePasswordRequest) error {
	s.logger.Info("Changing user password", map[string]interface{}{
		"user_id": userID,
	})

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewNotFoundError("User", "User not found")
		}
		return errors.NewDatabaseError(err)
	}

	// 現在のパスワード確認
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return errors.NewValidationError("current_password", "Current password is incorrect")
	}

	// 新しいパスワードハッシュ化
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.NewInternalError("Failed to hash password")
	}

	// パスワード更新
	if err := s.db.Model(&user).Update("password_hash", string(hashedPassword)).Error; err != nil {
		s.logger.Error("Failed to change password", err, map[string]interface{}{
			"user_id": userID,
		})
		return errors.NewDatabaseError(err)
	}

	s.logger.Info("Password changed successfully", map[string]interface{}{
		"user_id": userID,
	})

	return nil
}

// convertToUserResponse models.UserをUserResponseに変換
func (s *UserService) convertToUserResponse(user *models.User) *UserResponse {
	response := &UserResponse{
		ID:            user.ID,
		Name:          user.Name,
		Email:         user.Email,
		Status:        user.Status,
		DepartmentID:  user.DepartmentID,
		PrimaryRoleID: user.PrimaryRoleID,
		CreatedAt:     user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Department情報追加
	if user.Department.ID != uuid.Nil {
		response.Department = &DeptInfo{
			ID:   user.Department.ID,
			Name: user.Department.Name,
		}
	}

	// PrimaryRole情報追加
	if user.PrimaryRole != nil {
		response.PrimaryRole = &RoleInfo{
			ID:   user.PrimaryRole.ID,
			Name: user.PrimaryRole.Name,
		}
	}

	// ActiveRoles情報追加
	if len(user.UserRoles) > 0 {
		activeRoles := make([]RoleInfo, 0)
		for _, userRole := range user.UserRoles {
			if userRole.IsValidNow() {
				activeRoles = append(activeRoles, RoleInfo{
					ID:   userRole.Role.ID,
					Name: userRole.Role.Name,
				})
			}
		}
		response.ActiveRoles = activeRoles
	}

	return response
}
