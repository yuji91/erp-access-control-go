# ERP向けアクセス制御API - ロギング・監査ライブラリ選定

## 7. ロギング・監査

| ライブラリ | 用途 | 備考 |
|-----------|------|------|
| uber-go/zap | 高速な構造化ログ | 監査ログ（audit_logsテーブル）への書き込みにも応用可能 |
| sirupsen/logrus | 高機能ログ出力 | zapより柔らかいAPI。説明性やトレーサビリティ向上に便利 |

**ではどちらが適切？**

ERPアクセス制御システムにおける「ロギング + 監査性（Audit）」という要件に対しては、`uber-go/zap` を推奨します。とくに以下の理由から、監査ログテーブルへの記録や高パフォーマンスなAPI構成との親和性が高いためです。

## ✅ 結論：構造化・高速性・統一性の観点から zap を推奨

### 🔍 zap vs logrus 比較表

| 観点 | uber-go/zap | sirupsen/logrus |
|------|-------------|-----------------|
| ログ形式 | 構造化ログ（JSON推奨） | フィールド追加型ログ（やや柔らかめ） |
| パフォーマンス | ◎ 非常に高速（zero-allocation設計） | ◯ 標準的 |
| 構造化ログの精度 | ◎ 正確・明示的 | ◯ 柔軟だがやや曖昧さあり |
| 監査ログとの親和性 | ◎ JSON構造をそのままDBに保存しやすい | ◯ 必要に応じて整形 |
| 学習・導入コスト | 中（設計重視） | 低（感覚的に書ける） |
| 用途の適合性 | 🚀 高速・整形式で機械処理向け（監査・集約分析） | 🔎 人間向け出力として柔らかい（デバッグ向き） |

## ✳️ ERPシステム特有の要件との照合

| 要件 | zap 対応性 | 補足 |
|------|-----------|------|
| ❗️「なぜ拒否されたか」を構造化記録 | ✅ key-value構造で出力可 | `zap.String("reason", "NO_PERMISSION")` など |
| DB監査ログとの統一性（audit_logs） | ✅ ログ出力とDB構造が一致 | JSONログをそのまま保存・検索に応用可能 |
| 大規模ユーザー数に対応する性能 | ✅ 高速・GC負荷が低い | エンタープライズ向き |
| DevとOpsのロギング戦略統合 | ✅ Stackdriver, Loki などと統合しやすい | フィールド指定でフィルタしやすい |

## 🛠️ zap ログ出力例（監査用）

```go
logger.Info("Permission check failed",
    zap.String("user_id", "abc123"),
    zap.String("module", "inventory"),
    zap.String("action", "update"),
    zap.Bool("allowed", false),
    zap.String("reason", "NO_MATRIX_PERMISSION"),
)
```

→ そのままJSONとしてDBに保存可能：

```json
{
  "msg": "Permission check failed",
  "user_id": "abc123",
  "module": "inventory",
  "action": "update",
  "allowed": false,
  "reason": "NO_MATRIX_PERMISSION"
}
```

## ⚠️ logrus が向いている場合（参考）

| 条件 | コメント |
|------|----------|
| デバッグ用途が中心 | フィールド付き出力が直感的で読みやすい |
| 初学者でも扱いやすいログが必要 | `WithFields().Info()` などのAPIがシンプル |
| 小規模な構成、単純なログ要件 | 高速性より記述性を重視したいとき |

## ✅ まとめ

| 観点 | 推奨ライブラリ |
|------|---------------|
| 構造化・機械処理向けログ | ✅ uber-go/zap |
| パフォーマンス重視の本番システム | ✅ uber-go/zap |
| 柔軟なトレーシング / 説明性 | ◯ logrus も悪くはないが過剰になりやすい |

## 💡 補足：ログ → 監査テーブルへの連携

- `zapcore.Core` を拡張し、出力先に PostgreSQL INSERT を追加
- または `AuditLogger` をDIし、Policy評価 → DB保存を直接行う構成でもOK

必要であれば、zap による監査ログ → DB保存のインタフェース設計（例：AuditWriter）やログ→SQL変換の例も提供可能です。お気軽にお知らせください。






