# ERP向けアクセス制御API - OpenAPIライブラリ選定

ERP向けアクセス制御APIの要件と設計方針を踏まえると、OpenAPI対応ライブラリとしては以下のように使い分けが明確です。

## ✅ 結論：用途に応じて使い分けるのが最適

| ライブラリ名 | 主な役割 | 推奨用途 | 補足 |
|-------------|----------|----------|------|
| swaggo/swag | コメントベースでOpenAPI仕様を生成 | ✅ Ginでの自動生成ドキュメント | 実行時 /swagger/* でUI表示可能。Goコードから生成する方式。 |
| go-openapi/loads | OpenAPI YAMLをGo構造体にパース（runtime） | ✅ YAML定義ベースの静的検証やDSL構成 | 既存のOpenAPI YAMLをベースにロジックを駆動させたい場合に向く。 |

## 🎯 選定のポイント

| 観点 | swaggo/swag | go-openapi/loads |
|------|-------------|------------------|
| 目的 | Goコード→Swagger UI生成（開発者向け） | OpenAPI YAML→Go構造体パース（動的活用） |
| 開発フェーズ | API開発中・仕様共有 | テスト・Mock・バリデーション・DSL構築 |
| 親和性 | Gin / Echo に強い | 特定のFW依存なし（汎用ライブラリ） |
| Swagger UI表示 | ◎ 自動で対応可能 | ❌ 直接表示は非対応（別途UI必要） |

## 💡 組み合わせ方の実践例

| 利用ケース | 適切なライブラリ | 解説 |
|-----------|-----------------|------|
| GinベースでAPI仕様とSwagger UIを用意したい | swaggo/swag | コメントに @Summary や @Param を書くことで /swagger/index.html が自動生成 |
| 既存の openapi.yaml を読み取り、動的にPolicyResolverの対象としたい | go-openapi/loads | loads.Analyzed などで構造体を生成し、パスやパラメータに応じた権限DSLを組める |
| テストコードでAPI仕様に準拠しているか検証したい | go-openapi/validate, loads | スキーマ検証・テストとの整合性に便利（例：パスの存在、型一致） |

## 🛠️ ERPプロジェクトでの適用案

### 開発フェーズ（Ginベース）

- `swaggo/swag` を使用して `/me/permissions` などのAPIをコメントから生成
- Swagger UIでフロントと確認共有

### 検証 / ポリシー構築フェーズ

- `openapi.yaml` を `go-openapi/loads` でロードし、
  - "resource_type" に応じたPolicy分岐
  - "action" ごとの許可ルール生成
  - API定義と実装ロジックの差異チェック

## 🧩 最終整理：選定ガイド

| 利用目的 | ライブラリ | 採用有無（ERP案件） |
|----------|-----------|-------------------|
| API開発とSwagger UI生成 | swaggo/swag | ✅ 採用推奨 |
| OpenAPI YAMLベースのDSLパース・検証 | go-openapi/loads | ✅ 採用検討（特にテスト or DSL時） |

必要であれば、それぞれのライブラリを使ったコードテンプレートも提示可能です。ご希望があればお知らせください。

---

## GinベースのAPI構成について

Ginをルーティング / APIフレームワークに採用する場合、以下の構成がベストプラクティスになります。

### ✅ GinベースのAPI構成テンプレート（ERPアクセス制御）

| 層 | 概要 | 備考 |
|----|------|------|
| main.go | ルートエントリ、Routerの初期化 | Gin Engineの起動とDI |
| routes/ | 各APIグループのルーティング | /me, /resources, /audit-logs など |
| handlers/ | 各エンドポイントのハンドラ関数 | Ginの *gin.Context を受け取る |
| services/ | ビジネスロジック / Policy判断 | PolicyObject構成・PermissionMatrix評価 |
| middleware/ | 認証・ロギング・監査 | JWT解析・アクセスログ・拒否理由の出力 |
| models/ | GORMモデル | User, Role, Permission など |
| docs/ | swaggo/swag 生成ドキュメント | @Summary, @Param コメントで管理 |
| config/ | DB, Logger, 認可設定 | .env, struct, Viper等 |

### 🧩 Ginにおける主な技術選定

| 項目 | ライブラリ | 理由 |
|------|-----------|------|
| Web Framework | github.com/gin-gonic/gin | 高速、構造体バインド、豊富な拡張性 |
| Swagger生成 | github.com/swaggo/swag + github.com/swaggo/gin-swagger | コメント → OpenAPI UI |
| バリデーション | github.com/go-playground/validator/v10 | Ginに標準組込、binding:"required" で活用 |
| データベース | gorm.io/gorm | モデル管理、マイグレーションが豊富 |
| 認証 | github.com/appleboy/gin-jwt/v2 等 | JWT解析し user_id をContextに注入 |
| Policy構造 | 自作Interface（例：PolicyResolver） | Strategy Patternで柔軟な権限制御に対応 |

### 📂 具体的なディレクトリ構成（例）

```
.
├── main.go
├── routes/
│   └── router.go
├── handlers/
│   ├── permissions.go
│   ├── resources.go
│   └── audit_logs.go
├── services/
│   ├── policy_resolver.go
│   └── matrix_loader.go
├── middleware/
│   ├── auth.go
│   └── audit_logger.go
├── models/
│   ├── user.go
│   ├── permission.go
│   └── ...
├── docs/     ← `swag init` により生成
│   └── swagger.json
├── go.mod
└── config/
    └── config.go
```

### 📌 サンプルルーティング（Gin）

```go
func SetupRouter() *gin.Engine {
    r := gin.Default()

    // Swagger
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    // 認証ミドルウェア
    authorized := r.Group("/api/v1")
    authorized.Use(AuthMiddleware())

    // /me/permissions
    authorized.GET("/me/permissions", handlers.GetUserPermissions)

    // /resources/:type/:id/actions/:action
    authorized.POST("/resources/:type/:id/actions/:action", handlers.PostResourceAction)

    // /audit-logs
    authorized.GET("/audit-logs", handlers.GetAuditLogs)

    return r
}
```

### 🧪 テスト戦略（Gin + 権限評価）

| テスト対象 | 方式 | ツール |
|-----------|------|-------|
| ハンドラ単体テスト | テーブル駆動 | httptest + gin.CreateTestContext |
| PolicyResolverのロジックテスト | Strategyごとに分離 | go test ./services/... |
| API統合テスト | JWT発行〜レスポンス検証 | httptest + net/http/httptest |
| Swagger定義との整合性検証 | swag validate or go-openapi | 任意で導入 |

## ✨ まとめ：Gin選定のメリット in ERP ACL

- OpenAPIベースとの親和性が非常に高い（swaggo）
- 構造体ベースのBinding・Validationが充実
- Policy Objectの実装と分離しやすい
- 将来的にEchoへの移行も可能な設計が取りやすい

ご希望があれば、`/me/permissions` のGin + Swagger + GORM対応の具体コードスニペットも提供可能です。必要であればお申し付けください。