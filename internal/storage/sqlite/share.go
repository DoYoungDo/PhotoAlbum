package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"photoalbum/internal/storage"
)

// CreateShareLink 创建分享链接
func (s *DB) CreateShareLink(link *storage.ShareLink) error {
	result, err := s.db.Exec(`
		INSERT INTO share_links (token, type, target_id, created_by, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		link.Token, link.Type, link.TargetID,
		link.CreatedBy, link.ExpiresAt, link.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建分享链接失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	link.ID = id
	return nil
}

// GetShareLinkByToken 按 token 查询分享链接（自动过滤已过期）
func (s *DB) GetShareLinkByToken(token string) (*storage.ShareLink, error) {
	row := s.db.QueryRow(`
		SELECT id, token, type, target_id, created_by, expires_at, created_at
		FROM share_links
		WHERE token = ? AND (expires_at IS NULL OR expires_at > ?)`,
		token, time.Now())

	link, err := scanShareLink(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return link, err
}

// ListShareLinks 查询用户创建的所有分享链接
func (s *DB) ListShareLinks(userID int64) ([]*storage.ShareLink, error) {
	rows, err := s.db.Query(`
		SELECT id, token, type, target_id, created_by, expires_at, created_at
		FROM share_links
		WHERE created_by = ?
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("查询分享链接失败: %w", err)
	}
	defer rows.Close()

	var links []*storage.ShareLink
	for rows.Next() {
		l, err := scanShareLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

// DeleteShareLink 删除分享链接
func (s *DB) DeleteShareLink(id int64, userID int64) error {
	result, err := s.db.Exec(`DELETE FROM share_links WHERE id = ? AND created_by = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("删除分享链接失败: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("分享链接不存在")
	}
	return nil
}

// scanShareLink 从数据库行扫描 ShareLink 对象
func scanShareLink(row interface {
	Scan(...interface{}) error
}) (*storage.ShareLink, error) {
	var l storage.ShareLink
	var expiresAt sql.NullTime
	err := row.Scan(
		&l.ID, &l.Token, &l.Type, &l.TargetID,
		&l.CreatedBy, &expiresAt, &l.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		l.ExpiresAt = &expiresAt.Time
	}
	return &l, nil
}
