-- =============================================================================
-- 複数ロール対応マイグレーション: user_roles テーブル追加
-- =============================================================================

-- user_roles テーブル作成
CREATE TABLE IF NOT EXISTS user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    valid_from TIMESTAMPTZ DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    priority INT DEFAULT 1,
    is_active BOOLEAN DEFAULT TRUE,
    assigned_by UUID REFERENCES users(id),
    assigned_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT uq_user_roles_user_role UNIQUE(user_id, role_id),
    CONSTRAINT chk_user_roles_valid_period CHECK (valid_from < valid_to OR valid_to IS NULL),
    CONSTRAINT chk_user_roles_priority CHECK (priority > 0)
);

-- インデックス作成
CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_active_period ON user_roles(valid_from, valid_to) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_user_roles_priority ON user_roles(user_id, priority DESC) WHERE is_active = TRUE;

-- users テーブル変更（段階的移行）
DO $$
BEGIN
    -- role_idカラムが存在する場合のみNOT NULL制約を削除
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'role_id'
    ) THEN
        ALTER TABLE users ALTER COLUMN role_id DROP NOT NULL;
    END IF;
END $$;

-- primary_role_idカラムが存在しない場合のみ追加
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'primary_role_id'
    ) THEN
        ALTER TABLE users ADD COLUMN primary_role_id UUID REFERENCES roles(id);
    END IF;
END $$;

-- 既存データ移行（role_idが存在する場合のみ実行）
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'role_id'
    ) THEN
        INSERT INTO user_roles (user_id, role_id, valid_from, priority, is_active, assigned_reason)
        SELECT id, role_id, created_at, 1, true, 'Migration from single role'
        FROM users 
        WHERE role_id IS NOT NULL
        ON CONFLICT (user_id, role_id) DO NOTHING;

        -- primary_role_id設定
        UPDATE users SET primary_role_id = role_id 
        WHERE role_id IS NOT NULL AND primary_role_id IS NULL;
    END IF;
END $$;

-- コメント追加
COMMENT ON TABLE user_roles IS '複数ロール対応: ユーザー・ロール関連テーブル';
COMMENT ON COLUMN user_roles.valid_from IS 'ロール有効開始日時';
COMMENT ON COLUMN user_roles.valid_to IS 'ロール有効終了日時（NULL=無期限）';
COMMENT ON COLUMN user_roles.priority IS 'ロール優先度（高い値が優先）';
COMMENT ON COLUMN user_roles.is_active IS 'ロールアクティブ状態';
COMMENT ON COLUMN user_roles.assigned_by IS 'ロール割り当て実行者';
COMMENT ON COLUMN user_roles.assigned_reason IS 'ロール割り当て理由'; 