package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// DB SQLite 数据库封装
type DB struct {
	db *sql.DB
}

// New 打开 SQLite 数据库并执行迁移
func New(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// SQLite 最佳实践设置
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		return nil, fmt.Errorf("设置 WAL 模式失败: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		return nil, fmt.Errorf("启用外键失败: %w", err)
	}

	store := &DB{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return store, nil
}

// Close 关闭数据库连接
func (s *DB) Close() error {
	return s.db.Close()
}

// migrate 执行数据库建表迁移（幂等）
func (s *DB) migrate() error {
	_, err := s.db.Exec(schema)
	return err
}

// schema 数据库建表 SQL
const schema = `
CREATE TABLE IF NOT EXISTS photos (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid          TEXT    NOT NULL UNIQUE,
    original_name TEXT    NOT NULL,
    mime_type     TEXT    NOT NULL,
    size          INTEGER NOT NULL,
    width         INTEGER NOT NULL DEFAULT 0,
    height        INTEGER NOT NULL DEFAULT 0,
    taken_at      DATETIME NOT NULL,
    uploaded_at   DATETIME NOT NULL,
    uploaded_by   INTEGER  NOT NULL,
    deleted_at    DATETIME,
    deleted_by    INTEGER
);

CREATE INDEX IF NOT EXISTS idx_photos_uploaded_by_taken_at
    ON photos(uploaded_by, taken_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_photos_uploaded_by_deleted_at
    ON photos(uploaded_by, deleted_at DESC)
    WHERE deleted_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS albums (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    name           TEXT    NOT NULL,
    description    TEXT    NOT NULL DEFAULT '',
    cover_photo_id INTEGER,
    created_by     INTEGER NOT NULL,
    created_at     DATETIME NOT NULL,
    FOREIGN KEY (cover_photo_id) REFERENCES photos(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_albums_created_by
    ON albums(created_by);

CREATE TABLE IF NOT EXISTS album_photos (
    album_id INTEGER NOT NULL,
    photo_id INTEGER NOT NULL,
    added_at DATETIME NOT NULL,
    PRIMARY KEY (album_id, photo_id),
    FOREIGN KEY (album_id) REFERENCES albums(id) ON DELETE CASCADE,
    FOREIGN KEY (photo_id) REFERENCES photos(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_album_photos_photo_id
    ON album_photos(photo_id);

CREATE TABLE IF NOT EXISTS share_links (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    token      TEXT    NOT NULL UNIQUE,
    type       TEXT    NOT NULL CHECK(type IN ('photo', 'album')),
    target_id  INTEGER NOT NULL,
    created_by INTEGER NOT NULL,
    expires_at DATETIME,
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_share_links_token
    ON share_links(token);

CREATE INDEX IF NOT EXISTS idx_share_links_created_by
    ON share_links(created_by);
`
