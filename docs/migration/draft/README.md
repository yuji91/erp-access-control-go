# 📋 ERP Access Control Migration Scripts

ERPアクセス制御システムのデータベースマイグレーションについて、以下の段階で改善した。

- [01.sql](draft_migration_erp_acl_01.sql) - 基本版
- [02.sql](draft_migration_erp_acl_02.sql) - 拡張版  
- [03.sql](draft_migration_erp_acl_03.sql) - 完全版

## 📊 主要な改善ポイント

| 改善分野 | 01.sql | 02.sql | 03.sql | 改善効果 |
|----------|--------|--------|--------|----------|
| **パフォーマンス** | インデックスなし | インデックスなし | 15個の最適化インデックス | 🚀 クエリ速度大幅向上 |
| **データ整合性** | 基本制約のみ | 基本制約のみ | 6個の詳細制約 | 🛡️ データ品質確保 |
| **機能拡張** | 基本テーブルのみ | 拡張テーブル追加 | 時間制限+トークン管理 | ⚡ より高度な制御 |
| **運用性** | 手動クエリのみ | 手動クエリのみ | ビュー+関数提供 | 🔧 開発・運用効率向上 |
| **階層構造** | 基本リレーション | 基本リレーション | 再帰CTE最適化 | 🌳 階層検索の高速化 |
| **監査機能** | 基本ログ | 詳細監査ログ | IP・UserAgent対応 | 🔍 完全なトレーサビリティ |

## 🔄 進化の流れ

```
01.sql (基本版)
    ↓ 要件拡張・詳細化
02.sql (機能拡張版)
    ↓ パフォーマンス・運用最適化
03.sql (完全版)
```

---

# 01.sql -> 02.sql の改善内容

## 🎯 **02.sqlで追加・改善された内容**

### 1. **拡張テーブルの追加** ✅
- **time_restrictions**: 時間ベース制御（営業時間制限）
  - 開始・終了時間制御
  - 曜日配列での制限
  - タイムゾーン対応
- **revoked_tokens**: JWTトークン無効化管理
  - JWT ID (JTI) による無効化
  - 期限切れ管理

### 2. **監査機能の強化** ✅
- **IPアドレス記録**: `ip_address INET`
- **ユーザーエージェント**: `user_agent TEXT`
- **理由コード**: `reason_code TEXT`
- **結果分類の詳細化**: SUCCESS/DENIED/ERROR

### 3. **スコープ管理の拡張** ✅
- **リソースID対応**: 特定リソースへのスコープ制御
- **JSONB構造検証**: スコープ値の構造チェック制約
- **複合スコープ**: `{"department_id": "dpt-001", "project": "prj-XYZ"}`

### 4. **承認フローの詳細化** ✅
- **多段階承認**: `step_order INT`での順序管理
- **リソース単位制御**: `resource_type`での分類
- **スコープ条件**: 部門・拠点などの条件分岐

### 5. **データ整合性の向上** ✅
- **CHECK制約の追加**:
  - ユーザーステータス: `('active','inactive','suspended')`
  - 監査結果: `('SUCCESS','DENIED','ERROR')`
  - スコープタイプ: `('department','region','project','location')`
- **自己参照防止**: departments, roles
- **JSONB構造検証**: `jsonb_typeof(scope_value) = 'object'`

### 6. **タイムスタンプの統一** ✅
- **created_at/updated_at**: 全テーブルで一貫した時刻管理
- **TIMESTAMPTZ**: タイムゾーン対応

---

# 02.sql -> 03.sql の改善内容

## 🎯 **03.sqlで追加・改善された内容**

### 1. **パフォーマンス最適化** ✅
- **15個の最適化インデックス**追加
- **GINインデックス**: JSONB検索の高速化
- **複合インデックス**: `(user_id, timestamp DESC)`等
- **部分インデックス**: `WHERE status != 'active'`等

### 2. **階層構造最適化** ✅
- **department_hierarchy ビュー**: 部門階層の効率的検索
- **role_hierarchy ビュー**: ロール階層の効率的検索  
- **user_permissions_view**: 権限統合表示
- **再帰CTE**: 循環参照防止付き階層クエリ

### 3. **便利関数の追加** ✅
- `get_user_all_permissions()`: 階層ロール考慮の権限取得
- `revoke_token()`: トークン無効化
- `cleanup_expired_tokens()`: 期限切れトークンクリーンアップ

### 4. **運用機能の強化** ✅
- **統計ビュー**: パフォーマンス最適化された集計
- **クリーンアップ機能**: 自動メンテナンス対応
- **初期データ例**: サンプル部門・ロール・権限

### 5. **実装サポート** ✅
```go
// 階層ロール権限の取得
permissions := db.Raw("SELECT * FROM get_user_all_permissions(?)", userID)

// JSONB スコープ検索
scopes := db.Where("scope_value @> ?", `{"department": "sales"}`).Find(&userScopes)

// 時間制限チェック
timeRestrictions := db.Where("user_id = ? AND ? = ANY(allowed_days)", 
    userID, time.Now().Weekday()).Find(&restrictions)
```

---

## 🚀 **最終的な達成機能**

### 📋 **Permission Matrix + Policy Object のハイブリッド構成**
- **静的・構造的な制御**はマトリクスで高速対応
- **動的・条件分岐の複雑な要件**は関数型のポリシーで柔軟に対応
- **拡張時**もStrategyパターンで切り出しやすく、Golangの型システムと相性が良い

### 🏗️ **本格的なERPシステム対応**
- **部門階層**: 再帰的な組織構造
- **ロール継承**: 階層ロールでの権限継承
- **スコープ制御**: 部門・地域・プロジェクト単位
- **時間制限**: 営業時間・曜日制限
- **多段階承認**: 複雑な承認フロー
- **完全監査**: IP・UserAgent・理由コード
- **JWTセッション管理**: トークン無効化
