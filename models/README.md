🎯 GORMモデル構成（構造体 + タグ付与）

```plaintext
models/
├── base.go                  # 共通型・Enum定義・ヘルパー関数
├── departments.go           # Department構造体（階層構造・循環参照防止）
├── roles.go                 # Role構造体（階層・権限管理）
├── permissions.go           # Permission・RolePermission構造体
├── users.go                 # User構造体（ステータス・権限管理）
├── user_scopes.go           # UserScope構造体（JSONB対応）
├── approval_states.go       # ApprovalState構造体（多段階承認）
├── audit_logs.go            # AuditLog構造体（INET型対応）
├── time_restrictions.go     # TimeRestriction構造体（配列型対応）
└── revoked_tokens.go        # RevokedToken構造体（JWT管理）
```

🛠️ 主要な実装特徴
1. PostgreSQL特有型対応 ✅
JSONB型: base.goでカスタム型定義、UserScope・ApprovalStateで活用
INET型: AuditLogでIPアドレス管理
INTEGER[]型: TimeRestrictionで曜日配列管理
UUID型: 全モデルでPK・FK対応
2. 階層構造管理 ✅
Department・Role: 自己参照・循環参照防止
階層取得SQL: 祖先・子孫取得メソッド
権限継承: 階層ロール考慮の権限取得
3. バリデーション・制約 ✅
GORM Hooks: BeforeCreate・BeforeUpdate
CHECK制約: Enum値・配列範囲・ステップ順序
カスタムバリデーション: メール・IP・タイムゾーン
4. 豊富なヘルパー関数 ✅
CRUD操作: 各モデル専用の検索・作成・更新・削除
統計機能: リソース別・時間別・ユーザー別統計
権限チェック: 階層考慮・スコープ条件・時間制限

🚀 使用例
// 1. ユーザーの全権限取得（階層ロール考慮）
user, _ := FindUserByEmail(db, "user@example.com")
permissions, _ := user.GetAllPermissions(db)

// 2. JSONBスコープ検索
scopeConditions := JSONB{"department": "sales", "region": "tokyo"}
users, _ := GetUsersByScope(db, "orders", scopeConditions)

// 3. 時間制限チェック
isAllowed, _ := CheckTimeAccess(db, userID, "finance", time.Now())

// 4. 承認フロー取得
approvalFlow, _ := GetApprovalFlow(db, "orders", scopeConditions)

// 5. 監査ログ作成
CreateSuccessAuditLog(db, userID, "view", "orders", "ORD-001", &ipAddr, &userAgent)


📊 データベース機能活用
🔍 GINインデックス: JSONB検索の高速化
🌳 再帰CTE: 階層構造の効率的検索
📈 統計ビュー: パフォーマンス最適化された集計
🛡️ 制約チェック: データ整合性保証
⏰ 時間管理: タイムゾーン・営業時間制限
このGORMモデル群により、Permission Matrix + Policy Object のハイブリッド構成を完全にサポートし、本格的なERPシステムのアクセス制御要件に対応できます。