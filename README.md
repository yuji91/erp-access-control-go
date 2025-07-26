# 🔐 **ERP Access Control API**

> **Permission Matrix + Policy Object ハイブリッド構成**による本格的なERPアクセス制御システム

## ℹ️ **システム概要**

企業ERPシステムにおける **多層認可・動的権限制御** を実現するGolang製APIです。  

従来のRBAC（Role-Based Access Control）に加え、  
時間制限・スコープベース・承認フローなどの高度なポリシー制御を組み合わせた、実用性の高いアクセス制御システムを提供します。

**🎯 目標**: **Permission Matrix + Policy Object** の実用的なハイブリッド構成による、企業レベルのERPアクセス制御APIシステム

---

## 🚀 **クイックスタート**

**誰でも2コマンドで起動可能**：

```bash
# 1. プロジェクトクローン
git clone <repository-url>
cd erp-access-control-go

# 2. Docker環境起動
make docker-up-dev

# 🎉 完了！ - http://localhost:8080 でAPI利用可能
```

### **📋 詳細なセットアップ手順**

- 🐳 **Docker環境**: 最も簡単（推奨）
- ⚡ **Makefile自動化**: Go開発者向け
- 📋 **手動セットアップ**: 詳細制御・学習目的
- 🔧 **開発ツール設定**: 各種ツール・トラブルシューティング

**詳細は → [📚 環境セットアップガイド](./docs/setup/README.md)**

---

## 📊 **技術スタック**

| カテゴリ | 選定技術 | 選定理由 |
|----------|----------|----------|
| **言語・フレームワーク** | Go 1.24 + Gin | 高性能・OpenAPI親和性 |
| **データベース** | PostgreSQL + GORM | JSONB・階層クエリ・型安全 |
| **認証** | JWT (golang-jwt/jwt/v5) | カスタムクレーム・無効化管理 |
| **設定管理** | Viper + godotenv | 外部設定・環境変数 |
| **ログ** | uber-go/zap | 高速構造化ログ |
| **テスト** | testify + mock | TDD・モック |
| **API仕様** | OpenAPI 3.0.3 + Swagger | 自動ドキュメント生成 |


### **主要特徴**
- 🛡️ **Permission Matrix**: モジュール×アクション の権限マトリックス
- ⚙️ **Policy Object**: 時間制限・スコープ・承認フロー等の動的制御
- 🏢 **階層管理**: 部署・ロール階層構造サポート  
- 📊 **完全監査**: IP・User-Agent・理由コード付き詳細ログ
- 🔒 **JWT認証**: セキュアなトークンベース認証・無効化管理

---

## 📄 **設計資料**

| 項目 | 内容 | ファイル |
|------|------|----------|
| **要件定義** | ERPアクセス制御の複雑性と6つの要求事項 | [01_requirements_and_complexity.md](./docs/design/access_control/01_requirements_and_complexity.md) |
| **基本設計** | アクセス制御手法の比較検討と選定理由 | [02_strategy_comparison.md](./docs/design/access_control/02_strategy_comparison.md) |
| **詳細設計** | 開発に必要な全要素の網羅チェックリスト | [03_checklist.md](./docs/design/access_control/03_checklist.md) |
| **工程管理** | Phase別開発ロードマップ（MVPまで） | [04_roadmap.md](./docs/design/access_control/04_roadmap.md) |

---

## 📚 **ライブラリ選定**

| 項目 | 内容 | ファイル |
|------|------|----------|
| **選定概要** | Golang環境での候補ライブラリ一覧 | [00_selection_overview.md](./docs/design/golang_libraries/00_selection_overview.md) |
| **選定結論** | 最終選定ライブラリと技術スタック | [08_library_selection_summary.md](./docs/design/golang_libraries/08_library_selection_summary.md) |

> 🔗 **詳細**: [Full Library Selection Index](./docs/design/golang_libraries/README.md)

---

## 🌐 **API仕様書**

| 項目 | 内容 | リンク |
|------|------|--------|
| **OpenAPI仕様 v3.0.0** | 完全なRESTful API定義（41エンドポイント）🆕 **enum定義・型安全性強化** | [openapi.yaml](./api/openapi.yaml) |
| **API開発履歴** | 機能説明・v2→v3改善内容・次のステップ | [API README](./api/draft/README.md) |

> **🎯 v3.0.0 確定版**: レビュー改善完了・本格実装対応済み

### **v3.0.0 主要改善**
- **🔧 required フィールド完全化**: 全requestスキーマで型安全性強化
- **📝 enum定義追加**: ResourceType, ModuleName, ActionName, ReasonCode
- **🛡️ 403レスポンス統一**: セキュリティポリシー一貫性確保

### **開発・確認用ツール**

#### **📦 必要ツールのインストール**
```bash
# OpenAPI ツールをグローバルインストール
npm install -g @redocly/cli @openapitools/openapi-generator-cli
```

#### **🔍 API仕様の検証**
```bash
# Redocly CLI による詳細検証（推奨）
redocly lint api/openapi.yaml

# OpenAPI Generator による基本検証
openapi-generator-cli validate -i api/openapi.yaml
```

#### **📊 API仕様の確認・閲覧**
```bash
# Swagger UI でAPI仕様確認
# 1. OpenAPI Viewerで直接確認
# 2. Swagger Editor: https://editor.swagger.io/

# Redocly による静的HTMLドキュメント生成
redocly build-docs api/openapi.yaml --output docs/api.html
```

#### **🔧 クライアントコード生成**
```bash
# TypeScript型定義生成（✅ v3.0.0 enum対応済み）
openapi-generator-cli generate -i api/openapi.yaml -g typescript-fetch -o generated/typescript-client

# Go クライアント生成（実装用）
openapi-generator-cli generate -i api/openapi.yaml -g go -o generated/go-client
```

> **✅ 検証済み**: パスパラメータ不整合を修正し、全ツールでエラーなく動作確認済み

---

## 🗄️ **データベース設計**

| 項目 | 内容 | ファイル |
|------|------|----------|
| **マイグレーション概要** | 段階的改善プロセス（01→02→03） | [Migration README](./docs/migration/draft/README.md) |
| **最新SQL** | 本番用完全版マイグレーション | [init_migration_erp_acl_refine_02.sql](./migrations/init_migration_erp_acl_refine_02.sql) |

### **GORM モデル定義**
ERPアクセス制御システムの全テーブルに対応したGORMモデル
- 📁 **場所**: [models/](./models/)
- 🔗 **概要**: [models/README.md](./models/README.md)

---

## 📋 **プロジェクト構造**

### **設計思想**
[golang-standards/project-layout](https://github.com/golang-standards/project-layout) の公式推奨に準拠し、**API仕様中心の明確な配置**でコード生成ツール（swagger-codegen, oapi-codegen）が見つけやすく、他のGoプロジェクトとの一貫性を保持。

### **ディレクトリ構成**

#### **設計ドキュメント関連**
```
docs/
├── design/                          # 全体設計思想・アーキテクチャ
│   ├── access_control/              # アクセス制御ドメイン設計
│   │   ├── 01_requirements_and_complexity.md  # システム要件と複雑性分析
│   │   ├── 02_strategy_comparison.md           # 手法比較とハイブリッド選定
│   │   ├── 03_checklist.md                     # 実装要素網羅チェックリスト  
│   │   └── 04_roadmap.md                       # Phase別開発工程管理
│   └── golang_libraries/            # ライブラリ選定プロセス
│       ├── README.md                # 選定INDEX・推奨構造変更
│       ├── 00_selection_overview.md # 候補ライブラリ一覧
│       └── 08_library_selection_summary.md    # 最終選定結論
├── setup/                           # 環境セットアップ手順
│   └── README.md                    # 詳細セットアップガイド
└── migration/                       # DBマイグレーション設計
    ├── README.md                    # マイグレーション進化記録
    └── draft/                       # 段階的改善プロセス
        ├── draft_migration_erp_acl_01.sql  # 基本版
        ├── draft_migration_erp_acl_02.sql  # 拡張版
        └── draft_migration_erp_acl_03.sql  # 完全版
```

#### **標準ディレクトリ構成**
```
/project-root
├── cmd/                             # アプリケーションエントリポイント
│   └── server/                      # APIサーバー本体
├── internal/                        # 内部パッケージ（import制限付き）
│   ├── handlers/                    # HTTPハンドラ
│   ├── services/                    # ビジネスロジック・Policy
│   ├── middleware/                  # JWT認証・監査ログ
│   └── config/                      # Viper設定管理
├── pkg/                             # 外部にも公開可能なライブラリ
│   ├── logger/                      # 構造化ログ（zap）
│   ├── errors/                      # カスタムエラー型
│   └── jwt/                         # JWT認証サービス
├── models/                          # ✅ GORMモデル定義（10ファイル）
├── api/                             # OpenAPI / Swagger定義
│   ├── draft/                       # API仕様ドラフト
│   │   ├── draft_openapi_02.yaml    # 完全版API仕様
│   │   └── README.md                # API概要・次のステップ
│   └── openapi.yaml                 # 本番用API仕様（予定）
├── migrations/                      # ✅ DBマイグレーション（本番用）
│   └── init_migration_erp_acl_refine_02.sql  # 最新完全版
├── go.mod                           # ✅ 選定ライブラリ定義完了
└── README.md                        # このファイル
```

---

## ✅ **現在の完成状況**

| Phase | 状況 | 成果物 |
|-------|------|--------|
| **設計・要件定義** | ✅ 完了 | 要件・手法比較・チェックリスト・ロードマップ |
| **ライブラリ選定** | ✅ 完了 | go.mod定義・技術スタック確定 |
| **DB設計** | ✅ 完了 | PostgreSQLマイグレーション・GORMモデル |
| **API設計** | ✅ 完了 | OpenAPI仕様書（41エンドポイント） |
| **環境構築** | ✅ 完了 | Go 1.24.5・依存関係ダウンロード |
| **実装** | ⏳ 開始準備 | Phase 1: プロジェクト基盤構築から開始 |

> 🔗 **詳細**: [開発進捗状況](./docs/progress/README.md)

---

### **主要リソース**
- **📋 開発チェックリスト**: [03_checklist.md](./docs/design/access_control/03_checklist.md)
- **🗺️ 詳細ロードマップ**: [04_roadmap.md](./docs/design/access_control/04_roadmap.md)
- **🌐 API仕様書**: [openapi.yaml](./api/openapi.yaml)
- **🗄️ データベース設計**: [マイグレーション](./docs/migration/)
