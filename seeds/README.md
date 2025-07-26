# 🌱 **Seeds (シードデータ)**

このディレクトリには、開発・テスト環境用のサンプルデータファイルが含まれています。

## 📁 **ディレクトリ構成**

```
seeds/
├── README.md            # このファイル
└── 01_test_data.sql     # テストデータ（ユーザー、ロール、権限等）
```

## 🎯 **目的**

- **開発環境**: 開発者が機能テストに使用するサンプルデータ
- **テスト環境**: 自動テスト・手動テストで使用するテストデータ
- **デモ環境**: 機能デモンストレーション用のサンプルデータ

## 🚀 **使用方法**

### **シードデータ投入**
```bash
# シードデータのみ投入
make docker-seed

# マイグレーション + シードデータ投入
make docker-setup-dev
```

### **データリセット**
```bash
# データベースリセット + マイグレーション + シードデータ
make docker-migrate-reset
```

## 📋 **含まれるテストデータ**

### **👥 ユーザーアカウント**
| ユーザー | メールアドレス | パスワード | ロール |
|----------|----------------|------------|--------|
| システム管理者 | admin@example.com | password123 | システム管理者 + 部門管理者 |
| IT部門長 | it-manager@example.com | password123 | 部門管理者 |
| 人事部長 | hr-manager@example.com | password123 | 部門管理者 |
| 開発者A | developer-a@example.com | password123 | 開発者 + テスター(期限付き) |
| 開発者B | developer-b@example.com | password123 | 開発者 + PM(期限付き) |
| PM田中 | pm-tanaka@example.com | password123 | プロジェクトマネージャー |
| 一般ユーザーA | user-a@example.com | password123 | 一般ユーザー |
| 一般ユーザーB | user-b@example.com | password123 | 一般ユーザー |
| ゲストユーザー | guest@example.com | password123 | ゲストユーザー |

### **🏢 部門データ**
- IT部門
- 人事部門  
- 営業部門
- 経理部門

### **👤 ロールデータ**
- システム管理者（全権限）
- 部門管理者（管理権限）
- 開発者（開発関連権限）
- テスター（テスト関連権限）
- プロジェクトマネージャー（プロジェクト管理権限）
- 一般ユーザー（基本権限）
- ゲストユーザー（閲覧のみ）

### **🔐 権限データ**
- ユーザー管理権限 (user:read, user:write, user:delete)
- ロール管理権限 (role:read, role:write, role:delete)
- 部門管理権限 (department:read, department:write, department:delete)
- システム管理権限 (system:read, system:write, system:admin)
- その他の権限 (permission:*, audit:*)

### **🔄 複数ロール例**
- **システム管理者**: システム管理者ロール(優先度1) + 部門管理者ロール(優先度2)
- **開発者A**: 開発者ロール(優先度1) + テスターロール(期限付き・優先度2)
- **開発者B**: 開発者ロール(優先度1) + PMロール(期限付き・優先度2)

## 📝 **ファイル命名規則**

```
01_basic_data.sql        # 基本データ（部門、ロール、権限）
02_test_users.sql        # テストユーザー
03_sample_projects.sql   # サンプルプロジェクト（将来）
```

## ⚠️ **注意事項**

### **本番環境での使用禁止**
- このディレクトリのファイルは**開発・テスト環境専用**です
- 本番環境では**絶対に実行しないでください**

### **セキュリティ**
- テストユーザーのパスワードは全て `password123` です
- 本番環境では強力なパスワードを使用してください

### **データ整合性**
- シードファイルは `ON CONFLICT DO NOTHING` を使用しており、重複実行が可能です
- 既存データがある場合は、新しいデータのみが追加されます

## 🔄 **更新・追加**

新しいシードデータを追加する場合：

1. 適切な番号付きファイルを作成
2. `ON CONFLICT` 句を含める
3. このREADMEを更新
4. テストデータの一覧を更新

## 🧪 **APIテスト**

投入されたテストデータを使用してAPIテストを実行できます：

```bash
# ログインテスト
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com", "password": "password123"}'
```

詳細は [`docs/api-testing/README.md`](../docs/api-testing/README.md) を参照してください。 