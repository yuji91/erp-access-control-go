# JWT認証ミドルウェア - 詳細解説 (1.2)

## 概要
このドキュメントでは、ERPアクセス制御システムの **JWT認証ミドルウェア** (`AuthMiddleware.Authentication()`) について、各コンポーネントの役割と動作を詳細に解説します。このミドルウェアは、全ての保護されたAPIエンドポイントで実行され、リクエストの認証・認可を担当する重要なコンポーネントです。

## 🔄 **全体的な処理フロー**

```mermaid
sequenceDiagram
    participant Client as "クライアント"
    participant Middleware as "AuthMiddleware.Authentication()"
    participant JWT as "JWTService"
    participant Revocation as "TokenRevocationService"
    participant Context as "Ginコンテキスト"
    participant Handler as "実際のAPIハンドラ"
    
    Client->>Middleware: API Request + Authorization Header
    Middleware->>Middleware: 1. Authorization ヘッダー存在確認
    Middleware->>Middleware: 2. Bearer トークン形式確認
    Middleware->>JWT: 3. ValidateToken(tokenString)
    JWT-->>Middleware: CustomClaims (ユーザー情報・権限)
    Middleware->>Revocation: 4. ValidateTokenStatus(jti, userID, issuedAt)
    Revocation-->>Middleware: トークン状態確認結果
    Middleware->>Context: 5. ユーザー情報をコンテキストに保存
    Middleware->>Middleware: 6. アクセスログ記録
    Middleware->>Handler: 7. c.Next() - 次のハンドラへ
    Handler-->>Client: API レスポンス
    
    Note over Middleware: エラー時は c.Abort() でリクエスト停止
```

---

## 🏗️ **コンポーネント詳細解析**

### 1. **AuthMiddleware 構造体** - ミドルウェアの基盤
**ファイル**: `internal/middleware/auth.go:18-32`

#### **構成要素**
```go
type AuthMiddleware struct {
    jwtService        *jwt.Service                    // JWTトークンの検証・解析
    revocationService *services.TokenRevocationService // トークン失効管理
    logger            *logger.Logger                   // 構造化ログ出力
}

func NewAuthMiddleware(jwtService *jwt.Service, revocationService *services.TokenRevocationService, logger *logger.Logger) *AuthMiddleware {
    return &AuthMiddleware{
        jwtService:        jwtService,
        revocationService: revocationService,
        logger:            logger,
    }
}
```

#### **依存関係**
- **JWTService**: トークンの署名検証・クレーム解析
- **TokenRevocationService**: 明示的にログアウトされたトークンの管理
- **Logger**: セキュリティ監査・アクセスログ記録

---

### 2. **Authentication()** - 核となる認証処理
**ファイル**: `internal/middleware/auth.go:100-167`

#### **役割**
- HTTPリクエストからJWTトークンを抽出
- トークンの形式・署名・有効期限の検証
- トークンの失効状態確認
- ユーザー情報をリクエストコンテキストに保存

#### **詳細処理ステップ**

##### **2.1 Authorization ヘッダーの確認**
```go
func (m *AuthMiddleware) Authentication() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            m.logger.Warn("Missing authorization header", map[string]interface{}{
                "path": c.Request.URL.Path,
                "ip":   c.ClientIP(),
            })
            c.Error(errors.ErrInvalidToken)
            c.Abort()
            return
        }
    }
}
```

**チェック項目**:
- Authorization ヘッダーの存在確認
- 欠如時のセキュリティログ記録
- エラー時の即座リクエスト停止

##### **2.2 Bearer トークン形式の検証**
```go
tokenString := strings.TrimPrefix(authHeader, "Bearer ")
if tokenString == authHeader {
    m.logger.Warn("Invalid token format", map[string]interface{}{
        "path": c.Request.URL.Path,
        "ip":   c.ClientIP(),
    })
    c.Error(errors.ErrInvalidToken)
    c.Abort()
    return
}
```

**期待形式**: `Authorization: Bearer <JWT_TOKEN>`
**セキュリティ特徴**:
- 不正な形式のトークンを即座拒否
- アクセス試行ログでセキュリティ監視

##### **2.3 JWTトークンの検証・解析**
```go
claims, err := m.jwtService.ValidateToken(tokenString)
if err != nil {
    m.logger.Warn("Token validation failed", map[string]interface{}{
        "error": err.Error(),
        "path":  c.Request.URL.Path,
        "ip":    c.ClientIP(),
    })
    c.Error(errors.ErrInvalidToken)
    c.Abort()
    return
}
```

**JWTService.ValidateToken()の内部処理**:
```go
func (s *Service) ValidateToken(tokenString string) (*CustomClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
        // 1. 署名アルゴリズムの確認（HMAC系のみ許可）
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return s.secretKey, nil
    })
    
    if err != nil {
        return nil, err  // 署名検証失敗・有効期限切れ等
    }
    
    if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
        return claims, nil
    }
    
    return nil, fmt.Errorf("invalid token")
}
```

**検証内容**:
- **署名検証**: HMAC-SHA256による改ざん検知
- **有効期限確認**: `exp`クレームによる時刻ベース検証
- **発行時刻確認**: `iat`, `nbf`クレームによる使用開始時刻確認
- **形式確認**: カスタムクレーム構造体への変換可能性

##### **2.4 トークン失効状態の確認**
```go
if err := m.revocationService.ValidateTokenStatus(claims.ID, claims.UserID, claims.IssuedAt.Time); err != nil {
    m.logger.Warn("Token revoked", map[string]interface{}{
        "token_id": claims.ID,
        "user_id":  claims.UserID,
        "path":     c.Request.URL.Path,
    })
    c.Error(errors.ErrTokenRevoked)
    c.Abort()
    return
}
```

**TokenRevocationService の役割**:
- ログアウト時に失効したトークンの管理
- JTI（JWT ID）ベースの個別トークン失効
- データベースまたはRedisでの失効リスト管理

##### **2.5 ユーザー情報のコンテキスト保存**
```go
// ユーザー情報をコンテキストに保存
c.Set("user_id", claims.UserID)
c.Set("email", claims.Email)
c.Set("permissions", claims.Permissions)
c.Set("jti", claims.ID)
c.Set("primary_role_id", claims.PrimaryRoleID)
c.Set("active_roles", claims.ActiveRoles)
c.Set("highest_role", claims.HighestRole)
```

**保存される情報**:
- **基本情報**: `user_id`, `email`, `jti`
- **権限情報**: `permissions` (string配列)
- **ロール情報**: `primary_role_id`, `active_roles`, `highest_role`

##### **2.6 アクセスログ記録**
```go
m.logger.Info("Authenticated request", map[string]interface{}{
    "user_id": claims.UserID,
    "email":   claims.Email,
    "path":    c.Request.URL.Path,
    "method":  c.Request.Method,
})
```

**監査ログ項目**:
- ユーザー識別情報
- アクセス先パス・HTTPメソッド
- 認証成功時刻

---

### 3. **RequirePermissions()** - 権限チェックミドルウェア
**ファイル**: `internal/middleware/auth.go:169-196`

#### **役割**
- 特定のAPIエンドポイントに必要な権限の確認
- ワイルドカード権限対応
- 複数権限の AND 条件チェック

#### **処理ステップ**
```go
func RequirePermissions(permissions ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. コンテキストから権限情報取得
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
        
        // 2. 必要な権限をすべてチェック（AND条件）
        for _, requiredPerm := range permissions {
            if !hasPermission(userPermissions, requiredPerm) {
                c.Error(errors.NewAuthorizationError(fmt.Sprintf("Missing required permission: %s", requiredPerm)))
                c.Abort()
                return
            }
        }
        
        c.Next()
    }
}
```

**使用例**:
```go
// ユーザー作成には "users:write" 権限が必要
router.POST("/users", authMiddleware.Authentication(), RequirePermissions("users:write"), userHandler.CreateUser)

// ユーザー削除には複数権限が必要
router.DELETE("/users/:id", authMiddleware.Authentication(), RequirePermissions("users:delete", "admin:action"), userHandler.DeleteUser)
```

---

### 4. **hasPermission()** - 権限マッチング機能
**ファイル**: `internal/middleware/auth.go:313-334`

#### **高度な権限マッチング機能**
```go
func hasPermission(userPermissions []string, requiredPermission string) bool {
    for _, perm := range userPermissions {
        // 1. 完全一致チェック
        if perm == requiredPermission {
            return true
        }
        
        // 2. 全権限ワイルドカード
        if perm == "*" || perm == "*:*" {
            return true
        }
        
        // 3. モジュール別ワイルドカード（例: "users:*"）
        if strings.Contains(requiredPermission, ":") && strings.HasSuffix(perm, ":*") {
            requiredModule := strings.Split(requiredPermission, ":")[0]
            permModule := strings.TrimSuffix(perm, ":*")
            if requiredModule == permModule {
                return true
            }
        }
    }
    return false
}
```

#### **権限マッチングパターン**

| ユーザー権限 | 要求される権限 | マッチ結果 | 説明 |
|-------------|---------------|-----------|------|
| `users:read` | `users:read` | ✅ | 完全一致 |
| `users:*` | `users:read` | ✅ | モジュール別ワイルドカード |
| `users:*` | `users:write` | ✅ | モジュール別ワイルドカード |
| `*` | `departments:read` | ✅ | 全権限ワイルドカード |
| `*:*` | `roles:delete` | ✅ | 全権限ワイルドカード |
| `users:read` | `users:write` | ❌ | アクション不一致 |
| `departments:*` | `users:read` | ❌ | モジュール不一致 |

---

### 5. **RequireAnyPermission()** - OR条件権限チェック
**ファイル**: `internal/middleware/auth.go:198-225`

#### **役割**
- 複数権限のうち、いずれか1つでも持っていれば許可
- 柔軟な権限設計に対応

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
        
        // いずれかの権限を持っていれば OK
        for _, requiredPerm := range permissions {
            if hasPermission(userPermissions, requiredPerm) {
                c.Next()
                return
            }
        }
        
        c.Error(errors.NewAuthorizationError(fmt.Sprintf("Missing any of required permissions: %s", strings.Join(permissions, ", "))))
        c.Abort()
    }
}
```

**使用例**:
```go
// 管理者権限 OR ユーザー管理権限があれば許可
router.GET("/admin/users", authMiddleware.Authentication(), RequireAnyPermission("admin:*", "users:manage"), adminHandler.GetUsers)
```

---

### 6. **RequireOwnership()** - リソース所有権チェック
**ファイル**: `internal/middleware/auth.go:227-266`

#### **役割**
- ユーザーが特定のリソースの所有者かを確認
- URLパラメータからリソースIDを取得
- 自分自身のデータのみアクセス許可

#### **処理ステップ**
```go
func RequireOwnership() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. コンテキストから現在のユーザーID取得
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
        
        // 2. URLパラメータからリソースユーザーID取得
        resourceUserID := c.Param("user_id")
        if resourceUserID == "" {
            c.Error(errors.NewValidationError("user_id", "missing user ID parameter"))
            c.Abort()
            return
        }
        
        resourceUserUUID, err := uuid.Parse(resourceUserID)
        if err != nil {
            c.Error(errors.NewValidationError("user_id", "invalid user ID format"))
            c.Abort()
            return
        }
        
        // 3. 所有権確認
        if currentUserID != resourceUserUUID {
            c.Error(errors.NewAuthorizationError("You do not have permission to access this resource"))
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

**使用例**:
```go
// 自分自身のプロフィールのみアクセス可能
router.GET("/users/:user_id/profile", authMiddleware.Authentication(), RequireOwnership(), userHandler.GetProfile)
```

---

## 🔒 **セキュリティ機能**

### 1. **多層防御機構**
```go
// Layer 1: ヘッダー存在確認
if authHeader == "" { /* 拒否 */ }

// Layer 2: 形式確認
if tokenString == authHeader { /* Bearer形式でない */ }

// Layer 3: 署名・期限確認
claims, err := m.jwtService.ValidateToken(tokenString)

// Layer 4: 失効確認
if err := m.revocationService.ValidateTokenStatus(...) { /* 失効済み */ }

// Layer 5: 権限確認
if !hasPermission(userPermissions, requiredPermission) { /* 権限不足 */ }
```

### 2. **ログ・監査機能**
```go
// 認証失敗パターンごとの詳細ログ
m.logger.Warn("Missing authorization header", map[string]interface{}{
    "path": c.Request.URL.Path,
    "ip":   c.ClientIP(),
})

m.logger.Warn("Invalid token format", map[string]interface{}{
    "path": c.Request.URL.Path,
    "ip":   c.ClientIP(),
})

m.logger.Warn("Token validation failed", map[string]interface{}{
    "error": err.Error(),
    "path":  c.Request.URL.Path,
    "ip":    c.ClientIP(),
})

m.logger.Warn("Token revoked", map[string]interface{}{
    "token_id": claims.ID,
    "user_id":  claims.UserID,
    "path":     c.Request.URL.Path,
})
```

**監査レベル**:
- **Info**: 正常な認証成功
- **Warn**: 認証失敗・権限不足
- **Error**: システムエラー・予期しない状況

### 3. **攻撃対策**
- **トークン改ざん検知**: HMAC-SHA256署名による完全性確認
- **リプレイ攻撃対策**: JTI（JWT ID）による個別トークン管理
- **セッション固定攻撃対策**: 短期間有効期限（15分）
- **権限昇格攻撃対策**: 厳密な権限マッチング・ワイルドカード制御

---

## 🚨 **エラーハンドリング**

### 1. **エラーの種類と対応**
```go
// 1. 認証エラー
errors.ErrInvalidToken          // 無効なトークン
errors.ErrTokenRevoked          // 失効済みトークン

// 2. 認可エラー
errors.ErrPermissionDenied      // 権限不足
errors.NewAuthorizationError()  // 詳細な権限エラー

// 3. バリデーションエラー
errors.NewValidationError()     // パラメータ形式エラー
```

### 2. **エラーレスポンス統一**
```json
{
  "error": {
    "code": "AUTHENTICATION_ERROR",
    "message": "Invalid or expired token",
    "details": {
      "timestamp": "2025-01-29T12:00:00Z",
      "path": "/api/v1/users"
    }
  }
}
```

### 3. **セキュリティ配慮**
- **詳細なエラー情報の抑制**: 攻撃者に有用な情報を提供しない
- **統一されたエラーレスポンス**: 内部構造の推測を困難にする
- **適切なHTTPステータス**: `401 Unauthorized`, `403 Forbidden`

---

## ⚡ **パフォーマンス特徴**

### 1. **高速な認証処理**
```go
// JWT検証は暗号化処理のみ（DB接続不要）
claims, err := m.jwtService.ValidateToken(tokenString)

// トークン失効確認も高速（Redis/メモリキャッシュ使用）
err := m.revocationService.ValidateTokenStatus(...)
```

### 2. **効率的なコンテキスト管理**
```go
// 一度の認証で必要な情報をすべてコンテキストに保存
c.Set("user_id", claims.UserID)
c.Set("permissions", claims.Permissions)
c.Set("active_roles", claims.ActiveRoles)
// 後続のハンドラではDB接続不要
```

### 3. **権限チェックの最適化**
```go
// ワイルドカード権限による効率的なマッチング
if perm == "*" || perm == "*:*" {
    return true  // 早期リターンでループ短縮
}
```

---

## 🔄 **使用例とAPIルーティング**

### 1. **基本的な認証**
```go
// すべての /api/v1/* エンドポイントに認証を適用
api := router.Group("/api/v1")
api.Use(authMiddleware.Authentication())

api.GET("/profile", userHandler.GetProfile)         // 認証のみ
api.GET("/users", userHandler.GetUsers)             // 認証のみ
```

### 2. **権限ベースアクセス制御**
```go
// 特定の権限が必要なエンドポイント
api.POST("/users", RequirePermissions("users:write"), userHandler.CreateUser)
api.DELETE("/users/:id", RequirePermissions("users:delete"), userHandler.DeleteUser)
api.GET("/admin/logs", RequirePermissions("admin:read"), adminHandler.GetLogs)
```

### 3. **複雑な権限制御**
```go
// OR条件の権限チェック
api.GET("/reports", RequireAnyPermission("reports:read", "admin:read"), reportHandler.GetReports)

// 所有権チェック
api.GET("/users/:user_id/profile", RequireOwnership(), userHandler.GetUserProfile)

// 複数条件の組み合わせ
api.PUT("/users/:user_id/password", 
    RequireAnyPermission("users:manage", "admin:*"),  // 管理権限 OR
    RequireOwnership(),                                // 自分自身
    userHandler.ChangePassword)
```

### 4. **実際のHTTPリクエスト例**
```bash
# 正常な認証リクエスト
curl -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiNTUwZTg0MDAtZTI5Yi00MWQ0LWE3MTYtNDQ2NjU1NDQwMDAwIiwiZW1haWwiOiJhZG1pbkBleGFtcGxlLmNvbSIsInBlcm1pc3Npb25zIjpbIio6KiJdLCJpc3MiOiJlcnAtYWNjZXNzLWNvbnRyb2wtYXBpIiwiZXhwIjoxNjQzNzI4ODAwfQ.Xm9X9Q7ZpqYwQzKZ8vJ3cJ2a7VUgJ2RLVV7kW5P2QQQ"

# 認証ヘッダーなし（401エラー）
curl -X GET http://localhost:8080/api/v1/users

# 無効なトークン形式（401エラー）
curl -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: InvalidToken"

# 権限不足（403エラー）
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer <read_only_user_token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"New User"}'
```

---

## 📊 **ミドルウェアチェーンの実行順序**

### 1. **標準的なミドルウェアスタック**
```go
router := gin.New()

// 1. 基本ミドルウェア
router.Use(gin.Logger())
router.Use(gin.Recovery())
router.Use(corsMiddleware())

// 2. 認証グループ
api := router.Group("/api/v1")
api.Use(authMiddleware.Authentication())          // 認証
api.Use(authMiddleware.ErrorHandler())            // エラーハンドリング

// 3. エンドポイント固有の権限チェック
api.POST("/users", 
    RequirePermissions("users:write"),            // 権限チェック
    auditMiddleware(),                            // 監査ログ
    userHandler.CreateUser)                       // 実際の処理
```

### 2. **実行順序**
```
リクエスト受信
    ↓
gin.Logger() - アクセスログ
    ↓
gin.Recovery() - パニック回復
    ↓
corsMiddleware() - CORS設定
    ↓
authMiddleware.Authentication() - JWT認証
    ↓
RequirePermissions() - 権限チェック
    ↓
auditMiddleware() - 監査ログ
    ↓
userHandler.CreateUser() - 実際のAPI処理
    ↓
レスポンス返却
```

---

## 🎯 **まとめ**

JWT認証ミドルウェア (`AuthMiddleware.Authentication()`) は、以下の特徴を持つ堅牢で高性能な実装となっています：

### **技術的強み**
1. **多層セキュリティ**: ヘッダー → 形式 → 署名 → 失効 → 権限の5段階検証
2. **高度な権限制御**: ワイルドカード・OR条件・所有権チェック対応
3. **包括的監査**: 全認証試行・権限チェックの詳細ログ記録
4. **高性能設計**: JWT検証のみでDB接続不要・効率的なコンテキスト管理

### **実装品質**
1. **柔軟な権限設計**: モジュール×アクション + ワイルドカード権限
2. **堅牢なエラーハンドリング**: セキュリティ配慮・統一レスポンス
3. **拡張可能な設計**: 複数認証方式・カスタム権限チェック対応準備済み
4. **開発者体験**: 型安全・明確なAPI・包括的テスト

### **エンタープライズ機能**
1. **トークン失効管理**: 明示的ログアウト・セキュリティインシデント対応
2. **詳細監査ログ**: コンプライアンス・セキュリティ分析対応
3. **複数ロール対応**: 複雑な組織構造・権限継承
4. **パフォーマンス最適化**: 大規模トラフィック・低レイテンシー対応

この実装により、企業レベルのERPシステムに求められる認証・認可要件を満たしつつ、スケーラビリティ・保守性・セキュリティの観点からも優れたアーキテクチャを実現しています。