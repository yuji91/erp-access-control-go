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
		c.JSON(http.StatusOK, gin.H{
			"message": "🔐 ERP Access Control API",
			"status":  "running",
			"features": []string{
				"多重ロール管理",
				"期限付きロール",
				"階層的権限",
				"JWT認証",
			},
			"endpoints": []string{
				"GET /health                       - ヘルスチェック",
				"GET /version                      - バージョン情報",
				"POST /api/v1/auth/login           - ログイン",
				"POST /api/v1/auth/refresh         - トークンリフレッシュ",
				"POST /api/v1/auth/logout          - ログアウト",
				"GET /api/v1/auth/profile          - プロフィール取得",
				"POST /api/v1/users                - ユーザー作成",
				"GET /api/v1/users                 - ユーザー一覧",
				"GET /api/v1/users/{id}            - ユーザー詳細",
				"PUT /api/v1/users/{id}            - ユーザー更新",
				"DELETE /api/v1/users/{id}         - ユーザー削除",
				"PUT /api/v1/users/{id}/status     - ステータス変更",
				"PUT /api/v1/users/{id}/password   - パスワード変更",
				"POST /api/v1/users/roles          - ロール割り当て",
				"GET /api/v1/users/{id}/roles      - ユーザーロール一覧",
				"PATCH /api/v1/users/{id}/roles/{role_id} - ロール更新",
				"DELETE /api/v1/users/{id}/roles/{role_id} - ロール取り消し",
				"POST /api/v1/departments          - 部署作成",
				"GET /api/v1/departments           - 部署一覧",
				"GET /api/v1/departments/hierarchy - 部署階層構造",
				"GET /api/v1/departments/{id}      - 部署詳細",
				"PUT /api/v1/departments/{id}      - 部署更新",
				"DELETE /api/v1/departments/{id}   - 部署削除",
				"POST /api/v1/roles                - ロール作成",
				"GET /api/v1/roles                 - ロール一覧",
				"GET /api/v1/roles/hierarchy       - ロール階層構造",
				"GET /api/v1/roles/{id}            - ロール詳細",
				"PUT /api/v1/roles/{id}            - ロール更新",
				"DELETE /api/v1/roles/{id}         - ロール削除",
				"PUT /api/v1/roles/{id}/permissions - 権限割り当て",
				"GET /api/v1/roles/{id}/permissions - ロール権限一覧",
				"POST /api/v1/permissions          - 権限作成",
				"GET /api/v1/permissions           - 権限一覧",
				"GET /api/v1/permissions/matrix    - 権限マトリックス",
				"GET /api/v1/permissions/modules/{module} - モジュール別権限",
				"GET /api/v1/permissions/{id}      - 権限詳細",
				"PUT /api/v1/permissions/{id}      - 権限更新",
				"DELETE /api/v1/permissions/{id}   - 権限削除",
				"GET /api/v1/permissions/{id}/roles - 権限を持つロール一覧",
			},
		})
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
		permissions.POST("", middleware.RequirePermissions("permission:create"), permissionHandler.CreatePermission)                    // POST /api/v1/permissions
		permissions.GET("", middleware.RequirePermissions("permission:list"), permissionHandler.GetPermissions)                         // GET /api/v1/permissions
		permissions.GET("/matrix", middleware.RequirePermissions("permission:list"), permissionHandler.GetPermissionMatrix)             // GET /api/v1/permissions/matrix
		permissions.GET("/modules/:module", middleware.RequirePermissions("permission:list"), permissionHandler.GetPermissionsByModule) // GET /api/v1/permissions/modules/:module
		permissions.GET("/:id", middleware.RequirePermissions("permission:read"), permissionHandler.GetPermission)                      // GET /api/v1/permissions/:id
		permissions.PUT("/:id", middleware.RequirePermissions("permission:update"), permissionHandler.UpdatePermission)                 // PUT /api/v1/permissions/:id
		permissions.DELETE("/:id", middleware.RequirePermissions("permission:delete"), permissionHandler.DeletePermission)              // DELETE /api/v1/permissions/:id
		permissions.GET("/:id/roles", middleware.RequirePermissions("permission:read"), permissionHandler.GetRolesByPermission)         // GET /api/v1/permissions/:id/roles
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
