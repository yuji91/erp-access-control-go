package services

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"erp-access-control-go/models"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/pkg/logger"
)

// DepartmentService 部署管理サービス
type DepartmentService struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewDepartmentService 新しい部署サービスを作成
func NewDepartmentService(db *gorm.DB, logger *logger.Logger) *DepartmentService {
	return &DepartmentService{
		db:     db,
		logger: logger,
	}
}

// CreateDepartmentRequest 部署作成リクエスト
type CreateDepartmentRequest struct {
	Name     string     `json:"name" binding:"required,min=2,max=100"`
	ParentID *uuid.UUID `json:"parent_id" binding:"omitempty"`
}

// UpdateDepartmentRequest 部署更新リクエスト
type UpdateDepartmentRequest struct {
	Name     *string    `json:"name" binding:"omitempty,min=2,max=100"`
	ParentID *uuid.UUID `json:"parent_id"`
}

// DepartmentResponse 部署レスポンス
type DepartmentResponse struct {
	ID          uuid.UUID             `json:"id"`
	Name        string                `json:"name"`
	Code        string                `json:"code"`
	Description string                `json:"description"`
	ParentID    *uuid.UUID            `json:"parent_id,omitempty"`
	CreatedAt   string                `json:"created_at"`
	UpdatedAt   string                `json:"updated_at"`
	Parent      *DepartmentBasicInfo  `json:"parent,omitempty"`
	Children    []DepartmentBasicInfo `json:"children,omitempty"`
	Users       []UserBasicInfo       `json:"users,omitempty"`
}

// DepartmentBasicInfo 部署基本情報
type DepartmentBasicInfo struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// UserBasicInfo ユーザー基本情報
type UserBasicInfo struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// DepartmentListResponse 部署一覧レスポンス
type DepartmentListResponse struct {
	Departments []DepartmentResponse `json:"departments"`
	Total       int64                `json:"total"`
	Page        int                  `json:"page"`
	Limit       int                  `json:"limit"`
}

// DepartmentHierarchyNode 階層ツリーノード
type DepartmentHierarchyNode struct {
	ID       uuid.UUID                 `json:"id"`
	Name     string                    `json:"name"`
	Children []DepartmentHierarchyNode `json:"children,omitempty"`
}

// DepartmentHierarchyResponse 階層ツリーレスポンス
type DepartmentHierarchyResponse struct {
	Departments []DepartmentHierarchyNode `json:"departments"`
}

// CreateDepartment 部署を作成
func (s *DepartmentService) CreateDepartment(req CreateDepartmentRequest) (*DepartmentResponse, error) {
	s.logger.Info("Creating new department", map[string]interface{}{
		"name":      req.Name,
		"parent_id": req.ParentID,
	})

	// 名前の重複チェック
	var existingDept models.Department
	if err := s.db.Where("name = ?", req.Name).First(&existingDept).Error; err == nil {
		return nil, errors.NewValidationError("name", "Department name already exists")
	} else if err != gorm.ErrRecordNotFound {
		return nil, errors.NewDatabaseError(err)
	}

	// 親部署存在確認（指定された場合）
	if req.ParentID != nil {
		var parent models.Department
		if err := s.db.First(&parent, *req.ParentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.NewNotFoundError("Department", "Parent department does not exist")
			}
			return nil, errors.NewDatabaseError(err)
		}

		// 階層深度チェック
		depth, err := s.calculateDepth(*req.ParentID)
		if err != nil {
			return nil, err
		}
		if depth >= 5 {
			return nil, errors.NewValidationError("parent_id", "Maximum hierarchy depth (5 levels) exceeded")
		}
	}

	// 部署作成
	department := models.Department{
		Name:     req.Name,
		ParentID: req.ParentID,
	}

	if err := s.db.Create(&department).Error; err != nil {
		s.logger.Error("Failed to create department", err, map[string]interface{}{
			"name": req.Name,
		})
		return nil, errors.NewDatabaseError(err)
	}

	s.logger.Info("Department created successfully", map[string]interface{}{
		"department_id": department.ID,
		"name":          department.Name,
	})

	// 作成された部署を詳細付きで取得
	return s.GetDepartment(department.ID)
}

// GetDepartment 部署詳細を取得
func (s *DepartmentService) GetDepartment(departmentID uuid.UUID) (*DepartmentResponse, error) {
	var department models.Department
	if err := s.db.
		Preload("Parent").
		Preload("Children").
		Preload("Users").
		First(&department, departmentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Department", "Department not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	return s.convertToDepartmentResponse(&department), nil
}

// UpdateDepartment 部署情報を更新
func (s *DepartmentService) UpdateDepartment(departmentID uuid.UUID, req UpdateDepartmentRequest) (*DepartmentResponse, error) {
	s.logger.Info("Updating department", map[string]interface{}{
		"department_id": departmentID,
	})

	// 部署存在確認
	var department models.Department
	if err := s.db.First(&department, departmentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Department", "Department not found")
		}
		return nil, errors.NewDatabaseError(err)
	}

	// 名前重複チェック（自分以外）
	if req.Name != nil {
		var existingDept models.Department
		if err := s.db.Where("name = ? AND id != ?", *req.Name, departmentID).First(&existingDept).Error; err == nil {
			return nil, errors.NewValidationError("name", "Department name already exists")
		} else if err != gorm.ErrRecordNotFound {
			return nil, errors.NewDatabaseError(err)
		}
	}

	// 親部署変更時の検証
	if req.ParentID != nil {
		// 自己参照チェック
		if *req.ParentID == departmentID {
			return nil, errors.NewValidationError("parent_id", "Department cannot be its own parent")
		}

		// 循環参照チェック
		if err := s.checkCircularReference(departmentID, *req.ParentID); err != nil {
			return nil, err
		}

		// 親部署存在確認
		var parent models.Department
		if err := s.db.First(&parent, *req.ParentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.NewNotFoundError("Department", "Parent department does not exist")
			}
			return nil, errors.NewDatabaseError(err)
		}

		// 階層深度チェック
		depth, err := s.calculateDepth(*req.ParentID)
		if err != nil {
			return nil, err
		}
		if depth >= 5 {
			return nil, errors.NewValidationError("parent_id", "Maximum hierarchy depth (5 levels) exceeded")
		}
	}

	// 更新用のマップを作成
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.ParentID != nil {
		updates["parent_id"] = *req.ParentID
	}

	// 更新実行
	if err := s.db.Model(&department).Updates(updates).Error; err != nil {
		s.logger.Error("Failed to update department", err, map[string]interface{}{
			"department_id": departmentID,
		})
		return nil, errors.NewDatabaseError(err)
	}

	s.logger.Info("Department updated successfully", map[string]interface{}{
		"department_id": departmentID,
	})

	return s.GetDepartment(departmentID)
}

// DeleteDepartment 部署を削除
func (s *DepartmentService) DeleteDepartment(departmentID uuid.UUID) error {
	s.logger.Info("Deleting department", map[string]interface{}{
		"department_id": departmentID,
	})

	// 部署存在確認
	var department models.Department
	if err := s.db.First(&department, departmentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewNotFoundError("Department", "Department not found")
		}
		return errors.NewDatabaseError(err)
	}

	// 子部署存在チェック
	var childCount int64
	if err := s.db.Model(&models.Department{}).Where("parent_id = ?", departmentID).Count(&childCount).Error; err != nil {
		return errors.NewDatabaseError(err)
	}
	if childCount > 0 {
		return errors.NewValidationError("department", "Cannot delete department with child departments")
	}

	// 所属ユーザー存在チェック
	var userCount int64
	if err := s.db.Model(&models.User{}).Where("department_id = ?", departmentID).Count(&userCount).Error; err != nil {
		return errors.NewDatabaseError(err)
	}
	if userCount > 0 {
		return errors.NewValidationError("department", "Cannot delete department with assigned users")
	}

	// 削除実行
	if err := s.db.Delete(&department).Error; err != nil {
		s.logger.Error("Failed to delete department", err, map[string]interface{}{
			"department_id": departmentID,
		})
		return errors.NewDatabaseError(err)
	}

	s.logger.Info("Department deleted successfully", map[string]interface{}{
		"department_id": departmentID,
	})

	return nil
}

// GetDepartments 部署一覧を取得
func (s *DepartmentService) GetDepartments(page, limit int, parentID *uuid.UUID, search string) (*DepartmentListResponse, error) {
	offset := (page - 1) * limit

	query := s.db.Model(&models.Department{}).
		Preload("Parent").
		Preload("Children")

	// 親部署フィルタ
	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
	}

	// 検索フィルタ
	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(name) LIKE ?", searchTerm)
	}

	// 総件数取得
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// データ取得
	var departments []models.Department
	if err := query.
		Offset(offset).
		Limit(limit).
		Order("name ASC").
		Find(&departments).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// レスポンス変換
	departmentResponses := make([]DepartmentResponse, len(departments))
	for i, dept := range departments {
		departmentResponses[i] = *s.convertToDepartmentResponse(&dept)
	}

	return &DepartmentListResponse{
		Departments: departmentResponses,
		Total:       total,
		Page:        page,
		Limit:       limit,
	}, nil
}

// GetDepartmentHierarchy 部署階層ツリーを取得
func (s *DepartmentService) GetDepartmentHierarchy() (*DepartmentHierarchyResponse, error) {
	// ルート部署を取得
	var rootDepartments []models.Department
	if err := s.db.Where("parent_id IS NULL").Order("name ASC").Find(&rootDepartments).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// 階層ツリーを構築
	hierarchy := make([]DepartmentHierarchyNode, len(rootDepartments))
	for i, dept := range rootDepartments {
		node, err := s.buildHierarchyNode(&dept)
		if err != nil {
			return nil, err
		}
		hierarchy[i] = *node
	}

	return &DepartmentHierarchyResponse{
		Departments: hierarchy,
	}, nil
}

// =============================================================================
// ヘルパーメソッド
// =============================================================================

// calculateDepth 指定された部署の階層深度を計算
func (s *DepartmentService) calculateDepth(departmentID uuid.UUID) (int, error) {
	depth := 1
	currentID := departmentID

	for {
		var dept models.Department
		if err := s.db.Select("parent_id").First(&dept, currentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				break
			}
			return 0, errors.NewDatabaseError(err)
		}

		if dept.ParentID == nil {
			break
		}

		depth++
		currentID = *dept.ParentID

		// 無限ループ防止
		if depth > 10 {
			return 0, errors.NewValidationError("hierarchy", "Invalid hierarchy structure detected")
		}
	}

	return depth, nil
}

// checkCircularReference 循環参照をチェック
func (s *DepartmentService) checkCircularReference(departmentID, newParentID uuid.UUID) error {
	visited := make(map[uuid.UUID]bool)
	currentID := newParentID

	for {
		if visited[currentID] {
			return errors.NewValidationError("parent_id", "Circular reference detected in department hierarchy")
		}

		if currentID == departmentID {
			return errors.NewValidationError("parent_id", "Circular reference detected in department hierarchy")
		}

		visited[currentID] = true

		var dept models.Department
		if err := s.db.Select("parent_id").First(&dept, currentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				break
			}
			return errors.NewDatabaseError(err)
		}

		if dept.ParentID == nil {
			break
		}

		currentID = *dept.ParentID
	}

	return nil
}

// buildHierarchyNode 階層ノードを再帰的に構築
func (s *DepartmentService) buildHierarchyNode(dept *models.Department) (*DepartmentHierarchyNode, error) {
	node := &DepartmentHierarchyNode{
		ID:   dept.ID,
		Name: dept.Name,
	}

	// 子部署を取得
	var children []models.Department
	if err := s.db.Where("parent_id = ?", dept.ID).Order("name ASC").Find(&children).Error; err != nil {
		return nil, errors.NewDatabaseError(err)
	}

	// 子ノードを再帰的に構築
	if len(children) > 0 {
		node.Children = make([]DepartmentHierarchyNode, len(children))
		for i, child := range children {
			childNode, err := s.buildHierarchyNode(&child)
			if err != nil {
				return nil, err
			}
			node.Children[i] = *childNode
		}
	}

	return node, nil
}

// convertToDepartmentResponse Departmentモデルをレスポンス形式に変換
func (s *DepartmentService) convertToDepartmentResponse(dept *models.Department) *DepartmentResponse {
	response := &DepartmentResponse{
		ID:        dept.ID,
		Name:      dept.Name,
		ParentID:  dept.ParentID,
		CreatedAt: dept.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// 親部署情報
	if dept.Parent != nil {
		response.Parent = &DepartmentBasicInfo{
			ID:   dept.Parent.ID,
			Name: dept.Parent.Name,
		}
	}

	// 子部署情報
	if len(dept.Children) > 0 {
		response.Children = make([]DepartmentBasicInfo, len(dept.Children))
		for i, child := range dept.Children {
			response.Children[i] = DepartmentBasicInfo{
				ID:   child.ID,
				Name: child.Name,
			}
		}
	}

	// 所属ユーザー情報
	if len(dept.Users) > 0 {
		response.Users = make([]UserBasicInfo, len(dept.Users))
		for i, user := range dept.Users {
			response.Users[i] = UserBasicInfo{
				ID:   user.ID,
				Name: user.Name,
			}
		}
	}

	return response
}
