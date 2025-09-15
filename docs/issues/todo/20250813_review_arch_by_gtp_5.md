# アーキテクチャ設計レビュー（GPT-5による評価）

以下ファイルの「Handler → Service → Repository」設計メモの内容前提での、改善点・論点・即アクションの順に並べます
- `20250729_lookback_typical_api_1_1_archtechture.md`

## まず評価（良い点）
- 責務分離の基本線はOK（Handler=I/O、Service=ビジネス、Repository=永続化）
- 「HTTP詳細をServiceに持ち込まない」「DB詳細をServiceに持ち込まない」の方針は健全
- テストの層別方針（Handler=HTTP、Service=ロジック、Repo=DB）は筋が通っている
- 「エラー変換／ログ出力の層別」も方向性はよい

## 率直な突っ込み（構造上の抜け・弱さ）
### 1) "3層"止まりで、境界(Ports)の定義が弱い

Serviceが「他サービス連携」や「権限照会」へ直接依存する前提になっており、外部依存の切替・テスト差し替え・リトライ/タイムアウト/バックオフ等を吸収するポート層がない。

**→ Ports & Adapters（Hexagonal）を採り入れ、Serviceはポート(interface)にのみ依存させる。**

### 2) トランザクション境界の曖昧さ

「Serviceがトランザクション境界を担う」記述はあるが、Unit of Workの抽象がなく、db.Transaction()のハンドオフ設計が不明。

**→ UnitOfWorkインターフェイス（Do(ctx, func(tx Tx) error)）を導入し、同一Txを跨ぐ複数Repo呼び出しを安全に束ねる。Txは*gorm.DBを内包し、RepoはTxを受け取るオーバーロードを用意。**

### 3) Repositoryが"都合の良い形の集約"を返している

ログイン例でPreload("UserRoles.Role")等、API都合の広めのPreloadが入っており、Repoが読み取り用途に過度最適化。

**→ ドメイン集約の形を返すRepoと、読み取り投影（Query/Readモデル）を分ける（軽いCQRS）。ログイン用は最小投影で十分。**

### 4) 権限判定の所在が曖昧

HandlerでやるのかServiceでやるのかの基準がない。

**→ 認可はService（ユースケース）側が持つ。Handlerはトークンの検証・ID抽出まで。ServiceはPolicy Object/Permissionサービス（ポート）で判定。監査ログ/説明可能性を考えるとここが正解。**

### 5) エラー設計がHTTPと密coupling

「認証失敗はAuthenticationError」などHTTP名寄せのエラー名が散見。

**→ ドメインはErrInvalidCredentials等のユースケース語彙で表し、HTTPはミドルウェアでマッピング。ドメイン→アプリケーション（UseCase）→インターフェース（HTTP）の段階的マッピングを徹底。**

### 6) ドメインモデル（エンティティ/VO）の記述が薄い

user.CheckPassword()程度で、不変条件、値オブジェクト、ドメインイベントの扱いが見えない。

**→ エンティティは不変条件を内包。メールやパスワード、権限範囲などVO化。状態遷移（Active/Inactive）はメソッドで表現し、必要に応じてドメインイベント発行（Outboxへ）。**

### 7) イベント駆動/Outbox/Idempotencyの運用設計が未落ち

Asynqや他サービス連携が出てくるのに冪等性キー／重複排除／再送戦略／Poisonメッセージ処理が未定義。

**→ Outboxテーブル（aggregate_id,event_id(unique),type,payload,attempts,visible_at）＋idempotency keyを外部通知/ジョブ投入の全経路に通す。**

### 8) コンテキスト（context.Context）の扱いが甘い

Handler→Service→Repoの全層でctxを伝搬し、deadline/timeout、trace id、tenant idを載せる。Repoはctx付きDB呼び出しを徹底。

### 9) 観測可能性（Observability）の仕様不足

ログ方針はあるが、構造化ログのフィールド規約、相関ID、メトリクス/トレース、SLO/SLIがない。

**→ Zap + OpenTelemetry。request_id,user_id,tenant_id,usecase,aggregate,event_id等のキーを層ごと既定。失敗率/遅延/リトライ回数をメトリクス化。**

### 10) マルチテナンシの一貫性

以前の会話文脈ではRLS/テナントID運用だが、本メモはテナント伝播/検証/隔離の設計明記が不足。

**→ TenantIDをctxに必須化、Repoクエリは必ずテナント条件を含む／DB側でRLS。テストではテナント跨ぎ侵入の否定検証を用意。**

## GORMまわりの"地雷"指摘

### Preloadの問題
広いPreloadはログイン/一覧APIで即死要因（N+1→Preload過多→過剰JOIN）。

**→ Projection（SELECT col...）専用Repo関数を用意。**

### エラーハンドリング
ErrRecordNotFound→NotFoundError変換はOKだが、存在しないメール＝認証失敗は情報漏えいを避けるため常に同一メッセージで返す（マッピング層で吸収）。

### 楽観ロック
楽観ロック（version列 or updated_at比較）が未記載。

**→ 更新系RepoはWHERE id=? AND version=?で更新し、失敗時はErrVersionConflict。**

### 運用規約の未定義
- ソフトデリートの扱い規約（一覧に出す/出さない、ユニーク制約どうする）が未定義
- ページング規約（limit/offset vs keyset, 並び順フィールド、最大件数制限）が未定義

## Handlerの細かい指摘

- `c.ShouldBindJSON`直後の構造/形式バリデーションはOK。ただしセマンティックなバリデーション（業務制約）はService側に寄せる
- `c.Error(err)`運用はグローバルErrorMiddlewareの契約が前提。ログ重複やエラーの二重ラップに注意
- APIバージョニングで「V1<->V2変換をHandlerでやる」設計は妥当。ただし変換子は別パッケージ化して肥大化回避

## すぐ直せる"設計の芯"——最小構成サンプル

### 1) Ports層の導入（Serviceはポートだけを見る）

```go
// application/port/auth.go
type PermissionPort interface {
    GetUserPermissions(ctx context.Context, userID string) ([]Permission, error)
}

type UserRepo interface {
    FindByEmail(ctx context.Context, email string) (*User, error)
    // ...
}
```

### 2) UnitOfWork

```go
// application/tx/uow.go
type Tx interface {
    DB() *gorm.DB
}

type UoW interface {
    Do(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error
}
```
Serviceは必要なときだけ：

```go
func (s *AuthService) Login(ctx context.Context, req LoginReq) (*LoginResp, error) {
    // 認証は通常Readのみ → トランザクション不要
    u, err := s.users.FindByEmail(ctx, req.Email)
    if err != nil { return nil, app.ErrInvalidCredentials } // 詳細隠蔽
    if !u.CheckPassword(req.Password) { return nil, app.ErrInvalidCredentials }

    perms, err := s.perms.GetUserPermissions(ctx, u.ID)
    if err != nil { return nil, app.ErrUpstream } // 下位障害の抽象化

    return buildLoginResp(u, perms), nil
}
```

### 3) 読み取り投影Repo（最低限・速さ優先）

```go
// infrastructure/repo/user_query_repo.go
func (r *UserQueryRepo) FindAuthProjection(ctx context.Context, email string) (*AuthUser, error) {
    // SELECT 必要列のみ。Preloadしない
}
```

### 4) ドメインイベント＋Outbox（冪等性キー）

```go
// domain/event/user.go
type UserLoggedIn struct { UserID string; OccurredAt time.Time }

// infrastructure/outbox/outbox_repo.go
// Insert(eventID unique, type, payload, dedup_key, visible_at, attempts)
```
Asynqへ投げる際は**dedup_key（idempotency key）必須**。リトライはExponential Backoff＋DLQ。

### 5) エラー語彙とHTTPマッピング

```go
// app/errors.go
var (
  ErrInvalidCredentials = errors.New("invalid credentials")
  ErrPermissionDenied   = errors.New("permission denied")
  ErrVersionConflict    = errors.New("version conflict")
  ErrUpstream           = errors.New("upstream failure")
)

// interface/http/error_mapper.go
func MapError(err error) (int, any) {
  switch {
  case errors.Is(err, app.ErrInvalidCredentials):
    return http.StatusUnauthorized, gin.H{"error":"invalid credentials"}
  case errors.Is(err, app.ErrPermissionDenied):
    return http.StatusForbidden, gin.H{"error":"forbidden"}
  case errors.Is(err, app.ErrVersionConflict):
    return http.StatusConflict, gin.H{"error":"conflict"}
  case errors.Is(err, app.ErrUpstream):
    return http.StatusBadGateway, gin.H{"error":"upstream error"}
  default:
    return http.StatusInternalServerError, gin.H{"error":"internal error"}
  }
}
```

### 6) 観測可能性の最低ライン

- ctxにrequest_id,tenant_id,user_idを注入
- Zapのフィールドは層ごと定義（Handler: method,path,status,latency / Service: usecase,aggregate / Repo: table,rows,sql_ms）
- OpenTelemetryでHTTP span → Service span → Repo spanがつながる構成

## テスト戦略の締め直し

- **Service**：テーブル駆動でユースケースを網羅（成功/境界/拒否/例外）。Repo・外部はモック（ポート）
- **Repo**：Testcontainers + マイグレーション適用で実DB検証（インデックス/実行計画/競合）
- 並行系・リトライの確率的テストは避け、制御可能なフェイククロック/フェイクバックオフで決定的に
- 権限は権限表 × 操作の表テストをユースケースに直結させる（"仕様"として残る）

## 即アクション（チェックリスト）

- [ ] Serviceに依存する外部をすべてPort化（Permission, Mail, Payment, Cache, Queue, HTTP Clients…）
- [ ] UnitOfWork導入＋RepoのTx対応メソッド整備
- [ ] ログイン/一覧APIのPreload削減→専用Projection Repoに分離
- [ ] 認可判定はServiceで一元化、Policy Object実装
- [ ] エラー語彙＝ドメイン語彙に整理、HTTPはMapperで集約
- [ ] Outbox + Idempotency Key + DLQの運用仕様を書面化
- [ ] ctx必須化（tenant/user/request/trace）、全層伝搬
- [ ] 楽観ロック／ページング／ソフトデリートのプロジェクト規約をREADMEに明文化
- [ ] Observability（構造化ログキー、OTel、主要SLI/SLO）を定義
- [ ] マルチテナントの隔離保証（RLS or クエリ規約）＋侵入否定のテスト

## まとめ

現状の3層は"教科書としては正しい"ですが、**実戦投入（ERP×マイクロサービス×長期運用）** を考えると、

1. **ポート（外部依存の抽象）**
2. **Unit of Work（Tx境界の型安全化）**
3. **読取投影の分離（CQRSライト）**
4. **イベント/Outbox/冪等性/観測可能性の運用設計**

が足りていません。ここを埋めると、変更耐性・性能・運用信頼性が一段引き上がります。

---

上記の指摘事項について、妥当性、現在のコードベースでの対応状況、対応優先度の3点（他に記載すべき観点があれば、4-5点）でまとめた表を以下、記載

## 指摘事項の評価・対応状況・優先度一覧

| 指摘項目 | 妥当性 | 現在の対応状況 | 影響度 | 実装困難度 | 優先度 | 推奨アクション |
|---------|--------|---------------|--------|------------|-------|---------------|
| **1. Ports & Adapters不在** | ⭐⭐⭐ | ❌ 未対応<br/>- サービス層がGORM直接依存<br/>- 外部API・キュー等の抽象化なし | High | Medium | 🔴 Critical | 段階的にRepo interfaceから導入、外部サービスのPort化 |
| **2. トランザクション境界曖昧** | ⭐⭐⭐ | 🔶 部分対応<br/>- 一部でdb.Transaction()使用<br/>- UnitOfWork抽象化なし | High | Low | 🔴 Critical | UoWインターフェース導入、Tx境界の型安全化 |
| **3. Repository過度最適化** | ⭐⭐⭐ | ❌ 問題あり<br/>- Preload多用（Login等）<br/>- 読み取り投影分離なし | Medium | Medium | 🟡 High | Query/Command分離、Projection専用Repo作成 |
| **4. 権限判定所在曖昧** | ⭐⭐⭐ | 🔶 部分対応<br/>- Service層で権限チェック<br/>- Policy Object未実装 | High | Medium | 🟡 High | 認可をService層に統一、Policy Object導入 |
| **5. エラー設計HTTP密結合** | ⭐⭐ | 🔶 部分対応<br/>- pkg/errorsで抽象化済<br/>- HTTPマッピング層分離済 | Low | Low | 🟢 Medium | 既存実装の拡張・整理で十分 |
| **6. ドメインモデル薄い** | ⭐⭐⭐ | 🔶 部分対応<br/>- 基本的な不変条件あり<br/>- VO、ドメインイベント不足 | Medium | High | 🟡 High | VO導入、ドメインイベント設計、Aggregate強化 |
| **7. イベント駆動設計未落ち** | ⭐⭐⭐ | ❌ 未対応<br/>- Outboxなし<br/>- 冪等性キー運用なし | High | High | 🟡 High | Outboxパターン、idempotency key、DLQ設計 |
| **8. Context伝搬甘い** | ⭐⭐⭐ | 🔶 部分対応<br/>- ctxは各層で使用<br/>- trace/tenant ID未対応 | Medium | Low | 🟡 High | ctx必須化、trace/tenant/user ID伝搬 |
| **9. 観測可能性不足** | ⭐⭐⭐ | 🔶 部分対応<br/>- 構造化ログあり<br/>- OTel・メトリクス未対応 | Medium | Medium | 🟡 High | OpenTelemetry導入、SLI/SLO定義 |
| **10. マルチテナント一貫性** | ⭐⭐ | ❌ 未対応<br/>- tenant IDなし<br/>- RLS未実装 | Medium | High | 🟢 Medium | 要件明確化後、RLS設計・テナント隔離実装 |

### 📊 **評価基準説明**

#### 妥当性（⭐1-3）
- ⭐: 将来的に必要だが現時点では優先度低
- ⭐⭐: 現在のシステムでも改善効果があり有用
- ⭐⭐⭐: 現在のシステムで必須、未対応はリスク

#### 対応状況
- ❌ 未対応: 全く実装されていない
- 🔶 部分対応: 基本実装はあるが不十分
- ✅ 対応済: 適切に実装済み

#### 優先度
- 🔴 Critical: すぐに着手すべき（1-2週間以内）
- 🟡 High: 重要だが計画的に対応（1-2ヶ月以内）
- 🟢 Medium: 中長期で対応検討（3ヶ月以降）

### 🎯 **即座に実行すべきアクション（Critical項目）**

#### 1. Ports & Adapters導入（1-2週間）
```go
// 段階的実装案
// Phase 1: Repository interface化
type UserRepository interface {
    FindByEmail(ctx context.Context, email string) (*User, error)
    Create(ctx context.Context, user *User) error
}

// Phase 2: 外部サービスのPort化
type PermissionPort interface {
    GetUserPermissions(ctx context.Context, userID string) ([]Permission, error)
}
```

#### 2. Unit of Work導入（1週間）
```go
type UnitOfWork interface {
    Do(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error
}
```

### 📈 **改善の投資対効果**

| 項目 | 工数見積 | 効果 | ROI |
|------|----------|------|-----|
| Ports & Adapters | 2-3週間 | テスタビリティ・保守性大幅向上 | ⭐⭐⭐ |
| UnitOfWork | 1週間 | トランザクション安全性向上 | ⭐⭐⭐ |
| Context伝搬改善 | 3-5日 | 観測可能性・デバッグ性向上 | ⭐⭐⭐ |
| Query/Command分離 | 1-2週間 | パフォーマンス向上 | ⭐⭐ |

### 🚀 **実装ロードマップ**

#### フェーズ1（1ヶ月）- 基盤強化
1. **Week 1**: UnitOfWork導入
2. **Week 2-3**: Repository interface化（段階的）
3. **Week 4**: Context伝搬改善（trace ID等）

#### フェーズ2（2ヶ月）- 品質向上
1. **Month 2**: Query/Command分離、Policy Object導入
2. **Month 3**: 観測可能性強化（OTel、メトリクス）

#### フェーズ3（3ヶ月以降）- 高度機能
1. ドメインイベント・Outboxパターン
2. マルチテナント対応（要件次第）

---

# 重要アーキテクチャパターンの詳細解説

## 1. Ports & Adapters（Hexagonal Architecture）

### 📚 **理論的背景・出典**

**原典**: Alistair Cockburn "Hexagonal Architecture" (2005)
- **別名**: Ports and Adapters Pattern
- **関連文献**: 
  - Robert C. Martin "Clean Architecture" (2017)
  - Vaughn Vernon "Implementing Domain-Driven Design" (2013)

### 🏗️ **概念の核心**

Hexagonal Architectureは、ビジネスロジック（内側の六角形）を外部の技術的詳細（データベース、UI、外部API等）から完全に分離するパターンです。

```
    ┌─────────────────┐
    │   Web UI        │
    └─────┬───────────┘
          │ HTTP Port
    ┌─────▼───────────┐
    │   Application   │◄──► Database Port ──► │ PostgreSQL │
    │   Core          │                       └────────────┘
    │   (Domain)      │◄──► Queue Port ────► │ Redis/SQS  │
    └─────▲───────────┘                      └────────────┘
          │ CLI Port
    ┌─────┴───────────┐
    │   CLI           │
    └─────────────────┘
```

### ⚠️ **現状のままの問題**

#### 1. **テスタビリティの悪化**
```go
// 現状の問題例
func (s *AuthService) Login(req LoginRequest) (*LoginResponse, error) {
    // GORMに直接依存 - テスト時にDB必須
    var user models.User
    if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
        return nil, err
    }
    
    // 外部API呼び出しも直接依存 - テスト困難
    notificationResult := http.Post("https://notification-service/notify", ...)
}
```

**問題点**:
- 単体テストにDBが必要
- 外部API依存でテストが不安定
- テスト実行が遅い（DB接続・HTTP通信）

#### 2. **技術負債の蓄積**
- データベース変更時の影響範囲が予測困難
- 外部サービス仕様変更時の修正コストが高い
- モック作成が困難でテストカバレッジが低下

#### 3. **並行開発の阻害**
- チーム間で外部依存の準備を待つ必要
- 統合テスト環境の構築・維持コストが高い

### ✅ **改善後のメリット**

```go
// Ports & Adapters適用例
type UserRepository interface {
    FindByEmail(ctx context.Context, email string) (*User, error)
    Create(ctx context.Context, user *User) error
}

type NotificationPort interface {
    SendLoginAlert(ctx context.Context, userID string, loginInfo LoginInfo) error
}

type AuthService struct {
    users        UserRepository     // Port (interface)
    notification NotificationPort   // Port (interface)
}

// テスト時はモックを注入
func TestLogin(t *testing.T) {
    mockUsers := &MockUserRepository{}
    mockNotification := &MockNotificationPort{}
    
    service := &AuthService{
        users:        mockUsers,
        notification: mockNotification,
    }
    // 高速で安定したテストが可能
}
```

### 🔗 **補足知識**

- **Dependency Inversion Principle**: SOLID原則の「D」
- **関連パターン**: Repository Pattern, Strategy Pattern
- **実装ツール**: Go interfaces, Dependency Injection (Uber FX, Google Wire)

---

## 2. Unit of Work Pattern

### 📚 **理論的背景・出典**

**原典**: Martin Fowler "Patterns of Enterprise Application Architecture" (2002)
- **章**: Unit of Work Pattern
- **関連**: Transaction Script, Domain Model Pattern

### 🏗️ **概念の核心**

Unit of Workは、ビジネストランザクション中に影響を受けるオブジェクトのリストを維持し、変更の書き込みとコンカレンシー問題の解決を調整するパターンです。

```go
// Unit of Work の概念図
type UnitOfWork interface {
    // トランザクション境界を明確に定義
    Do(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error
}

type Tx interface {
    UserRepo() UserRepository
    RoleRepo() RoleRepository
    // 同一トランザクション内でのリポジトリアクセス
}
```

### ⚠️ **現状のままの問題**

#### 1. **トランザクション境界の曖昧さ**
```go
// 現状の問題例 - トランザクション境界が不明確
func (s *UserService) CreateUserWithRoles(req CreateUserRequest) error {
    // どこからトランザクションが始まる？
    user := &User{Name: req.Name, Email: req.Email}
    if err := s.db.Create(user).Error; err != nil {
        return err
    }
    
    // 別のサービス呼び出し - 同じトランザクション？
    if err := s.roleService.AssignRole(user.ID, req.RoleID); err != nil {
        // ユーザーは作成済みだが、ロール割り当て失敗
        // データ不整合状態！
        return err
    }
    
    return nil
}
```

#### 2. **データ不整合リスク**
- 部分的な更新で処理が中断
- ロールバック漏れによる不正なデータ状態
- 複数テーブル更新時の一貫性保証なし

#### 3. **デッドロック・パフォーマンス問題**
- トランザクション範囲が予測困難
- 長時間トランザクションによるロック競合
- Connection Pool枯渇

### ✅ **改善後のメリット**

```go
// Unit of Work適用例
func (s *UserService) CreateUserWithRoles(ctx context.Context, req CreateUserRequest) error {
    return s.uow.Do(ctx, func(ctx context.Context, tx Tx) error {
        // 明確なトランザクション境界
        user := &User{Name: req.Name, Email: req.Email}
        if err := tx.UserRepo().Create(ctx, user); err != nil {
            return err
        }
        
        // 同一トランザクション内で実行保証
        if err := tx.RoleRepo().AssignUserRole(ctx, user.ID, req.RoleID); err != nil {
            return err // 自動ロールバック
        }
        
        return nil // 自動コミット
    })
}
```

### 🔗 **補足知識**

- **ACID特性**: Atomicity（原子性）の保証が主目的
- **関連パターン**: Repository Pattern, Aggregate Pattern
- **実装例**: GORM Transaction, sqlx.Tx

---

## 7. イベント駆動設計（Event-Driven Architecture）

### 📚 **理論的背景・出典**

**主要文献**:
- Greg Young "Event Sourcing" (2010)
- Martin Fowler "Event Sourcing" (2005)
- Chris Richardson "Microservices Patterns" (2018) - Saga Pattern

### 🏗️ **概念の核心**

イベント駆動アーキテクチャは、システム内の重要な出来事（イベント）を明示的にモデル化し、それを通じてコンポーネント間の疎結合な通信を実現するパターンです。

```
┌─────────────┐    Event    ┌─────────────┐    Event    ┌─────────────┐
│   Service   │ ─────────► │   Outbox    │ ─────────► │  Consumer   │
│     A       │             │   Table     │             │   Service   │
└─────────────┘             └─────────────┘             └─────────────┘
```

### ⚠️ **現状のままの問題**

#### 1. **密結合による脆弱性**
```go
// 現状の同期処理による問題
func (s *UserService) CreateUser(req CreateUserRequest) error {
    user := &User{Name: req.Name, Email: req.Email}
    if err := s.userRepo.Create(user); err != nil {
        return err
    }
    
    // 同期的な外部サービス呼び出し
    if err := s.emailService.SendWelcomeEmail(user.Email); err != nil {
        // メール送信失敗でユーザー作成処理全体が失敗
        return err
    }
    
    if err := s.auditService.LogUserCreation(user.ID); err != nil {
        // 監査ログ失敗でもユーザー作成処理が失敗
        return err
    }
    
    return nil
}
```

**問題点**:
- 一つのサービス障害が全体に波及
- レスポンス時間が最も遅いサービスに依存
- サービス追加時に既存コードの修正が必要

#### 2. **冪等性・重複処理問題**
- ネットワーク障害時の重複実行リスク
- 部分的な処理成功時のリトライ戦略が複雑
- データ不整合状態の検出・修復が困難

#### 3. **運用監視の困難さ**
- 処理の進行状況が不透明
- エラー時の影響範囲特定が困難
- パフォーマンスボトルネックの原因特定が困難

### ✅ **改善後のメリット**

```go
// イベント駆動＋Outboxパターン適用例
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) error {
    return s.uow.Do(ctx, func(ctx context.Context, tx Tx) error {
        // 1. メインの業務処理
        user := &User{Name: req.Name, Email: req.Email}
        if err := tx.UserRepo().Create(ctx, user); err != nil {
            return err
        }
        
        // 2. ドメインイベント作成＋Outboxに保存
        event := &UserCreatedEvent{
            UserID:    user.ID,
            Email:     user.Email,
            OccurredAt: time.Now(),
        }
        
        if err := tx.OutboxRepo().Store(ctx, event, req.IdempotencyKey); err != nil {
            return err
        }
        
        return nil // トランザクション内で原子的に処理
    })
}

// 別プロセスでイベント処理
func (w *EventWorker) ProcessEvents() {
    events := w.outboxRepo.GetPendingEvents()
    for _, event := range events {
        switch e := event.Type {
        case "user.created":
            w.emailService.SendWelcomeEmail(e.Email)    // 非同期処理
            w.auditService.LogUserCreation(e.UserID)    // 非同期処理
        }
        w.outboxRepo.MarkAsProcessed(event.ID)
    }
}
```

### 🔗 **補足知識**

- **Outbox Pattern**: Transactional Outboxによる確実なイベント配信
- **Saga Pattern**: 分散トランザクションの管理
- **CQRS**: Command Query Responsibility Segregation
- **Event Sourcing**: イベントストリームを真実の源とする手法

---

## 8. Context伝搬改善

### 📚 **理論的背景・出典**

**Go公式文献**:
- Go Blog "Context" (2014) by Sameer Ajmani
- "Effective Go" - Context package
- OpenTelemetry Specification (2019)

### 🏗️ **概念の核心**

Contextは、リクエストスコープの値、キャンセレーション信号、デッドラインを API境界を越えて伝搬するためのGo標準の仕組みです。

```
HTTP Request → Handler → Service → Repository
     ↓            ↓          ↓          ↓
   ctx.WithValue → ctx伝搬 → ctx伝搬 → ctx利用
(trace_id, user_id, tenant_id, deadline)
```

### ⚠️ **現状のままの問題**

#### 1. **観測可能性の欠如**
```go
// 現状の問題例 - リクエスト追跡不可能
func (s *UserService) GetUser(userID string) (*User, error) {
    // どのHTTPリクエストからの呼び出しか不明
    // エラー時の根本原因特定が困難
    user, err := s.db.First(&User{}, userID).Error
    if err != nil {
        // ログにリクエストIDがない - デバッグ困難
        log.Error("Failed to get user", err)
        return nil, err
    }
    return user, nil
}
```

#### 2. **タイムアウト制御不能**
```go
// タイムアウトが制御できない
func (s *PaymentService) ProcessPayment(amount int) error {
    // 外部API呼び出しが無制限に待機する可能性
    resp, err := http.Post("https://payment-gateway/charge", ...)
    // ユーザーは既にタイムアウトしているのに処理継続
    return err
}
```

#### 3. **マルチテナント対応不能**
- テナントIDの伝搬なし
- データ漏洩リスク
- 認可チェックの一貫性なし

### ✅ **改善後のメリット**

```go
// Context伝搬改善例
func (s *UserService) GetUser(ctx context.Context, userID string) (*User, error) {
    // リクエストID、ユーザーID、テナントIDが利用可能
    traceID := ctx.Value("trace_id").(string)
    requestUser := ctx.Value("user_id").(string)
    tenantID := ctx.Value("tenant_id").(string)
    
    // 構造化ログで追跡可能
    s.logger.Info("Getting user", map[string]interface{}{
        "trace_id":      traceID,
        "request_user":  requestUser,
        "tenant_id":     tenantID,
        "target_user":   userID,
    })
    
    // タイムアウト制御
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    // テナント隔離
    user, err := s.db.WithContext(ctx).
        Where("id = ? AND tenant_id = ?", userID, tenantID).
        First(&User{}).Error
    
    if err != nil {
        s.logger.Error("Failed to get user", err, map[string]interface{}{
            "trace_id": traceID,
            "user_id":  userID,
        })
        return nil, err
    }
    
    return user, nil
}
```

### 🔗 **補足知識**

- **OpenTelemetry**: 標準的な観測可能性フレームワーク
- **Distributed Tracing**: Jaeger, Zipkin等のトレーシングシステム
- **Structured Logging**: zap, logrus等の構造化ログライブラリ

---

## 📊 概要比較表

| パターン | 未対応時の主要リスク | 対応困難度 | 投資対効果 | 実装優先度 | 関連技術 |
|---------|-------------------|-----------|-----------|-----------|----------|
| **Ports & Adapters** | ・テスト困難<br/>・技術負債蓄積<br/>・並行開発阻害 | Medium | ⭐⭐⭐ | 🔴 Critical | Go interfaces, DI |
| **Unit of Work** | ・データ不整合<br/>・デッドロック<br/>・パフォーマンス劣化 | Low | ⭐⭐⭐ | 🔴 Critical | GORM Transaction |
| **イベント駆動** | ・密結合<br/>・運用監視困難<br/>・冪等性問題 | High | ⭐⭐ | 🟡 High | Outbox, Kafka, SQS |
| **Context伝搬** | ・観測不能<br/>・タイムアウト制御不能<br/>・セキュリティリスク | Low | ⭐⭐⭐ | 🟡 High | OpenTelemetry, Zap |

### 🎯 **実装アプローチ**

1. **即効性重視**: Context伝搬（3-5日で効果実感）
2. **基盤強化**: Unit of Work → Ports & Adapters（1-2ヶ月）
3. **高度機能**: イベント駆動アーキテクチャ（3ヶ月以降）

各パターンは段階的に導入可能で、相互に補完し合う関係にあります。

