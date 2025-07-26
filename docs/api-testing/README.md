# 🧪 **API動作テスト - 複数ロール対応**

## 📋 **テスト概要**

複数ロール対応の認証・認可APIの動作確認手順です。

## 🚀 **前提条件**

1. **アプリケーション起動**
   ```bash
   make run  # または make docker-up-dev
   ```

2. **データベース準備**
   ```bash
   make docker-up
   make docker-migrate-sql
   ```

3. **Postman または curl 準備**

## 📊 **テストシナリオ**

### **Phase 1: 基本動作確認**

#### 1.1 ヘルスチェック
```bash
curl -X GET http://localhost:8080/health
```

**期待結果:**
```json
{
  "service": "erp-access-control-api",
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00Z",
  "version": "0.1.0-dev"
}
```

#### 1.2 バージョン情報
```bash
curl -X GET http://localhost:8080/version
```

### **Phase 2: 認証テスト（未実装）**

> **注意**: 現在、認証エンドポイントは未実装です。
> 実装が必要な項目：
> - `/api/v1/auth/login`
> - `/api/v1/auth/refresh`
> - `/api/v1/auth/logout`

#### 2.1 ログイン（実装後）
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin123"
  }'
```

**期待結果:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": "24h",
  "user": {
    "id": "uuid",
    "name": "Administrator",
    "email": "admin@example.com",
    "primary_role": {
      "id": "role-uuid",
      "name": "super_admin"
    },
    "active_roles": [
      {
        "id": "role-uuid",
        "name": "super_admin",
        "priority": 1,
        "valid_to": null
      }
    ]
  },
  "permissions": ["*:*"]
}
```

### **Phase 3: 複数ロール管理テスト（未実装）**

#### 3.1 ロール割り当て
```bash
curl -X POST http://localhost:8080/api/v1/users/roles \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-uuid",
    "role_id": "role-uuid",
    "priority": 2,
    "valid_from": "2024-01-01T00:00:00Z",
    "valid_to": "2024-12-31T23:59:59Z",
    "reason": "期限付きプロジェクトマネージャー権限"
  }'
```

#### 3.2 ユーザーロール一覧取得
```bash
curl -X GET "http://localhost:8080/api/v1/users/{user_id}/roles?active=true" \
  -H "Authorization: Bearer <token>"
```

#### 3.3 ロール更新
```bash
curl -X PATCH "http://localhost:8080/api/v1/users/{user_id}/roles/{role_id}" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "priority": 5,
    "reason": "権限優先度を最高レベルに変更"
  }'
```

#### 3.4 ロール取り消し
```bash
curl -X DELETE "http://localhost:8080/api/v1/users/{user_id}/roles/{role_id}" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "プロジェクト終了により権限取り消し"
  }'
```

## 🎯 **現在の実装状況**

### ✅ **実装済み**
- [x] 基本ヘルスチェック
- [x] JWT Claims 複数ロール対応
- [x] UserRole モデル
- [x] UserRoleService
- [x] UserRoleHandler
- [x] 認証ミドルウェア更新

### ❌ **未実装（要対応）**
- [ ] API ルーティング設定
- [ ] 認証エンドポイント実装
- [ ] テストデータ準備
- [ ] 実際のAPI動作確認

## 📋 **次のステップ**

1. **API ルーティング設定**
   - `cmd/server/main.go` にルーティング追加
   - 認証・ロール管理エンドポイント設定

2. **テストデータ準備**
   - データベースに初期データ投入
   - テスト用ユーザー・ロール作成

3. **Postman テスト実行**
   - 提供されたコレクションをインポート
   - シナリオベースでのテスト実行

## 🔧 **トラブルシューティング**

### よくある問題

1. **サーバー起動エラー**
   ```bash
   # ポート使用確認
   lsof -i :8080
   
   # Docker環境停止
   make docker-down
   ```

2. **データベース接続エラー**
   ```bash
   # Docker DB状態確認
   make docker-db-status
   
   # マイグレーション再実行
   make docker-migrate-reset
   ```

3. **ビルドエラー**
   ```bash
   # 依存関係更新
   go mod tidy
   
   # ビルドテスト
   go build ./...
   ```

## 📊 **テスト結果の記録**

テスト実行時は以下の情報を記録してください：

- テスト日時
- API レスポンス時間
- エラー内容（発生時）
- 期待動作との差異
- 改善提案

---

**🎯 目標**: 複数ロール機能の基本動作確認と API 正常性の検証 