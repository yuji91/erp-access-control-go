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
	// JWT サービス
	jwtService := jwt.NewService(cfg.JWT.Secret, cfg.JWT.AccessTokenDuration)

	// 基本サービス
	permissionService := services.NewPermissionService(db)
	revocationService := services.NewTokenRevocationService(db)
	userRoleService := services.NewUserRoleService(db)

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
			// ユーザーロール管理
			setupUserRoleRoutes(protected, services.UserRole)
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
				"GET /health                     - ヘルスチェック",
				"GET /version                    - バージョン情報",
				"POST /api/v1/auth/login         - ログイン",
				"POST /api/v1/auth/refresh       - トークンリフレッシュ",
				"POST /api/v1/auth/logout        - ログアウト",
				"POST /api/v1/users/roles        - ロール割り当て",
				"GET /api/v1/users/{id}/roles    - ユーザーロール一覧",
				"PATCH /api/v1/users/{id}/roles/{role_id} - ロール更新",
				"DELETE /api/v1/users/{id}/roles/{role_id} - ロール取り消し",
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

// setupUserRoleRoutes ユーザーロール管理エンドポイントを設定
func setupUserRoleRoutes(group *gin.RouterGroup, userRoleService *services.UserRoleService) {
	userRoleHandler := handlers.NewUserRoleHandler(userRoleService)

	group.POST("/users/roles", userRoleHandler.AssignRole)
	group.GET("/users/:user_id/roles", userRoleHandler.GetUserRoles)
	group.PATCH("/users/:user_id/roles/:role_id", userRoleHandler.UpdateRole)
	group.DELETE("/users/:user_id/roles/:role_id", userRoleHandler.RevokeRole)
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
	JWT        *jwt.Service
}

// MiddlewareContainer ミドルウェアコンテナ
type MiddlewareContainer struct {
	Auth *middleware.AuthMiddleware
}
