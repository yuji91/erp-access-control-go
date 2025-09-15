# 権限チェックミドルウェア - 詳細解説 (1.3)

## 概要
このドキュメントでは、ERPアクセス制御システムにおける **権限チェックミドルウェア** について、各コンポーネントの役割と動作を詳細に解説します。  
権限チェックは `RequirePermissions()`, `RequireAnyPermission()`, `RequireOwnership()` の3つの主要ミドルウェアで実現され、ワイルドカード権限や複雑な権限マトリックスに対応しています。

## 🔄 **全体的な処理フロー**

```mermaid
sequenceDiagram
    participant Client as "クライアント"
    participant Auth as "AuthMiddleware.Authentication()"
    participant Perm as "RequirePermissions()"
    participant Handler as "APIハンドラ"
    participant Context as "Ginコンテキスト"
    participant Error as "ErrorHandler"
    
    Client->>Auth: API Request<br/>Authorization: Bearer <token>
    Auth->>Auth: JWTトークン検証
    Auth->>Context: ユーザー情報・権限をコンテキストに保存
    Auth->>Perm: 権限チェック開始
    Perm->>Context: permissions配列を取得
    Perm->>Perm: hasPermission() 実行<br/>ワイルドカード・権限マトリックス判定
    
    alt 権限あり
        Perm->>Handler: c.Next() 実行
        Handler-->>Client: API レスポンス
    else 権限なし
        Perm->>Error: AuthorizationError 生成
        Error-->>Client: 403 Forbidden<br/>権限不足メッセージ
    end
```

---

## 🏗️ **コンポーネント詳細解析**

### 1. **RequirePermissions()** - 必須権限チェックミドルウェア
**ファイル**: `internal/middleware/auth.go:170-196`

#### **役割**
- **全権限要求**: 指定された全ての権限をユーザーが保持しているかをチェック
- **AND演算子**: すべての権限が必要（一つでも欠けていれば拒否）
- **最も厳格な権限チェック**: 高セキュリティが要求されるAPIで使用

#### **処理ステップ**
```go
func RequirePermissions(permissions ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. コンテキストから権限配列を取得
        userPerms, exists := c.Get("permissions")
        if !exists {
            c.Error(errors.ErrPermissionDenied)
            c.Abort()
            return
        }

        // 2. 型安全な変換
        userPermissions, ok := userPerms.([]string)
        if !ok {
            c.Error(errors.ErrPermissionDenied)
            c.Abort()
            return
        }

        // 3. 全ての必須権限をチェック（AND演算）
        for _, requiredPerm := range permissions {
            if !hasPermission(userPermissions, requiredPerm) {
                c.Error(errors.NewAuthorizationError(
                    fmt.Sprintf("Missing required permission: %s", requiredPerm)
                ))
                c.Abort()
                return
            }
        }

        c.Next() // 全権限OK → 次のハンドラへ
    }
}
```

#### **使用例**
```go
// ユーザー削除: user:delete 権限必須
users.DELETE("/:id", middleware.RequirePermissions("user:delete"), userHandler.DeleteUser)

// 複数権限要求: 全て必要
group.POST("/sensitive", middleware.RequirePermissions("admin:system", "audit:write"), handler)
```

---

### 2. **RequireAnyPermission()** - いずれか権限チェックミドルウェア
**ファイル**: `internal/middleware/auth.go:198-225`

#### **役割**
- **いずれか権限で許可**: 指定された権限のいずれかを保持していれば許可
- **OR演算子**: 複数の権限ルートがある場合に使用
- **柔軟な権限管理**: 異なる役割でも同じAPIアクセスを可能にする

#### **処理ステップ**
```go
func RequireAnyPermission(permissions ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userPerms, exists := c.Get("permissions")
        if !exists {
            c.Error(errors.ErrPermissionDenied)
            c.Abort()
            return
        }

        userPermissions, ok := userPerms.([]string)
        if !ok {
            c.Error(errors.ErrPermissionDenied)
            c.Abort()
            return
        }

        // いずれかの権限があれば許可（OR演算）
        for _, requiredPerm := range permissions {
            if hasPermission(userPermissions, requiredPerm) {
                c.Next() // 権限発見 → 即座に許可
                return
            }
        }

        // 全ての権限が見つからない場合は拒否
        c.Error(errors.NewAuthorizationError(
            fmt.Sprintf("Missing any of required permissions: %s", 
                strings.Join(permissions, ", "))
        ))
        c.Abort()
    }
}
```

#### **使用例**
```go
// 部署情報の読み取り: 管理者権限 OR 部署読み取り権限
departments.GET("/:id", middleware.RequireAnyPermission("admin:read", "department:read"), handler)

// レポート生成: いずれかの権限があれば生成可能
reports.GET("/", middleware.RequireAnyPermission("manager:reports", "analyst:reports", "admin:reports"), handler)
```

---

### 3. **RequireOwnership()** - リソース所有者チェックミドルウェア
**ファイル**: `internal/middleware/auth.go:227-266`

#### **役割**
- **自分のリソースのみアクセス許可**: ユーザーが自分のリソースにのみアクセスできるように制御
- **個人情報保護**: 他人のプロフィール・設定へのアクセスを防止
- **URLパラメータとの照合**: リクエストのuser_idと認証済みuser_idを比較

#### **処理ステップ**
```go
func RequireOwnership() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 認証済みユーザーIDを取得
        userID, exists := c.Get("user_id")
        if !exists {
            c.Error(errors.ErrPermissionDenied)
            c.Abort()
            return
        }

        currentUserID, ok := userID.(uuid.UUID)
        if !ok {
            c.Error(errors.ErrPermissionDenied)
            c.Abort()
            return
        }

        // 2. URLパラメータからリソースのuser_idを取得
        resourceUserID := c.Param("user_id")
        if resourceUserID == "" {
            c.Error(errors.NewValidationError("user_id", "missing user ID parameter"))
            c.Abort()
            return
        }

        // 3. UUID変換
        resourceUserUUID, err := uuid.Parse(resourceUserID)
        if err != nil {
            c.Error(errors.NewValidationError("user_id", "invalid user ID format"))
            c.Abort()
            return
        }

        // 4. 所有者チェック: 同一ユーザーかどうか
        if currentUserID != resourceUserUUID {
            c.Error(errors.NewAuthorizationError(
                "You do not have permission to access this resource"
            ))
            c.Abort()
            return
        }

        c.Next() // 所有者OK → 次のハンドラへ
    }
}
```

#### **使用例**
```go
// パスワード変更: 自分のパスワードのみ変更可能
users.PUT("/:user_id/password", middleware.RequireOwnership(), userHandler.ChangePassword)

// プロフィール更新: 自分のプロフィールのみ更新可能
users.PUT("/:user_id/profile", middleware.RequireOwnership(), userHandler.UpdateProfile)
```

---

### 4. **hasPermission()** - 権限マッチング核心エンジン
**ファイル**: `internal/middleware/auth.go:313-334`

#### **役割**
- **権限マッチングの核心ロジック**: 3種類の権限パターンに対応
- **Permission Matrix サポート**: ワイルドカード権限の高度なマッチング
- **セキュリティとパフォーマンスのバランス**: 効率的な権限判定

#### **権限マッチング詳細**

##### **1. 完全一致マッチング**
```go
// 完全一致をチェック
if perm == requiredPermission {
    return true
}
```
**例**: `user:read` 権限で `user:read` アクセス → ✅ 許可

##### **2. グローバルワイルドカード権限**
```go
// ワイルドカード権限をチェック
if perm == "*" || perm == "*:*" {
    return true
}
```
**例**: `*:*` 権限（システム管理者）で任意のアクセス → ✅ 許可

##### **3. モジュール別ワイルドカード権限**
```go
// モジュール別ワイルドカード（例: "user:*"）をチェック
if strings.Contains(requiredPermission, ":") && strings.HasSuffix(perm, ":*") {
    requiredModule := strings.Split(requiredPermission, ":")[0]
    permModule := strings.TrimSuffix(perm, ":*")
    if requiredModule == permModule {
        return true
    }
}
```
**例**: `user:*` 権限で `user:read`, `user:create`, `user:delete` → ✅ 全て許可

#### **Permission Matrix との統合**
```go
// Permission Matrix の例（internal/services/permission.go より）
permissionMatrix := map[string]map[string][]string{
    "システム管理者": {
        "*": {"*"},  // 全権限
    },
    "部門管理者": {
        "user":       {"read", "create", "update"},
        "department": {"read", "update"},
        "role":       {"read"},
    },
    "開発者": {
        "user":       {"read"},
        "permission": {"read"},
        "role":       {"read"},
    },
    "一般ユーザー": {
        "user": {"read"},
    },
}
```

---

## 🛡️ **セキュリティ特徴**

### **多層防御アーキテクチャ**
1. **認証チェック**: JWT トークンの有効性確認
2. **トークン失効チェック**: ログアウト済みトークンの排除
3. **権限チェック**: 必要権限の保持確認
4. **リソース所有者チェック**: 自分のリソースのみアクセス許可

### **ログとモニタリング**
```go
// AuthMiddleware での詳細ログ記録
m.logger.Info("Authenticated request", map[string]interface{}{
    "user_id": claims.UserID,
    "email":   claims.Email,
    "path":    c.Request.URL.Path,
    "method":  c.Request.Method,
})
```

### **攻撃対策**
- **トークン形式チェック**: Bearer プレフィックスの検証
- **型安全な変換**: interface{} から具体的型への安全な変換
- **詳細エラーメッセージ**: 適切なエラー理由の提供（デバッグ支援）

---

## ⚡ **パフォーマンス特性**

### **効率的な権限チェック**
- **O(n)線形探索**: ユーザー権限配列での高速検索
- **早期リターン**: 権限発見時の即座処理終了
- **メモリ効率**: コンテキスト内での権限情報共有

### **コンテキスト共有の最適化**
```go
// AuthMiddleware で一度設定されたコンテキスト情報を再利用
c.Set("user_id", claims.UserID)
c.Set("permissions", claims.Permissions)
c.Set("active_roles", claims.ActiveRoles)

// 権限チェックミドルウェアでの効率的な取得
userPerms, exists := c.Get("permissions")
```

---

## 🎯 **実用例とAPIルーティング**

### **完全なルーティング例**
```go
// internal/cmd/server/main.go でのルーティング設定例

// 認証が必要なエンドポイントグループ
protected := v1.Group("")
protected.Use(middlewares.Auth.Authentication())  // JWTトークン検証
{
    // ユーザー管理（権限レベル別）
    users := protected.Group("/users")
    {
        // 作成権限必須
        users.POST("", middleware.RequirePermissions("user:create"), userHandler.CreateUser)
        
        // 一覧表示権限必須
        users.GET("", middleware.RequirePermissions("user:list"), userHandler.GetUsers)
        
        // 読み取り権限必須
        users.GET("/:id", middleware.RequirePermissions("user:read"), userHandler.GetUser)
        
        // 更新権限必須
        users.PUT("/:id", middleware.RequirePermissions("user:update"), userHandler.UpdateUser)
        
        // 削除権限必須（高レベル権限）
        users.DELETE("/:id", middleware.RequirePermissions("user:delete"), userHandler.DeleteUser)
        
        // 管理権限必須（システム管理）
        users.PUT("/:id/status", middleware.RequirePermissions("user:manage"), userHandler.ChangeUserStatus)
        
        // 所有者のみ（自分のパスワード変更）
        users.PUT("/:user_id/password", middleware.RequireOwnership(), userHandler.ChangePassword)
    }
    
    // 権限管理（高レベル権限必須）
    permissions := protected.Group("/permissions")
    {
        permissions.POST("", middleware.RequirePermissions("permission:create"), permissionHandler.CreatePermission)
        permissions.GET("/matrix", middleware.RequirePermissions("permission:list"), permissionHandler.GetPermissionMatrix)
        permissions.DELETE("/:id", middleware.RequirePermissions("permission:delete"), permissionHandler.DeletePermission)
    }
}
```

### **HTTP リクエスト例**

#### **成功例**: 権限ありの場合
```bash
# ユーザー一覧取得（user:list 権限保持ユーザー）
curl -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json"

# レスポンス: 200 OK
{
  "data": {
    "users": [...],
    "pagination": {...}
  }
}
```

#### **失敗例**: 権限不足の場合
```bash
# ユーザー削除試行（user:delete 権限なし）
curl -X DELETE http://localhost:8080/api/v1/users/123e4567-e89b-12d3-a456-426614174000 \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json"

# レスポンス: 403 Forbidden
{
  "code": "AUTHORIZATION_ERROR",
  "message": "Authorization failed",
  "details": {
    "reason": "Missing required permission: user:delete"
  }
}
```

---

## 📊 **統計・メトリクス**

### **権限チェック実装統計**
- **総権限保護エンドポイント**: 23個
- **権限種別**: 6モジュール × 5-8アクション = 約40権限
- **ワイルドカード対応**: グローバル(`*:*`) + モジュール別(`module:*`)
- **所有者チェック対応**: ユーザー自身のリソースアクセス制御

### **セキュリティ統計**
- **多層認証**: 認証 → トークン失効チェック → 権限チェック → 所有者チェック
- **エラーハンドリング**: 6種類のエラーカテゴリ
- **ログ記録**: アクセス成功・失敗の詳細記録

---

## 🔗 **関連コンポーネント**

### **直接連携**
- **AuthMiddleware.Authentication()**: JWT認証とコンテキスト設定
- **LoginAPI**: JWT生成時の権限情報埋め込み
- **PermissionService**: 権限マトリックス管理

### **間接連携**
- **ErrorHandler**: 統一エラーレスポンス生成
- **Logger**: セキュリティイベントの記録
- **TokenRevocationService**: ログアウト時の権限失効

---

## 📝 **参考情報**

- **作成日**: 2025-07-29
- **対象バージョン**: v0.1.0-dev
- **実装ファイル**: 
  - メインロジック: `internal/middleware/auth.go:170-334`
  - エラー定義: `pkg/errors/errors.go:98-105`
  - ルーティング例: `cmd/server/main.go:648-727`
- **テストファイル**: `internal/middleware/auth_test.go` (予定)
- **関連ドキュメント**: 
  - [JWT認証ミドルウェア詳細解説](./20250729_lookback_typical_api_1_2_jwt_auth_mid.md)
  - [ログイン認証API詳細解説](./20250729_lookback_typical_api_1_1_login_api.md)