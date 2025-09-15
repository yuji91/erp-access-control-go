# ログイン認証API - 詳細解説 (1.1)

## 概要
このドキュメントでは、ERPアクセス制御システムの核となる **ログイン認証API** (`POST /api/v1/auth/login`) について、各コンポーネントの役割と動作を詳細に解説します。

## 🔄 **全体的な処理フロー**

```mermaid
sequenceDiagram
    participant Client as "クライアント"
    participant Handler as "AuthHandler.Login()"
    participant Service as "AuthService.LoginWithCredentials()"
    participant DB as "データベース"
    participant JWT as "JWTService"
    participant Perm as "PermissionService"
    
    Client->>Handler: POST /api/v1/auth/login<br/>{email, password}
    Handler->>Handler: リクエスト形式バリデーション
    Handler->>Service: LoginWithCredentials(email, password)
    Service->>DB: ユーザー検索 (email, 複数ロール情報付き)
    DB-->>Service: User + PrimaryRole + UserRoles
    Service->>Service: パスワード検証 (bcrypt)
    Service->>Service: ユーザーステータス確認
    Service->>Perm: GetUserPermissions(userID)
    Perm-->>Service: []string (権限一覧)
    Service->>Service: アクティブロール情報取得
    Service->>JWT: GenerateToken(userInfo, permissions, roles)
    JWT-->>Service: access_token (JWT)
    Service-->>Handler: UserInfo + access_token
    Handler->>Handler: レスポンス構築
    Handler-->>Client: LoginResponse (トークン + ユーザー情報)
```

---

## 🏗️ **コンポーネント詳細解析**

### 1. **AuthHandler.Login()** - HTTPハンドラ層
**ファイル**: `internal/handlers/auth.go:65-123`

#### **役割**
- HTTPリクエストの受信・バリデーション
- レスポンス形式の統一
- ログ記録・監査

#### **処理ステップ**
```go
func (h *AuthHandler) Login(c *gin.Context) {
    // 1. リクエストボディのJSON解析
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        // バリデーションエラー処理
    }
    
    // 2. ログイン試行ログ記録
    h.logger.Info("Login attempt", map[string]interface{}{
        "email": req.Email,
        "ip":    c.ClientIP(),
    })
    
    // 3. 認証サービス呼び出し
    userInfo, accessToken, err := h.authService.LoginWithCredentials(req.Email, req.Password)
    
    // 4. 権限情報取得
    permissions, err := h.authService.GetUserPermissions(userInfo.ID)
    
    // 5. レスポンス構築
    response := &LoginResponse{
        AccessToken: accessToken,
        TokenType:   "Bearer",
        ExpiresIn:   900, // 15分
        User:        userInfo,
        Permissions: permissions,
        ActiveRoles: userInfo.ActiveRoles,
        PrimaryRole: userInfo.PrimaryRole,
        HighestRole: userInfo.HighestRole,
    }
}
```

#### **データ構造**
```go
// LoginRequest - 入力データ
type LoginRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=6"`
}

// LoginResponse - 出力データ（複数ロール対応）
type LoginResponse struct {
    AccessToken string              `json:"access_token"`
    TokenType   string              `json:"token_type"`
    ExpiresIn   int64               `json:"expires_in"`
    User        *services.UserInfo  `json:"user"`
    Permissions []string            `json:"permissions"`
    ActiveRoles []services.RoleInfo `json:"active_roles"`
    PrimaryRole *services.RoleInfo  `json:"primary_role"`
    HighestRole *services.RoleInfo  `json:"highest_role"`
}
```

---

### 2. **AuthService.LoginWithCredentials()** - ビジネスロジック層
**ファイル**: `internal/services/auth.go:409-422`

#### **役割**
- 実際の認証処理の起点
- 内部の`Login()`メソッドを呼び出し、適切な形式でレスポンス

#### **処理ステップ**
```go
func (s *AuthService) LoginWithCredentials(email, password string) (*UserInfo, string, error) {
    req := LoginRequest{
        Email:    email,
        Password: password,
    }
    
    resp, err := s.Login(req)  // 内部のLogin()メソッド呼び出し
    if err != nil {
        return nil, "", err
    }
    
    return &resp.User, resp.Token, nil
}
```

---

### 3. **AuthService.Login()** - 核となる認証処理
**ファイル**: `internal/services/auth.go:80-183`

#### **役割**
- パスワード検証
- ユーザーステータス確認
- JWTトークン生成
- 複数ロール情報の管理

#### **詳細処理ステップ**

##### **3.1 ユーザー検索（複数ロール対応）**
```go
var user models.User
if err := s.db.Preload("PrimaryRole").Preload("Department").
    Preload("UserRoles.Role").  // 複数ロール情報も取得
    Where("email = ?", req.Email).First(&user).Error; err != nil {
    // ユーザーが見つからない場合の処理
}
```

**データベースクエリ**: 
- `users` テーブルからメールアドレスで検索
- 関連テーブルを事前読み込み:
  - `departments` (部署情報)
  - `roles` (プライマリロール)
  - `user_roles` → `roles` (複数ロール情報)

##### **3.2 ユーザーステータス確認**
```go
if user.Status != models.UserStatusActive {
    return nil, errors.NewAuthenticationError("user account is not active")
}
```

**チェック項目**:
- `active`: 通常利用可能
- `inactive`: 一時的な無効化
- `suspended`: 管理者による停止

##### **3.3 パスワード検証（bcrypt）**
```go
if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
    return nil, errors.NewAuthenticationError("invalid email or password")
}
```

**セキュリティ特徴**:
- ハッシュ化されたパスワードとの比較
- タイミング攻撃対策（常に一定時間でレスポンス）
- 平文パスワードは保存されない

##### **3.4 権限情報取得**
```go
permissions, err := s.permissionService.GetUserPermissions(user.ID)
if err != nil {
    return nil, errors.NewDatabaseError(err)
}
```

**取得内容**:
- ユーザーが持つ全権限の文字列配列
- モジュール×アクション形式 (`users:read`, `departments:write` など)

##### **3.5 アクティブロール情報取得**
```go
activeRoles, err := user.GetActiveRoles(s.db)
if err != nil {
    return nil, errors.NewDatabaseError(err)
}

highestRole, err := user.GetHighestPriorityRole(s.db)
if err != nil && err != gorm.ErrRecordNotFound {
    return nil, errors.NewDatabaseError(err)
}
```

**複数ロール管理**:
- **アクティブロール**: 現在有効なロール一覧
- **プライマリロール**: メインとなるロール
- **最高優先度ロール**: 権限レベルが最も高いロール

##### **3.6 JWTトークン生成**
```go
token, err := s.jwtService.GenerateToken(
    user.ID,
    user.Email,
    permissions,
    user.PrimaryRoleID,
    jwtActiveRoles,
    jwtHighestRole,
)
```

---

### 4. **JWT.Service.GenerateToken()** - トークン生成
**ファイル**: `pkg/jwt/jwt.go:44-65`

#### **役割**
- セキュアなJWTトークンの生成
- 複数ロール情報をクレームに埋め込み
- 有効期限・署名の管理

#### **処理ステップ**
```go
func (s *Service) GenerateToken(userID uuid.UUID, email string, permissions []string, 
    primaryRoleID *uuid.UUID, activeRoles []RoleInfo, highestRole *RoleInfo) (string, error) {
    
    // 1. クレーム構築
    claims := CustomClaims{
        UserID:        userID,
        Email:         email,
        Permissions:   permissions,
        PrimaryRoleID: primaryRoleID,
        ActiveRoles:   activeRoles,
        HighestRole:   highestRole,
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   userID.String(),
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiresIn)), // 15分
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            ID:        uuid.New().String(), // JTI (JWT ID)
            Issuer:    "erp-access-control-api",
        },
    }
    
    // 2. HS256署名でトークン生成
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.secretKey)
}
```

#### **JWTクレーム構造**
```go
type CustomClaims struct {
    UserID         uuid.UUID  `json:"user_id"`
    Email          string     `json:"email"`
    Permissions    []string   `json:"permissions"`      // ユーザーが持つ全権限
    PrimaryRoleID  *uuid.UUID `json:"primary_role_id"`  // メインロールID
    ActiveRoles    []RoleInfo `json:"active_roles"`     // アクティブロール一覧
    HighestRole    *RoleInfo  `json:"highest_role"`     // 最高優先度ロール
    jwt.RegisteredClaims                                 // 標準クレーム
}
```

---

### 5. **PermissionService.GetUserPermissions()** - 権限管理
**ファイル**: `internal/services/permission.go` (呼び出し元)

#### **役割**
- ユーザーが保有する権限の集約
- 複数ロールからの権限統合
- 権限の重複除去

#### **処理概要**
```go
func (s *PermissionService) GetUserPermissions(userID uuid.UUID) ([]string, error) {
    // 1. ユーザーのアクティブロールを取得
    // 2. 各ロールが持つ権限を取得
    // 3. 権限を統合・重複除去
    // 4. string配列で返却
}
```

---

## 🔒 **セキュリティ機能**

### 1. **パスワードセキュリティ**
```go
// ハッシュ化（登録時）
func (u *User) HashPassword() error {
    hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    u.PasswordHash = string(hash)
    return nil
}

// 検証（ログイン時）
if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
    return nil, errors.NewAuthenticationError("invalid email or password")
}
```

**特徴**:
- bcrypt による不可逆ハッシュ化
- ソルト自動生成による辞書攻撃対策
- 平文パスワードはメモリに残さない

### 2. **JWTセキュリティ**
```go
RegisteredClaims: jwt.RegisteredClaims{
    Subject:   userID.String(),
    ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiresIn)), // 15分で期限切れ
    IssuedAt:  jwt.NewNumericDate(time.Now()),
    NotBefore: jwt.NewNumericDate(time.Now()),
    ID:        uuid.New().String(), // 一意なJTI
    Issuer:    "erp-access-control-api",
}
```

**特徴**:
- 短期間有効期限（15分）によるリスク軽減
- JTI（JWT ID）による個別トークン識別
- HS256署名による改ざん検知

### 3. **ログ・監査**
```go
// ログイン試行ログ
h.logger.Info("Login attempt", map[string]interface{}{
    "email": req.Email,
    "ip":    c.ClientIP(),
})

// ログイン成功ログ
h.logger.Info("Login successful", map[string]interface{}{
    "user_id": userInfo.ID,
    "email":   userInfo.Email,
    "ip":      c.ClientIP(),
})

// ログイン失敗ログ
h.logger.Warn("Login failed", map[string]interface{}{
    "email": req.Email,
    "error": err.Error(),
    "ip":    c.ClientIP(),
})
```

**監査項目**:
- IPアドレス記録
- 試行・成功・失敗の全ログ
- エラー詳細（デバッグ用）

---

## ⚡ **認証後の流れ**

### 1. **認証ミドルウェア** - `AuthMiddleware.Authentication()`
**ファイル**: `internal/middleware/auth.go:100-167`

#### **処理フロー**
```go
func (m *AuthMiddleware) Authentication() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Authorization ヘッダー確認
        authHeader := c.GetHeader("Authorization")
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        
        // 2. JWTトークン検証
        claims, err := m.jwtService.ValidateToken(tokenString)
        
        // 3. トークン失効確認
        if err := m.revocationService.ValidateTokenStatus(claims.ID, claims.UserID, claims.IssuedAt.Time); err != nil {
            // 失効済みトークンの拒否
        }
        
        // 4. ユーザー情報をコンテキストに保存
        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)
        c.Set("permissions", claims.Permissions)
        c.Set("primary_role_id", claims.PrimaryRoleID)
        c.Set("active_roles", claims.ActiveRoles)
        c.Set("highest_role", claims.HighestRole)
        
        c.Next()
    }
}
```

### 2. **権限チェックミドルウェア** - `RequirePermissions()`
**ファイル**: `internal/middleware/auth.go:169-196`

#### **権限チェック機能**
```go
func RequirePermissions(permissions ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userPermissions, _ := c.Get("permissions")
        
        for _, requiredPerm := range permissions {
            if !hasPermission(userPermissions.([]string), requiredPerm) {
                c.Error(errors.NewAuthorizationError(fmt.Sprintf("Missing required permission: %s", requiredPerm)))
                c.Abort()
                return
            }
        }
        c.Next()
    }
}
```

**権限マッチング機能**:
```go
func hasPermission(userPermissions []string, requiredPermission string) bool {
    for _, perm := range userPermissions {
        // 完全一致
        if perm == requiredPermission { return true }
        
        // 全権限ワイルドカード
        if perm == "*" || perm == "*:*" { return true }
        
        // モジュール別ワイルドカード（例: "user:*"）
        if strings.Contains(requiredPermission, ":") && strings.HasSuffix(perm, ":*") {
            requiredModule := strings.Split(requiredPermission, ":")[0]
            permModule := strings.TrimSuffix(perm, ":*")
            if requiredModule == permModule { return true }
        }
    }
    return false
}
```

---

## 🚨 **エラーハンドリング**

### 1. **認証エラーの種類**
```go
// 1. バリデーションエラー
errors.NewValidationError("request", "Invalid request format")

// 2. 認証エラー
errors.NewAuthenticationError("invalid email or password")
errors.NewAuthenticationError("user account is not active")

// 3. データベースエラー
errors.NewDatabaseError(err)

// 4. 内部エラー
errors.NewInternalError("Failed to get user permissions")
```

### 2. **セキュリティ考慮事項**
- **タイミング攻撃対策**: 存在しないユーザーでも一定時間レスポンス
- **情報漏洩防止**: 「ユーザーが存在しない」「パスワードが間違い」を区別しない
- **ブルートフォース対策**: ログに IP アドレスを記録（将来的にレート制限実装予定）

---

## 📊 **パフォーマンス特徴**

### 1. **データベース最適化**
```go
// Preload による N+1 問題回避
s.db.Preload("PrimaryRole").Preload("Department").
    Preload("UserRoles.Role").
    Where("email = ?", req.Email).First(&user)
```

### 2. **メモリ効率**
- パスワード平文の即座削除
- JWT クレームの適切なサイズ管理
- 不要なデータの除外

### 3. **拡張性**
- 複数ロール対応
- ワイルドカード権限
- 将来的な MFA・レート制限対応準備

---

## 🔄 **使用例**

### 1. **正常なログインフロー**
```bash
# リクエスト
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "password123"
  }'

# レスポンス
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "管理者",
    "email": "admin@example.com",
    "status": "active",
    "primary_role": {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "name": "システム管理者",
      "priority": 1
    },
    "active_roles": [...],
    "highest_role": {...},
    "department": {
      "id": "550e8400-e29b-41d4-a716-446655440002",
      "name": "情報システム部"
    }
  },
  "permissions": [
    "*:*",
    "users:read",
    "users:write",
    "departments:read"
  ],
  "active_roles": [...],
  "primary_role": {...},
  "highest_role": {...}
}
```

### 2. **後続APIでの認証利用**
```bash
# 認証が必要なAPIの呼び出し
curl -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

## 🎯 **まとめ**

ログイン認証API (`POST /api/v1/auth/login`) は、以下の特徴を持つ堅牢で実用的な実装となっています：

### **技術的強み**
1. **多層セキュリティ**: bcrypt + JWT + 権限マトリックス
2. **複数ロール対応**: エンタープライズレベルの権限管理
3. **包括的監査**: 全アクセスログ・エラーログ記録
4. **高性能設計**: N+1問題回避・効率的なデータ取得

### **実装品質**
1. **明確な責任分離**: Handler → Service → Repository パターン
2. **堅牢なエラーハンドリング**: セキュリティ考慮事項込み
3. **拡張可能な設計**: 将来的な MFA・レート制限対応準備済み
4. **開発者体験**: 型安全・IDE補完・包括的テスト

この実装により、企業レベルのERPシステムに求められる認証・認可要件を満たしつつ、開発・運用・保守の観点からも優れたアーキテクチャを実現しています。

---

#### **🔰 Go言語初学者向け - 権限チェック機能の詳細解説**

---

### **パターン1: 権限チェック機能（RequirePermissions）**

```go
func RequirePermissions(permissions ...string) gin.HandlerFunc {
//   ↑関数名            ↑可変長引数        ↑戻り値の型
    return func(c *gin.Context) {
//    ↑匿名関数を返す     ↑Ginコンテキスト
        userPermissions, _ := c.Get("permissions")
//      ↑変数宣言,代入    ↑ブランク識別子   ↑コンテキストから値取得
        
        for _, requiredPerm := range permissions {
//      ↑for-range文  ↑ブランク識別子   ↑スライスをループ
            if !hasPermission(userPermissions.([]string), requiredPerm) {
//             ↑否定演算子      ↑型アサーション(断言)
                c.Error(errors.NewAuthorizationError(fmt.Sprintf("Missing required permission: %s", requiredPerm)))
//                     ↑カスタムエラー生成            ↑文字列フォーマット
                c.Abort()
//              ↑リクエスト処理を中断
                return
//              ↑関数から早期リターン
            }
        }
        c.Next()
//      ↑次のミドルウェア/ハンドラに処理を移す
    }
}
```

#### **📚 Go言語文法の解説**

**1. 可変長引数 (`...string`)**
```go
func RequirePermissions(permissions ...string) gin.HandlerFunc
//                                  ↑このドット3つが可変長引数の記法
```
- **意味**: 任意の数の`string`型引数を受け取る
- **実際の呼び出し例**:
  ```go
  RequirePermissions("user:read")                    // 1つの権限
  RequirePermissions("user:read", "user:write")      // 2つの権限
  RequirePermissions("admin:system", "user:manage")  // 管理権限
  ```
- **関数内では**: `permissions`は`[]string`型のスライスとして扱われる

**2. 高階関数（関数を返す関数）**
```go
func RequirePermissions(...) gin.HandlerFunc {
    return func(c *gin.Context) {  // ← この部分が「匿名関数」
        // 実際の権限チェック処理
    }
}
```
- **目的**: Ginミドルウェアの仕組みに合わせるため
- **動作**: 設定された権限要件を「記憶」した関数を生成
- **実用例**: `users.DELETE("/:id", RequirePermissions("user:delete"), handler)`

**3. 型アサーション (`.([]string)`)**
```go
userPermissions, _ := c.Get("permissions")              // interface{}型で取得
if !hasPermission(userPermissions.([]string), requiredPerm) {  // []string型に変換
//                               ↑型アサーション
```
- **必要な理由**: `c.Get()`は`interface{}`型を返すため
- **安全な変換**: Go言語の型安全性を保つための仕組み
- **失敗時**: パニックが発生（本来はok変数でチェックすべき）

**4. ブランク識別子 (`_`)**
```go
userPermissions, _ := c.Get("permissions")
//               ↑ここでエラー値を無視
for _, requiredPerm := range permissions {
//  ↑ここでインデックスを無視
```
- **用途**: 使わない戻り値やループ変数を無視
- **Go言語の特徴**: 使わない変数があるとコンパイルエラーになるため

#### **🎯 この関数の責務と動作**

1. **設定フェーズ**: 必要な権限リストを受け取り、チェック関数を生成
2. **実行フェーズ**: HTTPリクエスト時に以下を実行
   - コンテキストからユーザーの権限リストを取得
   - 必要な権限が全て含まれているかチェック（AND条件）
   - 権限不足の場合: エラーを設定してリクエストを中断
   - 権限十分の場合: 次の処理に進む

---

### **パターン2: 権限マッチング機能（hasPermission）**

```go
func hasPermission(userPermissions []string, requiredPermission string) bool {
//   ↑関数名        ↑スライス型引数      ↑文字列型引数           ↑bool型戻り値
    for _, perm := range userPermissions {
//  ↑for-range文で配列の各要素をループ
        // 完全一致チェック
        if perm == requiredPermission { 
//         ↑文字列の等価比較
            return true 
//          ↑条件に合致したら即座にtrueを返して関数終了
        }
        
        // 全権限ワイルドカードチェック
        if perm == "*" || perm == "*:*" { 
//                    ↑論理OR演算子
            return true 
        }
        
        // モジュール別ワイルドカードチェック（例: "user:*"）
        if strings.Contains(requiredPermission, ":") && strings.HasSuffix(perm, ":*") {
//         ↑文字列に":"が含まれるかチェック    ↑論理AND演算子  ↑文字列が":*"で終わるかチェック
            requiredModule := strings.Split(requiredPermission, ":")[0]
//          ↑新しい変数宣言    ↑文字列を":"で分割してスライス化  ↑最初の要素を取得
            permModule := strings.TrimSuffix(perm, ":*")
//          ↑変数宣言         ↑文字列の末尾から":*"を除去
            if requiredModule == permModule { 
                return true 
            }
        }
    }
    return false
//  ↑どの条件にも合致しなかった場合はfalseを返す
}
```

#### **📚 Go言語文法の解説**

**1. for-range文**
```go
for _, perm := range userPermissions {
//  ↑インデックス無視  ↑要素の値  ↑スライスをループ
```
- **他言語との比較**:
  ```javascript
  // JavaScript
  for (let perm of userPermissions) { ... }
  ```
  ```python
  # Python
  for perm in user_permissions:
  ```

**2. strings パッケージの関数**
```go
strings.Contains(requiredPermission, ":")    // 文字列に特定文字が含まれるか
strings.HasSuffix(perm, ":*")               // 文字列が特定文字列で終わるか
strings.Split(requiredPermission, ":")[0]   // 文字列を分割して最初の要素
strings.TrimSuffix(perm, ":*")              // 文字列末尾から特定文字列を除去
```

**3. 論理演算子**
```go
if perm == "*" || perm == "*:*" {           // OR演算子：どちらかがtrueなら
if strings.Contains(...) && strings.HasSuffix(...) { // AND演算子：両方がtrueなら
```

**4. 早期リターン（Early Return）パターン**
```go
if 条件1 { return true }   // 条件1に合致したら即座に終了
if 条件2 { return true }   // 条件2に合致したら即座に終了
if 条件3 { return true }   // 条件3に合致したら即座に終了
return false              // どの条件にも合致しなかった場合
```
- **メリット**: ネストが深くならず、読みやすいコード

#### **🎯 この関数の責務と動作例**

**基本的な動作**:
```go
// 例1: 完全一致
hasPermission(["user:read", "user:write"], "user:read")  // → true

// 例2: 権限なし
hasPermission(["user:read"], "user:delete")              // → false

// 例3: 全権限ワイルドカード
hasPermission(["*:*"], "任意の権限")                     // → true

// 例4: モジュール別ワイルドカード
hasPermission(["user:*"], "user:delete")                // → true
hasPermission(["user:*"], "department:read")            // → false
```

**詳細な処理フロー**:
1. **完全一致チェック**: `"user:read" == "user:read"` → 一致したらtrue
2. **全権限チェック**: `"*"` または `"*:*"` があれば全ての権限を許可
3. **モジュール別チェック**: 
   - 要求権限: `"user:delete"` 
   - ユーザー権限: `"user:*"`
   - モジュール部分 `"user"` が一致するかチェック → true

#### **🔧 実際の使用例**

```go
// APIルーティングでの実際の使用例
users := group.Group("/users")
{
    // "user:create" 権限が必要
    users.POST("", RequirePermissions("user:create"), userHandler.CreateUser)
    
    // "user:read" 権限が必要  
    users.GET("/:id", RequirePermissions("user:read"), userHandler.GetUser)
    
    // 複数権限が必要（全て必要）
    users.DELETE("/:id", RequirePermissions("user:delete", "admin:system"), userHandler.DeleteUser)
}

// 権限の実際の判定例
userPermissions := []string{"user:*", "department:read"}
hasPermission(userPermissions, "user:create")    // true（user:*にマッチ）
hasPermission(userPermissions, "user:delete")    // true（user:*にマッチ）
hasPermission(userPermissions, "role:create")    // false（権限なし）
```

#### **💡 初学者向けのポイント**

1. **型の重要性**: Go言語は静的型付け言語。型アサーションで安全に変換
2. **エラーハンドリング**: 本来は`userPermissions, ok := c.Get("permissions")`でokチェック
3. **早期リターン**: 条件に合致したら即座にreturnで処理を終了
4. **文字列操作**: stringsパッケージの豊富な関数を活用
5. **スライス操作**: for-rangeでシンプルにループ処理

この権限チェックシステムにより、APIエンドポイントごとに細かい権限制御が可能になり、セキュリティを確保しています。
```
