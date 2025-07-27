package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"erp-access-control-go/internal/config"
	"erp-access-control-go/internal/handlers"
	"erp-access-control-go/internal/middleware"
	"erp-access-control-go/internal/services"
	"erp-access-control-go/pkg/jwt"
	"erp-access-control-go/pkg/logger"
)

// TestDB テスト用データベース接続
var TestDB *gorm.DB

// TestRouter テスト用Ginルーター
var TestRouter *gin.Engine

// TestLogger テスト用ロガー
var TestLogger *logger.Logger

// TestAuthService テスト用認証サービス
var TestAuthService *services.AuthService

// TestJWTService テスト用JWTサービス
var TestJWTService *jwt.Service

// TestDepartmentID テスト用部門ID
var TestDepartmentID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// SetupTestEnvironment テスト環境のセットアップ
func SetupTestEnvironment(t *testing.T) {
	// テスト用設定の読み込み
	cfg, err := config.Load()
	require.NoError(t, err)

	// テスト用データベース接続
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s_test sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
	)
	TestDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// テスト用ロガー
	var logBuf bytes.Buffer
	TestLogger = logger.NewLogger(
		logger.WithOutput(&logBuf),
		logger.WithMinLevel(logger.DEBUG),
		logger.WithEnvironment("test"),
	)

	// テスト用JWTサービス
	TestJWTService = jwt.NewService("test-secret", 15)

	// テスト用サービス
	permissionService := services.NewPermissionService(TestDB)
	revocationService := services.NewTokenRevocationService(TestDB)
	TestAuthService = services.NewAuthService(
		TestDB,
		TestJWTService,
		permissionService,
		revocationService,
	)

	// テスト用ミドルウェア
	authMiddleware := middleware.NewAuthMiddleware(
		TestJWTService,
		revocationService,
		TestLogger,
	)

	// テスト用ルーター
	gin.SetMode(gin.TestMode)
	TestRouter = gin.New()
	TestRouter.Use(middleware.ErrorHandler(TestLogger))

	// 認証ハンドラー
	authHandler := handlers.NewAuthHandler(TestAuthService, TestLogger)

	// ルーティング設定
	v1 := TestRouter.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)

			protected := auth.Group("")
			protected.Use(authMiddleware.Authentication())
			{
				protected.GET("/profile", authHandler.GetProfile)
				protected.POST("/change-password", authHandler.ChangePassword)
			}
		}
	}
}

// CleanupTestEnvironment テスト環境のクリーンアップ
func CleanupTestEnvironment(t *testing.T) {
	// データベースのクリーンアップ
	sqlDB, err := TestDB.DB()
	require.NoError(t, err)
	err = sqlDB.Close()
	require.NoError(t, err)
}

// CreateTestRequest テストリクエストの作成
func CreateTestRequest(method, path string, body interface{}) (*http.Request, error) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// ExecuteRequest テストリクエストの実行
func ExecuteRequest(req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	TestRouter.ServeHTTP(w, req)
	return w
}

// LoadTestData テストデータの読み込み
func LoadTestData(t *testing.T, filename string) []byte {
	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	return data
}

// ParseResponse レスポンスのパース
func ParseResponse(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	err := json.Unmarshal(w.Body.Bytes(), v)
	require.NoError(t, err)
}

// TestMain テストのメインエントリーポイント
func TestMain(m *testing.M) {
	// テスト環境のセットアップ
	if err := setupTestDB(); err != nil {
		fmt.Printf("Failed to setup test database: %v\n", err)
		os.Exit(1)
	}

	// テストの実行
	code := m.Run()

	// テスト環境のクリーンアップ
	if err := cleanupTestDB(); err != nil {
		fmt.Printf("Failed to cleanup test database: %v\n", err)
	}

	os.Exit(code)
}

// setupTestDB テストデータベースのセットアップ
func setupTestDB() error {
	// テストデータベースの作成
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// システムデータベースに接続
	systemDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
	)
	systemDB, err := gorm.Open(postgres.Open(systemDSN), &gorm.Config{})
	if err != nil {
		return err
	}

	// テストデータベースの作成
	testDBName := fmt.Sprintf("%s_test", cfg.Database.Name)
	err = systemDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName)).Error
	if err != nil {
		return err
	}

	err = systemDB.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName)).Error
	if err != nil {
		return err
	}

	return nil
}

// cleanupTestDB テストデータベースのクリーンアップ
func cleanupTestDB() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// システムデータベースに接続
	systemDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
	)
	systemDB, err := gorm.Open(postgres.Open(systemDSN), &gorm.Config{})
	if err != nil {
		return err
	}

	// テストデータベースの削除
	testDBName := fmt.Sprintf("%s_test", cfg.Database.Name)
	err = systemDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName)).Error
	if err != nil {
		return err
	}

	return nil
}
