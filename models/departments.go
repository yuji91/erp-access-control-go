package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Department 部門テーブル
type Department struct {
	BaseModel
	Name     string     `gorm:"not null" json:"name"`
	ParentID *uuid.UUID `gorm:"type:uuid;index" json:"parent_id,omitempty"`

	// リレーション
	Parent   *Department  `gorm:"foreignKey:ParentID;constraint:OnDelete:SET NULL" json:"parent,omitempty"`
	Children []Department `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Users    []User       `gorm:"foreignKey:DepartmentID" json:"users,omitempty"`
}

// TableName テーブル名を指定
func (Department) TableName() string {
	return "departments"
}

// BeforeCreate 作成前のバリデーション
func (d *Department) BeforeCreate(tx *gorm.DB) error {
	// 自己参照チェック
	if d.ParentID != nil && *d.ParentID == d.ID {
		return gorm.ErrInvalidData
	}
	return nil
}

// BeforeUpdate 更新前のバリデーション
func (d *Department) BeforeUpdate(tx *gorm.DB) error {
	// 自己参照チェック
	if d.ParentID != nil && *d.ParentID == d.ID {
		return gorm.ErrInvalidData
	}
	return nil
}

// =============================================================================
// 部門階層関連のメソッド
// =============================================================================

// IsRoot ルート部門かどうかを判定
func (d *Department) IsRoot() bool {
	return d.ParentID == nil
}

// HasParent 親部門を持つかどうかを判定
func (d *Department) HasParent() bool {
	return d.ParentID != nil
}

// GetAncestors 祖先部門を取得（階層上位）
func (d *Department) GetAncestors(db *gorm.DB) ([]Department, error) {
	var ancestors []Department
	query := `
		WITH RECURSIVE dept_ancestors AS (
			SELECT id, name, parent_id, 1 as level
			FROM departments WHERE id = ?
			UNION ALL
			SELECT d.id, d.name, d.parent_id, da.level + 1
			FROM departments d
			JOIN dept_ancestors da ON d.id = da.parent_id
		)
		SELECT id, name, parent_id FROM dept_ancestors WHERE level > 1
		ORDER BY level DESC
	`

	err := db.Raw(query, d.ID).Scan(&ancestors).Error
	return ancestors, err
}

// GetDescendants 子孫部門を取得（階層下位）
func (d *Department) GetDescendants(db *gorm.DB) ([]Department, error) {
	var descendants []Department
	query := `
		WITH RECURSIVE dept_descendants AS (
			SELECT id, name, parent_id, 1 as level
			FROM departments WHERE id = ?
			UNION ALL
			SELECT d.id, d.name, d.parent_id, dd.level + 1
			FROM departments d
			JOIN dept_descendants dd ON d.parent_id = dd.id
		)
		SELECT id, name, parent_id FROM dept_descendants WHERE level > 1
		ORDER BY level ASC
	`

	err := db.Raw(query, d.ID).Scan(&descendants).Error
	return descendants, err
}

// =============================================================================
// クエリ用ヘルパー関数
// =============================================================================

// FindDepartmentByID IDで部門を検索
func FindDepartmentByID(db *gorm.DB, id uuid.UUID) (*Department, error) {
	var dept Department
	err := db.Where("id = ?", id).First(&dept).Error
	if err != nil {
		return nil, err
	}
	return &dept, nil
}

// FindDepartmentsByParentID 親IDで子部門を検索
func FindDepartmentsByParentID(db *gorm.DB, parentID *uuid.UUID) ([]Department, error) {
	var departments []Department
	query := db.Model(&Department{})

	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}

	err := query.Find(&departments).Error
	return departments, err
}

// FindRootDepartments ルート部門を取得
func FindRootDepartments(db *gorm.DB) ([]Department, error) {
	var departments []Department
	err := db.Where("parent_id IS NULL").Find(&departments).Error
	return departments, err
}

// GetDepartmentHierarchy 部門階層をツリー構造で取得
func GetDepartmentHierarchy(db *gorm.DB) ([]Department, error) {
	var hierarchy []Department
	query := `
		WITH RECURSIVE dept_tree AS (
			SELECT id, name, parent_id, created_at, 1 as level, 
				   ARRAY[id] as path, name as full_path
			FROM departments WHERE parent_id IS NULL
			UNION ALL
			SELECT d.id, d.name, d.parent_id, d.created_at, dt.level + 1,
				   dt.path || d.id, dt.full_path || ' > ' || d.name
			FROM departments d
			JOIN dept_tree dt ON d.parent_id = dt.id
			WHERE NOT d.id = ANY(dt.path)
		)
		SELECT id, name, parent_id, created_at FROM dept_tree
		ORDER BY level, name
	`

	err := db.Raw(query).Scan(&hierarchy).Error
	return hierarchy, err
}
