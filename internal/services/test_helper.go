package services

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB テスト用のインメモリSQLiteデータベースを作成
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// 手動でDepartmentテーブルを作成
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS departments (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			name TEXT NOT NULL,
			parent_id TEXT,
			FOREIGN KEY (parent_id) REFERENCES departments(id)
		)
	`).Error
	require.NoError(t, err)

	// Userテーブルを作成
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			name TEXT NOT NULL,
			email TEXT,
			password_hash TEXT,
			status TEXT DEFAULT 'active',
			department_id TEXT,
			primary_role_id TEXT,
			FOREIGN KEY (department_id) REFERENCES departments(id)
		)
	`).Error
	require.NoError(t, err)

	return db
}
