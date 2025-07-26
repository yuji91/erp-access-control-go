# 🚀 **ERP Access Control API - 環境セットアップガイド**

> **Permission Matrix + Policy Object ハイブリッド構成**の完全セットアップ手順

---

## 📋 **セットアップ方法一覧**

| 方法 | 難易度 | 対象者 | 特徴 |
|------|--------|--------|------|
| **🐳 Docker** | ⭐ | 全員 | 最も簡単・依存関係不要 |
| **⚡ Makefile** | ⭐⭐ | Go開発者 | 自動化・詳細制御 |
| **📋 手動** | ⭐⭐⭐ | 詳細理解したい方 | 完全制御・学習目的 |

---

## 🐳 **Docker環境（推奨・最も簡単）**

### **🎯 ポートフォリオ用クイックスタート**

**誰でも2コマンドで起動可能**：

```bash
# 1. プロジェクトクローン
git clone <repository-url>
cd erp-access-control-go

# 2. Docker環境起動
make docker-up-dev

# 🎉 完了！ - http://localhost:8080 でAPI利用可能
```

### **🛠️ Docker環境詳細**

#### **基本サービス起動**
```bash
# PostgreSQL + Redis のみ起動
make docker-up

# 全サービス起動（pgAdmin, Redis Commander含む）
make docker-up-all

# アプリケーション含む開発環境起動
make docker-up-dev
```

#### **管理・確認**
```bash
# サービス状況確認
make docker-ps

# ログ表示
make docker-logs         # 全サービス
make docker-logs-app     # アプリケーションのみ

# コンテナログイン
make docker-exec-app     # アプリケーションコンテナ
make docker-exec-db      # PostgreSQLコンテナ
```

#### **データベース管理**
```bash
# マイグレーション実行
make docker-migrate

# pgAdmin でDB管理
# → http://localhost:5050
# Email: admin@erp-demo.com
# Password: admin_password_2024
```

#### **環境リセット**
```bash
# Docker環境停止
make docker-down

# 完全リセット（ボリューム削除）
make docker-reset
```

### **🔧 利用可能なサービス**

| サービス | URL | 認証情報 |
|----------|-----|----------|
| **API サーバー** | http://localhost:8080 | - |
| **pgAdmin** | http://localhost:5050 | admin@erp-demo.com / admin_password_2024 |
| **Redis Commander** | http://localhost:8081 | - |
| **PostgreSQL** | localhost:5432 | erp_user / erp_password_2024 |
| **Redis** | localhost:6379 | erp_redis_password_2024 |

### **💡 Docker環境の特徴**

- ✅ **ワンコマンドセットアップ**: 依存関係不要
- ✅ **ポートフォリオ対応**: 認証情報直書きで簡単共有
- ✅ **ホットリロード**: Air使用で自動再起動
- ✅ **永続化**: PostgreSQL・Redisデータ保持
- ✅ **管理ツール**: pgAdmin・Redis Commander内蔵
- ✅ **開発効率**: Go mod cache・build cache活用

---

## ⚡ **Makefile自動セットアップ**

### **🎯 クイックスタート**

**Makefile**を使用した自動セットアップ：

```bash
# 1. 環境変数設定
export DB_PASSWORD=your_password_here

# 2. 全自動セットアップ（ディレクトリ作成・依存関係・ツール・DB準備）
make setup

# 3. マイグレーション実行
make migrate-up

# 4. 開発サーバー起動
make dev  # PostgreSQL起動→サーバー起動
```

**利用可能なコマンド確認**：
```bash
make help  # 📋 全コマンド一覧（カテゴリ別・カラー表示）
```

### **⚡ 主要開発コマンド**

| コマンド | 説明 | 用途 |
|----------|------|------|
| `make help` | 📋 全コマンド一覧表示 | コマンド確認 |
| `make dev` | 💻 開発モード起動 | 日常開発 |
| `make quality` | 🏆 コード品質チェック | CI/リリース前 |
| `make test-coverage` | 📊 カバレッジ測定 | テスト確認 |
| `make api-docs-open` | 🌐 APIドキュメント表示 | API確認 |
| `make db-status` | ℹ️ DB接続状態確認 | 接続問題診断 |
| `make clean` | 🧹 全クリーンアップ | 環境リセット |

---

## 📋 **手動セットアップ（詳細版）**

### **1. 依存関係のインストール**
```bash
# Go依存関係ダウンロード
go mod tidy && go mod download

# または
make setup-deps
```

### **2. PostgreSQL準備**
```bash
# PostgreSQL インストール（macOS）
brew install postgresql
brew services start postgresql

# データベース作成
createdb erp_access_control
createdb erp_access_control_test  # テスト用

# または
make db-setup
```

### **3. .env ファイル設定**
`.env`ファイルを作成して以下の設定を追加：

```bash
# アプリケーション基本設定
APP_NAME=erp-access-control-api
APP_ENV=development
SERVER_PORT=8080

# データベース設定
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password_here
DB_NAME=erp_access_control
DB_SSLMODE=disable

# JWT認証設定（開発用）
JWT_SECRET=your-256-bit-secret-key-here-change-in-production
JWT_ACCESS_TOKEN_DURATION=15m
JWT_REFRESH_TOKEN_DURATION=168h

# ログ設定
LOG_LEVEL=debug
LOG_FORMAT=json

# セキュリティ設定
BCRYPT_COST=10
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

# OpenAPI設定
SWAGGER_ENABLED=true
SWAGGER_HOST=localhost:8080
```

### **4. プロジェクト構造作成**
```bash
# 必要ディレクトリ作成
mkdir -p cmd/server internal/{handlers,services,middleware,config} pkg/{logger,errors,jwt}

# ログディレクトリ
mkdir -p logs

# テスト用ディレクトリ  
mkdir -p tests/{integration,unit}

# または
make setup-dirs
```

### **5. 初回マイグレーション実行**
```bash
# マイグレーションツール準備
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# マイグレーション実行
migrate -path ./migrations -database "postgres://postgres:your_password@localhost/erp_access_control?sslmode=disable" up

# または
make migrate-up
```

### **6. 開発サーバー起動**
```bash
# サーバー起動（実装完了後）
go run cmd/server/main.go

# または
make run
```

### **7. API仕様確認**
```bash
# OpenAPI仕様検証
redocly lint api/openapi.yaml

# APIドキュメント生成
redocly build-docs api/openapi.yaml --output docs/api.html

# または
make api-validate
make api-docs-open  # ブラウザで自動表示
```

---

## 🔧 **開発ツール・設定**

### **必要ツールの一括インストール**

#### **Go開発ツール**
```bash
# 開発ツール一括インストール
make setup-tools

# 個別インストール
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/swaggo/swag/cmd/swag@latest
```

#### **OpenAPI・Node.jsツール**
```bash
# NPMツール（グローバルインストール）
npm install -g @redocly/cli @openapitools/openapi-generator-cli

# 確認
redocly --version
openapi-generator-cli version
```

### **環境確認コマンド**
```bash
# 環境状況チェック
make env-check

# プロジェクト情報確認
make info

# データベース接続確認
make db-status
```

---

## 🔍 **トラブルシューティング**

### **よくある問題と解決方法**

#### **1. PostgreSQL接続エラー**
```bash
# 問題: connection refused
# 解決策:
brew services start postgresql
make db-status  # 接続確認
```

#### **2. Go依存関係エラー**
```bash
# 問題: module not found
# 解決策:
go mod tidy
go mod download
```

#### **3. Docker起動エラー**
```bash
# 問題: port already in use
# 解決策:
make docker-down
lsof -i :5432  # ポート使用確認
make docker-up
```

#### **4. マイグレーションエラー**
```bash
# 問題: dirty database
# 解決策:
make migrate-down
make migrate-up
```

### **ログファイル場所**
```bash
# アプリケーションログ
logs/app.log

# Docker環境ログ
make docker-logs

# ビルドエラーログ（Air使用時）
build-errors.log
```

---

## 🌐 **ポート・URL一覧**

| サービス | ポート | URL | 備考 |
|----------|--------|-----|------|
| **APIサーバー** | 8080 | http://localhost:8080 | メインAPI |
| **PostgreSQL** | 5432 | localhost:5432 | データベース |
| **PostgreSQL（テスト）** | 5433 | localhost:5433 | テスト用DB |
| **Redis** | 6379 | localhost:6379 | キャッシュ |
| **pgAdmin** | 5050 | http://localhost:5050 | DB管理 |
| **Redis Commander** | 8081 | http://localhost:8081 | Redis管理 |
| **Prometheus** | 9090 | http://localhost:9090 | メトリクス |

---

## 📝 **開発ワークフロー**

### **日常開発**
```bash
# 1. 環境起動
make docker-up-dev    # または make dev

# 2. コード変更（ホットリロード自動実行）

# 3. テスト実行
make test

# 4. 品質チェック
make quality

# 5. API確認
make api-docs-open
```

### **コード品質管理**
```bash
# フォーマット・静的解析・テスト
make quality

# カバレッジ測定
make test-coverage

# APIドキュメント更新
make api-docs
```

### **環境リセット**
```bash
# 開発環境リセット
make dev-reset

# Docker環境リセット
make docker-reset

# 完全クリーンアップ
make clean
```

---

**🎯 目標**: **Permission Matrix + Policy Object** の実用的なハイブリッド構成による、企業レベルのERPアクセス制御APIシステム
