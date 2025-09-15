# ERPアクセス制御API - 代表的なAPIと関数 一覧

## 概要
このドキュメントでは、ERPアクセス制御システムの中核となる代表的なAPIエンドポイントと関数を紹介します。  
各機能は認証・認可、ユーザー管理、権限管理、ロール管理、システム管理の5つのカテゴリに分類されています。

---

## 1. 🔐 認証・認可システム

### 1.1 ログイン認証 API
- **エンドポイント**: `POST /api/v1/auth/login`
- **ハンドラ**: `AuthHandler.Login()`
- **サービス**: `AuthService.Login(req LoginRequest) (*LoginResponse, error)`
- **概要**: ユーザー認証を行い、JWT トークンと権限情報を返却
- **特徴**: 
  - bcrypt によるパスワード検証
  - 複数ロール対応
  - アクティブロール・最高優先度ロール情報取得

### 1.2 JWT認証ミドルウェア
- **ミドルウェア**: `AuthMiddleware.Authentication()`
- **機能**: 
  - JWT トークン検証
  - トークン失効チェック
  - ユーザー情報のコンテキスト保存
- **場所**: `internal/middleware/auth.go:100-167`

### 1.3 権限チェックミドルウェア  
- **ミドルウェア**: `RequirePermissions(permissions ...string)`
- **機能**:
  - 指定された権限の保有確認
  - ワイルドカード権限対応 (`*:*`, `module:*`)
  - モジュール×アクション権限マトリックス
- **場所**: `internal/middleware/auth.go:169-196`

---

## 2. 👥 ユーザー管理システム

### 2.1 ユーザー作成 API
- **エンドポイント**: `POST /api/v1/users`
- **ハンドラ**: `UserHandler.CreateUser()`
- **サービス**: `UserService.CreateUser(req CreateUserRequest) (*UserResponse, error)`
- **概要**: 新規ユーザーの作成
- **特徴**:
  - メール重複チェック
  - bcrypt パスワードハッシュ化
  - 部署・ロール存在確認
- **場所**: `internal/services/user.go:87-146`

### 2.2 ユーザー一覧取得 API
- **エンドポイント**: `GET /api/v1/users`
- **権限要求**: `user:list`
- **概要**: ページネーション対応のユーザー一覧取得

### 2.3 ユーザーロール割り当て API
- **エンドポイント**: `POST /api/v1/users/roles`
- **サービス**: `UserRoleService.AssignRole(userID, roleID, validFrom, validTo, priority, assignedBy, reason)`
- **概要**: 期限付き・優先度付きロール割り当て
- **特徴**:
  - 重複チェック
  - 有効期間設定
  - 優先度管理
- **場所**: `internal/services/user_role.go:25-80`

---

## 3. 🔑 権限管理システム

### 3.1 権限マトリックス取得 API
- **エンドポイント**: `GET /api/v1/permissions/matrix`
- **サービス**: `PermissionService.GetPermissionMatrix() (*PermissionMatrixResponse, error)`
- **概要**: モジュール×アクション×ロールの3次元権限マトリックス
- **特徴**:
  - 動的マトリックス生成
  - ロール別権限可視化
  - 未使用権限数計算
- **場所**: `internal/services/permission.go:397-470`

### 3.2 ユーザー権限取得
- **サービス**: `PermissionService.GetUserPermissions(userID uuid.UUID) ([]string, error)`
- **概要**: ユーザーの総合権限リスト取得
- **特徴**:
  - 複数ロール権限集約
  - PermissionMatrix との統合
  - アクティブロール期限チェック
- **場所**: `internal/services/permission.go:984-1092`

### 3.3 権限作成（冪等性保証）API
- **エンドポイント**: `POST /api/v1/permissions/create-if-not-exists`
- **サービス**: `PermissionService.CreatePermissionIfNotExists(req CreatePermissionRequest) (*PermissionResponse, error)`
- **概要**: 存在しない場合のみ権限作成
- **特徴**:
  - UNIQUE制約対応
  - 重複回避
  - システム権限保護
- **場所**: `internal/services/permission.go:176`

---

## 4. 🏢 組織・ロール管理システム

### 4.1 部署階層構造取得 API
- **エンドポイント**: `GET /api/v1/departments/hierarchy`  
- **概要**: 部署の親子関係を含む階層構造取得

### 4.2 ロール階層構造取得 API
- **エンドポイント**: `GET /api/v1/roles/hierarchy`
- **概要**: ロールの継承関係を含む階層構造取得

### 4.3 ロール権限割り当て API
- **エンドポイント**: `PUT /api/v1/roles/{id}/permissions`
- **概要**: ロールに対する権限の一括割り当て・更新

---

## 5. 🏥 システム管理

### 5.1 ヘルスチェック API
- **エンドポイント**: `GET /health`
- **概要**: システム稼働状況確認
- **レスポンス**: サービス名、ステータス、タイムスタンプ、バージョン

### 5.2 API一覧表示 (Web UI)
- **エンドポイント**: `GET /`
- **概要**: 全APIエンドポイントの美しいHTML表示
- **特徴**:
  - カテゴリ別エンドポイント表示
  - HTTPメソッド別カラーリング
  - レスポンシブデザイン
- **場所**: `cmd/server/main.go:221-624`

---

## 6. 🔧 JWT・セキュリティ

### 6.1 JWT トークン生成
- **サービス**: `jwt.Service.GenerateToken(userID, email, permissions, primaryRoleID, activeRoles, highestRole)`
- **概要**: 複数ロール対応のJWTトークン生成
- **特徴**:
  - カスタムクレーム (ユーザー権限、アクティブロール)
  - JTI (JWT ID) によるトークン管理
  - 有効期限設定
- **場所**: `pkg/jwt/jwt.go:44-65`

### 6.2 トークン失効管理
- **サービス**: `TokenRevocationService.ValidateTokenStatus()`
- **概要**: ログアウト時のトークン無効化・検証

---

## 技術的特徴

### Permission Matrix システム
**場所**: `internal/services/permission.go:892-940`
- **システム管理者**: `*:*` (全権限)
- **部門管理者**: モジュール別詳細権限
- **開発者**: 読み取り・更新権限
- **一般ユーザー**: 基本読み取り権限

### エラーハンドリング
- **カスタムエラー型**: `pkg/errors/errors.go`
- **統一エラーレスポンス**: HTTP ステータス + 詳細メッセージ
- **エラーカテゴリ**: VALIDATION_ERROR, DATABASE_ERROR, AUTHORIZATION_ERROR

### 監査ログ
- **構造化ログ**: uber-go/zap
- **アクセス記録**: IP・User-Agent・操作理由付き
- **権限チェック履歴**: 成功・失敗の詳細ログ

---

## 統計情報

- **総APIエンドポイント数**: 40+
- **実装済みエンドポイント**: 28
- **権限保護エンドポイント**: 23
- **認証不要エンドポイント**: 5 (login, refresh, logout, health, version)
- **モジュール数**: 6 (user, department, role, permission, audit, auth)
- **サポートHTTPメソッド**: GET, POST, PUT, PATCH, DELETE

---

## 参考情報

- **作成日**: 2025-07-29
- **対象バージョン**: v0.1.0-dev  
- **OpenAPI仕様書**: [api/openapi.yaml](../../../api/openapi.yaml)
- **実装ファイル**: 
  - ハンドラ: `internal/handlers/`
  - サービス: `internal/services/`
  - ミドルウェア: `internal/middleware/`
  - モデル: `models/`
