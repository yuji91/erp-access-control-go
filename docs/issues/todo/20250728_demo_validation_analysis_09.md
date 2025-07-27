# デモ実行エラー調査結果 - 2025/07/28 分析09

## 概要
`make demo`実行時に発生した4つのエラーについて、コードベースを詳細に調査し、根本原因と修正方法を特定しました。

## 調査対象エラー

### 1. 権限作成エラー (inventory:read権限作成)
```json
{
  "code": "INTERNAL_ERROR",
  "message": "Internal server error",
  "details": {
    "reason": "DATABASE_ERROR: Database operation failed"
  }
}
```

### 2. ユーザー作成エラー (VALIDATION_ERROR)
```json
{
  "code": "VALIDATION_ERROR",
  "message": "Validation failed",
  "details": {
    "field": "request",
    "reason": "Invalid request format"
  }
}
```

### 3. ヘルスチェック・バージョン情報エラー (404)
```
404 page not found
```

### 4. 権限一覧取得の誤ったエラー判定
正常なレスポンスが[ERROR]として処理されている問題

## 詳細調査結果

### エラー1: 権限作成エラー
**調査ファイル**: `internal/services/permission.go`

**原因分析**:
- `inventory`モジュールは有効なモジュールとして定義済み (`ModuleInventory`)
- `read`アクションも有効なアクションとして定義済み (`ActionRead`)  
- `isValidModuleActionCombination`で基本CRUD操作が許可されているため、`inventory:read`は理論的には有効
- エラーはデータベース層で発生している可能性が高い
- 重複チェック時の`findPermissionByModuleAction`でのデータベースエラーが考えられる

**根本原因**: データベース接続またはテーブル構造の問題、または既存データとの整合性エラー

### エラー2: ユーザー作成エラー ⭐ **主要問題**
**調査ファイル**: 
- `scripts/demo-permission-system-final.sh:456-460`
- `internal/services/user.go:29-36`

**原因分析**:
デモスクリプトが送信するリクエストボディ:
```json
{
  "username": "demo_manager_082201",          // ❌ 間違ったフィールド名
  "email": "demo_manager_082201@example.com",
  "password": "password123", 
  "department_id": "6c8145a5-81ed-4d39-bbd6-06915e87b0fa"
  // ❌ primary_role_id フィールドが欠如
}
```

期待される`CreateUserRequest`構造体:
```go
type CreateUserRequest struct {
    Name          string    `json:"name" binding:"required,min=1,max=100"`           // ⚠️ "username"ではなく"name"
    Email         string    `json:"email" binding:"required,email,max=255"`
    Password      string    `json:"password" binding:"required,min=6,max=255"`
    DepartmentID  uuid.UUID `json:"department_id" binding:"required"`
    PrimaryRoleID uuid.UUID `json:"primary_role_id" binding:"required"`              // ⚠️ 必須フィールドが欠如
    Status        string    `json:"status" binding:"omitempty,oneof=active inactive suspended"`
}
```

**根本原因**: 
1. フィールド名の不一致 (`username` → `name`)
2. 必須フィールド`primary_role_id`の欠如

### エラー3: ヘルスチェック・バージョン情報エラー
**調査ファイル**: 
- `scripts/demo-permission-system-final.sh:485-489`
- `cmd/server/main.go:200-220`

**原因分析**:
デモスクリプトが呼び出すURL:
```bash
# デモスクリプト (487行目)
safe_api_call "GET" "../health" "" "ヘルスチェック"      # → /api/v1/../health
safe_api_call "GET" "../version" "" "バージョン情報"     # → /api/v1/../version
```

実際のエンドポイント:
```go
// main.go (200, 208行目)
router.GET("/health", ...)    // 正しいパス: /health
router.GET("/version", ...)   // 正しいパス: /version
```

**根本原因**: パス解決の問題。`/api/v1/../health`は`/health`ではなく、ルーティングで404になる

### エラー4: 権限一覧取得の誤ったエラー判定
**調査ファイル**: `scripts/demo-permission-system-final.sh:134-157`

**原因分析**:
- APIレスポンス自体は正常 (HTTPステータス200、有効なJSONレスポンス)
- デモスクリプトの`show_response`関数が正常レスポンスを[ERROR]として判定
- `c.Errors`に何かが設定されている可能性

## 修正方法

### 修正1: ユーザー作成リクエストの修正
**ファイル**: `scripts/demo-permission-system-final.sh:456-465`

**Before**:
```bash
-d "{
    \"username\": \"demo_manager_${TIMESTAMP}\",
    \"email\": \"demo_manager_${TIMESTAMP}@example.com\",
    \"password\": \"password123\",
    \"department_id\": \"$DEPT_SALES_ID\"
}"
```

**After**:
```bash
-d "{
    \"name\": \"demo_manager_${TIMESTAMP}\",
    \"email\": \"demo_manager_${TIMESTAMP}@example.com\",
    \"password\": \"password123\",
    \"department_id\": \"$DEPT_SALES_ID\",
    \"primary_role_id\": \"$MANAGER_ROLE_ID\"
}"
```

### 修正2: ヘルスチェックパスの修正
**ファイル**: `scripts/demo-permission-system-final.sh:485-489`

**Before**:
```bash
safe_api_call "GET" "../health" "" "ヘルスチェック"
safe_api_call "GET" "../version" "" "バージョン情報"
```

**After**:
```bash
# API_BASE変数を使わず直接呼び出し
curl -s -X GET "http://localhost:8080/health"
curl -s -X GET "http://localhost:8080/version"
```

### 修正3: 権限作成の調査継続
データベースエラーの詳細ログを確認し、以下を調査:
1. データベース接続状態
2. `permissions`テーブルの制約違反
3. 既存データとの重複状況

### 修正4: エラー判定ロジックの改善
`show_response`関数のエラー判定ロジックを見直し、HTTPステータスコードも考慮する

## 緊急度

1. **高**: ユーザー作成エラー（フィールド名・必須フィールド不一致）
2. **中**: ヘルスチェックパスエラー（機能的な問題）
3. **中**: 権限作成エラー（システムの安定性に影響）
4. **低**: 権限一覧エラー判定（表示上の問題）

## 次のアクション

1. ユーザー作成リクエストの修正を最優先で実施
2. ヘルスチェックパスの修正
3. 権限作成エラーのデータベースログ詳細調査
4. デモスクリプト全体のエラーハンドリング改善

## 参考情報

- 調査時刻: 2025-07-28 08:22:01
- エラー発生環境: make demo実行時
- 影響範囲: デモシステム全体の動作
- 調査方法: コードベース静的解析 + ログ分析
