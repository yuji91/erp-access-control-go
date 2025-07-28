-- =============================================================================
-- ERP Access Control - Enterprise Data Seeds v2.0
-- =============================================================================
-- エンタープライズ機能のサンプルデータ投入
-- 対象テーブル: user_scopes, approval_states, audit_logs, time_restrictions, revoked_tokens

-- =============================================================================
-- User Scopes（ユーザースコープ）- 細粒度アクセス制御
-- =============================================================================

-- ユーザー別のスコープ権限設定
INSERT INTO user_scopes (user_id, resource_type, scope_type, scope_value, created_at) VALUES
-- システム管理者：全社アクセス
('880e8400-e29b-41d4-a716-446655440001', 'system', 'department', 
 '{"departments": ["*"], "access_level": "admin", "full_control": true}', NOW()),

-- IT部門長：IT部門関連の完全権限
('880e8400-e29b-41d4-a716-446655440002', 'department', 'department',
 '{"departments": ["550e8400-e29b-41d4-a716-446655440001"], "management_level": "department_head", "can_approve": true}', NOW()),

-- 人事部長：人事部門 + ユーザー管理権限
('880e8400-e29b-41d4-a716-446655440003', 'department', 'department',
 '{"departments": ["550e8400-e29b-41d4-a716-446655440002"], "user_management": true, "hr_functions": true}', NOW()),

-- 開発者A：開発プロジェクト範囲
('880e8400-e29b-41d4-a716-446655440004', 'project', 'project',
 '{"projects": ["erp-dev", "api-development"], "access_level": "developer", "read_write": true}', NOW()),

-- 開発者B：異なるプロジェクト範囲
('880e8400-e29b-41d4-a716-446655440005', 'project', 'project',
 '{"projects": ["frontend-dev", "testing"], "access_level": "developer", "deploy_permission": false}', NOW()),

-- プロジェクトマネージャー：複数プロジェクト管理
('880e8400-e29b-41d4-a716-446655440006', 'project', 'project',
 '{"projects": ["*"], "access_level": "manager", "resource_management": true, "team_lead": true}', NOW()),

-- 一般ユーザーA：営業部門限定
('880e8400-e29b-41d4-a716-446655440007', 'department', 'department',
 '{"departments": ["550e8400-e29b-41d4-a716-446655440003"], "access_level": "user", "read_only": false}', NOW()),

-- 一般ユーザーB：経理部門限定
('880e8400-e29b-41d4-a716-446655440008', 'department', 'department',
 '{"departments": ["550e8400-e29b-41d4-a716-446655440004"], "access_level": "user", "financial_data": true}', NOW()),

-- ゲストユーザー：閲覧のみ
('880e8400-e29b-41d4-a716-446655440009', 'department', 'department',
 '{"departments": ["550e8400-e29b-41d4-a716-446655440001"], "access_level": "guest", "read_only": true}', NOW());

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
 '{"cross_department": true, "hr_approval": true, "final_approval": true}', NOW());

-- =============================================================================
-- Audit Logs（監査ログ）- セキュリティ・コンプライアンス用ログ
-- =============================================================================

-- サンプル監査ログ（開発・テスト用）
INSERT INTO audit_logs (user_id, action, resource_type, resource_id, result, reason, reason_code, ip_address, user_agent, timestamp) VALUES
-- ログイン成功ログ
('880e8400-e29b-41d4-a716-446655440001', 'LOGIN', 'auth', 'session_12345', 'SUCCESS', 
 'Administrator login successful', 'AUTH_SUCCESS', '192.168.1.10', 'Mozilla/5.0', NOW() - INTERVAL '2 hours'),

-- ユーザー作成ログ
('880e8400-e29b-41d4-a716-446655440001', 'CREATE_USER', 'user', '880e8400-e29b-41d4-a716-446655440008', 'SUCCESS',
 'New user created in accounting department', 'USER_CREATED', '192.168.1.10', 'curl/7.68.0', NOW() - INTERVAL '1 hour'),

-- 権限変更ログ
('880e8400-e29b-41d4-a716-446655440002', 'MODIFY_PERMISSIONS', 'role', '660e8400-e29b-41d4-a716-446655440003', 'SUCCESS',
 'General user permissions updated', 'PERM_UPDATED', '192.168.1.20', 'Postman/8.0', NOW() - INTERVAL '4 hours'),

-- ログイン失敗ログ
('880e8400-e29b-41d4-a716-446655440004', 'LOGIN', 'auth', 'failed_attempt_001', 'DENIED',
 'Invalid password provided', 'AUTH_FAILED', '192.168.1.50', 'Mozilla/5.0', NOW() - INTERVAL '30 minutes'),

-- システム操作ログ
('880e8400-e29b-41d4-a716-446655440001', 'SYSTEM_BACKUP', 'system', 'backup_20250728', 'SUCCESS',
 'Full system backup completed successfully', 'BACKUP_SUCCESS', '192.168.1.100',
 'curl/7.68.0', NOW() - INTERVAL '3 hours');

-- =============================================================================
-- Time Restrictions（時間制限）- 時間ベースアクセス制御
-- =============================================================================

-- 時間制限設定
INSERT INTO time_restrictions (user_id, resource_type, start_time, end_time, allowed_days, timezone, created_at) VALUES
-- 開発者A：平日9-18時のみシステムアクセス可能
('880e8400-e29b-41d4-a716-446655440004', 'system', '09:00:00', '18:00:00', '{1,2,3,4,5}', 'Asia/Tokyo', NOW()),

-- 開発者B：平日8-20時 + 土曜日のシステムアクセス
('880e8400-e29b-41d4-a716-446655440005', 'system', '08:00:00', '20:00:00', '{1,2,3,4,5,6}', 'Asia/Tokyo', NOW()),

-- プロジェクトマネージャー：制限なし（24/7アクセス）
('880e8400-e29b-41d4-a716-446655440006', 'system', '00:00:00', '23:59:59', '{1,2,3,4,5,6,7}', 'Asia/Tokyo', NOW()),

-- ゲストユーザー：平日日中のみ閲覧可能
('880e8400-e29b-41d4-a716-446655440009', 'read_only', '09:00:00', '17:00:00', '{1,2,3,4,5}', 'Asia/Tokyo', NOW());

-- =============================================================================
-- Revoked Tokens（無効化トークン）- JWT管理・セキュリティ
-- =============================================================================

-- 無効化されたJWTトークンのサンプル（開発・テスト用）
INSERT INTO revoked_tokens (token_jti, user_id, revoked_at, expires_at) VALUES
-- ログアウト時の無効化
('jti-abc123-def456-ghi789', '880e8400-e29b-41d4-a716-446655440004', NOW() - INTERVAL '1 hour', NOW() + INTERVAL '24 hours'),

-- セキュリティ侵害による無効化
('jti-xyz789-uvw456-rst123', '880e8400-e29b-41d4-a716-446655440005', NOW() - INTERVAL '30 minutes', NOW() + INTERVAL '24 hours'),

-- 権限変更による無効化
('jti-mno345-pqr678-stu901', '880e8400-e29b-41d4-a716-446655440006', NOW() - INTERVAL '15 minutes', NOW() + INTERVAL '24 hours'),

-- パスワード変更による無効化
('jti-jkl234-mno567-pqr890', '880e8400-e29b-41d4-a716-446655440007', NOW() - INTERVAL '45 minutes', NOW() + INTERVAL '24 hours');

-- =============================================================================
-- 追加設定・メタデータ
-- =============================================================================

-- バージョン情報・実行履歴
INSERT INTO audit_logs (user_id, action, resource_type, resource_id, result, reason, reason_code, ip_address, user_agent, timestamp) VALUES
('880e8400-e29b-41d4-a716-446655440001', 'SEED_DATA_EXECUTED', 'system', 'enterprise_data_v2.0', 'SUCCESS',
 'Enterprise seed data v2.0 executed successfully. Tables: user_scopes, approval_states, audit_logs, time_restrictions, revoked_tokens', 
 'SEED_SUCCESS', '127.0.0.1', 'PostgreSQL/seed-script', NOW()); 