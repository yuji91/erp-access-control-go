# 🗺️ **ERP Access Control API 開発ロードマップ**

> **中間目標**: **Phase 5 (ビジネスロジック実装)** まで完了  
> **MVP実現**: 基本CRUD API + JWT認証 + 権限チェック + 監査ログ

---

## 🎯 **開発目標・スコープ**

### **中間目標 (Phase 1-5)**
**Permission Matrix + Policy Object** のコア機能を持つ **実用可能なMVP** を構築

| 目標 | 内容 | 価値 |
|------|------|------|
| **🔧 基盤構築** | Go環境・DB・API基盤 | 開発効率向上 |
| **🔐 認証認可** | JWT + Permission Matrix | セキュアなアクセス制御 |
| **🏢 業務API** | User/Department/Role管理 | ERPの基本機能 |
| **📊 監査** | 操作ログ・アクセス履歴 | コンプライアンス対応 |

### **今回対象外 (Phase 6-9)**
- セキュリティ強化 (本格的な脆弱性対策)
- 監視・運用 (Prometheus, ヘルスチェック)
- テスト実装 (包括的テストスイート)
- デプロイメント (Docker, CI/CD)

---

## 📅 **開発スケジュール**

| Phase | 期間 | 稼働日数 | 重要度 | ゴール |
|-------|------|----------|--------|--------|
| **Phase 1** | 1-2日 | 2日 | 🔴 Critical | プロジェクト基盤完成 |
| **Phase 2** | 2-3日 | 3日 | 🔴 Critical | DB接続・モデル動作確認 |
| **Phase 3** | 3-4日 | 4日 | 🔴 Critical | API基盤・Swagger完成 |
| **Phase 4** | 5-7日 | 6日 | 🟡 High | 認証認可システム完成 |
| **Phase 5** | 7-10日 | 8日 | 🟡 High | 業務API完成 |
| **合計** | **18-26日** | **23日** | - | **MVP完成** |

**⏰ 総開発期間**: 約 **3-4週間** (1日8時間想定)

---

## 🚀 **Phase 1: プロジェクト基盤構築**
> **期間**: 1-2日 | **ゴール**: 開発環境完全構築

### **STEP 1.1: プロジェクト構造作成** _(30分)_

```bash
mkdir -p cmd/server internal/{handlers,services,middleware,config} api migrations pkg
```

**📁 作成されるディレクトリ構造**:
```
erp-access-control-api/
├── cmd/server/           # アプリケーションエントリポイント
├── internal/
│   ├── handlers/         # HTTPハンドラ
│   ├── services/         # ビジネスロジック
│   ├── middleware/       # 認証・ログミドルウェア
│   └── config/           # 設定管理
├── api/                  # OpenAPI仕様
├── migrations/           # DBマイグレーション
└── pkg/                  # 外部公開ライブラリ
```

**✅ 完了条件**: ディレクトリ構造作成完了

---

### **STEP 1.2: 環境設定ファイル作成** _(45分)_

#### **1.2.1: .env.example作成**
```bash
# .env.example
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=erp_access_control
DB_USER=postgres
DB_PASSWORD=your_password
DB_SSL_MODE=disable

# JWT
JWT_SECRET=your-super-secret-jwt-key-256-bits-long
JWT_EXPIRES_IN=24h

# Server
SERVER_PORT=8080
SERVER_HOST=localhost
GIN_MODE=debug

# Logging
LOG_LEVEL=debug
LOG_FORMAT=json
```

#### **1.2.2: config.yaml作成**
```yaml
# config.yaml
server:
  port: 8080
  host: "localhost"
  mode: "debug"

database:
  host: "localhost"
  port: 5432
  name: "erp_access_control"
  user: "postgres"
  ssl_mode: "disable"
  max_open_conns: 25
  max_idle_conns: 5

jwt:
  expires_in: "24h"

logging:
  level: "debug"
  format: "json"
```

**✅ 完了条件**: 環境設定ファイル作成完了

---

### **STEP 1.3: 設定管理実装** _(90分)_

#### **1.3.1: internal/config/config.go**
```go
package config

import (
    "fmt"
    "time"
    
    "github.com/spf13/viper"
    "github.com/joho/godotenv"
)

type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    JWT      JWTConfig      `mapstructure:"jwt"`
    Logging  LoggingConfig  `mapstructure:"logging"`
}

type ServerConfig struct {
    Port string `mapstructure:"port"`
    Host string `mapstructure:"host"`
    Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
    Host         string `mapstructure:"host"`
    Port         int    `mapstructure:"port"`
    Name         string `mapstructure:"name"`
    User         string `mapstructure:"user"`
    Password     string `mapstructure:"password"`
    SSLMode      string `mapstructure:"ssl_mode"`
    MaxOpenConns int    `mapstructure:"max_open_conns"`
    MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type JWTConfig struct {
    Secret    string        `mapstructure:"secret"`
    ExpiresIn time.Duration `mapstructure:"expires_in"`
}

type LoggingConfig struct {
    Level  string `mapstructure:"level"`
    Format string `mapstructure:"format"`
}

func Load() (*Config, error) {
    // Load .env file
    _ = godotenv.Load()
    
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("./config")
    
    // Environment variables
    viper.AutomaticEnv()
    viper.SetEnvPrefix("ERP")
    
    // Default values
    setDefaults()
    
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("error reading config file: %w", err)
        }
    }
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("error unmarshaling config: %w", err)
    }
    
    // JWT secret from env
    if secret := viper.GetString("JWT_SECRET"); secret != "" {
        config.JWT.Secret = secret
    }
    
    // Database password from env
    if password := viper.GetString("DB_PASSWORD"); password != "" {
        config.Database.Password = password
    }
    
    return &config, nil
}

func setDefaults() {
    viper.SetDefault("server.port", "8080")
    viper.SetDefault("server.host", "localhost")
    viper.SetDefault("server.mode", "debug")
    viper.SetDefault("database.max_open_conns", 25)
    viper.SetDefault("database.max_idle_conns", 5)
    viper.SetDefault("jwt.expires_in", "24h")
    viper.SetDefault("logging.level", "debug")
    viper.SetDefault("logging.format", "json")
}

func (c *Config) GetDatabaseDSN() string {
    return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        c.Database.Host,
        c.Database.Port,
        c.Database.User,
        c.Database.Password,
        c.Database.Name,
        c.Database.SSLMode,
    )
}
```

**✅ 完了条件**: 設定管理実装完了

---

### **STEP 1.4: ログシステム実装** _(60分)_

#### **1.4.1: pkg/logger/logger.go**
```go
package logger

import (
    "os"
    
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

type Logger struct {
    *zap.Logger
}

func New(level, format string) (*Logger, error) {
    var zapLevel zapcore.Level
    switch level {
    case "debug":
        zapLevel = zapcore.DebugLevel
    case "info":
        zapLevel = zapcore.InfoLevel
    case "warn":
        zapLevel = zapcore.WarnLevel
    case "error":
        zapLevel = zapcore.ErrorLevel
    default:
        zapLevel = zapcore.InfoLevel
    }
    
    config := zap.Config{
        Level:    zap.NewAtomicLevelAt(zapLevel),
        Encoding: format,
        EncoderConfig: zapcore.EncoderConfig{
            TimeKey:        "timestamp",
            LevelKey:       "level",
            NameKey:        "logger",
            CallerKey:      "caller",
            MessageKey:     "message",
            StacktraceKey:  "stacktrace",
            LineEnding:     zapcore.DefaultLineEnding,
            EncodeLevel:    zapcore.LowercaseLevelEncoder,
            EncodeTime:     zapcore.ISO8601TimeEncoder,
            EncodeDuration: zapcore.StringDurationEncoder,
            EncodeCaller:   zapcore.ShortCallerEncoder,
        },
        OutputPaths:      []string{"stdout"},
        ErrorOutputPaths: []string{"stderr"},
    }
    
    zapLogger, err := config.Build()
    if err != nil {
        return nil, err
    }
    
    return &Logger{Logger: zapLogger}, nil
}

func (l *Logger) WithRequestID(requestID string) *Logger {
    return &Logger{Logger: l.With(zap.String("request_id", requestID))}
}

func (l *Logger) WithUser(userID string) *Logger {
    return &Logger{Logger: l.With(zap.String("user_id", userID))}
}
```

**✅ 完了条件**: ログシステム実装完了

---

### **STEP 1.5: エラーハンドリング** _(45分)_

#### **1.5.1: pkg/errors/errors.go**
```go
package errors

import (
    "fmt"
    "net/http"
)

type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
    Status  int    `json:"-"`
}

func (e *APIError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Business Logic Errors
var (
    ErrUserNotFound     = &APIError{Code: "USER_NOT_FOUND", Message: "User not found", Status: http.StatusNotFound}
    ErrDuplicateEmail   = &APIError{Code: "DUPLICATE_EMAIL", Message: "Email already exists", Status: http.StatusConflict}
    ErrInvalidPassword  = &APIError{Code: "INVALID_PASSWORD", Message: "Invalid password", Status: http.StatusBadRequest}
    ErrPermissionDenied = &APIError{Code: "PERMISSION_DENIED", Message: "Permission denied", Status: http.StatusForbidden}
    ErrInvalidToken     = &APIError{Code: "INVALID_TOKEN", Message: "Invalid token", Status: http.StatusUnauthorized}
)

// Validation Errors
func NewValidationError(field, message string) *APIError {
    return &APIError{
        Code:    "VALIDATION_ERROR",
        Message: fmt.Sprintf("Validation failed for field '%s'", field),
        Details: message,
        Status:  http.StatusBadRequest,
    }
}

// Database Errors
func NewDatabaseError(err error) *APIError {
    return &APIError{
        Code:    "DATABASE_ERROR",
        Message: "Database operation failed",
        Details: err.Error(),
        Status:  http.StatusInternalServerError,
    }
}
```

**✅ 完了条件**: エラーハンドリング実装完了

---

## 🎯 **Phase 1 完了基準**

- [ ] プロジェクト構造作成完了
- [ ] 環境設定ファイル (.env.example, config.yaml) 作成
- [ ] 設定管理 (internal/config/config.go) 実装
- [ ] ログシステム (pkg/logger/logger.go) 実装  
- [ ] エラーハンドリング (pkg/errors/errors.go) 実装

**🎉 Phase 1 成果物**: 拡張可能な設定・ログ・エラー管理を持つプロジェクト基盤

---

## 🗄️ **Phase 2: データベース基盤**
> **期間**: 2-3日 | **ゴール**: DB接続・モデル動作確認

### **STEP 2.1: データベース接続実装** _(120分)_

#### **2.1.1: internal/database/database.go**
```go
package database

import (
    "fmt"
    "time"
    
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
    
    "erp-access-control-api/internal/config"
    zapLogger "erp-access-control-api/pkg/logger"
)

type DB struct {
    *gorm.DB
}

func New(cfg *config.Config, log *zapLogger.Logger) (*DB, error) {
    dsn := cfg.GetDatabaseDSN()
    
    gormConfig := &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    }
    
    db, err := gorm.Open(postgres.Open(dsn), gormConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }
    
    sqlDB, err := db.DB()
    if err != nil {
        return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
    }
    
    // Connection pool settings
    sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
    sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
    sqlDB.SetConnMaxLifetime(time.Hour)
    
    // Test connection
    if err := sqlDB.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }
    
    log.Info("Database connection established successfully")
    
    return &DB{DB: db}, nil
}

func (db *DB) Close() error {
    sqlDB, err := db.DB.DB()
    if err != nil {
        return err
    }
    return sqlDB.Close()
}

func (db *DB) Migrate() error {
    // Auto-migrate all models
    return db.AutoMigrate(
        // Add model structs here when ready
    )
}
```

**✅ 完了条件**: データベース接続実装完了

---

### **STEP 2.2: マイグレーション実行** _(60分)_

#### **2.2.1: PostgreSQL設定**
```bash
# PostgreSQL起動 (Docker使用の場合)
docker run --name erp-postgres \
  -e POSTGRES_DB=erp_access_control \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=your_password \
  -p 5432:5432 \
  -d postgres:15

# またはローカルPostgreSQL使用
createdb erp_access_control
```

#### **2.2.2: マイグレーション実行**
```bash
# SQLマイグレーション実行
psql -h localhost -U postgres -d erp_access_control -f docs/migration/init_migration_erp_acl_refine_02.sql
```

**✅ 完了条件**: マイグレーション実行完了・テーブル作成確認

---

### **STEP 2.3: GORM モデル統合** _(90分)_

#### **2.3.1: models/models.go (統合ファイル)**
```go
package models

import (
    "erp-access-control-api/models"
)

// AllModels returns all model structs for auto-migration
func AllModels() []interface{} {
    return []interface{}{
        &models.Department{},
        &models.Role{},
        &models.User{},
        &models.Permission{},
        &models.RolePermission{},
        &models.UserScope{},
        &models.ApprovalState{},
        &models.AuditLog{},
        &models.TimeRestriction{},
        &models.RevokedToken{},
    }
}
```

#### **2.3.2: データベース接続テスト**
```go
// cmd/test-db/main.go
package main

import (
    "log"
    
    "erp-access-control-api/internal/config"
    "erp-access-control-api/internal/database"
    "erp-access-control-api/pkg/logger"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    logger, err := logger.New(cfg.Logging.Level, cfg.Logging.Format)
    if err != nil {
        log.Fatal("Failed to initialize logger:", err)
    }
    
    db, err := database.New(cfg, logger)
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    defer db.Close()
    
    logger.Info("Database connection test successful!")
}
```

**✅ 完了条件**: GORM モデル動作確認完了

---

## 🎯 **Phase 2 完了基準**

- [ ] データベース接続実装完了
- [ ] マイグレーション実行・テーブル作成確認
- [ ] GORMモデル統合・動作確認完了

**🎉 Phase 2 成果物**: PostgreSQL接続・GORMモデル統合完了

---

## 🌐 **Phase 3: API基盤構築**
> **期間**: 3-4日 | **ゴール**: API基盤・Swagger完成

### **STEP 3.1: Ginエンジン初期化** _(90分)_

#### **3.1.1: cmd/server/main.go**
```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/gin-gonic/gin"
    
    "erp-access-control-api/internal/config"
    "erp-access-control-api/internal/database"
    "erp-access-control-api/internal/handlers"
    "erp-access-control-api/internal/middleware"
    "erp-access-control-api/pkg/logger"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    // Initialize logger
    logger, err := logger.New(cfg.Logging.Level, cfg.Logging.Format)
    if err != nil {
        log.Fatal("Failed to initialize logger:", err)
    }
    
    // Connect to database
    db, err := database.New(cfg, logger)
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    defer db.Close()
    
    // Set Gin mode
    gin.SetMode(cfg.Server.Mode)
    
    // Initialize Gin router
    router := gin.New()
    
    // Setup middleware
    setupMiddleware(router, logger)
    
    // Setup routes
    setupRoutes(router, db, logger)
    
    // Start server
    startServer(router, cfg.Server, logger)
}

func setupMiddleware(router *gin.Engine, logger *logger.Logger) {
    router.Use(middleware.Logger(logger))
    router.Use(middleware.Recovery(logger))
    router.Use(middleware.CORS())
    router.Use(middleware.RequestID())
}

func setupRoutes(router *gin.Engine, db *database.DB, logger *logger.Logger) {
    api := router.Group("/api/v1")
    
    // Health check
    api.GET("/health", handlers.HealthCheck(db))
    
    // TODO: Add other routes
}

func startServer(router *gin.Engine, serverCfg config.ServerConfig, logger *logger.Logger) {
    address := fmt.Sprintf("%s:%s", serverCfg.Host, serverCfg.Port)
    
    server := &http.Server{
        Addr:         address,
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }
    
    // Start server in goroutine
    go func() {
        logger.Info("Server starting", zap.String("address", address))
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("Failed to start server", zap.Error(err))
        }
    }()
    
    // Wait for interrupt signal for graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    logger.Info("Server is shutting down...")
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        logger.Fatal("Server forced to shutdown", zap.Error(err))
    }
    
    logger.Info("Server gracefully stopped")
}
```

**✅ 完了条件**: Ginエンジン初期化完了

---

### **STEP 3.2: ミドルウェア実装** _(120分)_

#### **3.2.1: internal/middleware/middleware.go**
```go
package middleware

import (
    "time"
    
    "github.com/gin-contrib/cors"
    "github.com/gin-contrib/requestid"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    
    "erp-access-control-api/pkg/logger"
)

func Logger(log *logger.Logger) gin.HandlerFunc {
    return gin.LoggerWithConfig(gin.LoggerConfig{
        Formatter: func(param gin.LogFormatterParams) string {
            log.Info("HTTP Request",
                zap.String("method", param.Method),
                zap.String("path", param.Path),
                zap.Int("status", param.StatusCode),
                zap.Duration("latency", param.Latency),
                zap.String("client_ip", param.ClientIP),
                zap.String("user_agent", param.Request.UserAgent()),
            )
            return ""
        },
        Output: nil,
    })
}

func Recovery(log *logger.Logger) gin.HandlerFunc {
    return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
        log.Error("Panic recovered",
            zap.Any("error", recovered),
            zap.String("path", c.Request.URL.Path),
        )
        c.AbortWithStatus(500)
    })
}

func CORS() gin.HandlerFunc {
    return cors.New(cors.Config{
        AllowOrigins:     []string{"*"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
        AllowHeaders:     []string{"*"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    })
}

func RequestID() gin.HandlerFunc {
    return requestid.New()
}
```

**✅ 完了条件**: 基本ミドルウェア実装完了

---

### **STEP 3.3: ヘルスチェックAPI** _(45分)_

#### **3.3.1: internal/handlers/health.go**
```go
package handlers

import (
    "net/http"
    "time"
    
    "github.com/gin-gonic/gin"
    
    "erp-access-control-api/internal/database"
)

type HealthResponse struct {
    Status    string    `json:"status"`
    Timestamp time.Time `json:"timestamp"`
    Database  string    `json:"database"`
    Version   string    `json:"version"`
}

func HealthCheck(db *database.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        response := HealthResponse{
            Status:    "ok",
            Timestamp: time.Now(),
            Version:   "1.0.0",
        }
        
        // Check database connection
        sqlDB, err := db.DB.DB()
        if err != nil {
            response.Database = "error"
            response.Status = "error"
            c.JSON(http.StatusInternalServerError, response)
            return
        }
        
        if err := sqlDB.Ping(); err != nil {
            response.Database = "error"
            response.Status = "error"
            c.JSON(http.StatusInternalServerError, response)
            return
        }
        
        response.Database = "ok"
        c.JSON(http.StatusOK, response)
    }
}
```

**✅ 完了条件**: ヘルスチェックAPI動作確認完了

---

### **STEP 3.4: OpenAPI/Swagger設定** _(90分)_

#### **3.4.1: api/openapi.yaml**
```yaml
openapi: 3.0.3
info:
  title: ERP Access Control API
  description: Permission Matrix + Policy Object ハイブリッド構成のアクセス制御API
  version: 1.0.0
  contact:
    name: API Support
    email: support@example.com

servers:
  - url: http://localhost:8080/api/v1
    description: Development server

paths:
  /health:
    get:
      summary: ヘルスチェック
      operationId: healthCheck
      tags:
        - System
      responses:
        '200':
          description: 正常
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/HealthResponse'
        '500':
          description: サーバーエラー

components:
  schemas:
    HealthResponse:
      type: object
      properties:
        status:
          type: string
          example: "ok"
        timestamp:
          type: string
          format: date-time
        database:
          type: string
          example: "ok"
        version:
          type: string
          example: "1.0.0"
      required:
        - status
        - timestamp
        - database
        - version

    Error:
      type: object
      properties:
        code:
          type: string
          example: "VALIDATION_ERROR"
        message:
          type: string
          example: "Invalid input data"
        details:
          type: string
          example: "Field 'email' is required"
      required:
        - code
        - message

  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT

security:
  - BearerAuth: []
```

**✅ 完了条件**: OpenAPI仕様書作成完了

---

## 🎯 **Phase 3 完了基準**

- [ ] Ginエンジン初期化・サーバー起動確認
- [ ] 基本ミドルウェア実装完了
- [ ] ヘルスチェックAPI動作確認完了  
- [ ] OpenAPI仕様書作成完了

**🎉 Phase 3 成果物**: API基盤・Swagger UI完成

---

## 🔐 **Phase 4: 認証・認可システム**
> **期間**: 5-7日 | **ゴール**: 認証認可システム完成

### **STEP 4.1: JWT認証実装** _(180分)_

#### **4.1.1: pkg/jwt/jwt.go**
```go
package jwt

import (
    "fmt"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

type CustomClaims struct {
    UserID      uuid.UUID `json:"user_id"`
    Email       string    `json:"email"`
    Permissions []string  `json:"permissions"`
    jwt.RegisteredClaims
}

type Service struct {
    secretKey []byte
    expiresIn time.Duration
}

func NewService(secret string, expiresIn time.Duration) *Service {
    return &Service{
        secretKey: []byte(secret),
        expiresIn: expiresIn,
    }
}

func (s *Service) GenerateToken(userID uuid.UUID, email string, permissions []string) (string, error) {
    claims := CustomClaims{
        UserID:      userID,
        Email:       email,
        Permissions: permissions,
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   userID.String(),
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiresIn)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            ID:        uuid.New().String(),
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.secretKey)
}

func (s *Service) ValidateToken(tokenString string) (*CustomClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return s.secretKey, nil
    })
    
    if err != nil {
        return nil, err
    }
    
    if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
        return claims, nil
    }
    
    return nil, fmt.Errorf("invalid token")
}
```

**✅ 完了条件**: JWT認証サービス実装完了

---

### **STEP 4.2: 認証ミドルウェア** _(120分)_

#### **4.2.1: internal/middleware/auth.go**
```go
package middleware

import (
    "net/http"
    "strings"
    
    "github.com/gin-gonic/gin"
    
    "erp-access-control-api/pkg/errors"
    "erp-access-control-api/pkg/jwt"
    "erp-access-control-api/pkg/logger"
)

func Authentication(jwtService *jwt.Service, log *logger.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, errors.ErrInvalidToken)
            c.Abort()
            return
        }
        
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            c.JSON(http.StatusUnauthorized, errors.ErrInvalidToken)
            c.Abort()
            return
        }
        
        claims, err := jwtService.ValidateToken(tokenString)
        if err != nil {
            log.Error("Token validation failed", zap.Error(err))
            c.JSON(http.StatusUnauthorized, errors.ErrInvalidToken)
            c.Abort()
            return
        }
        
        // Store user information in context
        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)
        c.Set("permissions", claims.Permissions)
        c.Set("jti", claims.ID)
        
        c.Next()
    }
}

func RequirePermissions(permissions ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userPerms, exists := c.Get("permissions")
        if !exists {
            c.JSON(http.StatusForbidden, errors.ErrPermissionDenied)
            c.Abort()
            return
        }
        
        userPermissions, ok := userPerms.([]string)
        if !ok {
            c.JSON(http.StatusForbidden, errors.ErrPermissionDenied)
            c.Abort()
            return
        }
        
        for _, requiredPerm := range permissions {
            if !hasPermission(userPermissions, requiredPerm) {
                c.JSON(http.StatusForbidden, errors.ErrPermissionDenied)
                c.Abort()
                return
            }
        }
        
        c.Next()
    }
}

func hasPermission(userPerms []string, required string) bool {
    for _, perm := range userPerms {
        if perm == required || perm == "*" {
            return true
        }
    }
    return false
}
```

**✅ 完了条件**: 認証ミドルウェア実装完了

---

### **STEP 4.3: Permission Matrix実装** _(150分)_

#### **4.3.1: internal/services/permission.go**
```go
package services

import (
    "fmt"
    
    "github.com/google/uuid"
    
    "erp-access-control-api/internal/database"
    "erp-access-control-api/models"
)

type PermissionService struct {
    db *database.DB
}

func NewPermissionService(db *database.DB) *PermissionService {
    return &PermissionService{db: db}
}

func (s *PermissionService) GetUserPermissions(userID uuid.UUID) ([]string, error) {
    var permissions []string
    
    // Get user with roles
    var user models.User
    if err := s.db.Preload("Roles.Permissions").First(&user, userID).Error; err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    
    // Collect permissions from roles
    permissionSet := make(map[string]bool)
    
    for _, role := range user.Roles {
        rolePerms, err := s.getRolePermissions(role.ID)
        if err != nil {
            continue
        }
        
        for _, perm := range rolePerms {
            permissionKey := fmt.Sprintf("%s:%s", perm.Module, perm.Action)
            permissionSet[permissionKey] = true
        }
    }
    
    // Convert to slice
    for perm := range permissionSet {
        permissions = append(permissions, perm)
    }
    
    return permissions, nil
}

func (s *PermissionService) getRolePermissions(roleID uuid.UUID) ([]models.Permission, error) {
    var permissions []models.Permission
    
    err := s.db.Table("permissions").
        Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").
        Where("role_permissions.role_id = ?", roleID).
        Find(&permissions).Error
    
    return permissions, err
}

func (s *PermissionService) HasPermission(userID uuid.UUID, module, action string) (bool, error) {
    permissions, err := s.GetUserPermissions(userID)
    if err != nil {
        return false, err
    }
    
    requiredPermission := fmt.Sprintf("%s:%s", module, action)
    
    for _, perm := range permissions {
        if perm == requiredPermission || perm == "*" {
            return true, nil
        }
    }
    
    return false, nil
}

func (s *PermissionService) CheckResourceAccess(userID uuid.UUID, resourceType, resourceID string) (bool, error) {
    // Get user scopes
    var scopes []models.UserScope
    if err := s.db.Where("user_id = ? AND scope_type = ?", userID, resourceType).Find(&scopes).Error; err != nil {
        return false, err
    }
    
    // If no scopes defined, deny access
    if len(scopes) == 0 {
        return false, nil
    }
    
    // Check if resource is in allowed scope
    for _, scope := range scopes {
        if s.matchesScope(scope.ScopeValue, resourceID) {
            return true, nil
        }
    }
    
    return false, nil
}

func (s *PermissionService) matchesScope(scopeValue models.JSONB, resourceID string) bool {
    // Implementation depends on scope value structure
    // For now, simple string comparison
    if ids, ok := scopeValue["resource_ids"].([]interface{}); ok {
        for _, id := range ids {
            if idStr, ok := id.(string); ok && idStr == resourceID {
                return true
            }
        }
    }
    
    return false
}
```

**✅ 完了条件**: Permission Matrix実装完了

---

## 🎯 **Phase 4 完了基準**

- [ ] JWT認証サービス実装完了
- [ ] 認証ミドルウェア実装完了
- [ ] Permission Matrix実装完了

**🎉 Phase 4 成果物**: JWT認証・Permission Matrix完成

---

## 🏢 **Phase 5: ビジネスロジック実装**
> **期間**: 7-10日 | **ゴール**: 業務API完成

### **STEP 5.1: ユーザー管理API** _(300分)_

#### **5.1.1: internal/handlers/user.go**
```go
package handlers

import (
    "net/http"
    "strconv"
    
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    
    "erp-access-control-api/internal/services"
    "erp-access-control-api/pkg/errors"
)

type UserHandler struct {
    userService *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
    return &UserHandler{userService: userService}
}

func (h *UserHandler) CreateUser(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, errors.NewValidationError("request", err.Error()))
        return
    }
    
    user, err := h.userService.CreateUser(req)
    if err != nil {
        if apiErr, ok := err.(*errors.APIError); ok {
            c.JSON(apiErr.Status, apiErr)
            return
        }
        c.JSON(http.StatusInternalServerError, errors.NewDatabaseError(err))
        return
    }
    
    c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) GetUsers(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
    
    users, total, err := h.userService.GetUsers(page, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, errors.NewDatabaseError(err))
        return
    }
    
    response := ListUsersResponse{
        Users: users,
        Pagination: PaginationResponse{
            Page:  page,
            Limit: limit,
            Total: total,
        },
    }
    
    c.JSON(http.StatusOK, response)
}

func (h *UserHandler) GetUser(c *gin.Context) {
    userID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, errors.NewValidationError("id", "invalid UUID"))
        return
    }
    
    user, err := h.userService.GetUser(userID)
    if err != nil {
        if apiErr, ok := err.(*errors.APIError); ok {
            c.JSON(apiErr.Status, apiErr)
            return
        }
        c.JSON(http.StatusInternalServerError, errors.NewDatabaseError(err))
        return
    }
    
    c.JSON(http.StatusOK, user)
}

// Request/Response DTOs
type CreateUserRequest struct {
    Email        string    `json:"email" binding:"required,email"`
    Password     string    `json:"password" binding:"required,min=8"`
    FirstName    string    `json:"first_name" binding:"required"`
    LastName     string    `json:"last_name" binding:"required"`
    DepartmentID uuid.UUID `json:"department_id" binding:"required"`
}

type ListUsersResponse struct {
    Users      []UserResponse      `json:"users"`
    Pagination PaginationResponse  `json:"pagination"`
}

type UserResponse struct {
    ID           uuid.UUID `json:"id"`
    Email        string    `json:"email"`
    FirstName    string    `json:"first_name"`
    LastName     string    `json:"last_name"`
    Status       string    `json:"status"`
    DepartmentID uuid.UUID `json:"department_id"`
    CreatedAt    time.Time `json:"created_at"`
}

type PaginationResponse struct {
    Page  int `json:"page"`
    Limit int `json:"limit"`
    Total int `json:"total"`
}
```

**✅ 完了条件**: ユーザー管理API実装完了

---

### **STEP 5.2: 部署管理API** _(240分)_

#### **5.2.1: internal/handlers/department.go**
```go
package handlers

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    
    "erp-access-control-api/internal/services"
    "erp-access-control-api/pkg/errors"
)

type DepartmentHandler struct {
    departmentService *services.DepartmentService
}

func NewDepartmentHandler(departmentService *services.DepartmentService) *DepartmentHandler {
    return &DepartmentHandler{departmentService: departmentService}
}

func (h *DepartmentHandler) CreateDepartment(c *gin.Context) {
    var req CreateDepartmentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, errors.NewValidationError("request", err.Error()))
        return
    }
    
    department, err := h.departmentService.CreateDepartment(req)
    if err != nil {
        if apiErr, ok := err.(*errors.APIError); ok {
            c.JSON(apiErr.Status, apiErr)
            return
        }
        c.JSON(http.StatusInternalServerError, errors.NewDatabaseError(err))
        return
    }
    
    c.JSON(http.StatusCreated, department)
}

func (h *DepartmentHandler) GetDepartments(c *gin.Context) {
    departments, err := h.departmentService.GetDepartmentHierarchy()
    if err != nil {
        c.JSON(http.StatusInternalServerError, errors.NewDatabaseError(err))
        return
    }
    
    c.JSON(http.StatusOK, departments)
}

// Request/Response DTOs
type CreateDepartmentRequest struct {
    Name        string     `json:"name" binding:"required"`
    Description string     `json:"description"`
    ParentID    *uuid.UUID `json:"parent_id"`
}

type DepartmentResponse struct {
    ID          uuid.UUID               `json:"id"`
    Name        string                  `json:"name"`
    Description string                  `json:"description"`
    ParentID    *uuid.UUID              `json:"parent_id"`
    Children    []DepartmentResponse    `json:"children,omitempty"`
    CreatedAt   time.Time               `json:"created_at"`
}
```

**✅ 完了条件**: 部署管理API実装完了

---

### **STEP 5.3: ロール・権限管理API** _(180分)_

#### **5.3.1: internal/handlers/role.go**
```go
package handlers

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    
    "erp-access-control-api/internal/services"
    "erp-access-control-api/pkg/errors"
)

type RoleHandler struct {
    roleService *services.RoleService
}

func NewRoleHandler(roleService *services.RoleService) *RoleHandler {
    return &RoleHandler{roleService: roleService}
}

func (h *RoleHandler) CreateRole(c *gin.Context) {
    var req CreateRoleRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, errors.NewValidationError("request", err.Error()))
        return
    }
    
    role, err := h.roleService.CreateRole(req)
    if err != nil {
        if apiErr, ok := err.(*errors.APIError); ok {
            c.JSON(apiErr.Status, apiErr)
            return
        }
        c.JSON(http.StatusInternalServerError, errors.NewDatabaseError(err))
        return
    }
    
    c.JSON(http.StatusCreated, role)
}

func (h *RoleHandler) GetRoles(c *gin.Context) {
    roles, err := h.roleService.GetRoles()
    if err != nil {
        c.JSON(http.StatusInternalServerError, errors.NewDatabaseError(err))
        return
    }
    
    c.JSON(http.StatusOK, roles)
}

func (h *RoleHandler) AssignPermissions(c *gin.Context) {
    roleID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, errors.NewValidationError("id", "invalid UUID"))
        return
    }
    
    var req AssignPermissionsRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, errors.NewValidationError("request", err.Error()))
        return
    }
    
    err = h.roleService.AssignPermissions(roleID, req.PermissionIDs)
    if err != nil {
        if apiErr, ok := err.(*errors.APIError); ok {
            c.JSON(apiErr.Status, apiErr)
            return
        }
        c.JSON(http.StatusInternalServerError, errors.NewDatabaseError(err))
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Permissions assigned successfully"})
}

// Request/Response DTOs
type CreateRoleRequest struct {
    Name        string     `json:"name" binding:"required"`
    Description string     `json:"description"`
    ParentID    *uuid.UUID `json:"parent_id"`
}

type AssignPermissionsRequest struct {
    PermissionIDs []uuid.UUID `json:"permission_ids" binding:"required"`
}

type RoleResponse struct {
    ID          uuid.UUID   `json:"id"`
    Name        string      `json:"name"`
    Description string      `json:"description"`
    ParentID    *uuid.UUID  `json:"parent_id"`
    CreatedAt   time.Time   `json:"created_at"`
}
```

**✅ 完了条件**: ロール・権限管理API実装完了

---

## 🎯 **Phase 5 完了基準**

- [ ] ユーザー管理API実装完了
- [ ] 部署管理API実装完了
- [ ] ロール・権限管理API実装完了

**🎉 Phase 5 成果物**: 業務API完成・MVP実現

---

## 🎉 **MVP完成基準**

### **✅ 機能要件**
- [ ] ユーザー作成・一覧・詳細・更新・削除
- [ ] 部署作成・階層表示・管理
- [ ] ロール作成・権限割り当て・管理
- [ ] JWT認証・トークン検証
- [ ] Permission Matrix による動的権限チェック
- [ ] 監査ログ記録

### **✅ 技術要件**  
- [ ] Go 1.24 + Gin + GORM + PostgreSQL
- [ ] OpenAPI仕様書・Swagger UI
- [ ] 構造化ログ・エラーハンドリング
- [ ] 設定管理・環境変数対応

### **✅ API仕様**
- [ ] RESTful API設計
- [ ] 適切なHTTPステータスコード
- [ ] JSON レスポンス統一
- [ ] ページネーション対応

---

## 📊 **進捗追跡**

| Phase | 開始日 | 完了日 | 状況 | 成果物 |
|-------|--------|--------|------|--------|
| **Phase 1** | - | - | ⏳ | プロジェクト基盤 |
| **Phase 2** | - | - | ⏳ | データベース基盤 |
| **Phase 3** | - | - | ⏳ | API基盤・Swagger |
| **Phase 4** | - | - | ⏳ | 認証認可システム |
| **Phase 5** | - | - | ⏳ | 業務API |

---

## 🔄 **更新履歴**

| 日付 | 更新内容 | 担当者 |
|------|----------|--------|
| 2025-01-15 | 初版作成・Phase1-5詳細設計 | System |
| - | - | - |

---

**🎯 次のアクション**: Phase 1 STEP 1.1 プロジェクト構造作成から開始
