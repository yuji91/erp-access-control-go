# ERP向けアクセス制御API - 追加検討ライブラリ

## 🔍 補完すべきライブラリ候補（用途別）

### 1. Validation強化（構造体レベルの入力検証）

**go-playground/validator** 🔧

Ginとの連携実績多数。OpenAPIコメントでは表現できない細かいルール（例：スコープ値の正規表現制約）も対応可能。

特に `/resources/{type}/{id}/actions/{action}` のように動的な入力を伴うAPIでは必須。

### 2. 構成管理 / 設定ファイル

**spf13/viper** 🔧

権限制御のポリシー切替や監査設定（有効/無効）などを `.yaml` や `.env` で外部管理する場合に便利。

### 3. テスト補助 / テーブル駆動テスト

**stretchr/testify** ✅

PolicyResolverの各ケースに対し、テーブル駆動テストを用いた単体検証が容易。

`require`, `assert` 等を使って説明性のあるテストが書ける。

### 4. 依存注入のMocking

**uber-go/mock or golang/mock**

`fx` を導入しているので、PolicyResolverなどのインターフェース注入時にモックテストを追加する場合はほぼ必須。

## 💡 （オプショナル）補助構成アイデア

| 分類 | 補助ライブラリ or 手段 | 用途 |
|------|---------------------|------|
| 🌐 RESTテスト | resty or httpmock | 外部APIとの連携テスト or POST /actions シミュレーション |
| 📄 Swagger Viewer | swagger-ui or redoc | openapi.yaml をWebで可視化する際のビューワ |
| 📊 権限マトリクス可視化 | React + Mermaid.js | 管理者用UIでのマトリクス表示補助（別プロジェクト） |

## ✅ 結論：追加を検討すべき最小セット

以下3つを追加すれば、現構成をより補完できます：

- **go-playground/validator** – 動的入力の型安全バリデーション
- **spf13/viper** – 権限・ログ設定の外部管理  
- **stretchr/testify** – Policy系のテスト強化

特にバリデーションは認可判定に直結するため、セキュリティと説明性の両立の観点で早期導入を推奨します。

必要に応じて、それぞれのライブラリ設定例や `fx` 統合例も提示できます。どこから進めるかご希望があればお知らせください。

---

## 🛠️ go-playground/validator 詳細評価

**Validation強化（構造体レベルの入力検証）用ライブラリ**

| 評価軸 | 理由・補足 |
|-------|-----------|
| 🔌 Ginとの統合性 | Gin公式が `binding:"required"` や `validate:"email"` など `validator.v9/10` をデフォルト統合。タグベースで `BindJSON` → `Validate` の流れがスムーズ。 |
| 📦 GORMとの整合性 | モデル定義が構造体中心のため、`gorm:""` と `validate:""` を共存させやすい。動的入力（JSON）にもマッチ。 |
| 🛠️ OpenAPIとの親和性 | コメントに記載する `example`, `required`, `enum` などと `validate:""` タグが意味的に対応しており、テスト時の整合が取りやすい。 |
| ✅ Policy Objectとの相性 | `PolicyResolver` に渡す前の入力チェックに使える。入力が不正なら `403` でなく `400 BadRequest` に分類でき、責務が明確になる。 |
| 🔍 フィールド条件バリデーション | `scope_type in ["department", "region"]` や `action must match /^[a-z_]+$/` など、**正規表現・条件分岐も含めて柔軟に記述可能**。 |
| 🧪 テスト性 | カスタムバリデータを実装・テスト可能（例：`IsDepartmentExists` のようなDB連携検証バリデータ）。Gin + fx でも容易に組み込める。 |

### ⚙️ 実装例（Ginとの連携）

```go
type ActionRequest struct {
    Status     string `json:"status" validate:"required,oneof=PENDING APPROVED REJECTED"`
    Department string `json:"department" validate:"required,alpha"`
    ScopeType  string `json:"scope_type" validate:"required,oneof=department region project"`
    ScopeValue string `json:"scope_value" validate:"required"`
}
```

Ginでは以下のように自然に連携：

```go
func (h *Handler) HandleAction(c *gin.Context) {
    var req ActionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := validator.New().Struct(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"validation_error": err.Error()})
        return
    }

    // → PolicyResolver に渡して認可判定へ
}
```

### ✅ 結論：導入すべき

| 項目 | 判定 |
|------|------|
| Ginとの統合 | ✅ 標準サポート済み |
| 動的入力の制約検証 | ✅ 正規表現・enum対応 |
| 権限判定との役割分離 | ✅ 400 vs 403 の責務分離 |
| カスタムバリデーション | ✅ fx + DIで容易に拡張可能 |
| ドキュメント連携（OpenAPI） | ✅ `example`, `required`, `enum` と整合性 |

→ go-playground/validator は、Gin + GORM + OpenAPI の構成に非常に相性が良く、現構成における動的バリデーションの最有力候補です。

---

## 🛠️ spf13/viper 詳細評価

**構成管理 / 設定ファイル管理ライブラリ**

| 評価軸 | 理由・補足 |
|-------|-----------|
| 🔌 Ginとの統合性 | Gin自身は設定機能を持たないが、`viper`は `config.yaml` や `.env` から読み込んだ設定をDIで渡す形で自然に統合可能。`fx.Provide(func(cfg Config) …)` 構成にも対応。 |
| 📦 GORMとの整合性 | DB接続情報（DSN）やタイムアウト設定、AutoMigrateの有効/無効などを `config.yaml` に分離することで、**GORM関連設定の柔軟性とテスト容易性が向上**。 |
| 🛠️ OpenAPIとの親和性 | OpenAPIの仕様やエンドポイントとは独立だが、**`enable_audit_logging: true` のようなフラグ管理**を環境・フェーズごとに柔軟に切り替えるのに最適。 |
| ✅ Policy Objectとの相性 | 権限の厳格度（例：柔軟モード vs 厳格モード）や、`PolicyResolver` 内部の閾値（例：承認可能金額上限）などを設定ファイルから注入可能。ポリシーを動的に切り替える柔軟性あり。 |
| 🔍 複数環境切替（dev/stg/prod） | `config.{env}.yaml` や `--env=production` のように切り替えることで、**DB・ログ・認可レベルなどの設定を環境単位で管理**できる（CI/CDやマルチテナントにも有用）。 |
| 🧪 テスト性 | `viper.Set()` によりテスト用設定を即時注入可能。`fx` と組み合わせることで、**本番設定とテスト設定の分離が容易**。モックサーバやMockPolicyとの組み合わせにも最適。 |

### ⚙️ 実装例（Viper + fx + Gin）

```go
type Config struct {
    Port                int    `mapstructure:"port"`
    DatabaseURL         string `mapstructure:"database_url"`
    EnableAuditLogging  bool   `mapstructure:"enable_audit_logging"`
    JWTSecret           string `mapstructure:"jwt_secret"`
}

func LoadConfig() (Config, error) {
    var config Config
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AutomaticEnv()

    if err := viper.ReadInConfig(); err != nil {
        return config, err
    }
    err := viper.Unmarshal(&config)
    return config, err
}
```

fx に組み込む：

```go
fx.Provide(func() (Config, error) {
    return LoadConfig()
}),
```

ハンドラやサービス層で注入して活用可能：

```go
func NewAuditService(cfg Config, db *gorm.DB) *AuditService {
    if !cfg.EnableAuditLogging {
        return NewNoopAuditService()
    }
    return NewPostgresAuditService(db)
}
```

### ✅ 結論：導入すべき

| 項目 | 判定 |
|------|------|
| Ginとの統合 | ✅ `fx`経由で自然に統合可能 |
| GORM/Policyとの整合 | ✅ 接続設定・閾値・監査モードの管理に有用 |
| 多環境切替（dev/stg/prod） | ✅ `config.{env}.yaml` 形式が柔軟 |
| 設定のテスタビリティ | ✅ `viper.Set()` によるMock注入が容易 |
| 認可構成の柔軟性（Policy連携） | ✅ モード・閾値の設定をコードから分離可能 |

→ ERPのように「監査ログ有無」や「権限制御の厳格度」を設定ファイルから切り替えたいユースケースが多い構成において、spf13/viper は非常に高い適合度を誇ります。

---

## 🧪 stretchr/testify 詳細評価

**テスト補助 / テーブル駆動テストライブラリ**

| 評価軸 | 理由・補足 |
|-------|-----------|
| 🧩 Gin/Handler層との統合 | ハンドラ単体テストで `httptest.NewRecorder()` と組み合わせて、**レスポンスステータスやJSONボディの検証**が直感的。`assert.Equal(t, 200, res.Code)` など。 |
| 🧪 Policy層との整合性 | `PolicyResolver.CanApprove(...)` のようなメソッドの戻り値と拒否理由の確認に最適。**入力 x 出力のテーブル駆動テスト**が容易。 |
| 📦 GORMとの組み合わせ | DB操作後の検証に `assert.NoError(t, err)` や `assert.Equal(t, wantUser, gotUser)` で**失敗理由の明確化がしやすい**。GORMの挙動を確認する回帰テストにも有効。 |
| 🧪 テーブル駆動テストとの相性 | Golang標準の `for _, tt := range tests` に `assert` を組み込むだけで、**意図の明確なケース別テストが書ける**。Policy, Matrix判定に特に有効。 |
| ✅ Auditログ/バリデーション | JSON出力に含まれる `reasonCode`, `allowed` などを `assert.JSONEq` や `assert.Contains` で検証でき、**API契約のテストにも使える**。 |
| ⚙️ CIとの連携性 | エラー時に詳細出力されるため、GitHub ActionsやGitLab CIでの**ログ追跡性・デバッグ性が高い**。テスト落ちの原因が明示的。 |

### ⚙️ 実装例（テーブル駆動 + Policyテスト）

```go
func TestApprovalPolicy_CanApprove(t *testing.T) {
    policy := NewApprovalPolicy()
    tests := []struct {
        name      string
        input     Application
        expected  bool
        reason    string
    }{
        {
            name: "一次承認者が承認中申請を処理可能",
            input: Application{Status: "PENDING", ApproverRole: "supervisor"},
            expected: true,
            reason: "APPROVER_MATCHED",
        },
        {
            name: "営業が経理の申請を承認できない",
            input: Application{Status: "PENDING", ApproverRole: "sales"},
            expected: false,
            reason: "ROLE_NOT_ALLOWED",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            allowed, reason := policy.CanApprove(tt.input)
            assert.Equal(t, tt.expected, allowed)
            assert.Equal(t, tt.reason, reason)
        })
    }
}
```

### ✅ 結論：導入すべき

| 項目 | 判定 |
|------|------|
| Gin/ハンドラテスト支援 | ✅ 標準的で豊富な実績あり |
| Policy / Matrix テスト適合性 | ✅ テーブル駆動と高相性 |
| バリデーション / レスポンス検証 | ✅ JSON, status, error の検証が柔軟 |
| CI/CDとの親和性 | ✅ エラー詳細が明示的、ログ追跡性も高い |
| 拡張性（testify/suiteの利用など） | ✅ モック・セットアップも容易に統合可能 |

→ ERPのように「権限制御のロジックが複雑で、分岐ケースが多い」システムにおいて、Policy判定やバリデーションの網羅的なテストを書くには stretchr/testify が最も適しているライブラリです。
