GolangでERP向け権限管理APIを実装するにあたり、現在の構成（Permission Matrix + Policy Object ハイブリッド構成）と設計方針に沿ったライブラリの候補を以下に整理します。

# 🧰 推奨ライブラリ一覧（カテゴリ別）

## 1. ルーティング / APIフレームワーク

| ライブラリ | 役割 | 備考 |
|-----------|------|------|
| Gin | 高速・軽量なHTTPルーター | OpenAPI定義と親和性あり |
| Echo | シンプルなミドルウェア構成 | Ginと並ぶ人気 |

✅ どちらもOpenAPI YAML（openapi.yaml）に対応可能。Gin + Swag が主流。

## 2. OpenAPI / Swagger 対応

| ライブラリ | 役割 | 備考 |
|-----------|------|------|
| swaggo/swag | OpenAPI (Swagger) ドキュメント生成 | コメントベースでAPI定義可能 |
| go-openapi/loads | YAML → Go構造体へ読み込み | DSL風にパースして使いたい場合に便利 |

## 3. データベース ORM / マイグレーション

| ライブラリ | 役割 | 備考 |
|-----------|------|------|
| gorm | ORMフレームワーク | 初期スキーマ（init_migration_erp_acl.sql）と互換性あり |
| golang-migrate/migrate | マイグレーション管理 | SQLベースでマイグレーション定義可能。GORMと併用可 |

## 4. RBAC / ACL 実装補助

| ライブラリ | 用途 | 補足 |
|-----------|------|------|
| 自作 Permission Matrix | map[string]map[string][]string などで柔軟に設計 | シンプルかつGoに最適な実装構成 |
| casbin | RBAC / ABAC エンジン | ABAC拡張が必要なら有力。ただし複雑すぎる可能性もある |

❗️ **casbinについて**: 柔軟ですが、DSL記述が必要でGolangの型システムから逸脱しやすいため、Policy Object構成と相性が悪い場合も。

## 5. 認証・ユーザー管理

| ライブラリ | 用途 | 備考 |
|-----------|------|------|
| golang-jwt/jwt | JWTベースの認証 | ロール・部門IDをJWTに含めることで、Contextに注入可能 |
| go-chi/jwtauth | chiベースJWT Middleware | Ginでも類似のものあり。アクセストークンの検証に使用 |

## 6. DI / ポリシー分離（Strategy パターン用）

| ライブラリ | 用途 | 備考 |
|-----------|------|------|
| uber-go/fx | DIコンテナフレームワーク | ポリシーの切替・登録が容易になる |
| google/wire | 静的DI構成 | interfaceを使ったポリシー分離に親和性あり |

## 7. ロギング・監査

| ライブラリ | 用途 | 備考 |
|-----------|------|------|
| uber-go/zap | 高速な構造化ログ | 監査ログ（audit_logsテーブル）への書き込みにも応用可能 |
| sirupsen/logrus | 高機能ログ出力 | zapより柔らかいAPI。説明性やトレーサビリティ向上に便利 |

## 📌 組み合わせイメージ（構成例）

| 目的 | ライブラリ構成例 |
|------|-----------------|
| APIルーティング | Gin + swaggo/swag |
| 認証 | jwt-go + Gin middleware |
| 権限チェック | 自作Permission Matrix + PolicyResolver interface |
| DB操作 | gorm + golang-migrate |
| ログ記録 | zap + DB記録用サービス層 |
| テスト | stretchr/testify + table-driven tests |

