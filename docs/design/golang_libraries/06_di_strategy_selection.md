# ERP向けアクセス制御API - DI/ポリシー分離ライブラリ選定

## 6. DI / ポリシー分離（Strategy パターン用）

| ライブラリ | 用途 | 備考 |
|-----------|------|------|
| uber-go/fx | DIコンテナフレームワーク | ポリシーの切替・登録が容易になる |
| google/wire | 静的DI構成 | interfaceを使ったポリシー分離に親和性あり |

**ではどちらが適切？**

ERPアクセス制御システムにおける「Policy Object パターン + Golang + Gin」構成では、柔軟性・開発効率・可読性の観点から `uber-go/fx` を推奨します。

## ✅ 結論：uber-go/fx の採用がより適切

### 🔍 比較表：fx vs wire

| 観点 | uber-go/fx | google/wire |
|------|------------|-------------|
| DI方式 | 実行時DI（フレームワーク型） | 静的DI（コード生成） |
| 柔軟なポリシー切替 | ◎ ランタイムで構成差し替え可能 | △ ビルド後に固定される |
| Ginとの統合 | ◎ 中央で *gin.Engine 注入・拡張可能 | △ 手動構成が多い |
| 導入・初期学習コスト | ◯ 低〜中 | △ 中〜高（コード生成＋理解が必要） |
| テスト容易性 | ◎ Module単位で依存注入しやすい | ◯ 構成が静的なため再生成が必要 |
| PolicyObject構成との相性 | ◎ ランタイムで差し替えや条件分岐に強い | ◯ 基本は一意のbindingに固定される |

## 🎯 ERPのような要件で fx を推す理由

- ✅ **PolicyResolver の差し替えや条件分岐**（部門別・ロール別など） に柔軟に対応可能  
  → `fx.Provide` による複数実装登録と `fx.Annotated` による名前付きバインドが可能

- ✅ **Gin + GORM + PolicyResolver の依存関係**を統一的に管理できる  
  → `fx.New(Module...)` でまとめて登録、起動順も自動制御

- ✅ **ミドルウェア / ハンドラへの依存注入**が自然に書ける  
  → `func(policy PolicyResolver) gin.HandlerFunc` のようにDIが直感的

## ✳️ wire を採用すべきケース

| 採用が適している状況 | 理由 |
|-------------------|------|
| 処理構成が完全に静的で、ポリシー切替が不要 | コンパイル時安全性が高く、余計なランタイム処理が不要 |
| AWS Lambda 等で軽量ビルドが必要 | fx による実行時処理がコストになるケース |
| 小規模な構成で、手動DIが負担にならない | 生成コードが明示的で、理解しやすい |

しかし、ERPのように「拡張性 / 柔軟性 / ポリシー差し替え」が求められる場合にはやや窮屈です。

## ✅ fx による構成イメージ（簡易例）

```go
// policy/sales.go
type SalesPolicy struct{}
func (s SalesPolicy) Can(...) (bool, string) { ... }

// di/policy_module.go
var Module = fx.Options(
  fx.Provide(
    fx.Annotated{Name: "sales", Target: func() PolicyResolver { return SalesPolicy{} }},
  ),
)

// handler/permission.go
func NewPermissionHandler(@Name("sales") policy PolicyResolver) gin.HandlerFunc {
  return func(c *gin.Context) {
    // policy.Can(...) を使う
  }
}
```

## ✅ 総まとめ

| 評価軸 | uber-go/fx | google/wire |
|--------|------------|-------------|
| 柔軟性（ポリシー切替） | ◎ | △ |
| Ginとの親和性 | ◎ | ◯ |
| 静的安全性 | ◯ | ◎ |
| 複雑な拡張性 | ◎ | △ |
| ERP構成への適合性 | ✅ 最適 | ⚠️ 限定的な採用向き |

## 🔚 補足

- `fx` はModuleごとのテストやMock切り替えも簡単です（`fx.Replace()` で可能）
- `wire` はビルド後の構成が変わらないユースケースに非常に強力

---

必要であれば `fx.Module` を使った Gin + PolicyObject DI 構成サンプルコードを提示できます。お気軽にお申し付けください。