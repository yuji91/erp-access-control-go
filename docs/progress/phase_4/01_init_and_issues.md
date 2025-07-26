# Phase 4: 認証・認可システム実装

## 📋 **実装概要**

Phase 4では、ERPアクセス制御システムの核となる認証・認可システムを実装しました。JWTベースの認証と、階層化された権限管理システムを構築し、セキュアなAPIアクセス制御を実現しています。

## 🏗️ **アーキテクチャ構成**

### 📁 **プロジェクト構造**
```
📁 プロジェクト構造
├── cmd/server/          # アプリケーションエントリポイント
├── internal/
│   ├── config/          # 設定管理 (Viper)
│   ├── middleware/      # JWT認証・権限チェック
│   └── services/        # ビジネスロジック
├── pkg/
│   ├── jwt/            # JWT認証サービス
│   ├── errors/         # 構造化エラー
│   └── logger/         # ログシステム
├── models/             # GORM データモデル
└── api/               # OpenAPI仕様
```

### 🛡️ **権限システム設計**

```go
// Permission Matrix 構造
Permission := Module + ":" + Action
例: "user:create", "department:read", "audit:list"

// 階層化権限
super_admin: ["*:*"]                    // 全権限
admin:       ["user:*", "department:*"] // モジュール別全権限
manager:     ["user:read", "user:update"] // アクション別制限
employee:    ["user:read"]              // 読み取りのみ
```

## 🔧 **実装詳細**

### 1. **データモデル設計**

#### **ユーザー管理**
- `User` モデル: 基本情報、部門、ロール、ステータス管理
- `Role` モデル: 権限マトリックス定義
- `Department` モデル: 組織階層管理
- `Permission` モデル: モジュール・アクション別権限定義

#### **権限・スコープ管理**
- `UserScope` モデル: ユーザー固有のスコープ制限
- `TimeRestriction` モデル: 時間帯別アクセス制御
- `AuditLog` モデル: 全操作の監査証跡
- `RevokedToken` モデル: トークン無効化管理

### 2. **認証システム**

#### **JWT認証サービス** (`pkg/jwt/`)
```go
// 主要機能
- GenerateToken: ユーザー用JWTトークン生成
- ValidateToken: トークン検証・解析
- RefreshToken: トークンリフレッシュ
- GetTokenID: JTI抽出（トークン追跡用）
```

#### **認証サービス** (`internal/services/auth.go`)
```go
// 認証フロー
- Login: ユーザー認証・トークン発行
- Logout: セッション終了・トークン無効化
- RefreshToken: トークン更新
- ChangePassword: パスワード変更
```

### 3. **認可システム**

#### **権限サービス** (`internal/services/permission.go`)
```go
// 権限チェック機能
- CheckPermission: 基本権限チェック
- CheckPermissionWithScope: スコープ付き権限チェック
- GetUserPermissions: ユーザー全権限取得
- ValidatePermission: 権限妥当性検証
```

#### **認可ミドルウェア** (`internal/middleware/auth.go`)
```go
// ミドルウェア機能
- AuthMiddleware: JWT認証
- RequirePermissions: 権限要求
- RequireAnyPermission: いずれかの権限要求
- RequireOwnership: 所有者権限要求
```

### 4. **トークン管理**

#### **トークン無効化サービス** (`internal/services/token_revocation.go`)
```go
// セキュリティ機能
- RevokeToken: 個別トークン無効化
- RevokeAllUserTokens: ユーザー全トークン無効化
- IsTokenRevoked: トークン無効化状態チェック
- CleanupExpiredTokens: 期限切れトークン削除
```

### 5. **エラーハンドリング**

#### **構造化エラー** (`pkg/errors/`)
```go
// エラー分類
- ValidationError: バリデーションエラー
- AuthenticationError: 認証エラー
- AuthorizationError: 認可エラー
- DatabaseError: データベースエラー
- InternalError: 内部サーバーエラー
```

### 6. **ログシステム**

#### **基本ログ** (`pkg/logger/`)
```go
// ログレベル
- Info: 情報ログ
- Error: エラーログ
- Warn: 警告ログ
- Debug: デバッグログ
```

## 🗄️ **データベース設計**

### **初期マイグレーション** (`migrations/init_migration_erp_acl.sql`)

#### **主要テーブル**
- `users`: ユーザー基本情報
- `roles`: ロール定義
- `permissions`: 権限定義
- `role_permissions`: ロール-権限関連
- `departments`: 部門情報
- `user_scopes`: ユーザースコープ
- `time_restrictions`: 時間制限
- `audit_logs`: 監査ログ
- `revoked_tokens`: 無効化トークン

#### **ストアドプロシージャ**
- `get_user_all_permissions()`: 階層的権限取得
- `check_user_permission()`: 権限チェック

#### **ビュー**
- `user_permissions_view`: 権限統合ビュー

## 🔐 **セキュリティ機能**

### **認証セキュリティ**
- JWT トークンベース認証
- トークン無効化機能
- セッション管理
- パスワードハッシュ化

### **認可セキュリティ**
- 階層化権限システム
- スコープベースアクセス制御
- 時間帯制限
- 所有者権限チェック

### **監査・追跡**
- 全操作の監査ログ
- トークン使用履歴
- セキュリティイベント記録

## 📊 **実装統計**

### **ファイル構成**
- **モデル**: 10ファイル
- **サービス**: 3ファイル
- **ミドルウェア**: 1ファイル
- **設定**: 1ファイル
- **パッケージ**: 3ファイル
- **マイグレーション**: 1ファイル

### **コード行数**
- **総行数**: 約1,500行
- **コメント**: 日本語統一済み
- **テスト**: 未実装（TODO）

## ⚠️ **制限事項・TODO**

### **現在の制限**
1. **単一ロール**: ユーザーは1つのロールのみ
2. **基本認証**: ユーザー名/パスワードのみ
3. **開発用ログ**: 本番環境には不適切
4. **テスト未実装**: 単体・統合テスト

### **今後の改善項目**

#### **セキュリティ強化**
```go
// TODO: JWT強化
- RSA公開鍵/秘密鍵方式への移行
- アクセストークン(短期) + リフレッシュトークン(長期)
- レート制限・ブルートフォース攻撃対策
- MFA (多要素認証) 対応

// TODO: パスワードポリシー
- 強度チェック (長さ、複雑性、辞書攻撃対策)
- bcrypt cost調整 (本番: 12-14)
- パスワード履歴管理
```

#### **アーキテクチャ拡張**
```go
// TODO: 複数ロール対応
type UserRole struct {
    UserID   uuid.UUID
    RoleID   uuid.UUID  
    ValidFrom *time.Time  // 期限付きロール
    ValidTo   *time.Time
}

// TODO: 階層的権限継承
- 部門長→課長→係長の自動権限継承
- 地理的・時間的制限
```

#### **パフォーマンス最適化**
```go
// TODO: キャッシュレイヤー
- Redis/Memcached による権限キャッシュ
- 階層的権限の事前計算
- N+1クエリ問題の解決

// TODO: 監査ログ強化
- 全API操作の詳細ログ
- セキュリティインシデント検知
- ELK Stack連携
```

## 🎯 **Phase 4 完了項目**

### ✅ **完了済み**
- [x] 基本データモデル設計・実装
- [x] JWT認証システム
- [x] 階層化権限管理
- [x] 認可ミドルウェア
- [x] トークン無効化機能
- [x] 構造化エラーハンドリング
- [x] 基本ログシステム
- [x] データベースマイグレーション
- [x] 日本語コメント統一

### 🔄 **進行中**
- [ ] APIエンドポイント実装
- [ ] OpenAPI仕様更新

### ⏳ **未着手**
- [ ] 単体テスト実装
- [ ] 統合テスト実装
- [ ] セキュリティ強化
- [ ] パフォーマンス最適化

## 📈 **次のフェーズ**

Phase 5では以下の項目に取り組みます：

1. **APIエンドポイント実装**
   - RESTful API設計
   - OpenAPI仕様完成
   - エンドポイントテスト

2. **テスト実装**
   - 単体テスト
   - 統合テスト
   - セキュリティテスト

3. **運用準備**
   - 本番環境設定
   - 監視・ログ設定
   - デプロイメント準備
