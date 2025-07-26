# =============================================================================
# ERP Access Control API - Makefile
# =============================================================================
# 開発タスク自動化用 Makefile
# Go + PostgreSQL + OpenAPI プロジェクト対応

# 環境変数設定
# -----------------------------------------------------------------------------
# Go環境変数
export GOPATH := $(shell go env GOPATH)
export GOROOT := $(shell go env GOROOT)
export GOBIN := $(GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

# 変数定義
# -----------------------------------------------------------------------------
APP_NAME := erp-access-control-api
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC')
GO_VERSION := $(shell go version | awk '{print $$3}')

# ディレクトリ
BUILD_DIR := build
BIN_DIR := bin
LOGS_DIR := logs
COVERAGE_DIR := coverage
GENERATED_DIR := generated

# データベース設定
DB_HOST := localhost
DB_PORT := 5432
DB_USER := postgres
DB_NAME := erp_access_control
DB_TEST_NAME := erp_access_control_test
DB_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable
DB_TEST_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_TEST_NAME)?sslmode=disable

# Goビルドフラグ
LDFLAGS := -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.goVersion=$(GO_VERSION)
BUILD_FLAGS := -ldflags "$(LDFLAGS)"

# カラー出力
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
MAGENTA := \033[35m
CYAN := \033[36m
WHITE := \033[37m
RESET := \033[0m

# デフォルトターゲット
.DEFAULT_GOAL := help

# -----------------------------------------------------------------------------
# ヘルプ
# -----------------------------------------------------------------------------
.PHONY: help
help: ## 📋 利用可能なコマンド一覧を表示
	@echo "$(CYAN)🔐 ERP Access Control API - Makefile$(RESET)"
	@echo "$(CYAN)================================================$(RESET)"
	@echo ""
	@echo "$(YELLOW)📦 基本コマンド:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '^(setup|build|run|test|clean):' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🗄️  データベース:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '^(db-|migrate):' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🌐 OpenAPI:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '^(api-|swagger-):' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(MAGENTA)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🧪 テスト・品質:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '^(test|lint|fmt|coverage):' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(RED)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🚀 デプロイ・その他:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -vE '^(setup|build|run|test|clean|db-|migrate|api-|swagger-|test|lint|fmt|coverage):' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(WHITE)%-20s$(RESET) %s\n", $$1, $$2}'

# -----------------------------------------------------------------------------
# 基本コマンド
# -----------------------------------------------------------------------------
.PHONY: setup
setup: ## 🚀 開発環境の初期セットアップ
	@echo "$(CYAN)🚀 開発環境セットアップ開始...$(RESET)"
	@$(MAKE) setup-dirs
	@$(MAKE) setup-deps
	@$(MAKE) setup-tools
	@$(MAKE) db-setup
	@echo "$(GREEN)✅ セットアップ完了！$(RESET)"

.PHONY: setup-dirs
setup-dirs: ## 📁 必要ディレクトリの作成
	@echo "$(BLUE)📁 ディレクトリ作成中...$(RESET)"
	@mkdir -p $(BUILD_DIR) $(BIN_DIR) $(LOGS_DIR) $(COVERAGE_DIR) $(GENERATED_DIR)
	@mkdir -p cmd/server internal/{handlers,services,middleware,config} pkg/{logger,errors,jwt}
	@mkdir -p tests/{integration,unit}
	@echo "$(GREEN)✅ ディレクトリ作成完了$(RESET)"

.PHONY: setup-deps
setup-deps: ## 📦 Go依存関係のインストール
	@echo "$(BLUE)📦 Go依存関係インストール中...$(RESET)"
	@go mod tidy
	@go mod download
	@echo "$(GREEN)✅ 依存関係インストール完了$(RESET)"

.PHONY: setup-tools
setup-tools: ## 🔧 開発ツールのインストール
	@echo "$(BLUE)🔧 開発ツールインストール中...$(RESET)"
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "$(YELLOW)📝 NPMツールも確認中...$(RESET)"
	@npm list -g @redocly/cli @openapitools/openapi-generator-cli >/dev/null 2>&1 || echo "$(YELLOW)⚠️  NPMツールが未インストール: npm install -g @redocly/cli @openapitools/openapi-generator-cli$(RESET)"
	@echo "$(GREEN)✅ 開発ツールインストール完了$(RESET)"

.PHONY: build
build: ## 🔨 アプリケーションのビルド
	@echo "$(BLUE)🔨 ビルド中...$(RESET)"
	@go build $(BUILD_FLAGS) -o $(BIN_DIR)/$(APP_NAME) cmd/server/main.go
	@echo "$(GREEN)✅ ビルド完了: $(BIN_DIR)/$(APP_NAME)$(RESET)"

.PHONY: run
run: ## 🏃 開発サーバーの起動
	@echo "$(BLUE)🏃 サーバー起動中...$(RESET)"
	@go run cmd/server/main.go

.PHONY: run-build
run-build: build ## 🏃 ビルド済みバイナリの実行
	@echo "$(BLUE)🏃 ビルド済みサーバー起動中...$(RESET)"
	@./$(BIN_DIR)/$(APP_NAME)

.PHONY: clean
clean: ## 🧹 ビルド成果物とキャッシュのクリーンアップ
	@echo "$(YELLOW)🧹 クリーンアップ中...$(RESET)"
	@rm -rf $(BUILD_DIR) $(BIN_DIR) $(COVERAGE_DIR) $(GENERATED_DIR)
	@go clean -cache -modcache -testcache
	@echo "$(GREEN)✅ クリーンアップ完了$(RESET)"

# -----------------------------------------------------------------------------
# データベース管理
# -----------------------------------------------------------------------------
.PHONY: db-setup
db-setup: ## 🗄️ PostgreSQL環境セットアップ（DB作成含む）
	@echo "$(BLUE)🗄️ PostgreSQL環境セットアップ中...$(RESET)"
	@createdb $(DB_NAME) 2>/dev/null || echo "$(YELLOW)ℹ️  DB $(DB_NAME) は既に存在$(RESET)"
	@createdb $(DB_TEST_NAME) 2>/dev/null || echo "$(YELLOW)ℹ️  DB $(DB_TEST_NAME) は既に存在$(RESET)"
	@echo "$(GREEN)✅ PostgreSQL環境準備完了$(RESET)"

.PHONY: db-start
db-start: ## 🟢 PostgreSQLサービス開始（macOS Homebrew）
	@echo "$(BLUE)🟢 PostgreSQL開始中...$(RESET)"
	@brew services start postgresql
	@echo "$(GREEN)✅ PostgreSQL開始完了$(RESET)"

.PHONY: db-stop
db-stop: ## 🔴 PostgreSQLサービス停止
	@echo "$(YELLOW)🔴 PostgreSQL停止中...$(RESET)"
	@brew services stop postgresql
	@echo "$(GREEN)✅ PostgreSQL停止完了$(RESET)"

.PHONY: db-status
db-status: ## ℹ️ PostgreSQL接続状態確認（Docker優先）
	@echo "$(BLUE)ℹ️  PostgreSQL状態確認中...$(RESET)"
	@echo "$(YELLOW)🐳 Dockerコンテナ確認:$(RESET)"
	@if docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep erp-postgres >/dev/null 2>&1; then \
		echo "$(GREEN)✅ PostgreSQLコンテナ起動中$(RESET)"; \
		docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | head -1; \
		docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep erp-postgres; \
		echo "$(YELLOW)📡 DB接続テスト:$(RESET)"; \
		if docker exec erp-postgres psql -U erp_user -d erp_access_control -c "SELECT version();" >/dev/null 2>&1; then \
			echo "$(GREEN)✅ PostgreSQL DB接続成功$(RESET)"; \
			echo "$(YELLOW)🗄️  データベース情報:$(RESET)"; \
			echo "📚 Database : $$(docker exec erp-postgres psql -U erp_user -d erp_access_control -t -A -c "SELECT current_database();" 2>/dev/null | tr -d ' ')"; \
			echo "👤 User     : $$(docker exec erp-postgres psql -U erp_user -d erp_access_control -t -A -c "SELECT current_user;" 2>/dev/null | tr -d ' ')"; \
			echo "🛠️  Version  : $$(docker exec erp-postgres psql -U erp_user -d erp_access_control -t -A -c "SELECT split_part(version(), ' ', 1) || ' ' || split_part(version(), ' ', 2);" 2>/dev/null | sed 's/PostgreSQL/PostgreSQL /')"; \
		else \
			echo "$(RED)❌ PostgreSQL DB接続失敗$(RESET)"; \
		fi; \
	else \
		echo "$(RED)❌ PostgreSQLコンテナが起動していません$(RESET)"; \
		echo "$(YELLOW)📋 ローカルPostgreSQL確認:$(RESET)"; \
		if command -v brew >/dev/null 2>&1 && brew services list | grep postgresql >/dev/null 2>&1; then \
			brew services list | grep postgresql; \
			if psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -c "SELECT version();" >/dev/null 2>&1; then \
				echo "$(GREEN)✅ ローカルPostgreSQL接続成功$(RESET)"; \
			else \
				echo "$(RED)❌ ローカルPostgreSQL接続失敗$(RESET)"; \
			fi; \
		else \
			echo "$(RED)❌ ローカルPostgreSQLも見つかりません$(RESET)"; \
			echo "$(YELLOW)💡 Dockerコンテナ起動: make docker-up$(RESET)"; \
		fi; \
	fi

.PHONY: migrate-up
migrate-up: ## ⬆️ データベースマイグレーション実行
	@echo "$(BLUE)⬆️ マイグレーション実行中...$(RESET)"
	@migrate -path ./migrations -database "$(DB_URL)" up
	@echo "$(GREEN)✅ マイグレーション完了$(RESET)"

.PHONY: migrate-down
migrate-down: ## ⬇️ データベースマイグレーション巻き戻し
	@echo "$(YELLOW)⬇️ マイグレーション巻き戻し中...$(RESET)"
	@migrate -path ./migrations -database "$(DB_URL)" down
	@echo "$(GREEN)✅ マイグレーション巻き戻し完了$(RESET)"

.PHONY: migrate-reset
migrate-reset: ## 🔄 データベースリセット（down→up）
	@echo "$(YELLOW)🔄 データベースリセット中...$(RESET)"
	@$(MAKE) migrate-down
	@$(MAKE) migrate-up
	@echo "$(GREEN)✅ データベースリセット完了$(RESET)"

.PHONY: db-seed
db-seed: ## 🌱 テストデータ投入
	@echo "$(BLUE)🌱 テストデータ投入中...$(RESET)"
	@psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f migrations/seed.sql 2>/dev/null || echo "$(YELLOW)⚠️  seed.sqlが見つかりません$(RESET)"
	@echo "$(GREEN)✅ テストデータ投入完了$(RESET)"

# -----------------------------------------------------------------------------
# OpenAPI・API関連
# -----------------------------------------------------------------------------
.PHONY: api-validate
api-validate: ## ✅ OpenAPI仕様書の検証
	@echo "$(BLUE)✅ OpenAPI検証中...$(RESET)"
	@redocly lint api/openapi.yaml && echo "$(GREEN)✅ Redocly検証成功$(RESET)" || echo "$(RED)❌ Redocly検証失敗$(RESET)"
	@openapi-generator-cli validate -i api/openapi.yaml && echo "$(GREEN)✅ OpenAPI Generator検証成功$(RESET)" || echo "$(RED)❌ OpenAPI Generator検証失敗$(RESET)"

.PHONY: api-docs
api-docs: ## 📊 APIドキュメント生成（HTML）
	@echo "$(BLUE)📊 APIドキュメント生成中...$(RESET)"
	@redocly build-docs api/openapi.yaml --output docs/api.html
	@echo "$(GREEN)✅ APIドキュメント生成完了: docs/api.html$(RESET)"

.PHONY: api-docs-open
api-docs-open: api-docs ## 🌐 APIドキュメントをブラウザで開く
	@echo "$(BLUE)🌐 APIドキュメントを開いています...$(RESET)"
	@open docs/api.html

.PHONY: swagger-gen
swagger-gen: ## 📝 Swaggerアノテーションから仕様書生成
	@echo "$(BLUE)📝 Swagger仕様書生成中...$(RESET)"
	@swag init -g cmd/server/main.go -o docs/swagger
	@echo "$(GREEN)✅ Swagger仕様書生成完了$(RESET)"

.PHONY: api-client-ts
api-client-ts: ## 🔧 TypeScriptクライアント生成
	@echo "$(BLUE)🔧 TypeScriptクライアント生成中...$(RESET)"
	@openapi-generator-cli generate -i api/openapi.yaml -g typescript-fetch -o $(GENERATED_DIR)/typescript-client
	@echo "$(GREEN)✅ TypeScriptクライアント生成完了: $(GENERATED_DIR)/typescript-client$(RESET)"

.PHONY: api-client-go
api-client-go: ## 🔧 Goクライアント生成
	@echo "$(BLUE)🔧 Goクライアント生成中...$(RESET)"
	@openapi-generator-cli generate -i api/openapi.yaml -g go -o $(GENERATED_DIR)/go-client
	@echo "$(GREEN)✅ Goクライアント生成完了: $(GENERATED_DIR)/go-client$(RESET)"

# -----------------------------------------------------------------------------
# テスト・品質管理
# -----------------------------------------------------------------------------
.PHONY: test
test: ## 🧪 全テスト実行
	@echo "$(BLUE)🧪 テスト実行中...$(RESET)"
	@go test -v ./...
	@echo "$(GREEN)✅ テスト完了$(RESET)"

.PHONY: test-unit
test-unit: ## 🧪 ユニットテスト実行
	@echo "$(BLUE)🧪 ユニットテスト実行中...$(RESET)"
	@go test -v ./tests/unit/...
	@echo "$(GREEN)✅ ユニットテスト完了$(RESET)"

.PHONY: test-integration
test-integration: ## 🧪 統合テスト実行
	@echo "$(BLUE)🧪 統合テスト実行中...$(RESET)"
	@go test -v ./tests/integration/...
	@echo "$(GREEN)✅ 統合テスト完了$(RESET)"

.PHONY: test-coverage
test-coverage: ## 📊 テストカバレッジ測定
	@echo "$(BLUE)📊 カバレッジ測定中...$(RESET)"
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out
	@echo "$(GREEN)✅ カバレッジ測定完了: $(COVERAGE_DIR)/coverage.html$(RESET)"

.PHONY: lint
lint: ## 🔍 コード静的解析（golangci-lint）
	@echo "$(BLUE)🔍 静的解析中...$(RESET)"
	@$(GOBIN)/golangci-lint run
	@echo "$(GREEN)✅ 静的解析完了$(RESET)"

.PHONY: fmt
fmt: ## 📝 コードフォーマット（gofmt + goimports）
	@echo "$(BLUE)📝 コードフォーマット中...$(RESET)"
	@go fmt ./...
	@$(GOBIN)/goimports -w . 2>/dev/null || echo "$(YELLOW)⚠️  goimportsが未インストール: go install golang.org/x/tools/cmd/goimports@latest$(RESET)"
	@echo "$(GREEN)✅ フォーマット完了$(RESET)"

.PHONY: vet
vet: ## 🔍 go vetによるコード検査
	@echo "$(BLUE)🔍 go vet実行中...$(RESET)"
	@go vet ./...
	@echo "$(GREEN)✅ go vet完了$(RESET)"

.PHONY: quality
quality: fmt vet lint test ## 🏆 コード品質チェック（fmt + vet + lint + test）
	@echo "$(GREEN)🏆 コード品質チェック完了$(RESET)"

# -----------------------------------------------------------------------------
# ログ・モニタリング
# -----------------------------------------------------------------------------
.PHONY: logs
logs: ## 📋 アプリケーションログの表示
	@echo "$(BLUE)📋 ログ表示中...$(RESET)"
	@tail -f $(LOGS_DIR)/app.log 2>/dev/null || echo "$(YELLOW)⚠️  ログファイルが見つかりません: $(LOGS_DIR)/app.log$(RESET)"

.PHONY: logs-clear
logs-clear: ## 🧹 ログファイルのクリア
	@echo "$(YELLOW)🧹 ログクリア中...$(RESET)"
	@rm -f $(LOGS_DIR)/*.log
	@echo "$(GREEN)✅ ログクリア完了$(RESET)"

# -----------------------------------------------------------------------------
# デプロイ・リリース
# -----------------------------------------------------------------------------
.PHONY: build-linux
build-linux: ## 🐧 Linux用バイナリビルド
	@echo "$(BLUE)🐧 Linux用ビルド中...$(RESET)"
	@GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 cmd/server/main.go
	@echo "$(GREEN)✅ Linux用ビルド完了: $(BUILD_DIR)/$(APP_NAME)-linux-amd64$(RESET)"

.PHONY: build-windows
build-windows: ## 🪟 Windows用バイナリビルド
	@echo "$(BLUE)🪟 Windows用ビルド中...$(RESET)"
	@GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe cmd/server/main.go
	@echo "$(GREEN)✅ Windows用ビルド完了: $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe$(RESET)"

.PHONY: build-all
build-all: build build-linux build-windows ## 🌍 全プラットフォーム用バイナリビルド
	@echo "$(GREEN)🌍 全プラットフォーム用ビルド完了$(RESET)"

.PHONY: release
release: clean quality build-all ## 🚀 リリース用ビルド（品質チェック込み）
	@echo "$(GREEN)🚀 リリース用ビルド完了$(RESET)"

# -----------------------------------------------------------------------------
# 開発便利コマンド
# -----------------------------------------------------------------------------
.PHONY: dev
dev: db-start run ## 💻 開発モード（DB起動→サーバー起動）

.PHONY: dev-reset
dev-reset: clean setup migrate-reset ## 🔄 開発環境リセット
	@echo "$(GREEN)🔄 開発環境リセット完了$(RESET)"

.PHONY: info
info: ## ℹ️ プロジェクト情報表示
	@echo "$(CYAN)📊 プロジェクト情報$(RESET)"
	@echo "$(CYAN)=================$(RESET)"
	@echo "$(YELLOW)アプリ名:$(RESET) $(APP_NAME)"
	@echo "$(YELLOW)バージョン:$(RESET) $(VERSION)"
	@echo "$(YELLOW)Goバージョン:$(RESET) $(GO_VERSION)"
	@echo "$(YELLOW)ビルド時刻:$(RESET) $(BUILD_TIME)"
	@echo "$(YELLOW)データベース:$(RESET) $(DB_NAME)"
	@echo "$(YELLOW)テストDB:$(RESET) $(DB_TEST_NAME)"

.PHONY: env-check
env-check: ## 🔍 環境変数・設定確認
	@echo "$(BLUE)🔍 環境確認中...$(RESET)"
	@echo "$(YELLOW)Go Version:$(RESET) $$(go version)"
	@echo "$(YELLOW)PostgreSQL:$(RESET) $$(psql --version 2>/dev/null || echo 'Not installed')"
	@echo "$(YELLOW)Migrate:$(RESET) $$(/Users/yuji91/go/bin/migrate -version 2>&1 || echo 'Not installed')"
	@echo "$(YELLOW)Redocly:$(RESET) $$(redocly --version 2>/dev/null || echo 'Not installed')"
	@echo "$(YELLOW)OpenAPI Generator:$(RESET) $$(openapi-generator-cli version 2>/dev/null || echo 'Not installed')"

# -----------------------------------------------------------------------------
# Docker・コンテナ関連
# -----------------------------------------------------------------------------
.PHONY: docker-build
docker-build: ## 🐳 Dockerイメージビルド（開発用）
	@echo "$(BLUE)🐳 開発用Dockerイメージビルド中...$(RESET)"
	@docker build -f Dockerfile.dev -t $(APP_NAME):dev .
	@echo "$(GREEN)✅ 開発用Dockerイメージビルド完了$(RESET)"

.PHONY: docker-build-prod
docker-build-prod: ## 🐳 Dockerイメージビルド（本番用）
	@echo "$(BLUE)🐳 本番用Dockerイメージビルド中...$(RESET)"
	@docker build -f Dockerfile -t $(APP_NAME):latest .
	@echo "$(GREEN)✅ 本番用Dockerイメージビルド完了$(RESET)"

.PHONY: docker-up
docker-up: ## 🚀 Docker Compose起動（基本サービス）
	@echo "$(BLUE)🚀 Docker Compose起動中...$(RESET)"
	@docker-compose up -d postgres redis
	@echo "$(GREEN)✅ Docker Compose起動完了$(RESET)"

.PHONY: docker-up-all
docker-up-all: ## 🚀 Docker Compose起動（全サービス）
	@echo "$(BLUE)🚀 Docker Compose全サービス起動中...$(RESET)"
	@docker-compose --profile app --profile tools up -d
	@echo "$(GREEN)✅ Docker Compose全サービス起動完了$(RESET)"

.PHONY: docker-up-dev
docker-up-dev: ## 💻 Docker開発環境起動（DB+Redis+App）
	@echo "$(BLUE)💻 Docker開発環境起動中...$(RESET)"
	@docker-compose --profile app up -d
	@echo "$(GREEN)✅ Docker開発環境起動完了$(RESET)"

.PHONY: docker-down
docker-down: ## 🛑 Docker Compose停止
	@echo "$(YELLOW)🛑 Docker Compose停止中...$(RESET)"
	@docker-compose down
	@echo "$(GREEN)✅ Docker Compose停止完了$(RESET)"

.PHONY: docker-down-volumes
docker-down-volumes: ## 🗑️ Docker Compose停止（ボリューム削除）
	@echo "$(YELLOW)🗑️ Docker Compose停止（ボリューム削除）中...$(RESET)"
	@docker-compose down -v --remove-orphans
	@echo "$(GREEN)✅ Docker Compose停止（ボリューム削除）完了$(RESET)"

.PHONY: docker-restart
docker-restart: docker-down docker-up ## 🔄 Docker Compose再起動
	@echo "$(GREEN)🔄 Docker Compose再起動完了$(RESET)"

.PHONY: docker-logs
docker-logs: ## 📋 Docker Composeログ表示
	@echo "$(BLUE)📋 Docker Composeログ表示中...$(RESET)"
	@docker-compose logs -f

.PHONY: docker-logs-app
docker-logs-app: ## 📋 アプリケーションログ表示（Docker）
	@echo "$(BLUE)📋 アプリケーションログ表示中...$(RESET)"
	@docker-compose logs -f app

.PHONY: docker-ps
docker-ps: ## 📊 Docker Composeサービス状況確認
	@echo "$(BLUE)📊 Docker Composeサービス状況確認中...$(RESET)"
	@docker-compose ps

.PHONY: docker-services-status
docker-services-status: ## 🔍 全Dockerサービス状態確認（PostgreSQL + Redis + App）
	@echo "$(CYAN)🔍 ERP Access Control Services 状態確認$(RESET)"
	@echo "$(CYAN)==============================================$(RESET)"
	@echo ""
	@echo "$(YELLOW)📊 Docker Compose サービス一覧:$(RESET)"
	@docker-compose ps 2>/dev/null || echo "$(RED)❌ Docker Composeサービスが見つかりません$(RESET)"
	@echo ""
	@echo "$(YELLOW)🗄️  PostgreSQL ステータス:$(RESET)"
	@if docker ps --format "table {{.Names}}\t{{.Status}}" | grep erp-postgres >/dev/null 2>&1; then \
		echo "$(GREEN)✅ PostgreSQLコンテナ起動中$(RESET)"; \
		if docker exec erp-postgres pg_isready -U erp_user >/dev/null 2>&1; then \
			echo "$(GREEN)✅ PostgreSQL Ready$(RESET)"; \
		else \
			echo "$(RED)❌ PostgreSQL Not Ready$(RESET)"; \
		fi; \
	else \
		echo "$(RED)❌ PostgreSQLコンテナ停止中$(RESET)"; \
	fi
	@echo ""
	@echo "$(YELLOW)🔵 Redis ステータス:$(RESET)"
	@if docker ps --format "table {{.Names}}\t{{.Status}}" | grep erp-redis >/dev/null 2>&1; then \
		echo "$(GREEN)✅ Redisコンテナ起動中$(RESET)"; \
		if docker exec erp-redis redis-cli -a erp_redis_password_2024 ping >/dev/null 2>&1; then \
			echo "$(GREEN)✅ Redis Ready$(RESET)"; \
		else \
			echo "$(RED)❌ Redis Not Ready$(RESET)"; \
		fi; \
	else \
		echo "$(RED)❌ Redisコンテナ停止中$(RESET)"; \
	fi
	@echo ""
	@echo "$(YELLOW)🚀 Application ステータス:$(RESET)"
	@if docker ps --format "table {{.Names}}\t{{.Status}}" | grep erp-app >/dev/null 2>&1; then \
		echo "$(GREEN)✅ Applicationコンテナ起動中$(RESET)"; \
		if curl -f http://localhost:8080/health >/dev/null 2>&1; then \
			echo "$(GREEN)✅ Application Ready$(RESET)"; \
		else \
			echo "$(YELLOW)⚠️  Application Not Ready（実装待ち？）$(RESET)"; \
		fi; \
	else \
		echo "$(YELLOW)⚠️  Applicationコンテナ停止中$(RESET)"; \
	fi
	@echo ""
	@echo "$(YELLOW)🔧 管理ツール:$(RESET)"
	@if docker ps --format "table {{.Names}}\t{{.Status}}" | grep erp-pgadmin >/dev/null 2>&1; then \
		echo "$(GREEN)✅ pgAdmin: http://localhost:5050$(RESET)"; \
	else \
		echo "$(YELLOW)⚠️  pgAdmin停止中 (make docker-up-all で起動)$(RESET)"; \
	fi
	@if docker ps --format "table {{.Names}}\t{{.Status}}" | grep erp-redis-commander >/dev/null 2>&1; then \
		echo "$(GREEN)✅ Redis Commander: http://localhost:8081$(RESET)"; \
	else \
		echo "$(YELLOW)⚠️  Redis Commander停止中 (make docker-up-all で起動)$(RESET)"; \
	fi

.PHONY: docker-exec-app
docker-exec-app: ## 🔧 アプリケーションコンテナにログイン
	@echo "$(BLUE)🔧 アプリケーションコンテナにログイン中...$(RESET)"
	@docker-compose exec app sh

.PHONY: docker-exec-db
docker-exec-db: ## 🗄️ PostgreSQLコンテナにログイン
	@echo "$(BLUE)🗄️ PostgreSQLコンテナにログイン中...$(RESET)"
	@docker-compose exec postgres psql -U erp_user -d erp_access_control

.PHONY: docker-db-status
docker-db-status: ## 📊 Docker PostgreSQL詳細情報
	@echo "$(BLUE)📊 Docker PostgreSQL詳細情報確認中...$(RESET)"
	@if docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep erp-postgres >/dev/null 2>&1; then \
		echo "$(GREEN)✅ PostgreSQLコンテナ情報:$(RESET)"; \
		docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}" | head -1; \
		docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}" | grep erp-postgres; \
		echo ""; \
		echo "$(YELLOW)📡 DB接続情報:$(RESET)"; \
		echo "Host: localhost"; \
		echo "Port: 5432"; \
		echo "Database: erp_access_control"; \
		echo "User: erp_user"; \
		echo "Password: erp_password_2024"; \
		echo ""; \
		echo "$(YELLOW)🔍 データベース詳細:$(RESET)"; \
		docker exec erp-postgres psql -U erp_user -d erp_access_control -c "\l" 2>/dev/null | grep -E "(Name|erp_|---|List)"; \
		echo ""; \
		echo "$(YELLOW)📋 テーブル一覧:$(RESET)"; \
		docker exec erp-postgres psql -U erp_user -d erp_access_control -c "\dt" 2>/dev/null || echo "$(YELLOW)⚠️  テーブルが見つかりません（マイグレーション未実行？）$(RESET)"; \
		echo ""; \
		echo "$(YELLOW)🏥 ヘルスチェック:$(RESET)"; \
		docker exec erp-postgres pg_isready -U erp_user -d erp_access_control && echo "$(GREEN)✅ PostgreSQL Ready$(RESET)" || echo "$(RED)❌ PostgreSQL Not Ready$(RESET)"; \
	else \
		echo "$(RED)❌ PostgreSQLコンテナが起動していません$(RESET)"; \
		echo "$(YELLOW)💡 起動方法:$(RESET)"; \
		echo "  make docker-up     # DB+Redisのみ"; \
		echo "  make docker-up-dev # 開発環境一式"; \
	fi

.PHONY: docker-migrate
docker-migrate: ## ⬆️ Dockerマイグレーション実行
	@echo "$(BLUE)⬆️ Dockerマイグレーション実行中...$(RESET)"
	@docker-compose --profile migrate run --rm migrate
	@echo "$(GREEN)✅ Dockerマイグレーション完了$(RESET)"

.PHONY: docker-test
docker-test: ## 🧪 Dockerテスト環境起動
	@echo "$(BLUE)🧪 Dockerテスト環境起動中...$(RESET)"
	@docker-compose --profile test up -d postgres-test
	@echo "$(GREEN)✅ Dockerテスト環境起動完了$(RESET)"

.PHONY: docker-clean
docker-clean: ## 🧹 Docker環境クリーンアップ
	@echo "$(YELLOW)🧹 Docker環境クリーンアップ中...$(RESET)"
	@docker-compose down -v --remove-orphans
	@docker system prune -f
	@docker volume prune -f
	@echo "$(GREEN)✅ Docker環境クリーンアップ完了$(RESET)"

.PHONY: docker-reset
docker-reset: docker-clean docker-up ## 🔄 Docker環境リセット
	@echo "$(GREEN)🔄 Docker環境リセット完了$(RESET)"

# Docker Compose便利エイリアス
.PHONY: dc-up dc-down dc-logs dc-ps
dc-up: docker-up ## 🚀 Docker Compose起動（短縮形）
dc-down: docker-down ## 🛑 Docker Compose停止（短縮形）
dc-logs: docker-logs ## 📋 Docker Composeログ表示（短縮形）
dc-ps: docker-ps ## 📊 Docker Composeサービス状況確認（短縮形）

# -----------------------------------------------------------------------------
# エラーハンドリング
# -----------------------------------------------------------------------------
# PostgreSQLパスワードが未設定の場合の警告
# TODO: 本格的なローカル開発時に必要に応じて有効化
# ifndef DB_PASSWORD
# $(warning DB_PASSWORD環境変数が未設定です。.envファイルで設定してください)
# endif 