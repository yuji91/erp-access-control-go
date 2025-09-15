# ERPシステムの認可ライブラリ選定評価

ERPシステムの要件（Gin + GORM + JWT + RBAC構成）を踏まえて、以下に **認可ライブラリの選定評価** と **それぞれの適性** を一覧で整理します。

## 🔍 ERP権限管理要件 × Go認可ライブラリ対応表

| 要件 | Casbin | Oso | ORY Ladon | ORY Keto | gopolicy |
|------|--------|-----|-----------|----------|----------|
| ✅ 部門別アクセス制御 | ◎ Domainモデル可 | ◎ 属性で可能 | ○ Rule条件で可 | ◎ Namespaceで可 | △ 条件式で実装可能 |
| ✅ 機能別アクション制御 | ◎ Actionベース管理 | ◎ Polar関数で可 | ○ JSON action定義 | ◎ Rel構成で可 | △ 条件分岐で可能 |
| ✅ ロール階層構造 | ◎ GroupingPolicyあり | ○ 条件記述で可能 | △ 明示的定義なし | ◎ Relベースで可能 | ❌ |
| ✅ スコープ制御（データフィルタ） | △ 外部スコープロジック必要 | ◎ 属性 or 関数で記述可 | ○ context + JSONで可 | ◎ rel + subjectSetで可 | △ |
| ✅ 動的状態・条件制御（準ABAC） | △ matcher関数で一応可能 | ◎ 状態変数で記述可 | ○ 条件式次第 | ◎ | △ |
| ✅ 説明可能性（ログ・トレース） | ○（Enforcerログ + policy） | ◎（評価トレース） | △ | ◎（Check API） | △ |
| **学習コスト** | 中 | 高 | 低〜中 | 高 | 低 |
| **導入のしやすさ** | 中 | 中 | ○ | △ | ◎ |

## 🏁 推奨構成：Gin + GORM + JWT + RBAC + 準ABAC

### ✅ 最適候補：Casbin（RBAC＋ハイブリッド設計）

**理由：**

- Ginとのミドルウェア統合（gin-authz）が容易
- GORMのモデルと合わせてスコープ制御や階層構造を定義しやすい
- ABAC的条件は matcher や custom function で対応可能
- PolicyAdapter を使えばDB管理も可能

```go
// policy例（CSV/DB）
p, role, resource, action
g, alice, sales_manager
g, sales_manager, sales_team
```

### 💡 準代替候補：Oso

**利点：**

- 柔軟な条件記述（関数ベース、所属、ステータスなど）
- 「なぜ拒否されたか」を説明しやすい構造（トレースあり）
- クエリフィルタも定義可能（e.g. `oso.authorize_query(...)`）

**ただし：**

- Gin / GORMとの統合が自前での設計になる
- Polar構文に学習コストあり

### ❌ 非推奨 or 限定適用

| ライブラリ | コメント |
|------------|----------|
| **ORY Keto** | Zanzibar構成が大規模向けで、構築・運用が複雑。シンプルなERPにはオーバースペック |
| **ORY Ladon** | RBACや階層構造には弱く、ポリシー条件の表現力も限定的 |
| **gopolicy** | 軽量だが、階層ロールやスコープベース認可はコード依存になり、メンテナンスが難しい |

## 🛠️ 補足：Gin + GORM + Casbin連携例

### ✅ Middleware（JWT → Casbin Enforcer）

```go
router.Use(func(c *gin.Context) {
    sub := getUserRoleFromJWT(c) // 例: "sales_manager"
    obj := c.Request.URL.Path
    act := c.Request.Method

    if ok, _ := enforcer.Enforce(sub, obj, act); !ok {
        c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
        return
    }
    c.Next()
})
```

### ✅ GORM連携によるスコープ制御（データフィルタ）

```go
db.Where("department_id = ?", user.DepartmentID).Find(&customers)
```

このように、Casbinのポリシー判定と、**GORMのスコープ制御（フィルタリング）**を分離して実装することで、RBAC + 準ABACのハイブリッド制御が可能になります。

## ✅ 結論（今回のERP要件に対して）

| 種類 | ライブラリ | 採用判断 |
|------|------------|----------|
| 基本RBAC + 階層 + スコープ + 準ABAC対応 | **Casbin** | ✅ 採用推奨（Gin/GORM親和性高） |
| 条件記述とトレーサビリティを強化したい | **Oso** | ⚠️ 採用検討（高柔軟だが導入に工数） |
| シンプル構成で最低限のみ | **gopolicy** | ❌ ERP要件では不足 |

---

**必要であれば次は以下もご提案できます：**

- ✅ Casbinのモデル定義（.conf）とポリシーデータ設計
- ✅ Enforcer設定とJWT認可ミドルウェアの統合例
- ✅ GORMでの部門スコープ制御との分離設計
- ✅ ロール階層・業務状態・承認フローの実装パターン

ご希望あればお知らせください！

