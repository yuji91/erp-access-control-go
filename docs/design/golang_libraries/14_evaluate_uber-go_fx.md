# ERP Access Control API - uber-go/fx評価

## 📊 現状分析：uber-go/fxは実際に使用されていない

### 🔍 コードベース調査結果

**重要な発見**：このERPアクセス制御APIプロジェクトでは、uber-go/fxは**実際には使用されていません**。

#### 1. go.mod確認結果
```go
// go.modには uber-go/fx の依存関係は含まれていない
require (
    github.com/gin-gonic/gin v1.10.1
    github.com/golang-jwt/jwt/v5 v5.2.3
    gorm.io/gorm v1.30.1
    // ... 他の依存関係
)
// uber-go/fx は一切記載されていない
```

#### 2. 実装状況
- **main.go**: 手動でのDI（Dependency Injection）を実装
- **DI方式**: カスタムの`ServiceContainer`構造体を使用
- **ライフサイクル管理**: 手動でサービス初期化を実行

## 🏗️ 現在の手動DI実装

### ServiceContainer パターン
```go
// ServiceContainer サービスコンテナ
type ServiceContainer struct {
    Auth       *services.AuthService
    Permission *services.PermissionService
    Revocation *services.TokenRevocationService
    UserRole   *services.UserRoleService
    User       *services.UserService
    Department *services.DepartmentService
    Role       *services.RoleService
    JWT        *jwt.Service
}

// MiddlewareContainer ミドルウェアコンテナ  
type MiddlewareContainer struct {
    Auth *middleware.AuthMiddleware
}
```

### 手動初期化プロセス
```go
func main() {
    // 1. 設定読み込み
    cfg, err := config.Load()
    
    // 2. ロガー初期化
    appLogger := initLogger(cfg)
    
    // 3. データベース接続
    db, err := initDatabase(cfg)
    
    // 4. サービス初期化（手動DI）
    services := initServices(db, cfg)
    
    // 5. ミドルウェア初期化
    middlewares := initMiddlewares(services, appLogger)
    
    // 6. ルーター設定
    router := setupRoutes(services, middlewares, appLogger)
    
    // 7. サーバー起動
    startServer(router, cfg.Server.Port)
}
```

### 手動依存注入の例
```go
func initServices(db *gorm.DB, cfg *config.Config) *ServiceContainer {
    // ロガー初期化（重複コード）
    appLogger := logger.NewLogger(...)
    
    // JWT サービス
    jwtService := jwt.NewService(cfg.JWT.Secret, cfg.JWT.AccessTokenDuration)
    
    // 基本サービス（依存関係を手動管理）
    permissionService := services.NewPermissionService(db, appLogger)
    revocationService := services.NewTokenRevocationService(db)
    userService := services.NewUserService(db, appLogger)
    // ...
    
    // 認証サービス（複数の依存関係を手動注入）
    authService := services.NewAuthService(
        db,
        jwtService, 
        permissionService,
        revocationService,
    )
    
    return &ServiceContainer{...}
}
```

## 🎯 uber-go/fx採用のメリット

### 1. 依存関係の自動解決
**現在の問題**：
- 依存関係を手動で管理する必要がある
- 初期化順序を開発者が制御する必要がある  
- サービス追加時に複数箇所の修正が必要

**fxでの解決**：
```go
// fx採用版のmain.go例
func main() {
    fx.New(
        // モジュール登録
        config.Module,
        database.Module, 
        logger.Module,
        services.Module,
        handlers.Module,
        middleware.Module,
        server.Module,
        
        // 起動時処理
        fx.Invoke(startServer),
    ).Run()
}
```

### 2. モジュール化された設計
**現在の問題**：
- 全サービスが単一ファイルで初期化される
- テスト時の部分的なMockが困難
- 設定変更時の影響範囲が広い

**fxでの解決**：
```go
// services/module.go
var Module = fx.Options(
    fx.Provide(
        NewPermissionService,
        NewUserService,
        NewAuthService,
        fx.Annotated{
            Name: "admin",
            Target: NewAdminPolicyResolver,
        },
        fx.Annotated{
            Name: "user", 
            Target: NewUserPolicyResolver,
        },
    ),
)

// handlers/module.go  
var Module = fx.Options(
    fx.Provide(
        NewAuthHandler,
        NewUserHandler,
        NewPermissionHandler,
    ),
)
```

### 3. 柔軟なポリシー切り替え
**現在の制限**：
- ポリシーの動的切り替えが困難
- 部門別・ロール別の権限ポリシーが固定化される

**fxでの解決**：
```go
// 名前付きバインディングによるポリシー分離
type PermissionHandler struct {
    adminPolicy fx.In
    userPolicy  fx.In
    
    AdminResolver PolicyResolver `name:"admin"`
    UserResolver  PolicyResolver `name:"user"`
}

func (h *PermissionHandler) CheckPermission(c *gin.Context) {
    userRole := getUserRole(c)
    
    switch userRole {
    case "admin":
        allowed := h.AdminResolver.Can(...)
    case "user":
        allowed := h.UserResolver.Can(...)
    }
}
```

### 4. テスト容易性の向上
**現在の問題**：
- 統合テスト時に全サービスの初期化が必要
- Mock注入が複雑

**fxでの解決**：
```go
// テスト専用構成
func TestApp() *fx.App {
    return fx.New(
        services.Module,
        
        // 本番サービスをMockに置き換え
        fx.Replace(
            fx.Annotated{
                Name: "db",
                Target: func() *gorm.DB { return mockDB },
            },
        ),
        
        fx.Populate(&testContainer),
    )
}
```

## ⚠️ uber-go/fx非採用時のトレードオフ

### 現在の実装で生じている課題

#### 1. 初期化処理の重複
```go
// initLogger関数とinitServices関数で重複したロガー初期化
func initLogger(cfg *config.Config) *logger.Logger {
    var minLevel logger.LogLevel
    switch cfg.Environment {
    case "production": minLevel = logger.WARN
    case "staging": minLevel = logger.INFO
    default: minLevel = logger.DEBUG
    }
    return logger.NewLogger(...)
}

func initServices(db *gorm.DB, cfg *config.Config) *ServiceContainer {
    // 同じロジックがここでも重複
    var minLevel logger.LogLevel
    switch cfg.Environment {
    case "production": minLevel = logger.WARN
    // ...
}
```

#### 2. 依存関係の線形初期化
```go
// 現在：依存関係を手動で線形に初期化
permissionService := services.NewPermissionService(db, appLogger)
revocationService := services.NewTokenRevocationService(db)
userRoleService := services.NewUserRoleService(db)
userService := services.NewUserService(db, appLogger)

// 新しいサービス追加時、全ての依存関係を再確認する必要がある
authService := services.NewAuthService(
    db,
    jwtService,
    permissionService,  // ← この順序を間違えるとpanicの可能性
    revocationService,
)
```

#### 3. 設定変更時の広範囲修正
部門別ポリシーを追加する場合：
```go
// 現在：複数箇所の修正が必要
type ServiceContainer struct {
    // 既存フィールド...
    
    // 新規追加：部門別ポリシー
    SalesPolicy      PolicyResolver  // ← 新規追加
    AccountingPolicy PolicyResolver  // ← 新規追加
    HRPolicy         PolicyResolver  // ← 新規追加
}

func initServices(...) *ServiceContainer {
    // 新規ポリシーの初期化を手動追加
    salesPolicy := policies.NewSalesPolicy(db, cfg)
    accountingPolicy := policies.NewAccountingPolicy(db, cfg)
    hrPolicy := policies.NewHRPolicy(db, cfg)
    
    return &ServiceContainer{
        // 既存フィールド...
        SalesPolicy:      salesPolicy,      // ← 手動追加
        AccountingPolicy: accountingPolicy, // ← 手動追加  
        HRPolicy:         hrPolicy,         // ← 手動追加
    }
}

// setupRoutes関数も修正が必要
func setupRoutes(services *ServiceContainer, ...) {
    // 各エンドポイントに部門別ポリシーを手動で渡す必要
    setupUserRoutes(protected, services.User, services.SalesPolicy, appLogger)
    setupDepartmentRoutes(protected, services.Department, services.HRPolicy, appLogger)
    // ...
}
```

## 🚀 uber-go/fx導入時の改善コード例

### 1. モジュール化された設計
```go
// config/module.go
var Module = fx.Options(
    fx.Provide(Load),
)

// database/module.go  
var Module = fx.Options(
    fx.Provide(NewConnection),
)

// policies/module.go
var Module = fx.Options(
    fx.Provide(
        // 部門別ポリシーを名前付きで登録
        fx.Annotated{Name: "sales", Target: NewSalesPolicy},
        fx.Annotated{Name: "accounting", Target: NewAccountingPolicy},
        fx.Annotated{Name: "hr", Target: NewHRPolicy},
        fx.Annotated{Name: "default", Target: NewDefaultPolicy},
    ),
)
```

### 2. 自動依存注入
```go
// services/auth.go
type AuthService struct {
    db                *gorm.DB
    jwtService        *jwt.Service
    permissionService *PermissionService
    revocationService *TokenRevocationService
    logger           *logger.Logger
}

// fx.Inタグで自動注入
func NewAuthService(
    db *gorm.DB,
    jwt *jwt.Service,
    perm *PermissionService,
    revoke *TokenRevocationService,
    log *logger.Logger,
) *AuthService {
    return &AuthService{
        db:                db,
        jwtService:        jwt,
        permissionService: perm,
        revocationService: revoke,
        logger:           log,
    }
}
```

### 3. 柔軟なハンドラ構成
```go
// handlers/user.go
type UserHandler struct {
    userService *services.UserService
    policies    map[string]PolicyResolver
    logger      *logger.Logger
}

// 複数ポリシーを自動注入
type PolicyResolvers struct {
    fx.In
    
    Sales      PolicyResolver `name:"sales"`
    Accounting PolicyResolver `name:"accounting"`
    HR         PolicyResolver `name:"hr"`
    Default    PolicyResolver `name:"default"`
}

func NewUserHandler(
    userService *services.UserService,
    resolvers PolicyResolvers,
    logger *logger.Logger,
) *UserHandler {
    return &UserHandler{
        userService: userService,
        policies: map[string]PolicyResolver{
            "sales":      resolvers.Sales,
            "accounting": resolvers.Accounting,
            "hr":         resolvers.HR,
            "default":    resolvers.Default,
        },
        logger: logger,
    }
}

func (h *UserHandler) CreateUser(c *gin.Context) {
    department := getDepartmentFromContext(c)
    
    // 部門に応じたポリシーを動的選択
    policy := h.policies[department]
    if policy == nil {
        policy = h.policies["default"]
    }
    
    if allowed, reason := policy.Can(c, "user:create"); !allowed {
        c.JSON(403, gin.H{"error": reason})
        return
    }
    
    // ユーザー作成処理...
}
```

### 4. 簡潔なmain.go
```go
// cmd/server/main.go  
func main() {
    fx.New(
        // モジュール自動登録
        config.Module,
        database.Module,
        logger.Module,
        policies.Module,
        services.Module,
        handlers.Module,
        middleware.Module,
        
        // Ginサーバー起動
        fx.Provide(NewGinEngine),
        fx.Invoke(RegisterRoutes),
        fx.Invoke(StartServer),
    ).Run()
}

// 起動処理をシンプル化
func StartServer(engine *gin.Engine, cfg *config.Config, lc fx.Lifecycle) {
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            go engine.Run(":" + cfg.Server.Port)
            return nil
        },
        OnStop: func(ctx context.Context) error {
            // グレースフルシャットダウン
            return nil
        },
    })
}
```

## 📊 比較まとめ

| 観点 | 現在の手動DI | uber-go/fx採用時 |
|------|-------------|-----------------|
| **コード量** | 大量の初期化コード | モジュール定義のみ |
| **依存関係管理** | 手動・エラーが起きやすい | 自動解決・型安全 |
| **新機能追加** | 複数箇所修正が必要 | モジュール追加のみ |
| **テスト性** | Mock注入が複雑 | fx.Replace()で簡単 |
| **ポリシー切り替え** | 固定的・変更困難 | 名前付きで柔軟 |
| **保守性** | 依存関係が不明確 | 明確な関係性 |
| **学習コスト** | 低（単純な構造） | 中（fxの概念理解） |

## 🎯 推奨事項

### 短期的対応（現状維持）
- 現在の手動DIでも十分機能している
- 小規模チームでは学習コストが優先される場合

### 長期的対応（fx導入推奨）
- チーム規模拡大時の保守性向上
- 複雑な権限ポリシーの柔軟な管理
- エンタープライズグレードの拡張性確保

**結論**：ERPアクセス制御システムの将来的な拡張性を考慮すると、uber-go/fx の導入は**長期的に大きなメリット**をもたらすと評価されます。
