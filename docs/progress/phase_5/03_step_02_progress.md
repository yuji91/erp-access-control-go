# 🔧 **Phase 5 Step 2: Department管理API実装** - 進捗レポート

## 📋 **概要**

Department管理APIの実装（Step 2）の計画を策定しました。DepartmentServiceとDepartmentHandlerの実装により、部署の階層構造を含む完全なCRUD操作を実現する予定です。

## ⬜️ **実装予定項目**

### **2.1 DepartmentService実装** ⬜️ **未実装**
- **ファイル**: `internal/services/department.go`
- **実装予定機能**:
  - ⬜️ `CreateDepartment()` - 部署作成（階層構造対応）
  - ⬜️ `GetDepartment()` - 部署詳細取得（親子関係込み）
  - ⬜️ `UpdateDepartment()` - 部署更新（名前・親部署変更）
  - ⬜️ `DeleteDepartment()` - 部署削除（子部署・所属ユーザーチェック）
  - ⬜️ `GetDepartments()` - 部署一覧取得（階層表示）
  - ⬜️ `GetDepartmentHierarchy()` - 部署階層ツリー取得

### **2.2 DepartmentHandler実装** ⬜️ **未実装**
- **ファイル**: `internal/handlers/department.go`
- **実装予定エンドポイント**:
  - ⬜️ `POST /api/v1/departments` - 部署作成
  - ⬜️ `GET /api/v1/departments` - 部署一覧・階層
  - ⬜️ `GET /api/v1/departments/:id` - 部署詳細
  - ⬜️ `PUT /api/v1/departments/:id` - 部署更新
  - ⬜️ `DELETE /api/v1/departments/:id` - 部署削除
  - ⬜️ `GET /api/v1/departments/hierarchy` - 階層ツリー

### **2.3 階層管理・バリデーション** ⬜️ **未実装**
- **ファイル**: `internal/services/department.go`
- **実装予定機能**:
  - ⬜️ **階層制限**: 循環参照防止・最大階層深度制限（5階層）
  - ⬜️ **削除制限**: 子部署存在時の削除禁止・所属ユーザー存在チェック
  - ⬜️ **移動制限**: 部署移動時の権限・承認フロー対応

## 🔧 **実装計画詳細**

### **バリデーション・セキュリティ機能**

#### **入力値検証**
- ⬜️ **部署名**: `binding:"required,min=2,max=100"`
- ⬜️ **親部署ID**: `binding:"omitempty,uuid"`
- ⬜️ **階層深度**: 最大5階層までの制限
- ⬜️ **必須項目**: 名前・コード・説明

#### **ビジネスルール**
- ⬜️ **循環参照チェック**: 親子関係の循環を防止
- ⬜️ **階層深度チェック**: 最大深度を超える構造を防止
- ⬜️ **削除前チェック**: 子部署・所属ユーザーの存在確認
- ⬜️ **コード重複**: 部署コードの一意性確保

#### **セキュリティ**
- ⬜️ **認証必須**: 全エンドポイントでJWT認証が必要
- ⬜️ **権限チェック**: 部署管理権限の確認
- ⬜️ **監査ログ**: 全操作でリクエストユーザー・IP記録

### **データ操作機能**

#### **CRUD操作**
- ⬜️ **作成**: 親部署指定可能な部署作成
- ⬜️ **取得**: 階層関係を含む詳細取得
- ⬜️ **更新**: 名前・親部署の変更対応
- ⬜️ **削除**: 安全な削除（依存関係チェック）

#### **階層管理**
- ⬜️ **親子関係**: 再帰的な階層構造管理
- ⬜️ **ツリー表示**: 階層構造のJSON表現
- ⬜️ **深度制限**: 最大5階層までの制限
- ⬜️ **移動検証**: 安全な部署移動処理

#### **フィルタリング・検索**
- ⬜️ **親部署フィルタ**: `parent_id`パラメータ
- ⬜️ **テキスト検索**: `search`パラメータ（名前・コード）
- ⬜️ **ページング**: `page`・`limit`パラメータ
- ⬜️ **ソート**: `sort`パラメータ（名前・作成日）

## 🎯 **実装予定機能**

### **エンドポイント一覧**
| メソッド | エンドポイント | 機能 | 認証 |
|---------|---------------|------|------|
| POST | `/api/v1/departments` | 部署作成 | ⬜️ |
| GET | `/api/v1/departments` | 部署一覧 | ⬜️ |
| GET | `/api/v1/departments/:id` | 部署詳細 | ⬜️ |
| PUT | `/api/v1/departments/:id` | 部署更新 | ⬜️ |
| DELETE | `/api/v1/departments/:id` | 部署削除 | ⬜️ |
| GET | `/api/v1/departments/hierarchy` | 階層ツリー | ⬜️ |

### **予定レスポンス形式**
```json
{
  "id": "uuid",
  "name": "部署名",
  "code": "DEP001",
  "description": "部署説明",
  "parent_id": "parent_uuid",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z",
  "parent": {
    "id": "parent_uuid",
    "name": "親部署名"
  },
  "children": [
    {
      "id": "child_uuid",
      "name": "子部署名"
    }
  ],
  "users": [
    {
      "id": "user_uuid",
      "name": "所属ユーザー名"
    }
  ]
}
```

### **予定階層ツリーレスポンス**
```json
{
  "departments": [
    {
      "id": "root_uuid",
      "name": "本社",
      "children": [
        {
          "id": "child_uuid",
          "name": "営業部",
          "children": [
            {
              "id": "grandchild_uuid",
              "name": "東京営業所"
            }
          ]
        }
      ]
    }
  ]
}
```

## 🧪 **テスト実装計画**

### **単体テスト予定**
```bash
=== RUN   TestDepartmentService_CreateDepartment
=== RUN   TestDepartmentService_HierarchyValidation
=== RUN   TestDepartmentService_DeleteRestrictions
=== RUN   TestDepartmentService_TreeStructure
```

### **統合テスト予定**
```bash
=== RUN   TestDepartmentHandler_CRUD
=== RUN   TestDepartmentHandler_Hierarchy
=== RUN   TestDepartmentHandler_Validation
```

## 📊 **実装予定統計**

### **予定コード行数**
- `internal/services/department.go`: 約350行
- `internal/handlers/department.go`: 約280行
- テストファイル: 約450行

### **実装予定機能**
- ⬜️ **6つのエンドポイント**: 完全なCRUD操作
- ⬜️ **7つのサービスメソッド**: ビジネスロジック実装
- ⬜️ **4つのリクエスト型**: バリデーション付きデータ構造
- ⬜️ **3つのレスポンス型**: 詳細・一覧・階層ツリー
- ⬜️ **15テストケース**: 包括的なテストカバレッジ

## 🎯 **Step 2完了基準**

- ⬜️ **DepartmentService実装完了**: 全CRUD操作対応
- ⬜️ **DepartmentHandler実装完了**: 全エンドポイント対応
- ⬜️ **階層管理実装完了**: ツリー構造・制限対応
- ⬜️ **バリデーション実装完了**: 入力値検証・存在確認
- ⬜️ **テスト実装完了**: 単体・統合テスト対応

## 🚀 **次のステップ**

### **Step 2.1 DepartmentService実装**
1. **基本CRUD操作**: Create・Read・Update・Delete
2. **階層管理機能**: 親子関係・深度制限
3. **バリデーション**: 入力値検証・ビジネスルール

### **Step 2.2 DepartmentHandler実装**
1. **APIエンドポイント**: 6つのRESTfulエンドポイント
2. **リクエスト処理**: バリデーション・エラーハンドリング
3. **レスポンス形式**: 統一されたJSON形式

### **Step 2.3 テスト実装**
1. **単体テスト**: サービス層のテスト
2. **統合テスト**: ハンドラー層のテスト
3. **権限テスト**: 認証・認可のテスト

## 📝 **実装時の注意点**

### **技術的考慮事項**
1. **階層管理の複雑性**:
   - 循環参照の検出アルゴリズム
   - 再帰的な階層構造の処理
   - パフォーマンス最適化

2. **データ整合性**:
   - 外部キー制約の管理
   - 削除時の依存関係チェック
   - トランザクション管理

3. **セキュリティ**:
   - 権限チェックの実装
   - 入力値検証の徹底
   - 監査ログの記録

### **将来的な改善点**
1. **パフォーマンス最適化**:
   - 大規模階層構造のキャッシュ
   - 階層クエリの最適化
   - インデックス追加検討

2. **機能拡張**:
   - 部署コード自動生成
   - 一括移動機能
   - 履歴管理機能

3. **UI/UX改善**:
   - ツリー表示の最適化
   - ドラッグ&ドロップ移動
   - 階層図エクスポート
