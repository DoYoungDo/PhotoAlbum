package sqlite

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"photoalbum/internal/storage"
)

// cursor 游标结构，用于分页
type cursor struct {
	TakenAt time.Time `json:"t"`
	ID      int64     `json:"i"`
}

// encodeCursor 将游标编码为字符串
func encodeCursor(takenAt time.Time, id int64) string {
	c := cursor{TakenAt: takenAt, ID: id}
	b, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(b)
}

// decodeCursor 解码游标字符串
func decodeCursor(s string) (*cursor, error) {
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("无效的游标: %w", err)
	}
	var c cursor
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("无效的游标: %w", err)
	}
	return &c, nil
}

// scanPhoto 从数据库行扫描 Photo 对象
func scanPhoto(row interface {
	Scan(...interface{}) error
}) (*storage.Photo, error) {
	var p storage.Photo
	var deletedAt sql.NullTime
	var deletedBy sql.NullInt64

	err := row.Scan(
		&p.ID, &p.UUID, &p.OriginalName, &p.MimeType,
		&p.Size, &p.Width, &p.Height,
		&p.TakenAt, &p.UploadedAt, &p.UploadedBy,
		&deletedAt, &deletedBy,
	)
	if err != nil {
		return nil, err
	}
	if deletedAt.Valid {
		p.DeletedAt = &deletedAt.Time
	}
	if deletedBy.Valid {
		p.DeletedBy = &deletedBy.Int64
	}
	return &p, nil
}

// SavePhoto 保存图片记录
func (s *DB) SavePhoto(photo *storage.Photo) error {
	result, err := s.db.Exec(`
		INSERT INTO photos (uuid, original_name, mime_type, size, width, height, taken_at, uploaded_at, uploaded_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		photo.UUID, photo.OriginalName, photo.MimeType,
		photo.Size, photo.Width, photo.Height,
		photo.TakenAt, photo.UploadedAt, photo.UploadedBy,
	)
	if err != nil {
		return fmt.Errorf("保存图片失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	photo.ID = id
	return nil
}

// GetPhotoByID 按 ID 查询图片
func (s *DB) GetPhotoByID(id int64, userID int64) (*storage.Photo, error) {
	row := s.db.QueryRow(`
		SELECT id, uuid, original_name, mime_type, size, width, height,
		       taken_at, uploaded_at, uploaded_by, deleted_at, deleted_by
		FROM photos
		WHERE id = ? AND uploaded_by = ? AND deleted_at IS NULL`, id, userID)
	p, err := scanPhoto(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// GetPhotoByUUID 按 UUID 查询图片（不含软删除）
func (s *DB) GetPhotoByUUID(uuid string, userID int64) (*storage.Photo, error) {
	row := s.db.QueryRow(`
		SELECT id, uuid, original_name, mime_type, size, width, height,
		       taken_at, uploaded_at, uploaded_by, deleted_at, deleted_by
		FROM photos
		WHERE uuid = ? AND uploaded_by = ? AND deleted_at IS NULL`, uuid, userID)
	p, err := scanPhoto(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// GetPhotoByUUIDAny 按 UUID 查询图片，包含已软删除（用于文件服务回收站图片）
func (s *DB) GetPhotoByUUIDAny(uuid string, userID int64) (*storage.Photo, error) {
	row := s.db.QueryRow(`
		SELECT id, uuid, original_name, mime_type, size, width, height,
		       taken_at, uploaded_at, uploaded_by, deleted_at, deleted_by
		FROM photos
		WHERE uuid = ? AND uploaded_by = ?`, uuid, userID)
	p, err := scanPhoto(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// ListPhotos 查询用户图片（时间线，游标分页）
func (s *DB) ListPhotos(params storage.ListPhotosParams) (*storage.PhotoPage, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 30
	}

	var rows *sql.Rows
	var err error

	if params.Cursor == "" {
		rows, err = s.db.Query(`
			SELECT id, uuid, original_name, mime_type, size, width, height,
			       taken_at, uploaded_at, uploaded_by, deleted_at, deleted_by
			FROM photos
			WHERE uploaded_by = ? AND deleted_at IS NULL
			ORDER BY taken_at DESC, id DESC
			LIMIT ?`, params.UserID, limit+1)
	} else {
		c, err2 := decodeCursor(params.Cursor)
		if err2 != nil {
			return nil, err2
		}
		rows, err = s.db.Query(`
			SELECT id, uuid, original_name, mime_type, size, width, height,
			       taken_at, uploaded_at, uploaded_by, deleted_at, deleted_by
			FROM photos
			WHERE uploaded_by = ? AND deleted_at IS NULL
			  AND (taken_at < ? OR (taken_at = ? AND id < ?))
			ORDER BY taken_at DESC, id DESC
			LIMIT ?`, params.UserID, c.TakenAt, c.TakenAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, fmt.Errorf("查询图片失败: %w", err)
	}
	defer rows.Close()

	return collectPhotoPage(rows, limit)
}

// ListTrashedPhotos 查询回收站图片
func (s *DB) ListTrashedPhotos(params storage.ListPhotosParams) (*storage.PhotoPage, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 30
	}

	var rows *sql.Rows
	var err error

	if params.Cursor == "" {
		rows, err = s.db.Query(`
			SELECT id, uuid, original_name, mime_type, size, width, height,
			       taken_at, uploaded_at, uploaded_by, deleted_at, deleted_by
			FROM photos
			WHERE uploaded_by = ? AND deleted_at IS NOT NULL
			ORDER BY deleted_at DESC, id DESC
			LIMIT ?`, params.UserID, limit+1)
	} else {
		c, err2 := decodeCursor(params.Cursor)
		if err2 != nil {
			return nil, err2
		}
		rows, err = s.db.Query(`
			SELECT id, uuid, original_name, mime_type, size, width, height,
			       taken_at, uploaded_at, uploaded_by, deleted_at, deleted_by
			FROM photos
			WHERE uploaded_by = ? AND deleted_at IS NOT NULL
			  AND (deleted_at < ? OR (deleted_at = ? AND id < ?))
			ORDER BY deleted_at DESC, id DESC
			LIMIT ?`, params.UserID, c.TakenAt, c.TakenAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, fmt.Errorf("查询回收站失败: %w", err)
	}
	defer rows.Close()

	return collectPhotoPage(rows, limit)
}

// collectPhotoPage 收集分页结果，判断是否有更多
func collectPhotoPage(rows *sql.Rows, limit int) (*storage.PhotoPage, error) {
	var photos []*storage.Photo
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	page := &storage.PhotoPage{}
	if len(photos) > limit {
		page.HasMore = true
		photos = photos[:limit]
		last := photos[len(photos)-1]
		page.NextCursor = encodeCursor(last.TakenAt, last.ID)
	}
	page.Photos = photos
	return page, nil
}

// SoftDeletePhoto 软删除图片
func (s *DB) SoftDeletePhoto(id int64, userID int64, deletedBy int64) error {
	result, err := s.db.Exec(`
		UPDATE photos SET deleted_at = ?, deleted_by = ?
		WHERE id = ? AND uploaded_by = ? AND deleted_at IS NULL`,
		time.Now(), deletedBy, id, userID)
	if err != nil {
		return fmt.Errorf("删除图片失败: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("图片不存在或已删除")
	}
	return nil
}

// RestorePhoto 从回收站恢复图片
func (s *DB) RestorePhoto(id int64, userID int64) error {
	result, err := s.db.Exec(`
		UPDATE photos SET deleted_at = NULL, deleted_by = NULL
		WHERE id = ? AND uploaded_by = ? AND deleted_at IS NOT NULL`,
		id, userID)
	if err != nil {
		return fmt.Errorf("恢复图片失败: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("图片不在回收站中")
	}
	return nil
}

// HardDeletePhoto 彻底删除单张图片记录
func (s *DB) HardDeletePhoto(id int64, userID int64) error {
	_, err := s.db.Exec(`DELETE FROM photos WHERE id = ? AND uploaded_by = ?`, id, userID)
	return err
}

// HardDeleteTrashedPhotos 清空回收站，返回需要删除文件的 UUID 列表
func (s *DB) HardDeleteTrashedPhotos(userID int64) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT uuid FROM photos WHERE uploaded_by = ? AND deleted_at IS NOT NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uuids []string
	for rows.Next() {
		var uuid string
		if err := rows.Scan(&uuid); err != nil {
			return nil, err
		}
		uuids = append(uuids, uuid)
	}

	if _, err := s.db.Exec(`
		DELETE FROM photos WHERE uploaded_by = ? AND deleted_at IS NOT NULL`, userID); err != nil {
		return nil, fmt.Errorf("清空回收站失败: %w", err)
	}

	return uuids, nil
}
