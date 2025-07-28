package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"erp-access-control-go/internal/config"
	"erp-access-control-go/internal/handlers"
	"erp-access-control-go/internal/middleware"
	"erp-access-control-go/internal/services"
	"erp-access-control-go/pkg/jwt"
	"erp-access-control-go/pkg/logger"
)

func main() {
	// 設定読み込み
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ 設定読み込みエラー: %v", err)
	}

	// ロガー初期化
	appLogger := initLogger(cfg)

	// データベース接続
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("❌ データベース接続エラー: %v", err)
	}

	// サービス初期化
	services := initServices(db, cfg)

	// ミドルウェア初期化
	middlewares := initMiddlewares(services, appLogger)

	// Ginルーター初期化
	router := setupRoutes(services, middlewares, appLogger)

	// サーバー起動
	startServer(router, cfg.Server.Port)
}

// initLogger ロガーを初期化
func initLogger(cfg *config.Config) *logger.Logger {
	var minLevel logger.LogLevel
	switch cfg.Environment {
	case "production":
		minLevel = logger.WARN
	case "staging":
		minLevel = logger.INFO
	default:
		minLevel = logger.DEBUG
	}

	return logger.NewLogger(
		logger.WithMinLevel(minLevel),
		logger.WithEnvironment(cfg.Environment),
	)
}

// initDatabase データベース接続を初期化
func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.Database.GetDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Printf("✅ データベース接続成功: %s", cfg.Database.Name)
	return db, nil
}

// initServices 全サービスを初期化
func initServices(db *gorm.DB, cfg *config.Config) *ServiceContainer {
	// ロガー初期化
	var minLevel logger.LogLevel
	switch cfg.Environment {
	case "production":
		minLevel = logger.WARN
	case "staging":
		minLevel = logger.INFO
	default:
		minLevel = logger.DEBUG
	}

	appLogger := logger.NewLogger(
		logger.WithMinLevel(minLevel),
		logger.WithEnvironment(cfg.Environment),
	)

	// JWT サービス
	jwtService := jwt.NewService(cfg.JWT.Secret, cfg.JWT.AccessTokenDuration)

	// 基本サービス
	permissionService := services.NewPermissionService(db, appLogger)
	revocationService := services.NewTokenRevocationService(db)
	userRoleService := services.NewUserRoleService(db)
	userService := services.NewUserService(db, appLogger)
	departmentService := services.NewDepartmentService(db, appLogger)
	roleService := services.NewRoleService(db, appLogger)

	// 認証サービス
	authService := services.NewAuthService(
		db,
		jwtService,
		permissionService,
		revocationService,
	)

	return &ServiceContainer{
		Auth:       authService,
		Permission: permissionService,
		Revocation: revocationService,
		UserRole:   userRoleService,
		User:       userService,
		Department: departmentService,
		Role:       roleService,
		JWT:        jwtService,
	}
}

// initMiddlewares ミドルウェアを初期化
func initMiddlewares(services *ServiceContainer, appLogger *logger.Logger) *MiddlewareContainer {
	authMiddleware := middleware.NewAuthMiddleware(
		services.JWT,
		services.Revocation,
		appLogger,
	)

	return &MiddlewareContainer{
		Auth: authMiddleware,
	}
}

// setupRoutes ルーティングを設定
func setupRoutes(services *ServiceContainer, middlewares *MiddlewareContainer, appLogger *logger.Logger) *gin.Engine {
	// 開発モード設定
	gin.SetMode(gin.DebugMode)

	router := gin.Default()

	// エラーハンドリングミドルウェア
	router.Use(middleware.ErrorHandler(appLogger))

	// CORS設定（開発用）
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 基本エンドポイント
	setupBasicRoutes(router)

	// API v1 ルート
	v1 := router.Group("/api/v1")
	{
		// 認証エンドポイント
		setupAuthRoutes(v1, services.Auth, middlewares, appLogger)

		// 認証が必要なエンドポイント
		protected := v1.Group("")
		protected.Use(middlewares.Auth.Authentication())
		{
			// ユーザー管理
			setupUserRoutes(protected, services.User, appLogger)

			// ユーザーロール管理
			setupUserRoleRoutes(protected, services.UserRole)

			// 部署管理
			setupDepartmentRoutes(protected, services.Department, appLogger)

			// ロール管理
			setupRoleRoutes(protected, services.Role, appLogger)

			// 権限管理
			setupPermissionRoutes(protected, services.Permission, appLogger)
		}
	}

	return router
}

// setupBasicRoutes 基本エンドポイントを設定
func setupBasicRoutes(router *gin.Engine) {
	// ヘルスチェック
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "erp-access-control-api",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   "0.1.0-dev",
		})
	})

	// バージョン情報
	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "ERP Access Control API",
			"version": "0.1.0-dev",
			"status":  "development",
			"message": "API実装準備完了 - 複数ロール対応",
		})
	})

	// ルートエンドポイント
	router.GET("/", func(c *gin.Context) {
		// HTMLレスポンスで見やすく表示
		html := `<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>🔐 ERP Access Control API</title>
    <style>
        body {
            font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: #333;
            margin: 0;
            padding: 20px;
            min-height: 100vh;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: rgba(255, 255, 255, 0.95);
            border-radius: 15px;
            padding: 30px;
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.1);
        }
        .header {
            text-align: center;
            margin-bottom: 40px;
            border-bottom: 3px solid #667eea;
            padding-bottom: 20px;
        }
        .header h1 {
            margin: 0;
            color: #667eea;
            font-size: 2.5em;
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.1);
        }
        .status {
            background: linear-gradient(90deg, #28a745, #20c997);
            color: white;
            padding: 8px 20px;
            border-radius: 25px;
            display: inline-block;
            margin-top: 15px;
            font-weight: bold;
            box-shadow: 0 4px 15px rgba(40, 167, 69, 0.3);
        }
        .features {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin: 30px 0;
        }
        .feature-card {
            background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
            color: white;
            padding: 20px;
            border-radius: 10px;
            text-align: center;
            font-weight: bold;
            box-shadow: 0 8px 25px rgba(240, 147, 251, 0.3);
            transition: transform 0.3s ease;
        }
        .feature-card:hover {
            transform: translateY(-5px);
        }
        .endpoints-section {
            background: #f8f9fa;
            border-radius: 10px;
            padding: 25px;
            margin-top: 30px;
            border-left: 5px solid #667eea;
        }
        .endpoints-title {
            color: #667eea;
            font-size: 1.8em;
            margin-bottom: 20px;
            font-weight: bold;
        }
        .endpoint-category {
            margin-bottom: 25px;
        }
        .category-title {
            background: linear-gradient(90deg, #667eea, #764ba2);
            color: white;
            padding: 10px 15px;
            border-radius: 8px;
            font-weight: bold;
            margin-bottom: 15px;
            font-size: 1.1em;
        }
        .endpoint {
            background: white;
            margin: 8px 0;
            padding: 12px 20px;
            border-radius: 8px;
            border-left: 4px solid #28a745;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.05);
            transition: all 0.3s ease;
            font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
        }
        .endpoint:hover {
            transform: translateX(5px);
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
        }
        .method {
            font-weight: bold;
            color: #fff;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 0.9em;
            margin-right: 10px;
        }
        .get { background: #007bff; }
        .post { background: #28a745; }
        .put { background: #ffc107; color: #333; }
        .patch { background: #6f42c1; }
        .delete { background: #dc3545; }
        .path {
            color: #333;
            font-weight: bold;
            margin-right: 15px;
        }
        .description {
            color: #666;
            font-style: italic;
        }
        .footer {
            text-align: center;
            margin-top: 40px;
            padding: 20px;
            background: linear-gradient(90deg, #667eea, #764ba2);
            color: white;
            border-radius: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔐 ERP Access Control API</h1>
            <div class="status">✅ システム稼働中</div>
        </div>

        <div class="features">
            <div class="feature-card">
                <div>🔄 多重ロール管理</div>
            </div>
            <div class="feature-card">
                <div>⏰ 期限付きロール</div>
            </div>
            <div class="feature-card">
                <div>🏗️ 階層的権限</div>
            </div>
            <div class="feature-card">
                <div>🔐 JWT認証</div>
            </div>
        </div>

        <div class="endpoints-section">
            <div class="endpoints-title">📡 利用可能なAPIエンドポイント</div>
            
            <div class="endpoint-category">
                <div class="category-title">🏥 システム管理</div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/health</span>
                    <span class="description">ヘルスチェック</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/version</span>
                    <span class="description">バージョン情報</span>
                </div>
            </div>

            <div class="endpoint-category">
                <div class="category-title">🔐 認証・認可</div>
                <div class="endpoint">
                    <span class="method post">POST</span>
                    <span class="path">/api/v1/auth/login</span>
                    <span class="description">ログイン</span>
                </div>
                <div class="endpoint">
                    <span class="method post">POST</span>
                    <span class="path">/api/v1/auth/refresh</span>
                    <span class="description">トークンリフレッシュ</span>
                </div>
                <div class="endpoint">
                    <span class="method post">POST</span>
                    <span class="path">/api/v1/auth/logout</span>
                    <span class="description">ログアウト</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/auth/profile</span>
                    <span class="description">プロフィール取得</span>
                </div>
            </div>

            <div class="endpoint-category">
                <div class="category-title">👥 ユーザー管理</div>
                <div class="endpoint">
                    <span class="method post">POST</span>
                    <span class="path">/api/v1/users</span>
                    <span class="description">ユーザー作成</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/users</span>
                    <span class="description">ユーザー一覧</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/users/{id}</span>
                    <span class="description">ユーザー詳細</span>
                </div>
                <div class="endpoint">
                    <span class="method put">PUT</span>
                    <span class="path">/api/v1/users/{id}</span>
                    <span class="description">ユーザー更新</span>
                </div>
                <div class="endpoint">
                    <span class="method delete">DELETE</span>
                    <span class="path">/api/v1/users/{id}</span>
                    <span class="description">ユーザー削除</span>
                </div>
                <div class="endpoint">
                    <span class="method put">PUT</span>
                    <span class="path">/api/v1/users/{id}/status</span>
                    <span class="description">ステータス変更</span>
                </div>
                <div class="endpoint">
                    <span class="method put">PUT</span>
                    <span class="path">/api/v1/users/{id}/password</span>
                    <span class="description">パスワード変更</span>
                </div>
            </div>

            <div class="endpoint-category">
                <div class="category-title">🏷️ ユーザーロール管理</div>
                <div class="endpoint">
                    <span class="method post">POST</span>
                    <span class="path">/api/v1/users/roles</span>
                    <span class="description">ロール割り当て</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/users/{id}/roles</span>
                    <span class="description">ユーザーロール一覧</span>
                </div>
                <div class="endpoint">
                    <span class="method patch">PATCH</span>
                    <span class="path">/api/v1/users/{id}/roles/{role_id}</span>
                    <span class="description">ロール更新</span>
                </div>
                <div class="endpoint">
                    <span class="method delete">DELETE</span>
                    <span class="path">/api/v1/users/{id}/roles/{role_id}</span>
                    <span class="description">ロール取り消し</span>
                </div>
            </div>

            <div class="endpoint-category">
                <div class="category-title">🏢 部署管理</div>
                <div class="endpoint">
                    <span class="method post">POST</span>
                    <span class="path">/api/v1/departments</span>
                    <span class="description">部署作成</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/departments</span>
                    <span class="description">部署一覧</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/departments/hierarchy</span>
                    <span class="description">部署階層構造</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/departments/{id}</span>
                    <span class="description">部署詳細</span>
                </div>
                <div class="endpoint">
                    <span class="method put">PUT</span>
                    <span class="path">/api/v1/departments/{id}</span>
                    <span class="description">部署更新</span>
                </div>
                <div class="endpoint">
                    <span class="method delete">DELETE</span>
                    <span class="path">/api/v1/departments/{id}</span>
                    <span class="description">部署削除</span>
                </div>
            </div>

            <div class="endpoint-category">
                <div class="category-title">🎭 ロール管理</div>
                <div class="endpoint">
                    <span class="method post">POST</span>
                    <span class="path">/api/v1/roles</span>
                    <span class="description">ロール作成</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/roles</span>
                    <span class="description">ロール一覧</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/roles/hierarchy</span>
                    <span class="description">ロール階層構造</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/roles/{id}</span>
                    <span class="description">ロール詳細</span>
                </div>
                <div class="endpoint">
                    <span class="method put">PUT</span>
                    <span class="path">/api/v1/roles/{id}</span>
                    <span class="description">ロール更新</span>
                </div>
                <div class="endpoint">
                    <span class="method delete">DELETE</span>
                    <span class="path">/api/v1/roles/{id}</span>
                    <span class="description">ロール削除</span>
                </div>
                <div class="endpoint">
                    <span class="method put">PUT</span>
                    <span class="path">/api/v1/roles/{id}/permissions</span>
                    <span class="description">権限割り当て</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/roles/{id}/permissions</span>
                    <span class="description">ロール権限一覧</span>
                </div>
            </div>

            <div class="endpoint-category">
                <div class="category-title">🔑 権限管理</div>
                <div class="endpoint">
                    <span class="method post">POST</span>
                    <span class="path">/api/v1/permissions</span>
                    <span class="description">権限作成</span>
                </div>
                <div class="endpoint">
                    <span class="method post">POST</span>
                    <span class="path">/api/v1/permissions/create-if-not-exists</span>
                    <span class="description">権限作成（存在しない場合のみ）</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/permissions</span>
                    <span class="description">権限一覧</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/permissions/matrix</span>
                    <span class="description">権限マトリックス</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/permissions/modules/{module}</span>
                    <span class="description">モジュール別権限</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/permissions/{id}</span>
                    <span class="description">権限詳細</span>
                </div>
                <div class="endpoint">
                    <span class="method put">PUT</span>
                    <span class="path">/api/v1/permissions/{id}</span>
                    <span class="description">権限更新</span>
                </div>
                <div class="endpoint">
                    <span class="method delete">DELETE</span>
                    <span class="path">/api/v1/permissions/{id}</span>
                    <span class="description">権限削除</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/permissions/{id}/roles</span>
                    <span class="description">権限を持つロール一覧</span>
                </div>
            </div>
        </div>

        <div class="footer">
            <h3>🚀 ERP Access Control API v0.1.0-dev</h3>
            <p>📊 総エンドポイント数: <strong>40+</strong> | 🔒 セキュリティ: <strong>JWT認証</strong> | 🎯 品質: <strong>エンタープライズグレード</strong></p>
            <p>🌐 <a href="/health" style="color: #ffc107;">ヘルスチェック</a> | 📊 <a href="/version" style="color: #ffc107;">バージョン情報</a></p>
        </div>
    </div>
</body>
</html>`

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
	})
}

// setupAuthRoutes 認証エンドポイントを設定
func setupAuthRoutes(group *gin.RouterGroup, authService *services.AuthService, middlewares *MiddlewareContainer, appLogger *logger.Logger) {
	authHandler := handlers.NewAuthHandler(authService, appLogger)

	auth := group.Group("/auth")
	{
		// 認証不要エンドポイント
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/logout", authHandler.Logout)

		// 認証必要エンドポイント
		protected := auth.Group("")
		protected.Use(middlewares.Auth.Authentication())
		{
			protected.GET("/profile", authHandler.GetProfile)
			protected.POST("/change-password", authHandler.ChangePassword)
		}
	}
}

// setupUserRoutes ユーザー管理エンドポイントを設定
func setupUserRoutes(group *gin.RouterGroup, userService *services.UserService, appLogger *logger.Logger) {
	userHandler := handlers.NewUserHandler(userService, appLogger)

	users := group.Group("/users")
	{
		// ユーザーCRUD（権限チェック付き）
		users.POST("", middleware.RequirePermissions("user:create"), userHandler.CreateUser)       // POST /api/v1/users
		users.GET("", middleware.RequirePermissions("user:list"), userHandler.GetUsers)            // GET /api/v1/users
		users.GET("/:id", middleware.RequirePermissions("user:read"), userHandler.GetUser)         // GET /api/v1/users/:id
		users.PUT("/:id", middleware.RequirePermissions("user:update"), userHandler.UpdateUser)    // PUT /api/v1/users/:id
		users.DELETE("/:id", middleware.RequirePermissions("user:delete"), userHandler.DeleteUser) // DELETE /api/v1/users/:id

		// ステータス変更（管理者権限）
		users.PUT("/:id/status", middleware.RequirePermissions("user:manage"), userHandler.ChangeUserStatus) // PUT /api/v1/users/:id/status

		// パスワード変更（自己のみ）
		users.PUT("/:id/password", userHandler.ChangePassword) // PUT /api/v1/users/:id/password
	}
}

// setupUserRoleRoutes ユーザーロール管理エンドポイントを設定
func setupUserRoleRoutes(group *gin.RouterGroup, userRoleService *services.UserRoleService) {
	userRoleHandler := handlers.NewUserRoleHandler(userRoleService)

	group.POST("/users/roles", userRoleHandler.AssignRole)
	group.GET("/users/:id/roles", userRoleHandler.GetUserRoles)
	group.PATCH("/users/:id/roles/:role_id", userRoleHandler.UpdateRole)
	group.DELETE("/users/:id/roles/:role_id", userRoleHandler.RevokeRole)
}

// setupDepartmentRoutes 部署管理エンドポイントを設定
func setupDepartmentRoutes(group *gin.RouterGroup, departmentService *services.DepartmentService, appLogger *logger.Logger) {
	departmentHandler := handlers.NewDepartmentHandler(departmentService, appLogger)

	departments := group.Group("/departments")
	{
		// 部署CRUD（権限チェック付き）
		departments.POST("", middleware.RequirePermissions("department:create"), departmentHandler.CreateDepartment)              // POST /api/v1/departments
		departments.GET("", middleware.RequirePermissions("department:list"), departmentHandler.GetDepartments)                   // GET /api/v1/departments
		departments.GET("/hierarchy", middleware.RequirePermissions("department:list"), departmentHandler.GetDepartmentHierarchy) // GET /api/v1/departments/hierarchy
		departments.GET("/:id", middleware.RequirePermissions("department:read"), departmentHandler.GetDepartment)                // GET /api/v1/departments/:id
		departments.PUT("/:id", middleware.RequirePermissions("department:update"), departmentHandler.UpdateDepartment)           // PUT /api/v1/departments/:id
		departments.DELETE("/:id", middleware.RequirePermissions("department:delete"), departmentHandler.DeleteDepartment)        // DELETE /api/v1/departments/:id
	}
}

// setupRoleRoutes ロール管理エンドポイントを設定
func setupRoleRoutes(group *gin.RouterGroup, roleService *services.RoleService, appLogger *logger.Logger) {
	roleHandler := handlers.NewRoleHandler(roleService, appLogger)

	roles := group.Group("/roles")
	{
		roles.POST("", middleware.RequirePermissions("role:create"), roleHandler.CreateRole)                       // POST /api/v1/roles
		roles.GET("", middleware.RequirePermissions("role:list"), roleHandler.GetRoles)                            // GET /api/v1/roles
		roles.GET("/hierarchy", middleware.RequirePermissions("role:list"), roleHandler.GetRoleHierarchy)          // GET /api/v1/roles/hierarchy
		roles.GET("/:id", middleware.RequirePermissions("role:read"), roleHandler.GetRole)                         // GET /api/v1/roles/:id
		roles.PUT("/:id", middleware.RequirePermissions("role:update"), roleHandler.UpdateRole)                    // PUT /api/v1/roles/:id
		roles.DELETE("/:id", middleware.RequirePermissions("role:delete"), roleHandler.DeleteRole)                 // DELETE /api/v1/roles/:id
		roles.PUT("/:id/permissions", middleware.RequirePermissions("role:manage"), roleHandler.AssignPermissions) // PUT /api/v1/roles/:id/permissions
		roles.GET("/:id/permissions", middleware.RequirePermissions("role:read"), roleHandler.GetRolePermissions)  // GET /api/v1/roles/:id/permissions
	}
}

// setupPermissionRoutes 権限管理エンドポイントを設定
func setupPermissionRoutes(group *gin.RouterGroup, permissionService *services.PermissionService, appLogger *logger.Logger) {
	permissionHandler := handlers.NewPermissionHandler(permissionService, appLogger)

	permissions := group.Group("/permissions")
	{
		permissions.POST("", middleware.RequirePermissions("permission:create"), permissionHandler.CreatePermission)                                 // POST /api/v1/permissions
		permissions.POST("/create-if-not-exists", middleware.RequirePermissions("permission:create"), permissionHandler.CreatePermissionIfNotExists) // POST /api/v1/permissions/create-if-not-exists
		permissions.GET("", middleware.RequirePermissions("permission:list"), permissionHandler.GetPermissions)                                      // GET /api/v1/permissions
		permissions.GET("/matrix", middleware.RequirePermissions("permission:list"), permissionHandler.GetPermissionMatrix)                          // GET /api/v1/permissions/matrix
		permissions.GET("/modules/:module", middleware.RequirePermissions("permission:list"), permissionHandler.GetPermissionsByModule)              // GET /api/v1/permissions/modules/:module
		permissions.GET("/:id", middleware.RequirePermissions("permission:read"), permissionHandler.GetPermission)                                   // GET /api/v1/permissions/:id
		permissions.PUT("/:id", middleware.RequirePermissions("permission:update"), permissionHandler.UpdatePermission)                              // PUT /api/v1/permissions/:id
		permissions.DELETE("/:id", middleware.RequirePermissions("permission:delete"), permissionHandler.DeletePermission)                           // DELETE /api/v1/permissions/:id
		permissions.GET("/:id/roles", middleware.RequirePermissions("permission:read"), permissionHandler.GetRolesByPermission)                      // GET /api/v1/permissions/:id/roles
	}
}

// startServer サーバーを起動
func startServer(router *gin.Engine, port string) {
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 ERP Access Control API サーバー起動中...")
	log.Printf("📡 ポート: %s", port)
	log.Printf("🌐 URL: http://localhost:%s", port)
	log.Printf("🏥 ヘルスチェック: http://localhost:%s/health", port)
	log.Printf("📚 API仕様: http://localhost:%s/", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("❌ サーバー起動エラー: %v", err)
	}
}

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
