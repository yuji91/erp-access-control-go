package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"erp-access-control-go/models"
	"erp-access-control-go/pkg/errors"
	"erp-access-control-go/pkg/logger"
)

// テスト用ヘルパー関数
func setupTestDepartment(t *testing.T) (*DepartmentService, *gorm.DB) {
	db := setupTestDB(t)
	log := logger.NewLogger()
	return NewDepartmentService(db, log), db
}

func createTestDepartment(t *testing.T, db *gorm.DB, name string, parentID *uuid.UUID) *models.Department {
	dept := &models.Department{
		Name:     name,
		ParentID: parentID,
	}
	require.NoError(t, db.Create(dept).Error)
	return dept
}

// TestDepartmentService_CreateDepartment 部署作成のテスト
func TestDepartmentService_CreateDepartment(t *testing.T) {
	svc, db := setupTestDepartment(t)

	t.Run("正常系: 親部署なしで作成", func(t *testing.T) {
		req := CreateDepartmentRequest{
			Name: "営業部",
		}

		resp, err := svc.CreateDepartment(req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, req.Name, resp.Name)
		assert.Nil(t, resp.ParentID)
	})

	t.Run("正常系: 親部署ありで作成", func(t *testing.T) {
		parent := createTestDepartment(t, db, "本社", nil)
		req := CreateDepartmentRequest{
			Name:     "東京支社",
			ParentID: &parent.ID,
		}

		resp, err := svc.CreateDepartment(req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, req.Name, resp.Name)
		assert.Equal(t, parent.ID, *resp.ParentID)
	})

	t.Run("異常系: 名前重複", func(t *testing.T) {
		name := "総務部"
		createTestDepartment(t, db, name, nil)

		req := CreateDepartmentRequest{
			Name: name,
		}

		resp, err := svc.CreateDepartment(req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.True(t, errors.IsValidationError(err))
	})

	t.Run("異常系: 存在しない親部署", func(t *testing.T) {
		invalidID := uuid.New()
		req := CreateDepartmentRequest{
			Name:     "経理部",
			ParentID: &invalidID,
		}

		resp, err := svc.CreateDepartment(req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.True(t, errors.IsNotFound(err))
	})
}

// TestDepartmentService_GetDepartment 部署取得のテスト
func TestDepartmentService_GetDepartment(t *testing.T) {
	svc, db := setupTestDepartment(t)

	t.Run("正常系: 部署取得", func(t *testing.T) {
		dept := createTestDepartment(t, db, "人事部", nil)

		resp, err := svc.GetDepartment(dept.ID)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, dept.ID, resp.ID)
		assert.Equal(t, dept.Name, resp.Name)
	})

	t.Run("正常系: 親子関係付きで取得", func(t *testing.T) {
		parent := createTestDepartment(t, db, "開発本部", nil)
		child := createTestDepartment(t, db, "システム開発部", &parent.ID)

		resp, err := svc.GetDepartment(child.ID)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, child.ID, resp.ID)
		assert.Equal(t, parent.ID, *resp.ParentID)
		assert.NotNil(t, resp.Parent)
		assert.Equal(t, parent.Name, resp.Parent.Name)
	})

	t.Run("異常系: 存在しない部署", func(t *testing.T) {
		resp, err := svc.GetDepartment(uuid.New())
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.True(t, errors.IsNotFound(err))
	})
}

// TestDepartmentService_UpdateDepartment 部署更新のテスト
func TestDepartmentService_UpdateDepartment(t *testing.T) {
	svc, db := setupTestDepartment(t)

	t.Run("正常系: 名前更新", func(t *testing.T) {
		dept := createTestDepartment(t, db, "旧部署名", nil)
		newName := "新部署名"
		req := UpdateDepartmentRequest{
			Name: &newName,
		}

		resp, err := svc.UpdateDepartment(dept.ID, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, newName, resp.Name)
	})

	t.Run("正常系: 親部署変更", func(t *testing.T) {
		dept := createTestDepartment(t, db, "移動対象部署", nil)
		newParent := createTestDepartment(t, db, "新親部署", nil)
		req := UpdateDepartmentRequest{
			ParentID: &newParent.ID,
		}

		resp, err := svc.UpdateDepartment(dept.ID, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, newParent.ID, *resp.ParentID)
	})

	t.Run("異常系: 循環参照", func(t *testing.T) {
		parent := createTestDepartment(t, db, "親部署", nil)
		child := createTestDepartment(t, db, "子部署", &parent.ID)
		req := UpdateDepartmentRequest{
			ParentID: &child.ID,
		}

		resp, err := svc.UpdateDepartment(parent.ID, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.True(t, errors.IsValidationError(err))
	})

	t.Run("異常系: 深度制限超過", func(t *testing.T) {
		// 5階層の部署を作成
		var parentID *uuid.UUID
		var lastDept *models.Department
		for i := 0; i < 5; i++ {
			lastDept = createTestDepartment(t, db, "Dept"+string(rune('A'+i)), parentID)
			parentID = &lastDept.ID
		}

		// 6階層目を作成しようとする
		newDept := createTestDepartment(t, db, "最下層部署", nil)
		req := UpdateDepartmentRequest{
			ParentID: parentID,
		}

		resp, err := svc.UpdateDepartment(newDept.ID, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.True(t, errors.IsValidationError(err))
	})
}

// TestDepartmentService_DeleteDepartment 部署削除のテスト
func TestDepartmentService_DeleteDepartment(t *testing.T) {
	svc, db := setupTestDepartment(t)

	t.Run("正常系: 部署削除", func(t *testing.T) {
		dept := createTestDepartment(t, db, "削除対象部署", nil)

		err := svc.DeleteDepartment(dept.ID)
		require.NoError(t, err)

		// 削除確認
		var found models.Department
		err = db.First(&found, dept.ID).Error
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("異常系: 子部署が存在する場合", func(t *testing.T) {
		parent := createTestDepartment(t, db, "親部署", nil)
		createTestDepartment(t, db, "子部署", &parent.ID)

		err := svc.DeleteDepartment(parent.ID)
		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
	})

	t.Run("異常系: 所属ユーザーが存在する場合", func(t *testing.T) {
		dept := createTestDepartment(t, db, "社員所属部署", nil)

		// ユーザーを作成
		user := &models.User{
			Name:         "テストユーザー",
			DepartmentID: dept.ID,
		}
		require.NoError(t, db.Create(user).Error)

		err := svc.DeleteDepartment(dept.ID)
		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
	})
}

// TestDepartmentService_GetDepartments 部署一覧取得のテスト
func TestDepartmentService_GetDepartments(t *testing.T) {
	svc, db := setupTestDepartment(t)

	t.Run("正常系: 全部署取得", func(t *testing.T) {
		// テストデータ作成
		createTestDepartment(t, db, "部署1", nil)
		createTestDepartment(t, db, "部署2", nil)
		createTestDepartment(t, db, "部署3", nil)

		resp, err := svc.GetDepartments(1, 10, nil, "")
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int64(3), resp.Total)
		assert.Len(t, resp.Departments, 3)
	})

	t.Run("正常系: 親部署でフィルタ", func(t *testing.T) {
		parent := createTestDepartment(t, db, "親部署", nil)
		createTestDepartment(t, db, "子部署1", &parent.ID)
		createTestDepartment(t, db, "子部署2", &parent.ID)

		resp, err := svc.GetDepartments(1, 10, &parent.ID, "")
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int64(2), resp.Total)
		for _, dept := range resp.Departments {
			assert.Equal(t, parent.ID, *dept.ParentID)
		}
	})

	t.Run("正常系: 名前で検索", func(t *testing.T) {
		createTestDepartment(t, db, "検索用部署A", nil)
		createTestDepartment(t, db, "検索用部署B", nil)
		createTestDepartment(t, db, "その他部署", nil)

		resp, err := svc.GetDepartments(1, 10, nil, "検索用")
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int64(2), resp.Total)
		for _, dept := range resp.Departments {
			assert.Contains(t, dept.Name, "検索用")
		}
	})

	t.Run("正常系: ページング", func(t *testing.T) {
		// 5件のテストデータ作成
		for i := 0; i < 5; i++ {
			createTestDepartment(t, db, "Page部署"+string(rune('A'+i)), nil)
		}

		// 2件ずつ取得
		resp1, err := svc.GetDepartments(1, 2, nil, "Page")
		require.NoError(t, err)
		assert.Len(t, resp1.Departments, 2)
		assert.Equal(t, int64(5), resp1.Total)

		resp2, err := svc.GetDepartments(2, 2, nil, "Page")
		require.NoError(t, err)
		assert.Len(t, resp2.Departments, 2)

		resp3, err := svc.GetDepartments(3, 2, nil, "Page")
		require.NoError(t, err)
		assert.Len(t, resp3.Departments, 1)
	})
}

// TestDepartmentService_GetDepartmentHierarchy 部署階層取得のテスト
func TestDepartmentService_GetDepartmentHierarchy(t *testing.T) {
	svc, db := setupTestDepartment(t)

	t.Run("正常系: 階層構造取得", func(t *testing.T) {
		// テスト用の階層構造を作成
		root := createTestDepartment(t, db, "本社", nil)
		div1 := createTestDepartment(t, db, "事業部1", &root.ID)
		div2 := createTestDepartment(t, db, "事業部2", &root.ID)
		createTestDepartment(t, db, "部署1", &div1.ID)
		createTestDepartment(t, db, "部署2", &div1.ID)
		createTestDepartment(t, db, "部署3", &div2.ID)

		resp, err := svc.GetDepartmentHierarchy()
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Departments, 1) // ルート部署

		rootNode := resp.Departments[0]
		assert.Equal(t, "本社", rootNode.Name)
		assert.Len(t, rootNode.Children, 2) // 2つの事業部

		// 事業部1の検証
		div1Node := findNodeByName(rootNode.Children, "事業部1")
		require.NotNil(t, div1Node)
		assert.Len(t, div1Node.Children, 2) // 2つの部署

		// 事業部2の検証
		div2Node := findNodeByName(rootNode.Children, "事業部2")
		require.NotNil(t, div2Node)
		assert.Len(t, div2Node.Children, 1) // 1つの部署
	})

	t.Run("正常系: 空の階層構造", func(t *testing.T) {
		// データを全て削除
		db.Exec("DELETE FROM departments")

		resp, err := svc.GetDepartmentHierarchy()
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Departments, 0)
	})
}

// findNodeByName 指定された名前のノードを探す補助関数
func findNodeByName(nodes []DepartmentHierarchyNode, name string) *DepartmentHierarchyNode {
	for i, node := range nodes {
		if node.Name == name {
			return &nodes[i]
		}
	}
	return nil
}

// TestDepartmentService_ValidationRules バリデーションルールのテスト
func TestDepartmentService_ValidationRules(t *testing.T) {
	svc, db := setupTestDepartment(t)

	t.Run("名前の長さ制限", func(t *testing.T) {
		// 1文字の名前
		req1 := CreateDepartmentRequest{
			Name: "A",
		}
		_, err := svc.CreateDepartment(req1)
		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))

		// 101文字の名前
		longName := string(make([]rune, 101))
		req2 := CreateDepartmentRequest{
			Name: longName,
		}
		_, err = svc.CreateDepartment(req2)
		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
	})

	t.Run("階層深度制限", func(t *testing.T) {
		// 5階層の部署を作成
		var parentID *uuid.UUID
		for i := 0; i < 5; i++ {
			dept := createTestDepartment(t, db, "Depth"+string(rune('A'+i)), parentID)
			parentID = &dept.ID
		}

		// 6階層目を作成しようとする
		req := CreateDepartmentRequest{
			Name:     "TooDeep",
			ParentID: parentID,
		}
		_, err := svc.CreateDepartment(req)
		assert.Error(t, err)
		assert.True(t, errors.IsValidationError(err))
	})
}

// TestDepartmentService_ErrorHandling エラーハンドリングのテスト
func TestDepartmentService_ErrorHandling(t *testing.T) {
	_, _ = setupTestDepartment(t)

	t.Run("バリデーションエラー", func(t *testing.T) {
		err := errors.NewValidationError("name", "Department name already exists")
		assert.Contains(t, err.Error(), "name")
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("NotFoundエラー", func(t *testing.T) {
		err := errors.NewNotFoundError("Department", "Department not found")
		assert.Contains(t, err.Error(), "Department")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("データベースエラー", func(t *testing.T) {
		err := errors.NewDatabaseError(gorm.ErrRecordNotFound)
		assert.Contains(t, err.Error(), "database")
	})
}
