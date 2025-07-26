# 📚 Golangライブラリ選定ドキュメント

ERP向けアクセス制御APIの実装における、技術スタック選定の記録です。カテゴリごとに検討内容を整理し、採用・比較・補完ライブラリを網羅しています。

## 📂 一覧（リンク付き）

### 🔰 0. 概要・全体構成
- [00_selection_overview.md](00_selection_overview.md)  
  技術選定の背景とカテゴリ別の候補一覧

### 🚀 1〜3. API基盤選定
- [01_api_framework_selection.md](01_api_framework_selection.md)  
  Gin / Echo などAPIフレームワークの比較と採用理由
- [02_openapi_tool_selection.md](02_openapi_tool_selection.md)  
  swag / go-openapi の使い分けとユースケース
- [03_orm_selection.md](03_orm_selection.md)  
  GORM / migrate などDB構成・マイグレーションツール選定

### 🔐 4〜6. 権限制御・認証・DI構成
- [04_rbac_library_selection.md](04_rbac_library_selection.md)  
  自作Permission Matrix vs casbin 比較
- [05_auth_library_selection.md](05_auth_library_selection.md)  
  JWTベースの認証構成（golang-jwt / chi-jwtauth）
- [06_di_strategy_selection.md](06_di_strategy_selection.md)  
  fx / wire などのDIツールとStrategyパターンの適用

### 📝 7. ログ・監査
- [07_logging_library_selection.md](07_logging_library_selection.md)  
  zap / logrus などの構造化ロギング比較と用途別整理

### 🧩 8〜10. 追加・補完ライブラリ
- [08_library_selection_summary.md](08_library_selection_summary.md)  
  全体の採用ライブラリ総括（まとめ）
- [09_library_options_considered.md](09_library_options_considered.md)  
  validator / viper / testify など補完ライブラリの検討
- [10_library_selection_extended.md](10_library_selection_extended.md)  
  セキュリティ・監視・パフォーマンス対策ライブラリの追加提案

---

## ✅ 対象要件（抜粋）

- 部門 / モジュール / ステータス別の複合RBAC
- PolicyObjectを用いた柔軟な権限制御
- OpenAPIベースのAPIドキュメント・DSL構築
- JWT認証 + Gin構成によるREST API設計
- 権限エラーの理由説明・ログ記録・監査対応

## 📌 備考

- 上記 `.md` ファイルはすべて [docs/design/golang_libraries/](.) に格納
- 実装コードやテンプレートは別途 `examples/`, `config/`, `testcases/` ディレクトリに整理予定

---

## 🎯 **作成されたgo.modの特徴**

### 📋 **ライブラリ構成（カテゴリ別）**

| カテゴリ | ライブラリ | 選定理由 |
|----------|------------|----------|
| **🚀 Core Framework** | `gin-gonic/gin` | 高性能・OpenAPI親和性 |
| **📋 OpenAPI/Swagger** | `swaggo/swag` + `gin-swagger` | 自動ドキュメント生成 |
| **🗄️ Database/ORM** | `gorm` + `golang-migrate/migrate` | 型安全・SQL管理分離 |
| **🔐 Auth** | `golang-jwt/jwt/v5` | JWT標準・カスタムクレーム |
| **🔧 DI** | `uber-go/fx` | 動的ポリシー切り替え |
| **📝 Logging** | `uber-go/zap` | 高速構造化ログ |
| **✅ Validation** | `go-playground/validator/v10` | 動的入力検証 |
| **⚙️ Config** | `spf13/viper` | 設定外部管理 |
| **🛡️ Security** | `gin-contrib/cors`, `secure`, `time/rate` | CORS・セキュリティヘッダー・Rate Limiting |
| **📊 Monitoring** | `prometheus/client_golang` | メトリクス監視 |
| **🧪 Testing** | `stretchr/testify`, `uber-go/mock` | テスト駆動開発 |

### 🚀 **実装準備完了**

```bash
# プロジェクト初期化
go mod tidy
go mod download

# 依存関係確認
go list -m all
```

### 📂 **対応するプロジェクト構造**

```
erp-access-control-api/
├── go.mod                    # ✅ 作成済み
├── models/                   # ✅ 作成済み（GORMモデル）
├── docs/migration/           # ✅ 作成済み（DBスキーマ）
├── cmd/                      # → アプリケーションエントリポイント
├── internal/
│   ├── handlers/             # → Ginハンドラ
│   ├── services/             # → ビジネスロジック・Policy
│   ├── middleware/           # → JWT認証・監査ログ
│   └── config/               # → Viper設定管理
├── api/                      # → OpenAPI定義
├── migrations/               # → golang-migrate SQL
└── pkg/                      # → 外部利用可能ライブラリ
```

## ✅ **次のステップ**

### 1. **依存関係のダウンロード**
```bash
go mod tidy
```

### 2. **プロジェクト構造の作成**
```bash
mkdir -p cmd/server internal/{handlers,services,middleware,config} api migrations pkg
```

### 3. **基本ファイルの作成**
- `cmd/server/main.go` - アプリケーションエントリポイント
- `internal/config/config.go` - Viper設定管理
- `api/openapi.yaml` - OpenAPI仕様

このgo.modにより、**Permission Matrix + Policy Object のハイブリッド構成**を完全にサポートする、本格的なERPアクセス制御APIの開発準備が整いました！
