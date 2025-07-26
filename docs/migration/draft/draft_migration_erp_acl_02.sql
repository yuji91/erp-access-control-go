-- 🔧 初期マイグレーションスクリプト（見直し02 - 拡張版）

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE departments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  parent_id UUID REFERENCES departments(id) ON DELETE SET NULL
);

CREATE TABLE roles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  parent_id UUID REFERENCES roles(id) ON DELETE SET NULL
);

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  email TEXT UNIQUE NOT NULL,
  department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
  role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE permissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  module TEXT NOT NULL,
  action TEXT NOT NULL,
  UNIQUE(module, action)
);

CREATE TABLE role_permissions (
  role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
  PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_scopes (
  id SERIAL PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  resource_type TEXT NOT NULL,
  resource_id TEXT,  -- 拡張: スコープ対象のリソースID
  scope_type TEXT NOT NULL,
  scope_value JSONB NOT NULL  -- 拡張: JSON構造で複合スコープ対応
);

CREATE TABLE approval_states (
  id SERIAL PRIMARY KEY,
  state_name TEXT NOT NULL,
  approver_role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  step_order INT NOT NULL DEFAULT 1,  -- 拡張: 多段階承認対応
  resource_type TEXT,                 -- 拡張: リソース単位での制御
  scope JSONB                        -- 拡張: スコープ条件（部門・拠点など）
);

CREATE TABLE audit_logs (
  id SERIAL PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  result TEXT NOT NULL,
  reason TEXT,
  reason_code TEXT,           -- 拡張: 拒否/成功理由のコード化
  ip_address INET,            -- 拡張: 操作元IP
  user_agent TEXT,            -- 拡張: クライアント情報
  timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ✨ 補足ポイント
-- user_scopes.scope_value は {"department_id": "dpt-001", "project": "prj-XYZ"} のように複合スコープ表現が可能。
-- approval_states.scope により「経理部のみ二次承認が必要」などのルールも記述可能。
-- audit_logs.reason_code によって NO_PERMISSION, INVALID_STATE, SUCCESS など機械判定も可能に。