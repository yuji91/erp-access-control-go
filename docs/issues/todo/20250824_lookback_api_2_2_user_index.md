# ユーザー一覧取得 API - 詳細解説 (2.2)

## 概要

このドキュメントでは、ERPアクセス制御システムのユーザー一覧取得API（`GET /api/v1/users`）の実装について、コードベースを詳細に解説します。このAPIは高度なフィルタリング、ページネーション、検索機能を提供し、大規模なユーザーデータベースを効率的に管理できるよう設計されています。

---

## 1. アーキテクチャ構成

### 1.1 レイヤー構造

```
HTTP Request → ハンドラ (UserHandler) → サービス (UserService) → モデル (User) → データベース
```

#### 各レイヤーの責務
- **ハンドラレイヤー**: クエリパラメータバインディング、監査ログ、HTTPレスポンス
- **サービスレイヤー**: フィルタリング、ページネーション、検索ロジック、レスポンス変換
- **モデルレイヤー**: データベースクエリ最適化、リレーション読み込み

### 1.2 関連ファイル構成

```
internal/handlers/user.go:82-110       # GetUsers ハンドラ
internal/services/user.go:53-61        # UserListFilters 構造体
internal/services/user.go:78-84        # UserListResponse 構造体
internal/services/user.go:269-331      # GetUsers サービスメソッド
models/users.go:238-277                # モデル検索ヘルパー関数
cmd/server/main.go:655                 # ルート設定
```

## 2. フィルタ構造体とクエリパラメータ

### 2.1 UserListFilters 構造体

```go
type UserListFilters struct {
    DepartmentID *uuid.UUID         `form:"department_id"`
    Status       *models.UserStatus `form:"status"`
    RoleID       *uuid.UUID         `form:"role_id"`
    Search       string             `form:"search"`
    Page         int                `form:"page,default=1"`
    Limit        int                `form:"limit,default=20"`
}
```

**場所**: `internal/services/user.go:53-61`

### 2.2 クエリパラメータ仕様

| パラメータ | 型 | 必須 | デフォルト | 説明 |
|-----------|---|------|-----------|------|
| `department_id` | UUID | No | - | 部署IDによる絞り込み |
| `status` | string | No | - | ユーザーステータス（active/inactive/suspended） |
| `role_id` | UUID | No | - | プライマリロールIDによる絞り込み |
| `search` | string | No | - | 名前・メールアドレスの部分一致検索 |
| `page` | int | No | 1 | ページ番号（1から開始） |
| `limit` | int | No | 20 | 1ページあたりの件数（最大100） |

### 2.3 クエリ例

#### 基本的なページネーション
```
GET /api/v1/users?page=2&limit=10
```

#### 部署フィルタリング
```
GET /api/v1/users?department_id=123e4567-e89b-12d3-a456-426614174000
```

#### 複合フィルタリング
```
GET /api/v1/users?status=active&role_id=987fcdeb-51d2-43a1-b456-426614174111&search=田中&page=1&limit=20
```

### 2.4 UserListResponse 構造体

```go
type UserListResponse struct {
    Users []UserResponse `json:"users"`
    Total int64          `json:"total"`
    Page  int            `json:"page"`
    Limit int            `json:"limit"`
}
```

**場所**: `internal/services/user.go:78-84`

## 3. ハンドラレイヤーの実装詳細

### 3.1 GetUsers ハンドラの全体構造

```go
func (h *UserHandler) GetUsers(c *gin.Context) {
    // 1. クエリパラメータバインディング
    // 2. 監査ログ（リクエスト）
    // 3. サービス層呼び出し
    // 4. エラーハンドリング
    // 5. HTTPレスポンス返却
}
```

**場所**: `internal/handlers/user.go:82-110`

### 3.2 詳細なフロー解説

#### ① クエリパラメータバインディング (`internal/handlers/user.go:84-92`)

```go
var filters services.UserListFilters
if err := c.ShouldBindQuery(&filters); err != nil {
    h.logger.Warn("Invalid get users query parameters", map[string]interface{}{
        "error": err.Error(),
        "ip":    c.ClientIP(),
    })
    c.Error(errors.NewValidationError("query", "Invalid query parameters"))
    return
}
```

- **Gin のクエリバインディング**機能でURLパラメータを構造体にマッピング
- **デフォルト値**は構造体タグで自動設定
- **バリデーションエラー**時は警告ログとカスタムエラー

#### ② 監査ログ（リクエスト記録） (`internal/handlers/user.go:94-97`)

```go
h.logger.Info("Get users request", map[string]interface{}{
    "filters": filters,
    "ip":      c.ClientIP(),
})
```

- **構造化ログ**でフィルタ条件とクライアント情報を記録
- **検索・フィルタ履歴**のセキュリティ監査

#### ③ サービス層呼び出しとエラーハンドリング (`internal/handlers/user.go:99-107`)

```go
users, err := h.userService.GetUsers(filters)
if err != nil {
    h.logger.Error("Failed to get users", err, map[string]interface{}{
        "filters": filters,
        "ip":      c.ClientIP(),
    })
    c.Error(err)
    return
}
```

- **サービス層の分離**でビジネスロジックをハンドラから切り離し
- **詳細なエラーログ**でトラブルシューティングを支援

#### ④ HTTPレスポンス返却 (`internal/handlers/user.go:109`)

```go
c.JSON(http.StatusOK, users)
```

- **200 OK**でJSON形式のレスポンス返却
- **構造化レスポンス**でフロントエンドでの処理を容易化

## 4. サービスレイヤーの高度な機能

### 4.1 GetUsers サービスメソッドの全体構造

```go
func (s *UserService) GetUsers(filters UserListFilters) (*UserListResponse, error) {
    // 1. ベースクエリ構築（Preload設定）
    // 2. フィルタ適用
    // 3. 総数取得（Count）
    // 4. ページング設定・バリデーション
    // 5. ユーザー取得（Offset・Limit・Order）
    // 6. レスポンス変換
    // 7. 構造化レスポンス返却
}
```

**場所**: `internal/services/user.go:269-331`

### 4.2 詳細な機能解説

#### ① ベースクエリ構築とPreload設定 (`internal/services/user.go:271-273`)

```go
query := s.db.Model(&models.User{}).
    Preload("Department").
    Preload("PrimaryRole")
```

- **GORM Preload**で関連テーブルのデータを効率的に取得
- **N+1問題**の回避
- **Department**と**PrimaryRole**の情報を一括取得

#### ② 動的フィルタ適用 (`internal/services/user.go:275-291`)

```go
// 部署フィルタ
if filters.DepartmentID != nil {
    query = query.Where("department_id = ?", *filters.DepartmentID)
}

// ステータスフィルタ
if filters.Status != nil {
    query = query.Where("status = ?", *filters.Status)
}

// ロールフィルタ
if filters.RoleID != nil {
    query = query.Where("primary_role_id = ?", *filters.RoleID)
}

// 検索フィルタ（名前・メール部分一致）
if filters.Search != "" {
    searchTerm := "%" + strings.ToLower(filters.Search) + "%"
    query = query.Where("LOWER(name) LIKE ? OR LOWER(email) LIKE ?", searchTerm, searchTerm)
}
```

- **動的WHERE句**でフィルタ条件を柔軟に組み合わせ
- **ポインタ型**でオプショナルパラメータを判定
- **LIKE検索**で部分一致検索を実現
- **LOWER関数**で大文字小文字を区別しない検索

#### ③ 総数取得と効率化 (`internal/services/user.go:293-297`)

```go
var total int64
if err := query.Count(&total).Error; err != nil {
    return nil, errors.NewDatabaseError(err)
}
```

- **Count()メソッド**でフィルタ適用後の総レコード数を取得
- **ページネーション情報**の生成に使用

#### ④ ページング設定とバリデーション (`internal/services/user.go:299-307`)

```go
// ページング設定
if filters.Page < 1 {
    filters.Page = 1
}
if filters.Limit < 1 || filters.Limit > 100 {
    filters.Limit = 20
}

offset := (filters.Page - 1) * filters.Limit
```

- **デフォルト値**の安全な設定
- **上限制限**（100件）でシステム負荷を制御
- **オフセット計算**で正確なページング

#### ⑤ ユーザー取得とソート (`internal/services/user.go:309-317`)

```go
var users []models.User
if err := query.
    Offset(offset).
    Limit(filters.Limit).
    Order("created_at DESC").
    Find(&users).Error; err != nil {
    return nil, errors.NewDatabaseError(err)
}
```

- **Offset・Limit**で効率的なページング
- **Order("created_at DESC")**で新しいユーザーから表示
- **Find()メソッド**で複数レコードを一括取得

#### ⑥ レスポンス変換 (`internal/services/user.go:319-323`)

```go
userResponses := make([]UserResponse, len(users))
for i, user := range users {
    userResponses[i] = *s.convertToUserResponse(&user)
}
```

- **convertToUserResponse**ヘルパー関数でデータ変換
- **配列事前確保**でメモリ効率を最適化
- **ポインタ参照**で不要なコピーを回避

## 5. レスポンス構造とAPIテスト例

### 5.1 正常レスポンス例

#### HTTPステータス: 200 OK

```json
{
  "users": [
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
        "name": "開発部"
      },
      "primary_role": {
        "id": "987fcdeb-51d2-43a1-b456-426614174111",
        "name": "開発者"
      }
    },
    {
      "id": "660f9511-f3ac-52e5-b827-557766551111",
      "name": "佐藤花子",
      "email": "sato@company.com",
      "status": "active",
      "department_id": "234f5678-f90c-23e4-b567-537725285111",
      "primary_role_id": "098gdebc-62e3-54b2-c567-537725285222",
      "created_at": "2025-01-23T15:45:00Z",
      "updated_at": "2025-01-23T15:45:00Z",
      "department": {
        "id": "234f5678-f90c-23e4-b567-537725285111",
        "name": "営業部"
      },
      "primary_role": {
        "id": "098gdebc-62e3-54b2-c567-537725285222",
        "name": "営業担当"
      }
    }
  ],
  "total": 25,
  "page": 1,
  "limit": 20
}
```

### 5.2 空の結果レスポンス

```json
{
  "users": [],
  "total": 0,
  "page": 1,
  "limit": 20
}
```

### 5.3 エラーレスポンス例

#### 400 Bad Request (不正なクエリパラメータ)

```json
{
  "error": {
    "type": "VALIDATION_ERROR",
    "field": "query",
    "message": "Invalid query parameters",
    "details": "Invalid UUID format for department_id parameter"
  }
}
```

### 5.4 curlテスト例

#### 基本的な一覧取得

```bash
curl -X GET "http://localhost:8080/api/v1/users" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### ページネーション付き取得

```bash
curl -X GET "http://localhost:8080/api/v1/users?page=2&limit=10" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### 部署フィルタリング

```bash
curl -X GET "http://localhost:8080/api/v1/users?department_id=123e4567-e89b-12d3-a456-426614174000" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### 複合フィルタリング

```bash
curl -X GET "http://localhost:8080/api/v1/users?status=active&search=田中&page=1&limit=5" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### ステータス別フィルタリング

```bash
curl -X GET "http://localhost:8080/api/v1/users?status=inactive" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### 5.5 JavaScript（フロントエンド）使用例

```javascript
// 基本的な一覧取得
async function getUsers(page = 1, limit = 20) {
  const response = await fetch(`/api/v1/users?page=${page}&limit=${limit}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    }
  });
  
  if (!response.ok) {
    throw new Error('Failed to fetch users');
  }
  
  return await response.json();
}

// フィルタ付き検索
async function searchUsers(filters) {
  const params = new URLSearchParams();
  
  if (filters.departmentId) params.append('department_id', filters.departmentId);
  if (filters.status) params.append('status', filters.status);
  if (filters.search) params.append('search', filters.search);
  if (filters.page) params.append('page', filters.page);
  if (filters.limit) params.append('limit', filters.limit);
  
  const response = await fetch(`/api/v1/users?${params}`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    }
  });
  
  return await response.json();
}

// 使用例
const result = await searchUsers({
  status: 'active',
  search: '田中',
  page: 1,
  limit: 10
});

console.log(`Total users: ${result.total}`);
console.log(`Current page: ${result.page}`);
result.users.forEach(user => {
  console.log(`${user.name} (${user.email}) - ${user.department.name}`);
});
```

---

## 6. まとめ

### 6.1 このAPIの特徴

✅ **高性能フィルタリング**: 部署・ステータス・ロール・検索の組み合わせ対応  
✅ **効率的ページネーション**: OFFSET/LIMIT + 総数取得  
✅ **LIKE検索**: 名前・メールアドレスの部分一致検索  
✅ **N+1問題回避**: GORM Preloadによる関連データ一括取得  
✅ **安全な上限制御**: 最大100件/ページでシステム保護  
✅ **権限制御**: `user:list`権限による認可  
✅ **監査ログ**: 全検索・フィルタ操作の記録  

### 6.2 パフォーマンス考慮事項

- **インデックス設定**: `department_id`, `status`, `primary_role_id`にインデックス
- **Preload最適化**: N+1問題を回避して関連データを効率取得
- **上限制限**: 1ページ最大100件でメモリ使用量を制御
- **Count最適化**: フィルタ条件と同じクエリで総数を高速取得

### 6.3 関連API・機能

- **ユーザー詳細取得**: `GET /api/v1/users/:id`
- **ユーザー作成**: `POST /api/v1/users`
- **ユーザー更新**: `PUT /api/v1/users/:id`
- **ユーザーステータス変更**: `PUT /api/v1/users/:id/status`

### 6.4 参考資料

- **OpenAPI仕様**: `api/openapi.yaml`
- **テストファイル**: `internal/services/user_service_test.go:150-183`
- **統合テスト**: `internal/handlers/user_integration_test.go`
- **モデル定義**: `models/users.go`

---

**作成日**: 2025-01-24  
**対象バージョン**: v0.1.0-dev  
**関連ドキュメント**: [代表的なAPIと関数 一覧](20250729_lookback_typical_api_0_index.md)
