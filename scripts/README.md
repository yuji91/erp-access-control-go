# 📋 **ERP Access Control API - デモスクリプト**

権限管理システムの全機能をcurlコマンドで実演するデモンストレーション用スクリプト集

---

## 🎯 **demo-permission-system.sh**

### **概要**
ERP Access Control APIの実装済み全権限管理機能を実際のAPIエンドポイントに対してcurlコマンドで呼び出し、システムの動作を実演するスクリプトです。

### **対応機能**
- ✅ **User管理API** - CRUD・ステータス管理・パスワード変更
- ✅ **Department管理API** - CRUD・階層構造管理
- ✅ **Role管理API** - CRUD・階層管理・権限割り当て
- ✅ **Permission管理API** - CRUD・マトリックス表示・統計
- ✅ **認証・認可システム** - JWT・複数ロール・権限チェック

### **前提条件**
```bash
# 1. サーバーが起動済み
make docker-up-dev
# または
go run cmd/server/main.go

# 2. jqコマンドがインストール済み（推奨）
# macOS
brew install jq
# Ubuntu/Debian
sudo apt-get install jq
```

### **使用方法**
```bash
# デモ実行
./scripts/demo-permission-system.sh

# ヘルプ表示
./scripts/demo-permission-system.sh --help
```

### **実演内容**

#### **1. Department管理システム**
```bash
# 階層構造を持つ部署作成
POST /api/v1/departments

# 部署階層構造取得
GET /api/v1/departments/hierarchy

# 部署一覧取得
GET /api/v1/departments
```

#### **2. Role管理システム**
```bash
# 階層構造を持つロール作成
POST /api/v1/roles

# ロール階層構造取得
GET /api/v1/roles/hierarchy

# ロール権限割り当て
PUT /api/v1/roles/{id}/permissions
```

#### **3. Permission管理システム**
```bash
# 権限作成（モジュール・アクション別）
POST /api/v1/permissions

# 権限マトリックス表示
GET /api/v1/permissions/matrix

# 権限一覧取得（フィルタリング・検索）
GET /api/v1/permissions?search=user&page=1&limit=10

# モジュール別権限取得
GET /api/v1/permissions/modules/{module}
```

#### **4. User管理・複数ロール割り当て**
```bash
# ユーザー作成
POST /api/v1/users

# 複数ロール割り当て（期限付き）
POST /api/v1/users/roles

# ユーザーロール一覧確認
GET /api/v1/users/{id}/roles
```

#### **5. 認証・権限チェック**
```bash
# 管理者ログイン
POST /api/v1/auth/login

# 権限不足でのAPI呼び出し（403エラー）
POST /api/v1/users (一般ユーザーToken)

# 自分のプロフィール取得
GET /api/v1/auth/profile
```

#### **6. システム統計・モニタリング**
```bash
# 権限マトリックス統計
GET /api/v1/permissions/matrix

# システムヘルスチェック
GET /health

# バージョン情報
GET /version
```

### **出力例**
```bash
[DEMO] === 1. Department管理システム デモ ===
[STEP] 1.1 部署作成（階層構造）

━━━ 本社部署作成 ━━━
{
  "department": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "name": "本社",
    "description": "本社部署",
    "parent_id": null,
    "created_at": "2024-01-15T10:30:00Z"
  }
}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[SUCCESS] 管理者ログイン成功
```

### **実演される機能一覧**

| 機能カテゴリ | 実演内容 | APIエンドポイント数 |
|-------------|----------|-------------------|
| **認証・認可** | JWT認証・権限チェック・ログイン/ログアウト | 4個 |
| **Department管理** | CRUD・階層構造・統計 | 6個 |
| **Role管理** | CRUD・階層管理・権限割り当て・継承 | 8個 |
| **Permission管理** | CRUD・マトリックス・統計・検索 | 8個 |
| **User管理** | CRUD・複数ロール・ステータス管理 | 7個 |
| **ユーザーロール** | 割り当て・更新・取り消し・一覧 | 4個 |
| **システム** | ヘルスチェック・バージョン・統計 | 3個 |
| **合計** | **30+ RESTful API** | **40個** |

### **デモシナリオ**

1. **システム初期化**
   - サーバー接続確認
   - 管理者認証

2. **組織構造構築**
   - 部署階層作成（本社→営業部）
   - ロール階層作成（管理者→マネージャー→一般）

3. **権限体系構築**
   - 権限作成（user:create, department:manage, sales:read）
   - 権限マトリックス表示
   - ロールへの権限割り当て

4. **ユーザー管理**
   - ユーザー作成（営業マネージャー・一般ユーザー）
   - 複数ロール割り当て（期限付き）
   - ユーザー詳細確認

5. **権限チェック実演**
   - 一般ユーザーログイン
   - 権限不足エラー（403）
   - 適切な権限でのAPI呼び出し

6. **システム統計**
   - 権限マトリックス統計
   - 組織統計・モニタリング

### **期待される結果**

✅ **実演される機能**
- 階層構造を持つ部署管理
- 権限継承付きロール管理  
- 詳細な権限管理とマトリックス表示
- 複数ロール・期限付きロール割り当て
- JWT認証・権限チェック
- 包括的なユーザー管理
- リアルタイム統計・モニタリング

✅ **技術的デモンストレーション**
- 30+ RESTful APIエンドポイント
- JWT認証 + 権限ベースアクセス制御
- エンタープライズグレード（200+テストケース）
- 階層管理・権限継承・統計機能

### **トラブルシューティング**

#### **サーバーが起動していない**
```bash
[ERROR] サーバーが起動していません
以下のコマンドでサーバーを起動してください:
  make docker-up-dev
  または
  go run cmd/server/main.go
```

#### **管理者ログインに失敗**
```bash
# シードデータを再投入
make docker-seed
# または
psql -h localhost -p 5432 -U erp_user -d erp_access_control -f seeds/01_test_data.sql
```

#### **JSON表示が整形されない**
```bash
# jqコマンドをインストール
brew install jq  # macOS
sudo apt-get install jq  # Ubuntu/Debian
```

### **カスタマイズ**

#### **APIベースURL変更**
```bash
# スクリプト内の設定を変更
BASE_URL="http://your-server:8080"
```

#### **認証情報変更**
```bash
# admin_login関数内の認証情報を変更
"email": "your-admin@example.com",
"password": "your-password"
```

---

## 🚀 **その他のスクリプト**

### **api-test.sh**
APIテスト用スクリプト（既存）

### **generate_password_hash.go**
パスワードハッシュ生成ツール（既存）

---

## 📚 **関連ドキュメント**

- [API仕様書](../api/openapi.yaml)
- [実装進捗](../docs/progress/README.md)
- [Phase 5詳細](../docs/progress/phase_5/)
- [Phase 6計画](../docs/progress/phase_6/01_init_and_planning.md)

---

**🎯 このデモスクリプトにより、ERP Access Control APIの全権限管理機能を実際に体験し、システムの完成度とエンタープライズレベルの品質を確認できます！** 