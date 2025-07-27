# 🧪 **API動作テスト - 複数ロール対応**

## 📋 **テスト概要**

複数ロール対応の認証・認可APIの動作確認手順です。

## 🚀 **前提条件**

### **📦 推奨セットアップ（1コマンド完結）**
```bash
# 完全環境構築（Docker + マイグレーション + シードデータ）
make docker-up-dev

# 🎉 完了！APIテスト実行可能
make test-api
```

### **🔧 手動セットアップ**
```bash
# 1. Docker環境起動
make docker-up

# 2. マイグレーション実行
make docker-migrate-sql

# 3. シードデータ投入
make docker-seed

# 4. アプリケーション起動
make run
```

### **🛡️ 安全セットアップ（他プロジェクト保護）**
```bash
make setup-dev-clean  # ERPプロジェクトのみ対象
```

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

### **Phase 2: 認証テスト（実装済み）**

> **✅ 実装完了**: 認証エンドポイントが実装されました。
> 実装済み項目：
> - `/api/v1/auth/login` - ユーザーログイン
> - `/api/v1/auth/refresh` - トークンリフレッシュ
> - `/api/v1/auth/logout` - ユーザーログアウト
> - `/api/v1/auth/profile` - プロフィール取得
> - `/api/v1/auth/change-password` - パスワード変更

#### 2.1 ログイン
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

### **Phase 3: 複数ロール管理テスト（実装済み）**

> **✅ 実装完了**: 複数ロール管理エンドポイントが実装されました。
> 実装済み項目：
> - `POST /api/v1/users/roles` - ロール割り当て
> - `GET /api/v1/users/{user_id}/roles` - ユーザーロール一覧取得
> - `PATCH /api/v1/users/{user_id}/roles/{role_id}` - ロール更新
> - `DELETE /api/v1/users/{user_id}/roles/{role_id}` - ロール取り消し

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
- [x] 基本ヘルスチェック (`/health`, `/version`)
- [x] JWT Claims 複数ロール対応
- [x] UserRole モデル・サービス・ハンドラー
- [x] 認証ハンドラー (`internal/handlers/auth.go`)
- [x] 認証ミドルウェア更新
- [x] API ルーティング設定 (`cmd/server/main.go`)
- [x] 認証エンドポイント実装
- [x] 複数ロール管理エンドポイント実装
- [x] テストデータ準備 (`seeds/01_test_data.sql`)

### 🔄 **準備完了（テスト実行可能）**
- [x] データベースマイグレーション
- [x] 初期データ投入
- [x] サーバー起動設定
- [x] CORS設定

### ✅ **動作確認完了**
- [x] **実際のAPI動作確認** - `make test-api`で全エンドポイント動作確認済み
- [x] **エラーハンドリング確認** - 無効トークン・認証エラー処理確認済み
- [x] **複数ロール機能** - 4つのロール同時保持・優先度制御確認済み
- [x] **認証フロー** - ログイン・プロフィール取得・ロール管理確認済み

### ⚠️ **今後の改善項目**
- [ ] パフォーマンステスト（負荷測定）
- [ ] セキュリティテスト（侵入テスト）
- [ ] 国際化対応テスト
- [ ] 並行処理テスト

## 📋 **次のステップ**

1. **環境セットアップ**
   ```bash
   # データベース起動
   make docker-up
   
   # マイグレーション実行
   make docker-migrate-sql
   
   # テストデータ投入
   make docker-seed
   
   # サーバー起動
   make run
   ```

2. **基本動作確認**
   ```bash
   # ヘルスチェック
   curl -X GET http://localhost:8080/health
   
   # バージョン情報
   curl -X GET http://localhost:8080/version
   ```

3. **API動作確認スクリプト実行**
   ```bash
   # 全APIテスト実行
   make test-api
   
   # 基本動作確認のみ
   make test-api-quick
   
   # 手動でスクリプト実行
   ./scripts/api-test.sh
   ```

4. **認証テスト実行**
   - ログイン機能のテスト
   - トークンリフレッシュのテスト
   - プロフィール取得のテスト

5. **複数ロール管理テスト**
   - ロール割り当てのテスト
   - ロール一覧取得のテスト
   - ロール更新・取り消しのテスト

6. **Postman テスト実行**
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

---

## 📊 **実装完了状況サマリー**

| 項目 | 状況 | 詳細 |
|------|------|------|
| **認証システム** | ✅ 完了 | ログイン・リフレッシュ・ログアウト・プロフィール・パスワード変更 |
| **複数ロール管理** | ✅ 完了 | 割り当て・一覧取得・更新・取り消し |
| **API ルーティング** | ✅ 完了 | 全エンドポイント設定済み |
| **テストデータ** | ✅ 完了 | 部門・ロール・権限・ユーザーロール関連データ |
| **データベース** | ✅ 完了 | マイグレーション・シードデータ準備済み |
| **単体テスト** | ✅ 完了 | 44関数・70テストケース実装済み |

**🚀 次のアクション**: 実際のAPI動作確認と統合テストの実行

---

## 👥 **テストユーザー情報**

APIテストで利用可能なテストユーザー（全ユーザーのパスワード: `password123`）:

| ユーザー | メールアドレス | ロール | 権限 |
|----------|----------------|--------|------|
| **システム管理者** | `admin@example.com` | システム管理者 | 全権限 |
| **IT部門長** | `it-manager@example.com` | 部門管理者 | ユーザー・ロール・部門管理 |
| **人事部長** | `hr-manager@example.com` | 部門管理者 | ユーザー・ロール・部門管理 |
| **開発者A** | `developer-a@example.com` | 開発者 | 開発関連権限 |
| **開発者B** | `developer-b@example.com` | 開発者 | 開発関連権限 |
| **PM田中** | `pm-tanaka@example.com` | プロジェクトマネージャー | プロジェクト管理権限 |
| **一般ユーザーA** | `user-a@example.com` | 一般ユーザー | 基本権限 |
| **一般ユーザーB** | `user-b@example.com` | 一般ユーザー | 基本権限 |
| **ゲストユーザー** | `guest@example.com` | ゲストユーザー | 閲覧のみ |

**複数ロール対応**: 一部のユーザーには複数のロールが割り当てられており、期限付きロールや優先度付きロールのテストが可能です。 