-- パスワードハッシュカラム追加マイグレーション
-- 認証機能に必要なパスワードハッシュカラムをusersテーブルに追加

-- usersテーブルにpassword_hashカラムを追加（存在しない場合のみ）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' AND column_name = 'password_hash'
    ) THEN
        ALTER TABLE users ADD COLUMN password_hash TEXT;
    END IF;
END $$;

-- 既存のユーザーに対してデフォルトパスワードハッシュを設定
-- パスワード: password123 のbcryptハッシュ
UPDATE users 
SET password_hash = '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi' 
WHERE password_hash IS NULL;

-- password_hashカラムをNOT NULLに変更（既に設定されていない場合のみ）
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'users' 
        AND column_name = 'password_hash' 
        AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
    END IF;
END $$;

-- コメント
COMMENT ON COLUMN users.password_hash IS 'パスワードハッシュ（bcrypt）'; 