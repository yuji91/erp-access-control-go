# 🛡️ ERP Access Control API

> **Permission Matrix + Policy Object ハイブリッド構成**による企業向け認証・認可システム

## 📊 **現在の実装状況**

### ✅ **Phase 4 完了: 認証・認可システム**

| コンポーネント | 実装状況 | 説明 |
|---------------|----------|------|
| **JWT認証** | ✅ 完了 | トークン生成・検証・リフレッシュ・無効化 |
| **Permission Matrix** | ✅ 完了 | 役割ベース権限管理 + ワイルドカード対応 |
| **ミドルウェア** | ✅ 完了 | 認証・権限チェック・所有権検証 |
| **スコープ権限** | ✅ 完了 | JSONB動的ポリシー評価 |
| **設定管理** | ✅ 完了 | Viper統合・環境変数対応 |
| **エラーハンドリング** | ✅ 完了 | 構造化APIエラー |
| **複数ロール対応** | ✅ 完了 | 期限付きロール・優先度管理 |

### 🎯 **技術スタック**
- **Language**: Go 1.23+
- **Framework**: Gin (HTTP), GORM (ORM)
- **Database**: PostgreSQL (JSONB, UUID, INET型)
- **Authentication**: JWT (golang-jwt/jwt/v5)
- **Security**: bcrypt, Token Revocation
- **Config**: Viper (環境変数・YAML)

## 🚀 **クイックスタート**

### **🐳 Docker環境（推奨・最も簡単）**
```bash
# 1. プロジェクトクローン
git clone <repository-url>
cd erp-access-control-go

# 2. ワンコマンドセットアップ
make setup-docker

# 🎉 完了！ - http://localhost:8080 でAPI利用可能
```

### **💻 開発環境（Docker + アプリ）**
```bash
# 完全開発環境セットアップ
make setup-dev
```

### **⚡ 手動セットアップ**
```bash
# 1. 依存関係インストール
go mod tidy

# 2. 環境設定
cp .env.example .env
# 環境変数を適切に設定

# 3. ビルド & テスト
go build ./...
go vet ./...

# 4. ローカル開発サーバー起動
make run  # または go run cmd/server/main.go
```

**⚠️ 注意**: Docker環境が起動中の場合は`make docker-down`で停止してから実行してください。

### **📋 詳細セットアップガイド**
詳細な手順は [docs/setup/README.md](docs/setup/README.md) を参照してください。

## 🐳 **Docker開発環境**

### **前提条件**
- Docker & Docker Compose がインストール済み

### **クイックスタート（Docker）**
```bash
# 1. データベース起動
make docker-up

# 2. マイグレーション実行
make docker-migrate-sql

# 3. 開発サーバー起動
make run

### **🌐 利用可能なサービス**
| サービス | URL | 認証情報 |
|----------|-----|----------|
| **API サーバー** | http://localhost:8080 | - |
| **pgAdmin** | http://localhost:5050 | admin@erp-demo.com / admin_password_2024 |
| **Redis Commander** | http://localhost:8081 | - |

### **便利なコマンド**
```bash
make help                   # 📋 全コマンド一覧（カテゴリ別・カラー表示）
make docker-up              # PostgreSQL + Redis起動
make docker-up-no-restart   # 自動再起動無効で起動
make docker-migrate-sql     # マイグレーション実行
make docker-seed            # シードデータ投入
make docker-setup-dev       # マイグレーション + シード実行
make docker-db-status       # データベース状態確認
make docker-logs            # ログ確認
make docker-down-force      # 強制停止（コンテナ削除）
```

### **開発ワークフロー**
```bash
# 新機能開発時
make setup-docker           # 環境セットアップ
make run                    # アプリケーション起動

# マイグレーション追加時
# 1. migrations/03_xxx.sql を作成
# 2. make docker-migrate-sql で実行
```

## 🔧 **アーキテクチャ概要**

```
📁 プロジェクト構造
├── cmd/server/          # アプリケーションエントリポイント
├── internal/
│   ├── config/          # 設定管理 (Viper)
│   ├── middleware/      # JWT認証・権限チェック
│   └── services/        # ビジネスロジック
├── pkg/
│   ├── jwt/            # JWT認証サービス
│   ├── errors/         # 構造化エラー
│   └── logger/         # ログシステム
├── models/             # GORM データモデル
└── api/               # OpenAPI仕様
```

### 🛡️ **権限システム**

```go
// Permission Matrix 構造
Permission := Module + ":" + Action
例: "user:create", "department:read", "audit:list"

// 階層化権限
super_admin: ["*:*"]                    // 全権限
admin:       ["user:*", "department:*"] // モジュール別全権限
manager:     ["user:read", "user:update"] // アクション別制限
employee:    ["user:read"]              // 読み取りのみ
```

## ⚠️ **重要: TODO改善項目**

現在の実装は**MVP品質**です。本番環境では以下の改善が必要です：

### 🔐 **セキュリティ強化**
```go
// TODO: JWT強化
- RSA公開鍵/秘密鍵方式への移行
- アクセストークン(短期) + リフレッシュトークン(長期)
- レート制限・ブルートフォース攻撃対策
- MFA (多要素認証) 対応

// TODO: パスワードポリシー
- 強度チェック (長さ、複雑性、辞書攻撃対策)
- bcrypt cost調整 (本番: 12-14)
- パスワード履歴管理
```

### 🏗️ **アーキテクチャ拡張**
```go
// ✅ 複数ロール対応 (完了)
type UserRole struct {
    UserID   uuid.UUID
    RoleID   uuid.UUID  
    ValidFrom *time.Time  // 期限付きロール
    ValidTo   *time.Time
    Priority  int         // 優先度管理
    IsActive  bool        // アクティブ状態
}

// TODO: 階層的権限継承
- 部門長→課長→係長の自動権限継承
- 地理的・時間的制限
```

### ⚡ **パフォーマンス最適化**
```go
// TODO: キャッシュレイヤー
- Redis/Memcached による権限キャッシュ
- 階層的権限の事前計算
- N+1クエリ問題の解決

// TODO: 監査ログ強化
- 全API操作の詳細ログ
- セキュリティインシデント検知
- ELK Stack連携
```

## 📋 **開発ロードマップ**

| Phase | 状況 | 次のステップ |
|-------|------|-------------|
| **Phase 1-3** | ✅ 完了 | プロジェクト基盤・DB・API設計 |
| **Phase 4** | ✅ 完了 | 認証・認可システム |
| **Phase 4.1** | ✅ 完了 | 複数ロール対応 |
| **Phase 5** | 🚧 次期 | ビジネスロジック・APIハンドラー |
| **Phase 6+** | 📋 予定 | セキュリティ強化・運用最適化 |

## 🤝 **開発ガイドライン**

### コード品質
- `go build ./...` - ビルド成功必須
- `go vet ./...` - 静的解析パス
- `gofmt -w .` - フォーマット適用

### セキュリティ
- パスワード・秘密鍵をコードに埋め込み禁止
- 全ての認証・認可処理で適切なエラーハンドリング
- ログイン試行・権限チェックの監査ログ記録

## 📚 **参考資料**

- [設計ドキュメント](docs/design/)
- [OpenAPI仕様](api/openapi.yaml)
- [データベース設計](docs/migration/)
- [開発進捗](docs/progress/README.md)

---

**⚠️ 本実装は開発・学習目的です。本番環境使用前にセキュリティ監査とTODO項目の対応を実施してください。**
