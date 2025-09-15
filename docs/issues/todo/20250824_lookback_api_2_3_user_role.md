# ユーザーロール割り当て API - 詳細解説 (2.3)

## 概要

このドキュメントでは、ERPアクセス制御システムのユーザーロール割り当てAPI（`POST /api/v1/users/roles`）の実装について、コードベースを詳細に解説します。このAPIは複数ロール対応の中核機能で、期限付き・優先度付きロール割り当て、重複チェック、監査ログ機能を提供します。

---

## 1. アーキテクチャ構成

### 1.1 レイヤー構造

```
HTTP Request → ハンドラ (UserRoleHandler) → サービス (UserRoleService) → モデル (UserRole) → データベース
```

#### 各レイヤーの責務
- **ハンドラレイヤー**: リクエスト処理、認証チェック、デフォルト値設定、レスポンス変換
- **サービスレイヤー**: ビジネスロジック、存在確認、重複チェック、データ整合性保証
- **モデルレイヤー**: 期限・優先度管理、バリデーション、リレーション定義

### 1.2 関連ファイル構成

```
internal/handlers/user_role.go:28-36       # AssignRoleRequest 構造体
internal/handlers/user_role.go:45-59       # UserRoleResponse 構造体  
internal/handlers/user_role.go:61-101      # AssignRole ハンドラ
internal/services/user_role.go:25-86       # AssignRole サービスメソッド
models/user_roles.go:10-26                 # UserRole モデル定義
models/users.go:345-359                    # User.AssignRole ヘルパーメソッド
migrations/02_add_user_roles_table.sql     # データベーステーブル定義
cmd/server/main.go:672                     # ルート設定
```

## 2. リクエスト構造体とバリデーション

### 2.1 AssignRoleRequest 構造体

```go
type AssignRoleRequest struct {
    UserID    uuid.UUID  `json:"user_id" binding:"required"`
    RoleID    uuid.UUID  `json:"role_id" binding:"required"`
    ValidFrom *time.Time `json:"valid_from,omitempty"`
    ValidTo   *time.Time `json:"valid_to,omitempty"`
    Priority  int        `json:"priority" binding:"min=1"`
    Reason    string     `json:"reason,omitempty"`
}
```

**場所**: `internal/handlers/user_role.go:28-36`

### 2.2 バリデーション仕様

| フィールド | 制約 | 説明 |
|-----------|------|------|
| `user_id` | required, UUID形式 | 割り当て対象のユーザーID |
| `role_id` | required, UUID形式 | 割り当てるロールID |
| `valid_from` | optional, 日時形式 | ロール有効開始日時（デフォルト：現在時刻） |
| `valid_to` | optional, 日時形式 | ロール有効終了日時（NULL可能：無期限） |
| `priority` | optional, 正整数 | ロール優先度（デフォルト：1） |
| `reason` | optional, 文字列 | 割り当て理由（監査ログ用） |

### 2.3 UserRoleResponse 構造体

```go
type UserRoleResponse struct {
    ID             uuid.UUID  `json:"id"`
    UserID         uuid.UUID  `json:"user_id"`
    RoleID         uuid.UUID  `json:"role_id"`
    RoleName       string     `json:"role_name"`
    ValidFrom      time.Time  `json:"valid_from"`
    ValidTo        *time.Time `json:"valid_to,omitempty"`
    Priority       int        `json:"priority"`
    IsActive       bool       `json:"is_active"`
    AssignedBy     *uuid.UUID `json:"assigned_by,omitempty"`
    AssignedReason string     `json:"assigned_reason,omitempty"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
}
```

**場所**: `internal/handlers/user_role.go:45-59`

### 2.4 リクエスト例

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "role_id": "987fcdeb-51d2-43a1-b456-426614174111",
  "valid_from": "2025-01-24T10:00:00Z",
  "valid_to": "2025-12-31T23:59:59Z",
  "priority": 5,
  "reason": "プロジェクトマネージャーへの昇格"
}
```

## 3. ハンドラレイヤーの実装詳細

### 3.1 AssignRole ハンドラの全体構造

```go
func (h *UserRoleHandler) AssignRole(c *gin.Context) {
    // 1. リクエストバインディング
    // 2. 認証ユーザー取得
    // 3. デフォルト値設定
    // 4. サービス層呼び出し
    // 5. レスポンス変換
    // 6. HTTPレスポンス返却
}
```

**場所**: `internal/handlers/user_role.go:61-101`

### 3.2 詳細なフロー解説

#### ① リクエストバインディング (`internal/handlers/user_role.go:63-67`)

```go
var req AssignRoleRequest
if err := c.ShouldBindJSON(&req); err != nil {
    c.Error(errors.NewValidationError("request", "Invalid request format"))
    return
}
```

- **JSON バインディング**でリクエストを構造体にマッピング
- **バリデーションエラー**は自動的にチェック（binding タグ）
- エラー時はカスタムエラー型で統一的な処理

#### ② 認証ユーザー取得 (`internal/handlers/user_role.go:69-74`)

```go
assignedBy, exists := c.Get("user_id")
if !exists {
    c.Error(errors.NewAuthenticationError("Authentication required"))
    return
}
```

- **JWTミドルウェア**で設定されたユーザーIDを取得
- **監査ログ**用に実行者の情報を記録
- 認証エラー時は適切なエラーレスポンス

#### ③ デフォルト値設定 (`internal/handlers/user_role.go:76-83`)

```go
// デフォルト値設定
validFrom := time.Now()
if req.ValidFrom != nil {
    validFrom = *req.ValidFrom
}
if req.Priority == 0 {
    req.Priority = 1
}
```

- **ValidFrom**: 未指定時は現在時刻を設定
- **Priority**: 未指定時はデフォルト優先度（1）を設定
- **安全なデフォルト値**でシステムの安定性を保証

#### ④ サービス層呼び出し (`internal/handlers/user_role.go:85-97`)

```go
userRole, err := h.userRoleService.AssignRole(
    req.UserID,
    req.RoleID,
    validFrom,
    req.ValidTo,
    req.Priority,
    assignedBy.(uuid.UUID),
    req.Reason,
)
if err != nil {
    c.Error(err)
    return
}
```

- **ビジネスロジック**をサービス層に移譲
- **型安全性**を保証（assignedBy の型アサーション）
- エラーハンドリングの統一

#### ⑤ レスポンス変換と返却 (`internal/handlers/user_role.go:99-100`)

```go
response := h.convertToResponse(userRole)
c.JSON(http.StatusCreated, response)
```

- **convertToResponse**ヘルパーでモデルをレスポンス形式に変換
- **201 Created**で作成成功を通知
- 構造化レスポンスでフロントエンド処理を支援

## 4. サービスレイヤーの検証ロジック

### 4.1 AssignRole サービスメソッドの全体構造

```go
func (s *UserRoleService) AssignRole(
    userID, roleID uuid.UUID,
    validFrom time.Time,
    validTo *time.Time,
    priority int,
    assignedBy uuid.UUID,
    reason string,
) (*models.UserRole, error) {
    // 1. ユーザー存在確認
    // 2. ロール存在確認
    // 3. 重複チェック（アクティブなロール）
    // 4. UserRole作成
    // 5. データベース保存
    // 6. ロール情報Preload
    // 7. 作成されたUserRole返却
}
```

**場所**: `internal/services/user_role.go:25-86`

### 4.2 詳細なビジネスロジック解説

#### ① ユーザー存在確認 (`internal/services/user_role.go:34-41`)

```go
var user models.User
if err := s.db.First(&user, userID).Error; err != nil {
    if err == gorm.ErrRecordNotFound {
        return nil, errors.NewNotFoundError("User", "User does not exist")
    }
    return nil, errors.NewDatabaseError(err)
}
```

- **外部キー制約**の事前バリデーション
- **存在しないリソース**への参照を防止
- **カスタムエラー型**で詳細なエラー情報を提供

#### ② ロール存在確認 (`internal/services/user_role.go:43-50`)

```go
var role models.Role
if err := s.db.First(&role, roleID).Error; err != nil {
    if err == gorm.ErrRecordNotFound {
        return nil, errors.NewNotFoundError("Role", "Role does not exist")
    }
    return nil, errors.NewDatabaseError(err)
}
```

- **ロールベースアクセス制御**の前提条件チェック
- **有効なロール**のみ割り当て可能
- **データ整合性**の保証

#### ③ 重複チェック（アクティブなロール） (`internal/services/user_role.go:52-62`)

```go
var count int64
err := s.db.Model(&models.UserRole{}).
    Where("user_id = ? AND role_id = ? AND is_active = ?", userID, roleID, true).
    Count(&count).Error
if err != nil {
    return nil, errors.NewDatabaseError(err)
}
if count > 0 {
    return nil, errors.NewValidationError("role", "User already has this role assigned")
}
```

- **重複割り当て**の防止
- **アクティブなロール**のみをチェック対象
- **ビジネスルール**の徹底

#### ④ UserRole作成 (`internal/services/user_role.go:64-74`)

```go
userRole := &models.UserRole{
    UserID:         userID,
    RoleID:         roleID,
    ValidFrom:      validFrom,
    ValidTo:        validTo,
    Priority:       priority,
    IsActive:       true,
    AssignedBy:     &assignedBy,
    AssignedReason: reason,
}
```

- **構造体初期化**で安全にモデルを作成
- **監査ログ項目**（AssignedBy, AssignedReason）を記録
- **アクティブフラグ**の設定

#### ⑤ データベース保存とPreload (`internal/services/user_role.go:76-85`)

```go
if err := s.db.Create(userRole).Error; err != nil {
    return nil, errors.NewDatabaseError(err)
}

// ロール情報をPreload
if err := s.db.Preload("Role").First(userRole, userRole.ID).Error; err != nil {
    return nil, errors.NewDatabaseError(err)
}
```

- **データベース保存**でレコード作成
- **Preload**で関連データ（Role）を効率的に取得
- **完全なレスポンス**のためのデータ準備

## 5. UserRole モデルの設計とデータベース制約

### 5.1 UserRole モデル構造

```go
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
    User           User  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
    Role           Role  `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE" json:"role,omitempty"`
    AssignedByUser *User `gorm:"foreignKey:AssignedBy" json:"assigned_by_user,omitempty"`
}
```

**場所**: `models/user_roles.go:10-26`

### 5.2 データベース制約とバリデーション

#### ① インデックスとパフォーマンス最適化

```sql
-- 複合インデックス（検索性能向上）
CREATE INDEX idx_user_roles_user_id_role_id ON user_roles(user_id, role_id);
CREATE INDEX idx_user_roles_active_valid ON user_roles(is_active, valid_from, valid_to);
```

#### ② CHECK制約

```sql
-- 優先度は正の整数のみ
CHECK (priority > 0)

-- 期限の妥当性（ValidFrom <= ValidTo）
CHECK (valid_to IS NULL OR valid_from <= valid_to)
```

#### ③ 外部キー制約

```sql
-- カスケード削除
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE

-- 割り当て者（NULL許可）
FOREIGN KEY (assigned_by) REFERENCES users(id)
```

### 5.3 ビジネスルールとバリデーション

#### ① BeforeCreate フック (`models/user_roles.go:33-53`)

```go
func (ur *UserRole) BeforeCreate(tx *gorm.DB) error {
    // 期限の妥当性チェック
    if ur.ValidTo != nil && ur.ValidFrom.After(*ur.ValidTo) {
        return gorm.ErrInvalidValue
    }

    // 重複チェック（同じユーザー・ロールの組み合わせでアクティブなレコード）
    var count int64
    err := tx.Model(&UserRole{}).
        Where("user_id = ? AND role_id = ? AND is_active = ?", ur.UserID, ur.RoleID, true).
        Count(&count).Error
    if err != nil {
        return err
    }
    if count > 0 {
        return gorm.ErrDuplicatedKey
    }

    return nil
}
```

- **期限の妥当性**をアプリケーションレベルでチェック
- **重複防止**のためのカスタムバリデーション
- **データベース制約**とアプリケーションロジックの二重チェック

#### ② UserRole ヘルパーメソッド

```go
// IsValidNow 現在時刻で有効かどうかを判定
func (ur *UserRole) IsValidNow() bool {
    now := time.Now()
    return ur.IsActive && 
           ur.ValidFrom.Before(now) && 
           (ur.ValidTo == nil || ur.ValidTo.After(now))
}

// IsExpired 期限切れかどうかを判定
func (ur *UserRole) IsExpired() bool {
    if ur.ValidTo == nil {
        return false
    }
    return ur.ValidTo.Before(time.Now())
}
```

**場所**: `models/user_roles.go:69-83`

### 5.4 複数ロール管理の特徴

✅ **期限管理**: 開始日時・終了日時による有効期間制御  
✅ **優先度制御**: 数値による重要度管理（高い数値 = 高優先度）  
✅ **アクティブ管理**: 論理削除によるロール無効化  
✅ **監査ログ**: 割り当て者・理由の記録  
✅ **重複防止**: 同一ユーザー・ロールの重複割り当て防止  
✅ **カスケード削除**: ユーザー・ロール削除時の整合性保証  

## 6. レスポンス構造とAPIテスト例

### 6.1 正常レスポンス例

#### HTTPステータス: 201 Created

```json
{
  "id": "123e4567-e89b-12d3-a456-426614174001",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "role_id": "987fcdeb-51d2-43a1-b456-426614174111",
  "role_name": "プロジェクトマネージャー",
  "valid_from": "2025-01-24T10:00:00Z",
  "valid_to": "2025-12-31T23:59:59Z",
  "priority": 5,
  "is_active": true,
  "assigned_by": "660f9511-f3ac-52e5-b827-557766551222",
  "assigned_reason": "プロジェクトマネージャーへの昇格",
  "created_at": "2025-01-24T10:30:00Z",
  "updated_at": "2025-01-24T10:30:00Z"
}
```

### 6.2 エラーレスポンス例

#### 400 Bad Request (重複エラー)

```json
{
  "error": {
    "type": "VALIDATION_ERROR",
    "field": "role",
    "message": "User already has this role assigned",
    "details": "This user already has an active assignment for the specified role"
  }
}
```

#### 404 Not Found (ユーザー未存在)

```json
{
  "error": {
    "type": "NOT_FOUND_ERROR",
    "resource": "User",
    "message": "User does not exist",
    "details": "The specified user ID was not found"
  }
}
```

#### 404 Not Found (ロール未存在)

```json
{
  "error": {
    "type": "NOT_FOUND_ERROR",
    "resource": "Role",
    "message": "Role does not exist",
    "details": "The specified role ID was not found"
  }
}
```

### 6.3 curlテスト例

#### 基本的なロール割り当て

```bash
curl -X POST http://localhost:8080/api/v1/users/roles \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "role_id": "987fcdeb-51d2-43a1-b456-426614174111",
    "priority": 3,
    "reason": "新プロジェクト参加"
  }'
```

#### 期限付きロール割り当て

```bash
curl -X POST http://localhost:8080/api/v1/users/roles \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "role_id": "987fcdeb-51d2-43a1-b456-426614174111",
    "valid_from": "2025-02-01T00:00:00Z",
    "valid_to": "2025-06-30T23:59:59Z",
    "priority": 5,
    "reason": "期間限定プロジェクトリーダー"
  }'
```

#### 高優先度ロール割り当て

```bash
curl -X POST http://localhost:8080/api/v1/users/roles \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "role_id": "998gdebc-62e3-54b2-c567-537725285333",
    "priority": 10,
    "reason": "緊急対応チームリーダー"
  }'
```

### 6.4 JavaScript（フロントエンド）使用例

```javascript
// ロール割り当て関数
async function assignUserRole(assignment) {
  const response = await fetch('/api/v1/users/roles', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      user_id: assignment.userId,
      role_id: assignment.roleId,
      valid_from: assignment.validFrom,
      valid_to: assignment.validTo,
      priority: assignment.priority || 1,
      reason: assignment.reason
    })
  });
  
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error.message);
  }
  
  return await response.json();
}

// 使用例
try {
  const result = await assignUserRole({
    userId: '550e8400-e29b-41d4-a716-446655440000',
    roleId: '987fcdeb-51d2-43a1-b456-426614174111',
    validFrom: '2025-01-24T10:00:00Z',
    validTo: '2025-12-31T23:59:59Z',
    priority: 5,
    reason: 'プロジェクトマネージャーへの昇格'
  });
  
  console.log('ロール割り当て成功:', result);
  console.log(`${result.role_name} が ${result.priority} の優先度で割り当てられました`);
} catch (error) {
  console.error('ロール割り当て失敗:', error.message);
}
```

---

## 7. まとめ

### 7.1 このAPIの特徴

✅ **期限付きロール**: 開始・終了日時による柔軟なロール管理  
✅ **優先度制御**: 数値による複数ロールの重要度管理  
✅ **重複防止**: 同一ユーザー・ロールの重複割り当て防止  
✅ **監査ログ**: 割り当て者・理由の完全な記録  
✅ **データ整合性**: 事前バリデーション + データベース制約  
✅ **カスケード管理**: ユーザー・ロール削除時の安全な処理  
✅ **アクティブ管理**: 論理削除による安全なロール無効化  

### 7.2 関連API・機能

- **ロール取り消し**: `DELETE /api/v1/users/:id/roles/:role_id`
- **ロール更新**: `PATCH /api/v1/users/:id/roles/:role_id`
- **ユーザーロール一覧**: `GET /api/v1/users/:id/roles`
- **ユーザー作成**: `POST /api/v1/users`
- **ロール管理**: `/api/v1/roles/*`

### 7.3 参考資料

- **マイグレーション**: `migrations/02_add_user_roles_table.sql`
- **モデル定義**: `models/user_roles.go`
- **テストファイル**: `models/user_roles_test.go`
- **設計ドキュメント**: `docs/progress/phase_4/02_fix_multi_role_support.md`

---

**作成日**: 2025-01-24  
**対象バージョン**: v0.1.0-dev  
**関連ドキュメント**: [代表的なAPIと関数 一覧](20250729_lookback_typical_api_0_index.md)
