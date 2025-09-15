# ユーザー作成 API - 詳細解説 (2.1)

## 概要

このドキュメントでは、ERPアクセス制御システムのユーザー作成API（`POST /api/v1/users`）の実装について、コードベースを詳細に解説します。このAPIは新規ユーザーの作成を担当し、認証・認可、バリデーション、セキュリティ、監査ログなど、多くの重要な機能を含んでいます。

---

## 1. アーキテクチャ構成

### 1.1 レイヤー構造

```
HTTP Request → ハンドラ (UserHandler) → サービス (UserService) → モデル (User) → データベース
```

#### 各レイヤーの責務
- **ハンドラレイヤー**: HTTP リクエスト/レスポンス処理、監査ログ、認証チェック
- **サービスレイヤー**: ビジネスロジック、バリデーション、データ整合性チェック  
- **モデルレイヤー**: データ構造定義、データベース操作、バリデーション

### 1.2 関連ファイル構成

```
internal/handlers/user.go:30-80        # CreateUser ハンドラ
internal/services/user.go:29-37        # CreateUserRequest 構造体
internal/services/user.go:87-155       # CreateUser サービスメソッド
models/users.go:13-37                  # User モデル定義
cmd/server/main.go:654                 # ルート設定
```

## 2. リクエスト構造体とバリデーション

### 2.1 CreateUserRequest 構造体

```go
type CreateUserRequest struct {
    Name          string    `json:"name" binding:"required,min=1,max=100"`
    Email         string    `json:"email" binding:"required,email,max=255"`
    Password      string    `json:"password" binding:"required,min=6,max=255"`
    DepartmentID  uuid.UUID `json:"department_id" binding:"required"`
    PrimaryRoleID uuid.UUID `json:"primary_role_id" binding:"required"`
    Status        string    `json:"status" binding:"omitempty,oneof=active inactive suspended"`
}
```

**場所**: `internal/services/user.go:29-37`

### 2.2 バリデーション仕様

| フィールド | 制約 | 説明 |
|-----------|------|------|
| `name` | required, 1-100文字 | ユーザー表示名 |
| `email` | required, email形式, 最大255文字 | ログイン用メールアドレス |
| `password` | required, 6-255文字 | プレーンテキストパスワード（後でハッシュ化） |
| `department_id` | required, UUID形式 | 所属部署のID |
| `primary_role_id` | required, UUID形式 | プライマリロールのID |
| `status` | optional, enum値 | ユーザーステータス（デフォルト：active） |

### 2.3 リクエスト例

```json
{
  "name": "田中太郎", 
  "email": "tanaka@company.com",
  "password": "SecurePassword123",
  "department_id": "123e4567-e89b-12d3-a456-426614174000",
  "primary_role_id": "987fcdeb-51d2-43a1-b456-426614174111",
  "status": "active"
}
```

## 3. ハンドラレイヤーの実装詳細

### 3.1 CreateUser ハンドラの全体構造

```go
func (h *UserHandler) CreateUser(c *gin.Context) {
    // 1. リクエストバインディング
    // 2. 認証ユーザーの取得
    // 3. 監査ログ（リクエスト）
    // 4. サービス層呼び出し
    // 5. エラーハンドリング
    // 6. 監査ログ（成功）
    // 7. HTTPレスポンス返却
}
```

**場所**: `internal/handlers/user.go:30-80`

### 3.2 詳細なフロー解説

#### ① リクエストバインディング (`internal/handlers/user.go:32-40`)

```go
var req services.CreateUserRequest
if err := c.ShouldBindJSON(&req); err != nil {
    h.logger.Warn("Invalid create user request format", map[string]interface{}{
        "error": err.Error(),
        "ip":    c.ClientIP(),
    })
    c.Error(errors.NewValidationError("request", "Invalid request format"))
    return
}
```

- **Gin のバインディング機能**を使用してJSONリクエストを構造体にマッピング
- **バリデーションエラー**時は警告ログとカスタムエラーでレスポンス
- **クライアントIP**も合わせてログに記録

#### ② 認証ユーザー取得 (`internal/handlers/user.go:42-51`)

```go
requestUserID, err := middleware.GetCurrentUserID(c)
if err != nil {
    h.logger.Warn("Failed to get current user ID", map[string]interface{}{
        "error": err.Error(),
        "ip":    c.ClientIP(),
    })
    c.Error(errors.NewAuthenticationError("Authentication required"))
    return
}
```

- **JWTミドルウェア**で設定されたユーザーIDを取得
- **監査ログ**用に実行者の情報を記録
- 認証エラー時は適切なエラーレスポンス

#### ③ 監査ログ（リクエスト記録） (`internal/handlers/user.go:53-59`)

```go
h.logger.Info("Create user request", map[string]interface{}{
    "email":           req.Email,
    "department_id":   req.DepartmentID,
    "primary_role_id": req.PrimaryRoleID,
    "requested_by":    requestUserID,
    "ip":              c.ClientIP(),
})
```

- **構造化ログ**で重要なリクエストパラメータを記録
- **セキュリティ監査**のためIP・実行者・対象データを記録
- パスワードは**ログに記録しない**セキュリティ配慮

## 4. サービスレイヤーのビジネスロジック

### 4.1 CreateUser サービスメソッドの全体構造

```go
func (s *UserService) CreateUser(req CreateUserRequest) (*UserResponse, error) {
    // 1. 作成開始ログ
    // 2. メールアドレス重複チェック
    // 3. 部署存在確認
    // 4. ロール存在確認
    // 5. パスワードハッシュ化
    // 6. ユーザーモデル作成
    // 7. データベース保存
    // 8. 作成成功ログ
    // 9. 詳細情報付きレスポンス返却
}
```

**場所**: `internal/services/user.go:87-155`

### 4.2 詳細なビジネスロジック解説

#### ① メールアドレス重複チェック (`internal/services/user.go:94-100`)

```go
var existingUser models.User
if err := s.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
    return nil, errors.NewValidationError("email", "Email address already exists")
} else if err != gorm.ErrRecordNotFound {
    return nil, errors.NewDatabaseError(err)
}
```

- **UNIQUE制約**をアプリケーションレベルでも事前チェック
- **データベースエラー**と**既存レコード**を区別して処理
- カスタムエラー型で**詳細なエラー情報**を提供

#### ② 部署存在確認 (`internal/services/user.go:102-109`)

```go
var department models.Department
if err := s.db.First(&department, req.DepartmentID).Error; err != nil {
    if err == gorm.ErrRecordNotFound {
        return nil, errors.NewNotFoundError("Department", "Department does not exist")
    }
    return nil, errors.NewDatabaseError(err)
}
```

- **外部キー制約**の事前バリデーション
- **存在しないリソース**への参照を防止
- **データ整合性**の保証

#### ③ ロール存在確認 (`internal/services/user.go:111-118`)

```go
var role models.Role
if err := s.db.First(&role, req.PrimaryRoleID).Error; err != nil {
    if err == gorm.ErrRecordNotFound {
        return nil, errors.NewNotFoundError("Role", "Role does not exist")
    }
    return nil, errors.NewDatabaseError(err)
}
```

- **ロールベースアクセス制御**の前提条件チェック
- **有効なロール**のみ割り当て可能

#### ④ パスワードハッシュ化 (`internal/services/user.go:120-124`)

```go
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
if err != nil {
    return nil, errors.NewInternalError("Failed to hash password")
}
```

- **bcrypt**アルゴリズムで安全にハッシュ化
- **平文パスワード**は絶対にデータベースに保存しない
- **DefaultCost (10)**で適切なセキュリティレベルを提供

#### ⑤ ユーザーモデル作成とデフォルト値設定 (`internal/services/user.go:126-139`)

```go
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
```

- **構造体初期化**で安全にモデルを作成
- **デフォルト値**の明示的な設定
- **NULL可能フィールド**（PrimaryRoleID）はポインタで管理

## 5. エラーハンドリングとセキュリティ

### 5.1 カスタムエラー型の使用

プロジェクトでは `pkg/errors/errors.go` で定義されたカスタムエラー型を使用：

| エラー型 | 用途 | HTTPステータス |
|---------|------|---------------|
| `ValidationError` | バリデーション失敗 | 400 Bad Request |
| `AuthenticationError` | 認証失敗 | 401 Unauthorized |
| `NotFoundError` | リソース不存在 | 404 Not Found |
| `DatabaseError` | データベースエラー | 500 Internal Server Error |
| `InternalError` | 内部処理エラー | 500 Internal Server Error |

### 5.2 セキュリティ考慮事項

#### ① パスワードセキュリティ
- **bcrypt**によるハッシュ化（コスト10）
- **平文パスワード**のログ出力禁止
- **レスポンス**にパスワード情報を含めない

#### ② 認証・認可
```go
// ルート設定での権限チェック
users.POST("", middleware.RequirePermissions("user:create"), userHandler.CreateUser)
```

- **JWT認証**が必須
- **`user:create`権限**が必要
- **ミドルウェア**で自動的に権限チェック

#### ③ 監査ログ
```go
h.logger.Info("Create user request", map[string]interface{}{
    "email":           req.Email,
    "department_id":   req.DepartmentID,
    "primary_role_id": req.PrimaryRoleID,
    "requested_by":    requestUserID,
    "ip":              c.ClientIP(),
})
```

- **誰が・何を・いつ**の記録
- **IPアドレス**の追跡
- **機密情報**（パスワード）の除外

### 5.3 データ整合性の保証

#### ① トランザクション管理
- 現在は単一テーブル操作のため**明示的トランザクション**は未使用
- 複雑な操作時は`db.Transaction()`を使用可能

#### ② 外部キー制約
```sql
-- migrations/01_init_migration_erp_acl.sql
FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE CASCADE
FOREIGN KEY (primary_role_id) REFERENCES roles(id) ON DELETE SET NULL
```

#### ③ UNIQUE制約
```sql
-- users テーブル
CONSTRAINT users_email_unique UNIQUE (email)
```

## 6. レスポンス構造とAPIテスト例

### 6.1 UserResponse 構造体

```go
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
}
```

**場所**: `internal/services/user.go:64-74`

### 6.2 正常レスポンス例

#### HTTPステータス: 201 Created

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "田中太郎",
  "email": "tanaka@company.com",
  "status": "active",
  "department_id": "123e4567-e89b-12d3-a456-426614174000",
  "primary_role_id": "987fcdeb-51d2-43a1-b456-426614174111",
  "created_at": "2025-01-24T10:30:00Z",
  "updated_at": "2025-01-24T10:30:00Z",
  "department": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "name": "開発部",
    "description": "システム開発チーム"
  },
  "primary_role": {
    "id": "987fcdeb-51d2-43a1-b456-426614174111", 
    "name": "開発者",
    "description": "アプリケーション開発者"
  }
}
```

### 6.3 エラーレスポンス例

#### 400 Bad Request (バリデーションエラー)

```json
{
  "error": {
    "type": "VALIDATION_ERROR",
    "field": "email",
    "message": "Email address already exists",
    "details": "The provided email address is already registered in the system"
  }
}
```

#### 404 Not Found (部署未存在)

```json
{
  "error": {
    "type": "NOT_FOUND_ERROR", 
    "resource": "Department",
    "message": "Department does not exist",
    "details": "The specified department ID was not found"
  }
}
```

#### 401 Unauthorized (権限不足)

```json
{
  "error": {
    "type": "AUTHORIZATION_ERROR",
    "message": "Permission denied",
    "details": "user:create permission is required"
  }
}
```

### 6.4 curlテスト例

#### 正常ケース

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "name": "田中太郎",
    "email": "tanaka@company.com", 
    "password": "SecurePassword123",
    "department_id": "123e4567-e89b-12d3-a456-426614174000",
    "primary_role_id": "987fcdeb-51d2-43a1-b456-426614174111",
    "status": "active"
  }'
```

#### メール重複エラーケース

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "name": "既存ユーザー",
    "email": "existing@company.com",
    "password": "password123", 
    "department_id": "123e4567-e89b-12d3-a456-426614174000",
    "primary_role_id": "987fcdeb-51d2-43a1-b456-426614174111"
  }'
```

---

## 7. まとめ

### 7.1 このAPIの特徴

✅ **レイヤー分離**: ハンドラ・サービス・モデルの明確な責務分離  
✅ **セキュリティ**: bcryptハッシュ化、JWT認証、権限チェック  
✅ **監査ログ**: 全操作の追跡可能性  
✅ **データ整合性**: 事前バリデーション、外部キー制約  
✅ **エラーハンドリング**: 詳細なカスタムエラー型  
✅ **複数ロール対応**: プライマリロール + UserRoles テーブル  

### 7.2 参考資料

- **OpenAPI仕様**: `api/openapi.yaml:1820-1849`
- **テストファイル**: `internal/services/user_service_test.go`
- **統合テスト**: `internal/handlers/user_integration_test.go`
- **モデル定義**: `models/users.go`
- **マイグレーション**: `migrations/01_init_migration_erp_acl.sql`

### 7.3 関連API

- **ユーザー一覧取得**: `GET /api/v1/users`
- **ユーザー詳細取得**: `GET /api/v1/users/:id`
- **ユーザー更新**: `PUT /api/v1/users/:id`
- **ユーザーロール割り当て**: `POST /api/v1/users/roles`

---

**作成日**: 2025-01-24  
**対象バージョン**: v0.1.0-dev  
**関連ドキュメント**: [代表的なAPIと関数 一覧](20250729_lookback_typical_api_0_index.md)
