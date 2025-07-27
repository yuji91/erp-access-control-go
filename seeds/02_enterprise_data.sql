-- エンタープライズ機能用テストデータ
-- 企業レベルのRBACシステム・スコープ管理・監査機能用のサンプルデータ

-- =============================================================================
-- User Scopes（ユーザースコープ）- 部門・地域・プロジェクト別アクセス制御
-- =============================================================================

-- 部門スコープ：IT部門管理者は自部門とその子部門のユーザーのみ管理可能
INSERT INTO user_scopes (user_id, resource_type, scope_type, scope_value, created_at) VALUES
-- IT部門長の部門スコープ
('880e8400-e29b-41d4-a716-446655440002', 'user', 'department', 
 '{"department_ids": ["550e8400-e29b-41d4-a716-446655440001"], "include_children": true, "access_level": "manager"}', NOW()),

-- 人事部長の部門スコープ  
('880e8400-e29b-41d4-a716-446655440003', 'user', 'department',
 '{"department_ids": ["550e8400-e29b-41d4-a716-446655440002"], "include_children": true, "access_level": "manager"}', NOW()),

-- 開発者Aのプロジェクトスコープ
('880e8400-e29b-41d4-a716-446655440004', 'project', 'project',
 '{"project_ids": ["proj-erp-001", "proj-api-002"], "access_level": "developer", "permissions": ["read", "write"]}', NOW()),

-- 開発者Bの地域スコープ
('880e8400-e29b-41d4-a716-446655440005', 'user', 'region',
 '{"regions": ["tokyo", "kanagawa"], "permissions": ["read", "write"], "time_restricted": true}', NOW()),

-- プロジェクトマネージャーの複合スコープ
('880e8400-e29b-41d4-a716-446655440006', 'user', 'department',
 '{"department_ids": ["550e8400-e29b-41d4-a716-446655440001", "550e8400-e29b-41d4-a716-446655440003"], "include_children": false, "access_level": "project_manager"}', NOW()),

-- テスターの限定スコープ
('880e8400-e29b-41d4-a716-446655440007', 'system', 'feature',
 '{"features": ["testing", "bug_tracking"], "access_level": "tester", "read_only": false}', NOW())
ON CONFLICT (user_id, resource_type, scope_type) DO NOTHING;

-- =============================================================================
-- Approval States（承認状態）- ワークフロー・承認フロー管理
-- =============================================================================

-- 承認フロー状態定義
INSERT INTO approval_states (state_name, approver_role_id, step_order, resource_type, scope, created_at) VALUES
-- ユーザー作成承認フロー
('初期申請', '660e8400-e29b-41d4-a716-446655440003', 1, 'user_creation', 
 '{"max_amount": 0, "auto_approve": false, "required_fields": ["name", "email", "department_id"]}', NOW()),
 
('部門長承認', '660e8400-e29b-41d4-a716-446655440002', 2, 'user_creation',
 '{"max_amount": 1000000, "auto_approve": false, "approval_timeout": 48}', NOW()),
 
('システム管理者最終承認', '660e8400-e29b-41d4-a716-446655440001', 3, 'user_creation',
 '{"final_approval": true, "can_override": true, "notification_required": true}', NOW()),

-- ロール変更承認フロー
('ロール変更申請', '660e8400-e29b-41d4-a716-446655440002', 1, 'role_assignment',
 '{"role_restrictions": ["no_admin_roles"], "requires_justification": true}', NOW()),
 
('上級管理者承認', '660e8400-e29b-41d4-a716-446655440001', 2, 'role_assignment',
 '{"final_approval": true, "can_assign_admin": true, "audit_required": true}', NOW()),

-- 部門変更承認フロー
('部門変更申請', '660e8400-e29b-41d4-a716-446655440002', 1, 'department_change',
 '{"cross_department": false, "notification_required": true}', NOW()),
 
('人事部承認', '660e8400-e29b-41d4-a716-446655440003', 2, 'department_change',
 '{"cross_department": true, "hr_approval": true, "final_approval": true}', NOW())
ON CONFLICT (state_name, resource_type, step_order) DO NOTHING;

-- =============================================================================
-- Audit Logs（監査ログ）- セキュリティ・コンプライアンス用ログ
-- =============================================================================

-- サンプル監査ログ（開発・テスト用）
INSERT INTO audit_logs (user_id, action, resource_type, resource_id, result, details, ip_address, user_agent, timestamp, created_at) VALUES
-- システム管理者のログイン・操作ログ
('880e8400-e29b-41d4-a716-446655440001', 'LOGIN', 'auth', '880e8400-e29b-41d4-a716-446655440001', 'SUCCESS',
 '{"login_method": "password", "session_duration": "15min", "two_factor": false}', '192.168.1.100', 
 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)', NOW() - INTERVAL '2 hours', NOW()),

('880e8400-e29b-41d4-a716-446655440001', 'CREATE_USER', 'user', '880e8400-e29b-41d4-a716-446655440009', 'SUCCESS',
 '{"created_user": "新規開発者", "assigned_role": "開発者", "department": "IT部門"}', '192.168.1.100',
 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)', NOW() - INTERVAL '1 hour', NOW()),

-- 部門管理者の操作ログ
('880e8400-e29b-41d4-a716-446655440002', 'UPDATE_USER_ROLE', 'user_role', '880e8400-e29b-41d4-a716-446655440004', 'SUCCESS',
 '{"old_role": "一般ユーザー", "new_role": "開発者", "reason": "昇進に伴う権限変更"}', '192.168.1.101',
 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)', NOW() - INTERVAL '30 minutes', NOW()),

-- 開発者の失敗ログ
('880e8400-e29b-41d4-a716-446655440004', 'DELETE_USER', 'user', '880e8400-e29b-41d4-a716-446655440008', 'FAILED',
 '{"error": "PERMISSION_DENIED", "required_permission": "user:delete", "user_permission": "user:read"}', '192.168.1.102',
 'Mozilla/5.0 (Linux; Ubuntu)', NOW() - INTERVAL '15 minutes', NOW()),

-- セキュリティ関連ログ
('880e8400-e29b-41d4-a716-446655440005', 'LOGIN_FAILED', 'auth', NULL, 'FAILED',
 '{"reason": "INVALID_PASSWORD", "attempt_count": 3, "ip_blocked": false}', '203.0.113.42',
 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_0)', NOW() - INTERVAL '5 minutes', NOW()),

-- システム操作ログ
('880e8400-e29b-41d4-a716-446655440001', 'SYSTEM_BACKUP', 'system', 'backup_20250728', 'SUCCESS',
 '{"backup_type": "full", "size_mb": 2048, "duration_seconds": 120}', '192.168.1.100',
 'curl/7.68.0', NOW() - INTERVAL '3 hours', NOW())
ON CONFLICT (id) DO NOTHING;

-- =============================================================================
-- Time Restrictions（時間制限）- 時間ベースアクセス制御
-- =============================================================================

-- 時間制限設定
INSERT INTO time_restrictions (user_id, resource_type, start_time, end_time, allowed_days, timezone, restriction_type, created_at) VALUES
-- 開発者A：平日9-18時のみシステムアクセス可能
('880e8400-e29b-41d4-a716-446655440004', 'system', '09:00:00', '18:00:00', '{1,2,3,4,5}', 'Asia/Tokyo', 'work_hours', NOW()),

-- 開発者B：平日8-20時 + 土曜日のシステムアクセス
('880e8400-e29b-41d4-a716-446655440005', 'system', '08:00:00', '20:00:00', '{1,2,3,4,5,6}', 'Asia/Tokyo', 'extended_hours', NOW()),

-- プロジェクトマネージャー：制限なし（24/7アクセス）
('880e8400-e29b-41d4-a716-446655440006', 'system', '00:00:00', '23:59:59', '{0,1,2,3,4,5,6}', 'Asia/Tokyo', 'unrestricted', NOW()),

-- テスター：特定機能への時間制限
('880e8400-e29b-41d4-a716-446655440007', 'feature', '10:00:00', '16:00:00', '{1,2,3,4,5}', 'Asia/Tokyo', 'testing_hours', NOW()),

-- ゲストユーザー：平日日中のみ閲覧可能
('880e8400-e29b-41d4-a716-446655440008', 'read_only', '09:00:00', '17:00:00', '{1,2,3,4,5}', 'Asia/Tokyo', 'guest_hours', NOW())
ON CONFLICT (user_id, resource_type) DO NOTHING;

-- =============================================================================
-- Revoked Tokens（無効化トークン）- JWT管理・セキュリティ
-- =============================================================================

-- 無効化されたJWTトークンのサンプル（開発・テスト用）
INSERT INTO revoked_tokens (token_id, user_id, revoked_at, reason, revoked_by, created_at) VALUES
-- ログアウト時の無効化
('token-abc123-def456-ghi789', '880e8400-e29b-41d4-a716-446655440004', NOW() - INTERVAL '1 hour', 'USER_LOGOUT', '880e8400-e29b-41d4-a716-446655440004', NOW()),

-- セキュリティ侵害による無効化
('token-xyz789-uvw456-rst123', '880e8400-e29b-41d4-a716-446655440005', NOW() - INTERVAL '30 minutes', 'SECURITY_BREACH', '880e8400-e29b-41d4-a716-446655440001', NOW()),

-- 権限変更による無効化
('token-mno345-pqr678-stu901', '880e8400-e29b-41d4-a716-446655440006', NOW() - INTERVAL '15 minutes', 'ROLE_CHANGE', '880e8400-e29b-41d4-a716-446655440002', NOW()),

-- パスワード変更による無効化
('token-jkl234-mno567-pqr890', '880e8400-e29b-41d4-a716-446655440007', NOW() - INTERVAL '45 minutes', 'PASSWORD_CHANGE', '880e8400-e29b-41d4-a716-446655440007', NOW())
ON CONFLICT (token_id) DO NOTHING;

-- =============================================================================
-- 追加設定・メタデータ
-- =============================================================================

-- バージョン情報・実行履歴
INSERT INTO audit_logs (user_id, action, resource_type, resource_id, result, details, ip_address, user_agent, timestamp, created_at) VALUES
('880e8400-e29b-41d4-a716-446655440001', 'SEED_DATA_EXECUTED', 'system', 'enterprise_data_v1.0', 'SUCCESS',
 '{"seed_file": "02_enterprise_data.sql", "version": "1.0", "tables_affected": ["user_scopes", "approval_states", "audit_logs", "time_restrictions", "revoked_tokens"]}', 
 '127.0.0.1', 'PostgreSQL/seed-script', NOW(), NOW())
ON CONFLICT (id) DO NOTHING; 