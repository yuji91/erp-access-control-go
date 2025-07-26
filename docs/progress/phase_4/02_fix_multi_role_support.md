# Phase 4 Fix: 複数ロール対応

## 📋 **対応概要**

現在の単一ロールシステムを複数ロール対応に拡張します。期限付きロール機能も含めて、より柔軟で実用的な権限管理システムを構築します。

## 🎯 **対応範囲**

### **現在の課題**
- ユーザーは1つのロールのみ割り当て可能
- 期限付きロールが未対応
- 複雑な組織構造での権限管理が困難

### **対応後の機能**
- 複数ロールの同時割り当て
- 期限付きロール（開始日・終了日）
- ロール優先度管理
- 一時的権限の付与・取り消し

## 🗄️ **データベース設計変更**

### **1. 新規テーブル追加**

#### **user_roles テーブル**
```sql
CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    valid_from TIMESTAMPTZ DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    priority INT DEFAULT 1,
    is_active BOOLEAN DEFAULT TRUE,
    assigned_by UUID REFERENCES users(id),
    assigned_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT uq_user_roles_user_role UNIQUE(user_id, role_id),
    CONSTRAINT chk_user_roles_valid_period CHECK (valid_from < valid_to OR valid_to IS NULL),
    CONSTRAINT chk_user_roles_priority CHECK (priority > 0)
);

-- インデックス
CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX idx_user_roles_active_period ON user_roles(valid_from, valid_to) WHERE is_active = TRUE;
```

### **2. 既存テーブル変更**

#### **users テーブル**
```sql
-- 段階的移行のため、role_idを削除せずnullable化
ALTER TABLE users ALTER COLUMN role_id DROP NOT NULL;
ALTER TABLE users ADD COLUMN primary_role_id UUID REFERENCES roles(id);

-- 移行後はrole_idを削除予定
-- ALTER TABLE users DROP COLUMN role_id;
```

## 🏗️ **モデル設計変更**

### **1. UserRole モデル新規追加**

```go
// UserRole ユーザー・ロール関連テーブル
type UserRole struct {
    BaseModelWithUpdate
    UserID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
    RoleID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"role_id"`
    ValidFrom      time.Time  `gorm:"default:NOW()" json:"valid_from"`
    ValidTo        *time.Time `gorm:"default:null" json:"valid_to,omitempty"`
    Priority       int        `gorm:"default:1;check:priority > 0" json:"priority"`
    IsActive       bool       `gorm:"default:true" json:"is_active"`
    AssignedBy     *uuid.UUID `gorm:"type:uuid" json:"assigned_by,omitempty"`
    AssignedReason string     `gorm:"type:text" json:"assigned_reason,omitempty"`

    // リレーション
    User       User  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
    Role       Role  `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE" json:"role,omitempty"`
    AssignedByUser *User `gorm:"foreignKey:AssignedBy" json:"assigned_by_user,omitempty"`
}
```

### **2. User モデル変更**

```go
// User ユーザーテーブル（複数ロール対応版）
type User struct {
    BaseModelWithUpdate
    Name           string     `gorm:"not null" json:"name"`
    Email          string     `gorm:"uniqueIndex;not null" json:"email"`
    PasswordHash   string     `gorm:"not null" json:"-"`
    DepartmentID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"department_id"`
    PrimaryRoleID  *uuid.UUID `gorm:"type:uuid;index" json:"primary_role_id,omitempty"` // メインロール
    Status         UserStatus `gorm:"not null;default:'active'" json:"status"`

    // リレーション
    Department       Department        `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
    PrimaryRole      *Role             `gorm:"foreignKey:PrimaryRoleID" json:"primary_role,omitempty"`
    UserRoles        []UserRole        `gorm:"foreignKey:UserID" json:"user_roles,omitempty"`
    ActiveUserRoles  []UserRole        `gorm:"foreignKey:UserID;where:is_active = true AND (valid_to IS NULL OR valid_to > NOW())" json:"active_user_roles,omitempty"`
    // ... 既存フィールド
}
```

### **3. Role モデル変更**

```go
// Role ロールテーブル（複数ロール対応版）
type Role struct {
    BaseModel
    Name     string     `gorm:"not null" json:"name"`
    ParentID *uuid.UUID `gorm:"type:uuid;index" json:"parent_id,omitempty"`

    // リレーション
    Parent           *Role             `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
    Children         []Role            `gorm:"foreignKey:ParentID" json:"children,omitempty"`
    UserRoles        []UserRole        `gorm:"foreignKey:RoleID" json:"user_roles,omitempty"`
    PrimaryUsers     []User            `gorm:"foreignKey:PrimaryRoleID" json:"primary_users,omitempty"`
    Permissions      []Permission      `gorm:"many2many:role_permissions" json:"permissions,omitempty"`
}
```

## 🔧 **実装変更詳細**

### **1. 権限チェックロジック変更**

#### **権限集約メソッド**
```go
// GetAllPermissions 複数ロールから全権限を集約取得
func (u *User) GetAllPermissions(db *gorm.DB) ([]Permission, error) {
    var permissions []Permission
    
    // アクティブなロールの権限を全て取得
    query := `
        SELECT DISTINCT p.id, p.module, p.action, p.created_at
        FROM permissions p
        JOIN role_permissions rp ON p.id = rp.permission_id
        JOIN user_roles ur ON rp.role_id = ur.role_id
        WHERE ur.user_id = ? 
            AND ur.is_active = true
            AND ur.valid_from <= NOW()
            AND (ur.valid_to IS NULL OR ur.valid_to > NOW())
        ORDER BY p.module, p.action
    `
    
    err := db.Raw(query, u.ID).Scan(&permissions).Error
    return permissions, err
}

// GetActiveRoles アクティブなロールを取得
func (u *User) GetActiveRoles(db *gorm.DB) ([]Role, error) {
    var roles []Role
    
    err := db.Joins("JOIN user_roles ur ON roles.id = ur.role_id").
        Where("ur.user_id = ? AND ur.is_active = ? AND ur.valid_from <= ? AND (ur.valid_to IS NULL OR ur.valid_to > ?)", 
              u.ID, true, time.Now(), time.Now()).
        Order("ur.priority DESC, ur.created_at ASC").
        Find(&roles).Error
    
    return roles, err
}

// GetHighestPriorityRole 最高優先度のロールを取得
func (u *User) GetHighestPriorityRole(db *gorm.DB) (*Role, error) {
    var role Role
    
    err := db.Joins("JOIN user_roles ur ON roles.id = ur.role_id").
        Where("ur.user_id = ? AND ur.is_active = ? AND ur.valid_from <= ? AND (ur.valid_to IS NULL OR ur.valid_to > ?)", 
              u.ID, true, time.Now(), time.Now()).
        Order("ur.priority DESC, ur.created_at ASC").
        First(&role).Error
    
    if err != nil {
        return nil, err
    }
    return &role, nil
}
```

#### **ロール管理メソッド**
```go
// AssignRole ロールを割り当て
func (u *User) AssignRole(db *gorm.DB, roleID uuid.UUID, validFrom time.Time, validTo *time.Time, priority int, assignedBy uuid.UUID, reason string) error {
    userRole := UserRole{
        UserID:         u.ID,
        RoleID:         roleID,
        ValidFrom:      validFrom,
        ValidTo:        validTo,
        Priority:       priority,
        IsActive:       true,
        AssignedBy:     &assignedBy,
        AssignedReason: reason,
    }
    
    return db.Create(&userRole).Error
}

// RevokeRole ロールを取り消し
func (u *User) RevokeRole(db *gorm.DB, roleID uuid.UUID, revokedBy uuid.UUID, reason string) error {
    return db.Model(&UserRole{}).
        Where("user_id = ? AND role_id = ? AND is_active = ?", u.ID, roleID, true).
        Updates(map[string]interface{}{
            "is_active":       false,
            "valid_to":        time.Now(),
            "assigned_by":     revokedBy,
            "assigned_reason": reason,
            "updated_at":      time.Now(),
        }).Error
}

// UpdateRolePriority ロール優先度を更新
func (u *User) UpdateRolePriority(db *gorm.DB, roleID uuid.UUID, newPriority int) error {
    return db.Model(&UserRole{}).
        Where("user_id = ? AND role_id = ? AND is_active = ?", u.ID, roleID, true).
        Update("priority", newPriority).Error
}

// ExtendRole ロール期限を延長
func (u *User) ExtendRole(db *gorm.DB, roleID uuid.UUID, newValidTo *time.Time) error {
    return db.Model(&UserRole{}).
        Where("user_id = ? AND role_id = ? AND is_active = ?", u.ID, roleID, true).
        Update("valid_to", newValidTo).Error
}
```

### **2. サービス層変更**

#### **PermissionService 変更**
```go
// GetUserPermissions 複数ロールから権限を集約
func (s *PermissionService) GetUserPermissions(userID uuid.UUID) ([]string, error) {
    var permissions []string
    
    query := `
        SELECT DISTINCT CONCAT(p.module, ':', p.action) as permission
        FROM permissions p
        JOIN role_permissions rp ON p.id = rp.permission_id
        JOIN user_roles ur ON rp.role_id = ur.role_id
        WHERE ur.user_id = ? 
            AND ur.is_active = true
            AND ur.valid_from <= NOW()
            AND (ur.valid_to IS NULL OR ur.valid_to > NOW())
    `
    
    err := s.db.Raw(query, userID).Pluck("permission", &permissions).Error
    return permissions, err
}

// GetUserRoleHierarchyPermissions 階層ロール権限を含めて取得
func (s *PermissionService) GetUserRoleHierarchyPermissions(userID uuid.UUID) ([]string, error) {
    var permissions []string
    
    query := `
        WITH RECURSIVE role_hierarchy AS (
            -- アクティブなユーザーロール
            SELECT ur.role_id, ur.priority
            FROM user_roles ur
            WHERE ur.user_id = ? 
                AND ur.is_active = true
                AND ur.valid_from <= NOW()
                AND (ur.valid_to IS NULL OR ur.valid_to > NOW())
            
            UNION
            
            -- 親ロールを辿る
            SELECT r.parent_id, rh.priority
            FROM roles r
            JOIN role_hierarchy rh ON r.id = rh.role_id
            WHERE r.parent_id IS NOT NULL
        )
        SELECT DISTINCT CONCAT(p.module, ':', p.action) as permission
        FROM permissions p
        JOIN role_permissions rp ON p.id = rp.permission_id
        JOIN role_hierarchy rh ON rp.role_id = rh.role_id
        ORDER BY permission
    `
    
    err := s.db.Raw(query, userID).Pluck("permission", &permissions).Error
    return permissions, err
}
```

### **3. マイグレーション戦略**

#### **段階的移行スクリプト**
```sql
-- Phase 1: 新しいテーブル作成
-- （上記のuser_rolesテーブル作成）

-- Phase 2: 既存データ移行
INSERT INTO user_roles (user_id, role_id, valid_from, priority, is_active, assigned_reason)
SELECT id, role_id, created_at, 1, true, 'Migration from single role'
FROM users 
WHERE role_id IS NOT NULL;

-- Phase 3: primary_role_id設定
UPDATE users SET primary_role_id = role_id WHERE role_id IS NOT NULL;

-- Phase 4: 段階的にrole_idをnullable化（すでに実行済み想定）
-- Phase 5: 最終的にrole_id削除（後のフェーズで実行）
```

## 🧪 **テスト計画**

### **1. 単体テスト**
```go
// ロール割り当てテスト
func TestUserRole_AssignRole(t *testing.T) {
    // 複数ロール割り当て
    // 期限付きロール
    // 優先度設定
    // 重複ロール割り当て防止
}

// 権限取得テスト  
func TestUser_GetAllPermissions(t *testing.T) {
    // 複数ロール権限集約
    // 階層ロール権限継承
    // 期限切れロール除外
    // 非アクティブロール除外
}

// 権限チェックテスト
func TestPermissionService_CheckPermission(t *testing.T) {
    // 複数ロール権限チェック
    // 最高優先度ロール適用
    // 期間限定権限
}
```

### **2. 統合テスト**
- API エンドポイントでの複数ロール動作確認
- 認証ミドルウェアの複数ロール対応
- 権限チェックパフォーマンステスト

## ⚠️ **注意事項・リスク**

### **1. 後方互換性**
- 既存の単一ロール前提のコードが一時的に動作する
- `User.RoleID` から `User.PrimaryRoleID` への段階移行
- 既存のAPIレスポンスフォーマット維持

### **2. パフォーマンス**
- 複数ロール権限取得のクエリ複雑化
- N+1 問題の回避（Preload使用）
- 権限キャッシュの導入検討

### **3. セキュリティ**
- 権限昇格の防止（高優先度ロール割り当て制限）
- 期限切れロールの自動無効化
- ロール変更の監査ログ強化

## 📋 **実装手順**

### **Phase 1: データベース・モデル準備**
1. `user_roles` テーブル作成
2. `UserRole` モデル実装
3. `User`, `Role` モデル更新
4. 既存データ移行スクリプト実行

### **Phase 2: サービス層実装**
1. 複数ロール権限取得メソッド実装
2. ロール管理メソッド実装  
3. `PermissionService` 更新
4. 既存メソッドの後方互換性確保

### **Phase 3: ミドルウェア・API更新**
1. 認証ミドルウェアの複数ロール対応
2. JWT Claims への複数ロール情報追加
3. API エンドポイントでのロール管理

### **Phase 4: テスト・検証**
1. 単体テスト実装
2. 統合テスト実装
3. パフォーマンステスト
4. セキュリティテスト

### **Phase 5: 旧システム削除**
1. `User.RoleID` フィールド削除
2. 旧コードのクリーンアップ
3. 最終マイグレーション実行

## 🎯 **期待効果**

### **機能面**
- 複雑な組織構造への対応
- 一時的権限付与の柔軟性
- ロール管理の細粒度制御

### **運用面**
- 権限管理の自動化
- 監査トレーサビリティの向上
- セキュリティポリシーの強化

この実装により、より実用的で柔軟なアクセス制御システムが実現できます。
