# ERP向けアクセス制御API - RBAC/ACL実装補助ライブラリ選定

今回のERPアクセス制御システムでは「自作 Permission Matrix + Policy Object」の構成が最も適切です。casbin は柔軟すぎて今回の要件に対しては過剰設計かつ型の恩恵を受けづらくなる可能性があります。

## ✅ 結論：Golang × ERP × Policy Object構成では「自作 Permission Matrix」を推奨

### 🔍 理由と比較

| 観点 | 自作 Permission Matrix | casbin |
|------|----------------------|--------|
| Golangとの親和性 | ◎ map構造で型安全・IDE補完も効く | △ DSLベースで補完効きづらい |
| シンプルなRBAC要件への適合性 | ◎ 十分に対応可能 | ◎ 標準RBACモデルあり |
| 階層構造・ABAC的要素対応 | ◯ PolicyObjectと併用で対応可 | ◎ 条件付きABACに強い |
| DSL必要性 | ❌ 不要 | ✅ g, p ルールなどDSL定義が必要 |
| 柔軟性 | ◯ Goの関数で柔軟に設計可能 | ◎ 非常に高いがオーバーヘッドにもなり得る |
| 学習コスト・運用負荷 | 低 | 高（ルール設計、デバッグ、テストなど複雑） |
| ユニットテスト容易性 | ◎ 普通のGo関数として容易 | △ DSLやAdapterのMockが必要 |

## 🎯 今回のシステムに特有のポイント

- ✅ **部門別 × 機能別の複合権限** → Permission Matrix で静的に対応可能
- ✅ **スコープ制御・状態による動的制御** → Policy Object（関数ベース）で柔軟に対応
- ✅ **拒否理由・説明性が重要** → Goコードで明示的に書いた方が追跡・テストが容易
- ❗️ **DSLベースの外部定義は保守が重くなる**

## ✳️ casbin を使うべきケース

以下のような要件が主軸である場合に限り casbin の導入を検討できます：

| 適用場面 | 備考 |
|----------|------|
| 完全ABACが必要 | 属性 × 属性の複雑な条件式（例：sub.department == obj.department && obj.status == "OPEN"） |
| 外部管理者がルールをDSLで定義したい | ノーコードでポリシーを更新・反映したい運用 |
| 他言語との構成統一が必要 | Rust/Pythonなどと統一したRBACエンジンとして |

ただし、これらは今回のERP要件において必須ではなく、構成を重くするリスクの方が大きいです。

## ✅ 推奨構成：自作 Permission Matrix + Policy Object（Hybrid）

### Permission Matrix の実装例

```go
// permission_matrix.go
var PermissionMatrix = map[string]map[string][]string{
  "sales": {
    "inventory": {"view", "update"},
    "orders":    {"create", "approve"},
  },
  "hr": {
    "profile": {"view", "edit"},
  },
}
```

### Policy Resolver インターフェース

```go
// policy_resolver.go
type PolicyResolver interface {
    Can(user User, action string, resource Resource) (bool, string)
}
```

### 部門別 Policy 実装例

```go
// 部門・スコープ別にResolverを差し替え
type SalesPolicy struct{}
func (p SalesPolicy) Can(user User, action string, res Resource) (bool, string) {
    if user.Department != "sales" { return false, "NOT_SALES" }
    if action == "approve" && res.Status != "PENDING" { return false, "INVALID_STATE" }
    return true, "ALLOWED"
}
```

## 🔚 まとめ

| 項目 | 推奨 |
|------|------|
| ERP要件への適合性 | ✅ 自作 Permission Matrix + Policy Object |
| 型安全・構造化 | ✅ Goの型＋関数で定義可能 |
| シンプルさ | ✅ casbinより理解・保守・テストが容易 |
| 将来的な拡張 | ✅ PolicyResolverを差し替えて柔軟に対応可能 |

必要であれば、casbin を部分的に使った構成や、自作Permission Matrixの初期設計テンプレートも提示できます。お気軽にどうぞ。