# 📋 **テストヘルパー関数命名規則ガイドライン**

**更新日**: 2025/01/27  
**対象**: すべてのテストファイル (unit tests, integration tests)  
**目的**: テストヘルパー関数の命名競合防止・可読性向上・保守性確保

---

## 🎯 **命名規則の基本方針**

### **統一パターン**: `{動作}{リソース}For{テスト対象}Test`

#### **例**:
```go
// ✅ 推奨パターン
func createRoleForRoleTest(t *testing.T, db *gorm.DB, name string, parentID *uuid.UUID) *models.Role
func createPermissionForPermissionTest(t *testing.T, db *gorm.DB, module, action string) *models.Permission
func createDepartmentForDepartmentTest(t *testing.T, db *gorm.DB, name string, parentID *uuid.UUID) *models.Department
func createRoleForRoleIntegrationTest(t *testing.T, db *gorm.DB, name string, parentID *string) string

// ❌ 非推奨パターン（競合リスク）
func createTestRole(...)          // 複数ファイルで重複
func createTestPermission(...)    // 複数ファイルで重複
func createTestRoleViaDB(...)     // 命名パターン不統一
```

---

## 📝 **命名要素の定義**

### **1. 動作 (Action)**
| 動作 | 用途 | 例 |
|------|------|-----|
| `create` | リソース作成 | `createRoleForRoleTest` |
| `setup` | テスト環境構築 | `setupTestPermission` |
| `assign` | 関連付け作成 | `assignPermissionToRole` |
| `clear` | データクリア | `clearTestData` |

### **2. リソース (Resource)**
| リソース | 対象エンティティ | 例 |
|----------|-----------------|-----|
| `Role` | ロール | `createRoleForRoleTest` |
| `Permission` | 権限 | `createPermissionForPermissionTest` |
| `Department` | 部署 | `createDepartmentForDepartmentTest` |
| `User` | ユーザー | `createUserForRoleTest` |

### **3. テスト対象 (Test Target)**
| テスト対象 | 範囲 | 例 |
|------------|------|-----|
| `{Service}Test` | 単体テスト | `RoleTest`, `PermissionTest` |
| `{Service}IntegrationTest` | 統合テスト | `RoleIntegrationTest` |
| `{Handler}Test` | ハンドラーテスト | `RoleHandlerTest` |

---

## 🗂️ **ファイル別命名マップ**

### **単体テスト (services layer)**
| ファイル | Setup関数 | Create関数パターン |
|----------|-----------|-------------------|
| `role_test.go` | `setupTestRole` | `create{Resource}ForRoleTest` |
| `permission_test.go` | `setupTestPermission` | `create{Resource}ForPermissionTest` |
| `department_test.go` | `setupTestDepartment` | `create{Resource}ForDepartmentTest` |
| `user_test.go` | `setupTestUser` | `create{Resource}ForUserTest` |

### **統合テスト (handlers layer)**
| ファイル | Setup関数 | Create関数パターン |
|----------|-----------|-------------------|
| `role_integration_test.go` | `setupRoleIntegrationTest` | `create{Resource}ForRoleIntegrationTest` |
| `department_integration_test.go` | `setupTestDepartmentHandler` | `create{Resource}ForDepartmentIntegrationTest` |

---

## ✅ **実装済み関数一覧**

### **Role関連テスト**
```go
// internal/services/role_test.go
func setupTestRole(t *testing.T) (*RoleService, *gorm.DB)
func createRoleForRoleTest(t *testing.T, db *gorm.DB, name string, parentID *uuid.UUID) *models.Role
func createPermissionForRoleTest(t *testing.T, db *gorm.DB, module, action string) *models.Permission  
func createUserForRoleTest(t *testing.T, db *gorm.DB, name, email string, primaryRoleID *uuid.UUID) *models.User

// internal/handlers/role_integration_test.go
func setupRoleIntegrationTest(t *testing.T) (*gin.Engine, *services.RoleService, *gorm.DB)
func createRoleForRoleIntegrationTest(t *testing.T, db *gorm.DB, name string, parentID *string) string
```

### **Permission関連テスト**
```go
// internal/services/permission_test.go
func setupTestPermission(t *testing.T) (*PermissionService, *gorm.DB)
func createPermissionForPermissionTest(t *testing.T, db *gorm.DB, module, action string) *models.Permission
func createRoleForPermissionTest(t *testing.T, db *gorm.DB, name string, parentID *uuid.UUID) *models.Role
func assignPermissionToRole(t *testing.T, db *gorm.DB, roleID, permissionID uuid.UUID)
```

### **Department関連テスト**
```go
// internal/services/department_test.go
func setupTestDepartment(t *testing.T) (*DepartmentService, *gorm.DB)
func createDepartmentForDepartmentTest(t *testing.T, db *gorm.DB, name string, parentID *uuid.UUID) *models.Department
```

---

## 🔧 **実装ガイドライン**

### **1. 新しいヘルパー関数作成時**
```go
// テンプレート
func create{Resource}For{TestTarget}(
    t *testing.T, 
    db *gorm.DB, 
    /* リソース固有のパラメータ */
) *models.{Resource} {
    // 実装
}

// 具体例: User用ヘルパー
func createUserForUserTest(
    t *testing.T,
    db *gorm.DB,
    name, email string,
    departmentID, primaryRoleID *uuid.UUID,
) *models.User {
    user := &models.User{
        Name:          name,
        Email:         email,
        DepartmentID:  departmentID,
        PrimaryRoleID: primaryRoleID,
        Status:        models.UserStatusActive,
    }
    require.NoError(t, db.Create(user).Error)
    return user
}
```

### **2. 既存ヘルパー関数のリネーム**
```bash
# 一括置換例
sed -i '' 's/createTestRole(/createRoleForRoleTest(/g' internal/services/role_test.go
sed -i '' 's/createTestPermission(/createPermissionForPermissionTest(/g' internal/services/permission_test.go
```

### **3. Setup関数の統一**
```go
// 推奨パターン
func setupTest{Service}(t *testing.T) (*{Service}Service, *gorm.DB) {
    db := setupTestDB(t)
    // データクリア
    // サービス初期化
    return service, db
}

// 統合テスト用
func setup{Service}IntegrationTest(t *testing.T) (*gin.Engine, *services.{Service}Service, *gorm.DB) {
    // 統合テスト環境構築
    return router, service, db
}
```

---

## 🎯 **命名の利点**

### **1. 競合回避**
- ✅ 複数ファイル間での関数名重複防止
- ✅ コンパイルエラーの事前回避
- ✅ 明確な責任分界の確立

### **2. 可読性向上**
- ✅ 関数名からテスト対象が明確
- ✅ IDE補完での候補絞り込み
- ✅ テストファイル間の移動時の混乱防止

### **3. 保守性確保**
- ✅ 新しいテストファイル追加時の安全性
- ✅ リファクタリング時の影響範囲明確化
- ✅ コードレビューでの理解しやすさ

---

## 📋 **チェックリスト**

### **新しいテストファイル作成時**
- [ ] Setup関数は `setupTest{Service}` または `setup{Service}IntegrationTest` パターンか？
- [ ] Create関数は `create{Resource}For{TestTarget}` パターンか？
- [ ] 既存の同名関数との競合はないか？
- [ ] 統一されたパラメータ順序 (`t *testing.T, db *gorm.DB, ...`) か？

### **既存テストファイル修正時**
- [ ] 非推奨パターンの関数名はないか？
- [ ] 全ての呼び出し箇所が更新されているか？
- [ ] コンパイルエラーが発生していないか？
- [ ] テストが正常に実行されるか？

---

## 🔮 **今後の拡張**

### **予定される新規テスト**
```go
// User管理テスト (予定)
func setupTestUser(t *testing.T) (*UserService, *gorm.DB)
func createUserForUserTest(t *testing.T, db *gorm.DB, ...) *models.User
func createDepartmentForUserTest(t *testing.T, db *gorm.DB, ...) *models.Department

// PermissionHandler統合テスト (予定)  
func setupPermissionIntegrationTest(t *testing.T) (*gin.Engine, *services.PermissionService, *gorm.DB)
func createPermissionForPermissionIntegrationTest(t *testing.T, db *gorm.DB, ...) string
```

### **ヘルパー関数ライブラリ化 (検討中)**
```go
// pkg/testhelpers/helpers.go (検討中)
func CreateGenericResource[T any](t *testing.T, db *gorm.DB, resource T) T
func SetupGenericService[S any](t *testing.T, constructor func(*gorm.DB, *logger.Logger) S) (S, *gorm.DB)
```

---

**🎯 この命名規則により、テストコードの品質・保守性・開発効率が大幅に向上し、エンタープライズグレードのテスト実装基盤が確立されます。** 