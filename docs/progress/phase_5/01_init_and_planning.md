# **Phase 5: ビジネスロジック実装計画** 

## 📋 **概要**

Phase 4で認証・認可の基盤が完成したため、Phase 5では実際のビジネスロジックとなるCRUD APIの実装を行います。

**🎯 現在の進捗**: Step 1（User管理API）とStep 2（Department管理API）が完了済み（40%完了）。
Step 3（Role管理API）の実装に向けて準備中です。

## 🎯 **目標・スコープ**

### **達成目標**
- **完全なRESTful API実装**: User/Department/Role/Permission の CRUD 操作
- **ビジネスルール実装**: データ検証、権限チェック、操作制限
- **統一されたエラーハンドリング**: Phase 4で実装した標準形式の活用
- **OpenAPI仕様書の更新**: 新しいエンドポイントの仕様書作成

### **対象範囲**
- ✅ **完成済**: 認証API (`AuthHandler`) - ログイン/ログアウト/プロフィール
- ✅ **完成済**: ユーザーロール管理API (`UserRoleHandler`) - 複数ロール操作
- ✅ **完成済**: User管理API - ユーザーCRUD操作 _(Step 1完了)_
- ✅ **完成済**: Department管理API - 部署CRUD操作・階層管理 _(Step 2完了)_
- 🔧 **実装対象**: Role管理API - ロールCRUD操作・階層管理・権限割り当て
- 🔧 **実装対象**: Permission管理API - 権限CRUD操作・権限マトリックス

### **対象外**
- 監査ログAPI（Phase 6で実装予定）
- 承認フローAPI（Phase 6で実装予定）
- 時間制限API（Phase 6で実装予定）
- ユーザースコープAPI（Phase 6で実装予定）

## 🗂️ **現在の実装状況**

### **✅ 完成済コンポーネント**

#### **モデル層**
- ✅ **全モデル定義完了**: User/Department/Role/Permission/UserRole
- ✅ **リレーション設定完了**: 外部キー・多対多関係・階層構造
- ✅ **カスタムメソッド実装済**: 階層取得・権限チェック・CRUD操作
- ✅ **バリデーション実装済**: BeforeCreate/BeforeUpdate フック

#### **サービス層**
- ✅ **AuthService**: 認証・認可サービス完成
- ✅ **UserRoleService**: 複数ロール管理サービス完成
- ✅ **PermissionService**: 権限チェックサービス完成

#### **ハンドラー層**
- ✅ **AuthHandler**: 認証APIハンドラー完成
- ✅ **UserRoleHandler**: ユーザーロール管理APIハンドラー完成

#### **インフラ層**
- ✅ **エラーハンドリング**: 標準化されたエラーレスポンス
- ✅ **ログ機能**: 構造化ログ・環境別設定
- ✅ **認証ミドルウェア**: JWT認証・権限チェック
- ✅ **データベース接続**: GORM・PostgreSQL接続

### **🔧 実装が必要なコンポーネント**

#### **サービス層（一部実装済）**
- ✅ **UserService**: ユーザーCRUD操作 _(Step 1完了)_
- ✅ **DepartmentService**: 部署CRUD操作・階層管理 _(Step 2完了)_
- ❌ **RoleService**: ロールCRUD操作・権限割り当て
- ❌ **PermissionService（拡張）**: 権限CRUD操作

#### **ハンドラー層（一部実装済）**
- ✅ **UserHandler**: ユーザー管理API _(Step 1完了)_
- ✅ **DepartmentHandler**: 部署管理API _(Step 2完了)_
- ❌ **RoleHandler**: ロール管理API
- ❌ **PermissionHandler**: 権限管理API

## 📋 **実装計画・Step分解**

### **Step 1: User管理API実装** 
**優先度**: 🔴 Critical | **工数**: 3-4日

#### **1.1 UserService実装** _(1日)_
- **ファイル**: `internal/services/user.go`
- **機能**:
  - `CreateUser()` - ユーザー作成（部署・プライマリロール設定）
  - `GetUser()` - ユーザー詳細取得（リレーション込み）
  - `UpdateUser()` - ユーザー更新（メール・名前・部署・ステータス）
  - `DeleteUser()` - ユーザー削除（ソフトデリート検討）
  - `GetUsers()` - ユーザー一覧取得（フィルタリング・ページング）
  - `ChangeUserStatus()` - ステータス変更（アクティブ・非アクティブ・停止）

#### **1.2 UserHandler実装** _(1日)_
- **ファイル**: `internal/handlers/user.go`
- **エンドポイント**:
  - `POST /api/v1/users` - ユーザー作成
  - `GET /api/v1/users` - ユーザー一覧
  - `GET /api/v1/users/:id` - ユーザー詳細
  - `PUT /api/v1/users/:id` - ユーザー更新
  - `DELETE /api/v1/users/:id` - ユーザー削除
  - `PUT /api/v1/users/:id/status` - ステータス変更

#### **1.3 バリデーション・権限チェック** _(1日)_
- **入力値検証**: メール形式・パスワード強度・必須項目
- **権限チェック**: ユーザー管理権限・所属部署制限・自己編集制限
- **ビジネスルール**: 部署移動制限・ステータス変更制限

#### **1.4 テスト実装** _(1日)_
- **単体テスト**: UserServiceの各メソッドテスト
- **統合テスト**: UserHandlerのAPIエンドポイントテスト
- **権限テスト**: 認証・認可チェックテスト

### **Step 2: Department管理API実装** ✅ **完了**
**優先度**: 🟡 High | **工数**: 2-3日

#### **2.1 DepartmentService実装** ✅ **完了** _(1日)_
- **ファイル**: `internal/services/department.go`
- **実装済の機能**:
  - ✅ `CreateDepartment()` - 部署作成（階層構造対応）
  - ✅ `GetDepartment()` - 部署詳細取得（親子関係込み）
  - ✅ `UpdateDepartment()` - 部署更新（名前・親部署変更）
  - ✅ `DeleteDepartment()` - 部署削除（子部署・所属ユーザーチェック）
  - ✅ `GetDepartments()` - 部署一覧取得（階層表示）
  - ✅ `GetDepartmentHierarchy()` - 部署階層ツリー取得

#### **2.2 DepartmentHandler実装** ✅ **完了** _(1日)_
- **ファイル**: `internal/handlers/department.go`
- **実装済のエンドポイント**:
  - ✅ `POST /api/v1/departments` - 部署作成
  - ✅ `GET /api/v1/departments` - 部署一覧・階層
  - ✅ `GET /api/v1/departments/:id` - 部署詳細
  - ✅ `PUT /api/v1/departments/:id` - 部署更新
  - ✅ `DELETE /api/v1/departments/:id` - 部署削除
  - ✅ `GET /api/v1/departments/hierarchy` - 階層ツリー

#### **2.3 階層管理・バリデーション** ✅ **完了** _(1日)_
- ✅ **階層制限**: 循環参照防止・最大階層深度制限（5階層）
- ✅ **削除制限**: 子部署存在時の削除禁止・所属ユーザー存在チェック
- ✅ **移動制限**: 部署移動時の循環参照・深度チェック
- ✅ **テスト実装**: 単体テスト26ケース、統合テスト22ケース（全て成功）
- ✅ **サーバー統合**: ルーティング設定・権限チェック完了

### **Step 3: Role管理API実装**
**優先度**: 🟡 High | **工数**: 3-4日

#### **3.1 RoleService実装** _(1-2日)_
- **ファイル**: `internal/services/role.go`
- **機能**:
  - `CreateRole()` - ロール作成（階層構造・権限設定）
  - `GetRole()` - ロール詳細取得（権限・階層込み）
  - `UpdateRole()` - ロール更新（名前・親ロール変更）
  - `DeleteRole()` - ロール削除（ユーザー割り当てチェック）
  - `GetRoles()` - ロール一覧取得（階層表示）
  - `AssignPermissions()` - ロールへの権限割り当て
  - `GetRolePermissions()` - ロール権限一覧取得

#### **3.2 RoleHandler実装** _(1日)_
- **ファイル**: `internal/handlers/role.go`
- **エンドポイント**:
  - `POST /api/v1/roles` - ロール作成
  - `GET /api/v1/roles` - ロール一覧・階層
  - `GET /api/v1/roles/:id` - ロール詳細
  - `PUT /api/v1/roles/:id` - ロール更新
  - `DELETE /api/v1/roles/:id` - ロール削除
  - `PUT /api/v1/roles/:id/permissions` - 権限割り当て
  - `GET /api/v1/roles/:id/permissions` - ロール権限一覧

#### **3.3 権限管理・バリデーション** _(1日)_
- **階層権限継承**: 親ロールからの権限継承
- **権限競合チェック**: 複数ロール間の権限競合解決
- **削除制限**: ユーザー割り当て済ロールの削除禁止

### **Step 4: Permission管理API実装**
**優先度**: 🟢 Medium | **工数**: 2-3日

#### **4.1 PermissionService拡張** _(1日)_
- **ファイル**: `internal/services/permission.go`（既存拡張）
- **追加機能**:
  - `CreatePermission()` - 権限作成
  - `GetPermission()` - 権限詳細取得
  - `UpdatePermission()` - 権限更新
  - `DeletePermission()` - 権限削除
  - `GetPermissions()` - 権限一覧取得
  - `GetPermissionMatrix()` - 権限マトリックス取得

#### **4.2 PermissionHandler実装** _(1日)_
- **ファイル**: `internal/handlers/permission.go`
- **エンドポイント**:
  - `POST /api/v1/permissions` - 権限作成
  - `GET /api/v1/permissions` - 権限一覧
  - `GET /api/v1/permissions/:id` - 権限詳細
  - `PUT /api/v1/permissions/:id` - 権限更新
  - `DELETE /api/v1/permissions/:id` - 権限削除
  - `GET /api/v1/permissions/matrix` - 権限マトリックス

#### **4.3 権限マトリックス・バリデーション** _(1日)_
- **Module・Action検証**: 有効なモジュール・アクション名
- **重複チェック**: 同一権限の重複作成防止
- **削除制限**: ロール割り当て済権限の削除制限

### **Step 5: 統合・最適化**
**優先度**: 🟢 Medium | **工数**: 2-3日

#### **5.1 API ルーティング統合** _(1日)_
- **ファイル**: `cmd/server/main.go`
- **追加**:
  - User管理エンドポイント登録
  - Department管理エンドポイント登録
  - Role管理エンドポイント登録
  - Permission管理エンドポイント登録

#### **5.2 OpenAPI仕様書更新** _(1日)_
- **ファイル**: `api/openapi.yaml`
- **追加**:
  - 新しいエンドポイントの仕様
  - リクエスト・レスポンス定義
  - エラーレスポンス定義

#### **5.3 統合テスト・パフォーマンステスト** _(1日)_
- **APIシナリオテスト**: 複数エンドポイント連携テスト
- **パフォーマンステスト**: レスポンス時間・同時リクエスト処理
- **負荷テスト**: 大量データでのCRUD操作性能

## ⚠️ **注意点・考慮事項**

### **データ整合性**
- **トランザクション管理**: 複数テーブル更新時の整合性保証
- **外部キー制約**: 削除時の参照整合性チェック
- **楽観的ロック**: 同時更新時の競合回避

### **セキュリティ**
- **権限チェック**: 各操作での適切な権限確認
- **入力値検証**: SQLインジェクション・XSS対策
- **ログ記録**: 重要操作の監査ログ記録

### **パフォーマンス**
- **N+1問題**: Preloadによる効率的なクエリ
- **ページング**: 大量データでのメモリ使用量制限
- **インデックス**: 検索クエリの最適化

### **ユーザビリティ**
- **エラーメッセージ**: 分かりやすいエラー情報
- **バリデーション**: フロントエンド向けの詳細バリデーション
- **レスポンス形式**: 一貫したJSON構造

## 🎯 **完了基準**

### **機能要件**
- [ ] User CRUD API（6エンドポイント）実装完了
- [ ] Department CRUD API（6エンドポイント）実装完了
- [ ] Role CRUD API（7エンドポイント）実装完了
- [ ] Permission CRUD API（6エンドポイント）実装完了
- [ ] 全エンドポイントでの認証・認可チェック

### **品質要件**
- [ ] 単体テストカバレッジ80%以上
- [ ] 統合テスト全エンドポイント実装
- [ ] APIレスポンス時間200ms以内
- [ ] OpenAPI仕様書完全更新

### **ドキュメント要件**
- [ ] API仕様書（OpenAPI）更新
- [ ] 実装ドキュメント作成
- [ ] テストケース一覧作成

## 📊 **工数見積もり**

| Step | 内容 | 工数 | 優先度 | 状況 |
|------|------|------|--------|------|
| Step 1 | User管理API | 3-4日 | 🔴 Critical | ✅ **完了** |
| Step 2 | Department管理API | 2-3日 | 🟡 High | ✅ **完了** |
| Step 3 | Role管理API | 3-4日 | 🟡 High | 🔧 **実装中** |
| Step 4 | Permission管理API | 2-3日 | 🟢 Medium | ⬜️ **未着手** |
| Step 5 | 統合・最適化 | 2-3日 | 🟢 Medium | ⬜️ **未着手** |
| **合計** | - | **12-17日** | - | **40%完了** |

**📅 予想期間**: ~~約 2.5-3.5週間~~ → **残り1.5-2週間** （Step 1-2完了済み）

## 🚀 **次のステップ**

1. ✅ ~~Step 1実装開始~~ → **Step 1完了**（User管理API）
2. ✅ ~~Step 2実装開始~~ → **Step 2完了**（Department管理API）
3. 🔧 **Step 3実装開始**（Role管理API実装）
4. **Step 4実装**（Permission管理API実装）
5. **Step 5統合・最適化**
6. **Phase 6準備**（監査ログ・セキュリティ強化）
