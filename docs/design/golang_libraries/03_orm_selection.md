# ERP向けアクセス制御API - ORM・マイグレーション選定

ERPアクセス制御システムにおけるデータベース操作では、以下の構成が非常に適切です。

## ✅ 結論：gorm + golang-migrate/migrate の併用が最適

| ライブラリ | 役割 | 採用理由 |
|-----------|------|----------|
| gorm | ORM（構造体 ⇔ DB） | 型安全で柔軟なクエリ構築が可能。init_migration_erp_acl.sql の構造に対応したモデルが組める。 |
| golang-migrate/migrate | マイグレーション管理（SQL） | 明示的なバージョン管理と履歴の記録が可能。GORMのAutoMigrateに依存しない堅牢な運用が可能。 |

## 🔍 採用構成の補足

### gorm の利用ポイント
- `users`, `roles`, `permissions` などをGo構造体で定義し、GORMタグでDBスキーマと対応させる
- 関連構造（belongsTo, hasMany）にも対応
- ABAC・RBACのスコープ判断に必要な `user_scopes`, `approval_states` などの動的クエリも組みやすい

### golang-migrate の利用ポイント
- `init_migration_erp_acl.sql` を `0001_init_schema.up.sql` にして初期マイグレーションに使用
- 手動でDDLを定義することで、テーブル構造の制御と監査がしやすい
- マイグレーション履歴（schema_migrations）によって安全な移行管理が可能

## 🛠️ 推奨構成・運用フロー

### ディレクトリ例

```
.
├── migrations/
│   ├── 0001_init_schema.up.sql
│   └── 0001_init_schema.down.sql
├── models/
│   ├── user.go
│   ├── role.go
│   └── ...
├── internal/db/
│   └── connect.go
```

### マイグレーションコマンド（例）

```bash
# マイグレーション実行
migrate -path ./migrations -database "postgres://user:pass@localhost:5432/dbname?sslmode=disable" up

# ダウン
migrate -path ./migrations -database ... down 1
```

## ✨ なぜ併用が良いのか？

| 目的 | GORM | golang-migrate |
|------|------|----------------|
| クエリ構築 | ✅ | ❌（SQL手書き） |
| データ操作 | ✅ | ❌ |
| スキーマ更新 | ⚠️（AutoMigrateは暗黙的） | ✅（SQLで明示管理） |
| チーム開発 | ⚠️（変更の記録が薄い） | ✅（履歴で管理・共有が容易） |
| 本番運用 | ❌（AutoMigrateは危険） | ✅（バージョン管理で安全） |

→ GORMは実行時の操作、migrateは移行と履歴管理に特化。それぞれの強みを活かせます。

## ✅ まとめ

| 項目 | 推奨ライブラリ | 備考 |
|------|---------------|------|
| ORMフレームワーク | gorm | 型安全、構造体ベースの柔軟なクエリ生成 |
| マイグレーション管理 | golang-migrate/migrate | 手動SQLでの確実なマイグレーション管理 |
| 初期スキーマ連携 | ✅ init_migration_erp_acl.sql をそのまま .up.sql に変換可能 | スムーズに初期導入可能 |

ご希望であれば、gorm モデル定義（構造体）＋ golang-migrate 初期スクリプトの実装例も提示可能です。必要でしたらお申し付けください。

---

## Gin採用時の構成について

Ginを採用する場合でも、gorm + golang-migrate/migrate の併用方針はそのまま有効かつ推奨です。むしろGinとの組み合わせにより、疎結合・責務分離されたモジュール構成を実現しやすくなります。

### ✅ Gin採用時の構成：gorm + migrate の統合案

| レイヤ | 役割 | 備考 |
|--------|------|------|
| main.go | DB接続 + Ginルーター初期化 | DB接続をDI（依存注入）で handlers に渡す |
| models/ | GORMモデル | User, Role, Permission などを定義 |
| handlers/ | APIエンドポイント（Gin） | GET /me/permissions, POST /resources/... を担当 |
| services/ | ビジネスロジック・ポリシー評価 | DBクエリ + 権限判定（PolicyObject） |
| migrations/ | SQLファイル管理（golang-migrate） | init_migration_erp_acl.sql から作成可能 |

### 📂 ディレクトリ構成例（Gin + GORM + Migrate）

```
.
├── main.go
├── config/
│   └── config.go         # DB接続設定
├── db/
│   └── connect.go        # GORM + DB接続用
├── models/
│   ├── user.go
│   ├── permission.go
│   └── ...
├── handlers/
│   ├── permissions.go
│   └── resources.go
├── services/
│   ├── policy.go
│   └── matrix_loader.go
├── middleware/
│   └── auth.go
├── routes/
│   └── router.go         # Ginのルーティング設定
├── migrations/
│   └── 0001_init_schema.up.sql
├── go.mod
└── go.sum
```

### 🔧 Gin × GORM 統合ポイント（例）

```go
// db/connect.go
func ConnectDB() *gorm.DB {
  dsn := os.Getenv("DATABASE_URL")
  db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
  if err != nil {
    log.Fatalf("DB接続エラー: %v", err)
  }
  return db
}
```

```go
// main.go
func main() {
  db := db.ConnectDB()
  router := routes.SetupRouter(db)
  router.Run(":8080")
}
```

```go
// handlers/permissions.go
func GetUserPermissions(c *gin.Context) {
  db := c.MustGet("db").(*gorm.DB)
  userID := c.GetString("userID")

  // Permission取得処理
  var permissions []models.Permission
  db.Joins("JOIN role_permissions ...").
    Where("users.id = ?", userID).
    Find(&permissions)

  c.JSON(http.StatusOK, gin.H{"permissions": permissions})
}
```

### 🧪 マイグレーション運用例（SQL管理）

```bash
# 初回マイグレーション登録（init_migration_erp_acl.sqlを使う）
cp init_migration_erp_acl.sql migrations/0001_init_schema.up.sql

# golang-migrate コマンド
migrate -path ./migrations -database "$DATABASE_URL" up
```

GORMの `AutoMigrate()` は開発初期には便利ですが、本番環境では SQLベースの明示的な管理（migrate） が望まれます。

### ✅ Gin構成でもベストな理由

| 項目 | gorm + migrate | 理由（Gin構成において） |
|------|----------------|----------------------|
| DB操作 | gorm | ハンドラやサービス層からシンプルにクエリ実行 |
| マイグレーション | golang-migrate | SQLでの厳格な管理と履歴追跡が可能 |
| Gin連携 | OK | *gorm.DB を c.Set("db", db) で渡して使用可能 |
| テスト性 | 高 | DBだけモックしてビジネスロジックを独立してテスト可能 |

## ✨ 総まとめ：Gin構成での役割分担

| 層 | フレームワーク / ライブラリ | 備考 |
|----|---------------------------|------|
| ルーティング / API | gin-gonic/gin | 各エンドポイント定義（REST） |
| ドキュメント / UI | swaggo/swag | Swagger UI生成 |
| ORM | gorm | モデル定義 + クエリ構築 |
| マイグレーション | golang-migrate/migrate | 明示的なDDL制御、履歴追跡 |
| 権限評価 | 独自 PolicyResolver | RBAC/ABAC対応の判断ロジック |

必要であれば `models.User` や `0001_init_schema.up.sql` への変換例、Gin用の依存注入コード例もすぐご提供できます。ご希望があればどうぞ。


