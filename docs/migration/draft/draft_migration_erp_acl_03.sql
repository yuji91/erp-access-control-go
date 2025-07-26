-- 🔧 初期マイグレーションスクリプト（見直し03 - 完全版）
-- ERP向けアクセス制御システム用PostgreSQL DDL

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================================================
-- 基本テーブル定義
-- =============================================================================

CREATE TABLE departments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  parent_id UUID REFERENCES departments(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  CONSTRAINT chk_departments_no_self_reference CHECK (id != parent_id)
);

CREATE TABLE roles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  parent_id UUID REFERENCES roles(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  CONSTRAINT chk_roles_no_self_reference CHECK (id != parent_id)
);

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  email TEXT UNIQUE NOT NULL,
  department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
  role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  CONSTRAINT chk_users_status CHECK (status IN ('active', 'inactive', 'suspended'))
);

CREATE TABLE permissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  module TEXT NOT NULL,
  action TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(module, action)
);

CREATE TABLE role_permissions (
  role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_scopes (
  id SERIAL PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  resource_type TEXT NOT NULL,
  resource_id TEXT,  -- 特定リソースへのスコープ制御
  scope_type TEXT NOT NULL,
  scope_value JSONB NOT NULL,  -- 複合スコープ対応
  created_at TIMESTAMPTZ DEFAULT NOW(),
  CONSTRAINT chk_user_scopes_scope_type CHECK (scope_type IN ('department', 'region', 'project', 'location')),
  CONSTRAINT chk_user_scopes_scope_value_structure CHECK (jsonb_typeof(scope_value) = 'object')
);

CREATE TABLE approval_states (
  id SERIAL PRIMARY KEY,
  state_name TEXT NOT NULL,
  approver_role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  step_order INT NOT NULL DEFAULT 1,  -- 多段階承認対応
  resource_type TEXT,                 -- リソース単位での制御
  scope JSONB,                       -- スコープ条件（部門・拠点など）
  created_at TIMESTAMPTZ DEFAULT NOW(),
  CONSTRAINT chk_approval_states_step_order CHECK (step_order > 0)
);

CREATE TABLE audit_logs (
  id SERIAL PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  result TEXT NOT NULL,
  reason TEXT,
  reason_code TEXT,           -- 拒否/成功理由のコード化
  ip_address INET,            -- 操作元IP
  user_agent TEXT,            -- クライアント情報
  timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_audit_logs_result CHECK (result IN ('SUCCESS', 'DENIED', 'ERROR'))
);

-- =============================================================================
-- 拡張テーブル（時間制限・セッション管理）
-- =============================================================================

-- 時間ベース制御テーブル
CREATE TABLE time_restrictions (
  id SERIAL PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  resource_type TEXT NOT NULL,
  start_time TIME,
  end_time TIME,
  allowed_days INTEGER[], -- [1,2,3,4,5] = 月-金
  timezone TEXT DEFAULT 'UTC',
  created_at TIMESTAMPTZ DEFAULT NOW(),
  CONSTRAINT chk_time_restrictions_days CHECK (
    array_length(allowed_days, 1) IS NULL OR 
    (allowed_days <@ ARRAY[1,2,3,4,5,6,7] AND array_length(allowed_days, 1) > 0)
  )
);

-- JWTトークン無効化管理
CREATE TABLE revoked_tokens (
  id SERIAL PRIMARY KEY,
  token_jti TEXT NOT NULL UNIQUE, -- JWT ID
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  revoked_at TIMESTAMPTZ DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL
);

-- =============================================================================
-- インデックス作成（パフォーマンス最適化）
-- =============================================================================

-- 基本検索用インデックス
CREATE INDEX idx_users_department_role ON users(department_id, role_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status) WHERE status != 'active';

-- スコープ検索用インデックス
CREATE INDEX idx_user_scopes_user_resource ON user_scopes(user_id, resource_type);
CREATE INDEX idx_user_scopes_resource_type ON user_scopes(resource_type);
CREATE INDEX idx_user_scopes_scope_value ON user_scopes USING GIN(scope_value);

-- 承認フロー検索用インデックス
CREATE INDEX idx_approval_states_role_step ON approval_states(approver_role_id, step_order);
CREATE INDEX idx_approval_states_resource ON approval_states(resource_type);
CREATE INDEX idx_approval_states_scope ON approval_states USING GIN(scope);

-- 監査ログ検索用インデックス
CREATE INDEX idx_audit_logs_user_timestamp ON audit_logs(user_id, timestamp DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_result ON audit_logs(result);

-- 権限検索用インデックス
CREATE INDEX idx_role_permissions_role ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission ON role_permissions(permission_id);

-- 時間制限検索用インデックス
CREATE INDEX idx_time_restrictions_user_resource ON time_restrictions(user_id, resource_type);

-- トークン管理用インデックス
CREATE INDEX idx_revoked_tokens_jti ON revoked_tokens(token_jti);
CREATE INDEX idx_revoked_tokens_expires ON revoked_tokens(expires_at);
CREATE INDEX idx_revoked_tokens_user ON revoked_tokens(user_id);

-- =============================================================================
-- ビュー作成（階層構造最適化）
-- =============================================================================

-- 部門階層ビュー
CREATE OR REPLACE VIEW department_hierarchy AS
WITH RECURSIVE dept_tree AS (
  SELECT id, name, parent_id, 1 as level, ARRAY[id] as path, name as full_path
  FROM departments WHERE parent_id IS NULL
  UNION ALL
  SELECT d.id, d.name, d.parent_id, dt.level + 1, dt.path || d.id, dt.full_path || ' > ' || d.name
  FROM departments d
  JOIN dept_tree dt ON d.parent_id = dt.id
  WHERE NOT d.id = ANY(dt.path) -- 循環参照防止
)
SELECT * FROM dept_tree;

-- ロール階層ビュー
CREATE OR REPLACE VIEW role_hierarchy AS
WITH RECURSIVE role_tree AS (
  SELECT id, name, parent_id, 1 as level, ARRAY[id] as path, name as full_path
  FROM roles WHERE parent_id IS NULL
  UNION ALL
  SELECT r.id, r.name, r.parent_id, rt.level + 1, rt.path || r.id, rt.full_path || ' > ' || r.name
  FROM roles r
  JOIN role_tree rt ON r.parent_id = rt.id
  WHERE NOT r.id = ANY(rt.path) -- 循環参照防止
)
SELECT * FROM role_tree;

-- ユーザー権限統合ビュー
CREATE OR REPLACE VIEW user_permissions_view AS
SELECT 
  u.id as user_id,
  u.name as user_name,
  u.email,
  d.name as department_name,
  r.name as role_name,
  p.module,
  p.action,
  u.status as user_status
FROM users u
JOIN departments d ON u.department_id = d.id
JOIN roles r ON u.role_id = r.id
JOIN role_permissions rp ON r.id = rp.role_id
JOIN permissions p ON rp.permission_id = p.id
WHERE u.status = 'active';

-- =============================================================================
-- 関数作成（便利機能）
-- =============================================================================

-- ユーザーの全権限取得関数（階層ロール考慮）
CREATE OR REPLACE FUNCTION get_user_all_permissions(user_uuid UUID)
RETURNS TABLE(module TEXT, action TEXT) AS $$
BEGIN
  RETURN QUERY
  WITH user_role_hierarchy AS (
    SELECT rh.id
    FROM users u
    JOIN role_hierarchy rh ON (u.role_id = rh.id OR u.role_id = ANY(rh.path))
    WHERE u.id = user_uuid
  )
  SELECT DISTINCT p.module, p.action
  FROM user_role_hierarchy urh
  JOIN role_permissions rp ON urh.id = rp.role_id
  JOIN permissions p ON rp.permission_id = p.id;
END;
$$ LANGUAGE plpgsql;

-- トークン無効化関数
CREATE OR REPLACE FUNCTION revoke_token(jti TEXT, user_uuid UUID, exp_timestamp TIMESTAMPTZ)
RETURNS VOID AS $$
BEGIN
  INSERT INTO revoked_tokens (token_jti, user_id, expires_at)
  VALUES (jti, user_uuid, exp_timestamp)
  ON CONFLICT (token_jti) DO NOTHING;
END;
$$ LANGUAGE plpgsql;

-- 期限切れトークンクリーンアップ関数
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS INTEGER AS $$
DECLARE
  deleted_count INTEGER;
BEGIN
  DELETE FROM revoked_tokens WHERE expires_at < NOW();
  GET DIAGNOSTICS deleted_count = ROW_COUNT;
  RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 初期データ投入例
-- =============================================================================

-- サンプル部門
INSERT INTO departments (id, name) VALUES 
  ('00000000-0000-0000-0000-000000000001', 'ルート部門'),
  ('00000000-0000-0000-0000-000000000002', '営業部'),
  ('00000000-0000-0000-0000-000000000003', '経理部'),
  ('00000000-0000-0000-0000-000000000004', '人事部');

-- サンプルロール
INSERT INTO roles (id, name) VALUES 
  ('00000000-0000-0000-0000-000000000001', 'admin'),
  ('00000000-0000-0000-0000-000000000002', 'manager'),
  ('00000000-0000-0000-0000-000000000003', 'employee');

-- 階層関係設定
UPDATE roles SET parent_id = '00000000-0000-0000-0000-000000000002' 
WHERE name = 'employee';

UPDATE roles SET parent_id = '00000000-0000-0000-0000-000000000001' 
WHERE name = 'manager';

-- サンプル権限
INSERT INTO permissions (module, action) VALUES 
  ('inventory', 'view'),
  ('inventory', 'update'),
  ('orders', 'create'),
  ('orders', 'approve'),
  ('reports', 'export');

-- ✨ 補足ポイント
-- 1. JSONB活用: user_scopes.scope_value で {"department_id": "dpt-001", "project": "prj-XYZ"} のような複合スコープ
-- 2. 多段階承認: approval_states.step_order + scope で「経理部のみ二次承認が必要」等の制御
-- 3. 詳細監査: reason_code + ip_address + user_agent で完全なトレーサビリティ
-- 4. パフォーマンス: GINインデックスでJSONB検索高速化
-- 5. 時間制御: time_restrictions で営業時間外アクセス制限
-- 6. セッション管理: revoked_tokens でJWT無効化管理

-- 🗺️ ER図（PlantUML / dbdiagram.io / dbml）
-- （ここにER図を貼り付ける）

-- 📜 テスト用初期データINSERT文（モック部門・ロール・ユーザーなど）
-- （ここにテスト用初期データINSERT文を貼り付ける）
