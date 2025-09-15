# 認証・認可ライブラリの実装評価と導入検討

## 概要

現在のERPアクセス制御システムは **Permission Matrix + Policy Object** のハイブリッド構成で85%完了しています。本ドキュメントでは、検討された認証・認可ライブラリと現在の実装を比較し、追加導入の意義を評価します。

## 🔍 現在の実装状況分析

### ✅ 実装済み機能

| 機能 | 実装状況 | 詳細 |
|------|----------|------|
| **JWT認証** | ✅ 完了 | `pkg/jwt/` + `internal/middleware/auth.go` |
| **Permission Matrix** | ✅ 完了 | `internal/services/permission.go:899-942` |
| **RBAC基本機能** | ✅ 完了 | 複数ロール、ロール階層、継承 |
| **スコープ制御** | ✅ 完了 | `CheckPermissionWithScope()` + JSONB条件 |
| **ワイルドカード権限** | ✅ 完了 | `*:*`, `user:*`, `*:read` パターン |
| **動的権限集約** | ✅ 完了 | 複数ロールからの権限マージ |
| **トークン無効化** | ✅ 完了 | `TokenRevocationService` |
| **監査ログ** | ✅ 完了 | アクセス・権限チェック記録 |

### 🏗️ 現在のアーキテクチャの特徴

```go
// 自作実装の核心部分
PermissionMatrix = map[string][]Permission{
    "システム管理者": {"*:*"},
    "部門管理者": {NewPermission(ModuleUser, ActionCreate), ...},
    "開発者": {NewPermission(ModuleUser, ActionRead), ...},
    "一般ユーザー": {NewPermission(ModuleUser, ActionRead)},
}

// 動的スコープ制御
func (s *PermissionService) CheckPermissionWithScope(
    userID uuid.UUID, 
    requiredPermission string, 
    resourceScope map[string]interface{}
) (bool, error)
```

## 📊 ライブラリ別評価：現実装との比較

### 1. **Casbin** 

#### 🔄 既存コードとの共通点
- **Permission Matrix**: Casbinの `p, role, resource, action` と同じ概念
- **ロール階層**: Casbinの `g, user, role` と同様の実装済み
- **ワイルドカード**: Casbin同様 `*` パターン対応済み

#### 🆚 現実装との相違点
| 項目 | 現実装 | Casbin |
|------|--------|--------|
| **ポリシー記述** | Go構造体 + マップ | CSV/JSON/DB |
| **評価エンジン** | 自作 `hasPermission()` | 最適化済みエンジン |
| **動的ポリシー** | DB + JSONB | PolicyAdapter |
| **パフォーマンス** | 単純マップ検索 | インデックス最適化 |

#### 📈 導入意義評価

**✅ 既存コードリプレイス不要で導入可能**
- **段階的移行**: 現在のPermissionMatrixを保持しつつ、Enforcerを並行導入
- **ミドルウェア統合**: `gin-authz`で既存のJWTトークンをそのまま活用

```go
// 段階的導入例
func (m *AuthMiddleware) CasbinEnforcement() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 既存のJWT認証は保持
        userRoles := getCurrentUserRoles(c)
        
        // Casbinで追加チェック
        for _, role := range userRoles {
            if ok, _ := enforcer.Enforce(role, c.Request.URL.Path, c.Request.Method); ok {
                c.Next()
                return
            }
        }
        c.Abort()
    }
}
```

**期待効果：**
- ポリシー記述の標準化・明確化
- 条件付きアクセス制御の強化
- 管理UIでの動的ポリシー変更

---

### 2. **Oso**

#### 🔄 既存コードとの共通点
- **条件ベース制御**: 現在の `CheckPermissionWithScope()` と類似
- **トレーサビリティ**: 既存の監査ログと同じ目的

#### 🆚 現実装との相違点
| 項目 | 現実装 | Oso |
|------|--------|-----|
| **条件記述** | JSONB + Go関数 | Polar言語 |
| **説明性** | 基本ログ | 詳細トレース |
| **複雑な条件** | 手動実装 | 宣言的記述 |

#### 📈 導入意義評価

**⚠️ 既存コードリプレイス必要**
- **大規模変更**: PermissionMatrixとスコープ制御ロジックの再実装が必要
- **学習コスト**: Polar構文の習得が必要

**期待効果：**
- より柔軟な条件記述（現在のJSONB制約を超える）
- 権限拒否理由の詳細説明
- 複雑なビジネス条件の宣言的記述

---

### 3. **ORY Keto (Zanzibar)**

#### 🔄 既存コードとの共通点
- **関係ベース**: 現在のロール-ユーザー関係と概念的に類似

#### 🆚 現実装との相違点
| 項目 | 現実装 | ORY Keto |
|------|--------|----------|
| **スケール** | 中小規模向け | Google規模向け |
| **複雑性** | シンプル | 高複雑 |
| **運用コスト** | 低 | 高 |

#### 📈 導入意義評価

**❌ 現時点で導入意義低**
- **オーバーエンジニアリング**: 現在の要件に対して過剰
- **運用負荷**: インフラ・監視コストが大幅増加

---

### 4. **ORY Ladon & gopolicy**

#### 📈 導入意義評価

**❌ 既存実装が上位互換**
- **機能不足**: 現在の実装の方が高機能
- **後退**: 導入することで機能低下

## 🎯 導入推奨度評価

### ✅ **これから導入する意義があるライブラリ（既存コードリプレイス不要）**

#### 1. **Casbin** - 推奨度: ⭐⭐⭐⭐⭐

**導入理由：**
- 現在のPermissionMatrixを保持しつつ段階的導入可能
- ポリシー管理の標準化・可視化
- パフォーマンス向上（大量ユーザー対応）

**具体的導入パス：**
```go
// Phase 1: 並行実行
middleware1 := existingPermissionCheck()  // 既存
middleware2 := casbinEnforcement()        // 新規追加

// Phase 2: 段階的置換
middleware := hybridEnforcement()         // 両方チェック

// Phase 3: 完全移行
middleware := casbinEnforcement()         // Casbinのみ
```

---

### ⚠️ **これから導入する意義があるライブラリ（既存コードリプレイス必要）**

#### 1. **Oso** - 推奨度: ⭐⭐⭐

**導入条件：**
- より複雑なビジネス条件が必要になった場合
- 権限拒否理由の詳細説明が必須要件になった場合
- 現在のJSONBスコープ制御では表現できない条件が増えた場合

**リプレイス範囲：**
- `CheckPermissionWithScope()` 全体
- PermissionMatrix → Polar policy
- スコープ制御ロジック → 宣言的条件

---

### 🤔 **最初から導入しておけば良かったかもしれないライブラリ**

#### 1. **Casbin** - 後悔度: ⭐⭐⭐

**理由：**
- 現在の自作実装は本質的にCasbinの再実装
- 最初からCasbinを使っていれば、以下の工数削減が可能だった：
  - PermissionMatrix定義・保守 (約2週間)
  - hasPermission()ロジック実装 (約1週間)  
  - ワイルドカード処理 (約3日)
  - 階層ロール処理最適化 (約1週間)

**現在との違い：**
```go
// 現在の自作実装 (約150行)
func (s *PermissionService) hasPermission(userPermissions []string, requiredPermission string) bool {
    // 複雑な自作ロジック...
}

// Casbinだった場合 (約5行)
if ok, _ := enforcer.Enforce(user, resource, action); ok {
    return true
}
```

#### 2. **Oso** - 後悔度: ⭐⭐

**理由：**
- スコープ制御の柔軟性で先を見越していれば有用だった
- ただし学習コストと運用コストを考慮すると、現在の選択も妥当

## 📋 結論・推奨アクション

### 短期（3ヶ月以内）
1. **Casbin段階的導入** を検討
   - 既存コードを保持したまま並行テスト
   - ポリシー管理UIの検討

### 中期（6ヶ月〜1年）
1. **Casbin完全移行** の実施
   - パフォーマンス改善効果の測定
   - 管理コスト削減効果の検証

### 長期（1年以上）
1. **Oso導入検討** の条件監視
   - ビジネス条件の複雑化状況
   - 説明可能性要件の増加

### 現実装の継続改善
現在の実装は十分高品質であり、**無理にライブラリ導入する必要はない**。以下の改善で十分：

1. Permission Matrixの動的DB管理化
2. スコープ制御条件の可視化UI
3. 権限チェックパフォーマンス最適化

**総評**: 現在の実装は適切で、Casbin以外の急速な導入は不要。段階的改善が最適解。
