# 🔧 **Phase 5 Step 3: Role管理API実装** - 進捗レポート

## 📋 **概要**

Role管理APIの実装（Step 3）を開始します。RoleServiceとRoleHandlerの実装により、ロールの階層構造と権限割り当てを含む完全なCRUD操作を実現します。

## ⬜️ **実装予定項目**

### **3.1 RoleService実装** ✅ **完了**
- **ファイル**: `internal/services/role.go`
- **実装済の機能**:
  - ✅ `CreateRole()` - ロール作成（階層構造・権限設定対応）
  - ✅ `GetRole()` - ロール詳細取得（権限・階層込み）
  - ✅ `UpdateRole()` - ロール更新（名前・親ロール変更）
  - ✅ `DeleteRole()` - ロール削除（ユーザー割り当てチェック）
  - ✅ `GetRoles()` - ロール一覧取得（階層表示）
  - ✅ `AssignPermissions()` - ロールへの権限割り当て
  - ✅ `GetRolePermissions()` - ロール権限一覧取得
  - ✅ `GetRoleHierarchy()` - ロール階層ツリー取得

### **3.2 RoleHandler実装** ✅ **完了**
- **ファイル**: `internal/handlers/role.go`
- **実装済のエンドポイント**:
  - ✅ `POST /api/v1/roles` - ロール作成
  - ✅ `GET /api/v1/roles` - ロール一覧・階層
  - ✅ `GET /api/v1/roles/:id` - ロール詳細
  - ✅ `PUT /api/v1/roles/:id` - ロール更新
  - ✅ `DELETE /api/v1/roles/:id` - ロール削除
  - ✅ `PUT /api/v1/roles/:id/permissions` - 権限割り当て
  - ✅ `GET /api/v1/roles/:id/permissions` - ロール権限一覧
  - ✅ `GET /api/v1/roles/hierarchy` - 階層ツリー取得

### **3.3 階層管理・権限継承** ⬜️ **未着手**
- **ファイル**: `internal/services/role.go`
- **実装予定機能**:
  - ⬜️ **階層制限**: 循環参照防止・最大階層深度制限（5階層）
  - ⬜️ **削除制限**: ユーザー割り当て済みロールの削除禁止
  - ⬜️ **権限継承**: 親ロールからの権限継承システム
  - ⬜️ **権限競合解決**: 複数ロール間の権限競合処理

## 🔧 **実装計画詳細**

### **バリデーション・セキュリティ機能**

#### **入力値検証**
- ✅ **ロール名**: `binding:"required,min=2,max=100"`
- ✅ **親ロールID**: `binding:"omitempty,uuid"`
- ✅ **権限ID配列**: `binding:"omitempty,dive,uuid"`
- ✅ **階層深度**: 最大5階層までの制限

#### **ビジネスルール**
- ✅ **循環参照チェック**: 親子関係の循環を防止
- ✅ **階層深度チェック**: 最大深度を超える構造を防止
- ✅ **削除前チェック**: ユーザー割り当て・子ロールの存在確認
- ✅ **ロール名重複**: ロール名の一意性確保
- ✅ **権限存在確認**: 割り当て権限IDの存在確認

#### **セキュリティ**
- ✅ **認証必須**: 全エンドポイントでJWT認証が必要
- ✅**権限チェック**: ロール管理権限の確認
- ✅ **監査ログ**: 全操作でリクエストユーザー・IP記録
- ✅ **権限継承制御**: 適切な権限継承ルール実装

### **データ操作機能**

#### **CRUD操作**
- ✅ **作成**: 親ロール・権限設定可能なロール作成
- ✅ **取得**: 階層関係・権限を含む詳細取得
- ✅ **更新**: 名前・親ロール・権限の変更対応
- ✅ **削除**: 安全な削除（依存関係チェック）

#### **階層管理**
- ✅ **親子関係**: 再帰的な階層構造管理
- ✅ **ツリー表示**: 階層構造のJSON表現
- ✅ **深度制限**: 最大5階層までの制限
- ✅ **移動検証**: 安全なロール移動処理

#### **権限管理**
- ✅ **権限割り当て**: ロールへの権限付与・削除
- ✅ **権限継承**: 親ロールからの自動権限継承
- ✅ **権限解決**: 複数ロール間の権限マージ・競合解決
- ✅ **権限検索**: 権限別ロール検索機能

#### **フィルタリング・検索**
- ✅ **親ロールフィルタ**: `parent_id`パラメータ
- ✅ **権限フィルタ**: `permission_id`パラメータ
- ✅ **テキスト検索**: `search`パラメータ（名前・説明）
- ✅ **ページング**: `page`・`limit`パラメータ
- ✅ **ソート**: 名前順・作成日順

## 🎯 **実装予定機能**

### **エンドポイント一覧**
| メソッド | エンドポイント | 機能 | 認証 |
|---------|---------------|------|------|
| POST | `/api/v1/roles` | ロール作成 | ⬜️ |
| GET | `/api/v1/roles` | ロール一覧 | ⬜️ |
| GET | `/api/v1/roles/:id` | ロール詳細 | ⬜️ |
| PUT | `/api/v1/roles/:id` | ロール更新 | ⬜️ |
| DELETE | `/api/v1/roles/:id` | ロール削除 | ⬜️ |
| PUT | `/api/v1/roles/:id/permissions` | 権限割り当て | ⬜️ |
| GET | `/api/v1/roles/:id/permissions` | ロール権限一覧 | ⬜️ |
| GET | `/api/v1/roles/hierarchy` | 階層ツリー | ⬜️ |

### **予定レスポンス形式**
```json
{
  "id": "uuid",
  "name": "ロール名",
  "code": "ROLE_CODE",
  "description": "ロール説明",
  "parent_id": "parent_uuid",
  "level": 2,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z",
  "parent": {
    "id": "parent_uuid",
    "name": "親ロール名",
    "code": "PARENT_CODE"
  },
  "children": [
    {
      "id": "child_uuid",
      "name": "子ロール名",
      "code": "CHILD_CODE"
    }
  ],
  "permissions": [
    {
      "id": "permission_uuid",
      "module": "user",
      "action": "read",
      "description": "ユーザー閲覧権限",
      "inherited": false
    }
  ],
  "inherited_permissions": [
    {
      "id": "inherited_uuid",
      "module": "department",
      "action": "read",
      "description": "部署閲覧権限",
      "inherited_from": "parent_uuid"
    }
  ],
  "user_count": 5
}
```

### **予定階層ツリーレスポンス**
```json
{
  "roles": [
    {
      "id": "root_uuid",
      "name": "システム管理者",
      "code": "SYSTEM_ADMIN",
      "level": 0,
      "permission_count": 15,
      "user_count": 2,
      "children": [
        {
          "id": "child_uuid",
          "name": "部署管理者",
          "code": "DEPT_ADMIN",
          "level": 1,
          "permission_count": 8,
          "user_count": 5,
          "children": [
            {
              "id": "grandchild_uuid",
              "name": "一般ユーザー",
              "code": "GENERAL_USER",
              "level": 2,
              "permission_count": 3,
              "user_count": 50
            }
          ]
        }
      ]
    }
  ]
}
```

### **権限割り当てリクエスト**
```json
{
  "permission_ids": [
    "uuid1",
    "uuid2",
    "uuid3"
  ],
  "replace": true
}
```

## 🧪 **テスト実装計画**

### **単体テスト実装** ✅ **実装中**
- **ファイル**: `internal/services/role_test.go`
- **実装済テスト**:
  - ✅ `TestRoleService_InputValidation` - 入力値検証テスト（18サブテスト）
    - CreateRole入力値検証：10テストケース（名前、親ID、権限ID、階層深度）
    - UpdateRole入力値検証：4テストケース（名前重複、自己参照、親変更）
    - AssignPermissions検証：3テストケース（権限存在、ロール存在）
  - ✅ `TestRoleService_HierarchyValidation` - 階層構造検証テスト（5サブテスト）
    - 循環参照検証：3テストケース（直接・間接循環参照、正常移動）
    - 階層深度検証：2テストケース（5階層OK、6階層NG）
  - ✅ `TestRoleService_DeleteValidation` - 削除制限検証テスト（5サブテスト）
    - 削除制限：子ロール存在、プライマリロール割り当て、ユーザーロール割り当て
    - 正常削除：順次削除、存在しないロール
- **実装予定テスト**:
  - ⬜️ `TestRoleService_GetRole` - ロール取得のテスト（4サブテスト）
  - ⬜️ `TestRoleService_GetRoles` - ロール一覧のテスト（5サブテスト）
  - ⬜️ `TestRoleService_GetRolePermissions` - ロール権限取得のテスト（3サブテスト）
  - ⬜️ `TestRoleService_GetRoleHierarchy` - 階層構造のテスト（3サブテスト）
  - ⬜️ `TestRoleService_PermissionInheritance` - 権限継承のテスト（4サブテスト）

### **統合テスト実装** ⬜️ **未着手**
- **ファイル**: `internal/handlers/role_integration_test.go`
- **実装予定テスト**:
  - ⬜️ `TestRoleHandler_CreateRole_Validation` - ロール作成バリデーション（6サブテスト）
  - ⬜️ `TestRoleHandler_GetRoles_QueryParams` - クエリパラメータテスト（8サブテスト）
  - ⬜️ `TestRoleHandler_GetRole_PathParams` - パスパラメータテスト（3サブテスト）
  - ⬜️ `TestRoleHandler_CRUD_Flow` - CRUD操作フロー（6サブテスト）
  - ⬜️ `TestRoleHandler_PermissionManagement` - 権限管理テスト（5サブテスト）
  - ⬜️ `TestRoleHandler_GetRoleHierarchy` - 階層構造取得（2サブテスト）
  - ⬜️ `TestRoleHandler_PermissionInheritance` - 権限継承テスト（4サブテスト）

## 📊 **実装予定統計**

### **予定コード行数**
- `internal/services/role.go`: 約600行（権限継承ロジック含む）
- `internal/handlers/role.go`: 約400行（8エンドポイント）
- テストファイル: 約800行（単体500行・統合300行）

### **実装予定機能**
- ⬜️ **8つのエンドポイント**: 完全なロールCRUD操作・権限管理
- ⬜️ **8つのサービスメソッド**: ビジネスロジック実装・権限継承
- ⬜️ **6つのリクエスト型**: バリデーション付きデータ構造
- ⬜️ **4つのレスポンス型**: 詳細・一覧・階層ツリー・権限一覧
- ⬜️ **約70テストケース**: 包括的なテストカバレッジ（単体44+統合26）

## 🎯 **Step 3完了基準**

- ⬜️ **RoleService実装完了**: 全CRUD操作・権限管理対応
- ⬜️ **RoleHandler実装完了**: 全エンドポイント対応
- ⬜️ **階層管理実装完了**: ツリー構造・制限対応
- ⬜️ **権限継承実装完了**: 親ロールからの権限継承
- ⬜️ **バリデーション実装完了**: 入力値検証・存在確認
- ⬜️ **テスト実装完了**: 単体・統合テスト対応
- ⬜️ **サーバー統合完了**: ルーティング設定・権限チェック

## 🚀 **実装ステップ**

### **Step 3.1 RoleService実装** ⬜️ **未着手**
1. **基本CRUD操作**: Create・Read・Update・Delete
2. **階層管理機能**: 親子関係・深度制限・循環参照防止
3. **権限管理**: 権限割り当て・取得・継承ロジック
4. **バリデーション**: 入力値検証・ビジネスルール

### **Step 3.2 RoleHandler実装** ⬜️ **未着手**
1. **APIエンドポイント**: 8つのRESTfulエンドポイント
2. **リクエスト処理**: バリデーション・エラーハンドリング
3. **レスポンス形式**: 統一されたJSON形式
4. **権限チェック**: 適切な認証・認可実装

### **Step 3.3 テスト実装** ⬜️ **未着手**
1. **単体テスト**: サービス層のテスト（権限継承含む）
2. **統合テスト**: ハンドラー層のテスト
3. **権限テスト**: 認証・認可のテスト

### **Step 3.4 サーバー統合** ⬜️ **未着手**
1. **ルーティング設定**: `cmd/server/main.go`への統合
2. **権限設定**: 各エンドポイントの権限要件設定
3. **動作確認**: 実際のAPI動作テスト

## 📝 **実装時の注意点**

### **技術的考慮事項**
1. **権限継承の複雑性**:
   - 親ロールからの権限継承アルゴリズム
   - 継承権限と直接権限の区別
   - パフォーマンス最適化（キャッシュ考慮）

2. **階層管理**:
   - 循環参照の検出アルゴリズム（Department実装参考）
   - 再帰的な階層構造の処理
   - 深度制限の実装

3. **権限競合解決**:
   - 複数ロール間の権限マージロジック
   - 権限の優先順位決定
   - 明示的拒否の処理

4. **データ整合性**:
   - 外部キー制約の管理
   - 削除時の依存関係チェック
   - トランザクション管理

### **Department実装からの学習ポイント**
1. **階層管理**: `checkCircularReference()`・`calculateDepth()`の活用
2. **テスト戦略**: SQLiteメモリDB・テストデータ分離
3. **エラーハンドリング**: カスタムエラー型の活用
4. **バリデーション**: Gin binding + 独自ビジネスルール

### **セキュリティ考慮事項**
1. **権限エスカレーション防止**: 子ロールが親ロールより強い権限を持たない
2. **権限継承の透明性**: 継承元の明確化
3. **監査ログ**: 権限変更の追跡可能性
4. **最小権限の原則**: 必要最小限の権限割り当て

## 🔄 **依存関係・前提条件**

### **完了済み前提条件**
- ✅ **User管理API**: Step 1完了（ユーザー・ロール関係）
- ✅ **Department管理API**: Step 2完了（階層管理パターン）
- ✅ **Permission基盤**: 既存のPermissionサービス
- ✅ **認証・認可基盤**: JWT・ミドルウェア

### **必要なモデル**
- ✅ **models.Role**: 既存実装済み
- ✅ **models.Permission**: 既存実装済み
- ✅ **models.RolePermission**: 多対多中間テーブル
- ✅ **models.UserRole**: ユーザー・ロール関係

## 🎯 **成功指標**

### **機能要件**
- [ ] ロールCRUD操作（8エンドポイント）実装完了
- [ ] 階層管理（最大5階層）実装完了
- [ ] 権限継承システム実装完了
- [ ] 権限割り当て・取得機能実装完了

### **品質要件**
- [ ] 単体テストカバレッジ80%以上
- [ ] 統合テスト全エンドポイント実装
- [ ] 権限継承ロジックの正確性検証
- [ ] パフォーマンス要件（階層取得200ms以内）

### **セキュリティ要件**
- [ ] 全エンドポイントでの認証・認可チェック
- [ ] 権限エスカレーション防止の検証
- [ ] 監査ログの完全性確保

## 🚧 **リスク・課題**

### **技術的リスク**
1. **権限継承の複雑性**: アルゴリズムの正確性・パフォーマンス
2. **循環参照検出**: 階層構造の複雑化
3. **権限競合**: 複数ロール間の競合解決ロジック

### **対応策**
1. **段階的実装**: 基本CRUD→階層→権限継承の順
2. **既存実装活用**: Department階層管理ロジックの参考
3. **包括的テスト**: エッジケースを含む網羅的テスト

## 🎉 **期待される成果**

Step 3完了により、以下が実現されます：

1. **完全なロール管理**: 階層構造を持つロールの完全管理
2. **柔軟な権限制御**: 継承・割り当てによる柔軟な権限設計
3. **スケーラブルなRBAC**: 大規模組織対応の権限システム
4. **Phase 5の80%完了**: User・Department・Roleの基盤完成

---

**🚀 Phase 5 Step 3 (Role管理API実装) 開始準備完了！**
