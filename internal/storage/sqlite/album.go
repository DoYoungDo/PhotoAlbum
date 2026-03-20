package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"photoalbum/internal/storage"
)

// CreateAlbum 创建相册
func (s *DB) CreateAlbum(album *storage.Album) error {
	result, err := s.db.Exec(`
		INSERT INTO albums (name, description, cover_photo_id, created_by, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		album.Name, album.Description, album.CoverPhotoID,
		album.CreatedBy, album.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建相册失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	album.ID = id
	return nil
}

// GetAlbumByID 按 ID 查询相册，附带图片数量（不含已软删除的图片）
func (s *DB) GetAlbumByID(id int64, userID int64) (*storage.Album, error) {
	row := s.db.QueryRow(`
		SELECT a.id, a.name, a.description, a.cover_photo_id, a.created_by, a.created_at,
		       COUNT(p.id) as photo_count
		FROM albums a
		LEFT JOIN album_photos ap ON ap.album_id = a.id
		LEFT JOIN photos p ON p.id = ap.photo_id AND p.deleted_at IS NULL
		WHERE a.id = ? AND a.created_by = ?
		GROUP BY a.id`, id, userID)

	album, err := scanAlbum(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return album, err
}

// ListAlbums 查询用户所有相册（photo_count 不含已软删除的图片）
func (s *DB) ListAlbums(userID int64) ([]*storage.Album, error) {
	rows, err := s.db.Query(`
		SELECT a.id, a.name, a.description, a.cover_photo_id, a.created_by, a.created_at,
		       COUNT(p.id) as photo_count
		FROM albums a
		LEFT JOIN album_photos ap ON ap.album_id = a.id
		LEFT JOIN photos p ON p.id = ap.photo_id AND p.deleted_at IS NULL
		WHERE a.created_by = ?
		GROUP BY a.id
		ORDER BY a.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("查询相册失败: %w", err)
	}
	defer rows.Close()

	var albums []*storage.Album
	for rows.Next() {
		a, err := scanAlbum(rows)
		if err != nil {
			return nil, err
		}
		albums = append(albums, a)
	}
	return albums, rows.Err()
}

// UpdateAlbum 更新相册信息
func (s *DB) UpdateAlbum(album *storage.Album) error {
	result, err := s.db.Exec(`
		UPDATE albums SET name = ?, description = ?, cover_photo_id = ?
		WHERE id = ? AND created_by = ?`,
		album.Name, album.Description, album.CoverPhotoID,
		album.ID, album.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("更新相册失败: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("相册不存在")
	}
	return nil
}

// DeleteAlbum 删除相册（级联删除 album_photos 关联，不删除图片本身）
func (s *DB) DeleteAlbum(id int64, userID int64) error {
	result, err := s.db.Exec(`DELETE FROM albums WHERE id = ? AND created_by = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("删除相册失败: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("相册不存在")
	}
	return nil
}

// AddPhotoToAlbum 将图片添加到相册
func (s *DB) AddPhotoToAlbum(albumID int64, photoID int64, userID int64) error {
	// 验证相册属于当前用户
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM albums WHERE id = ? AND created_by = ?`,
		albumID, userID).Scan(&count); err != nil || count == 0 {
		return fmt.Errorf("相册不存在")
	}
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO album_photos (album_id, photo_id, added_at) VALUES (?, ?, ?)`,
		albumID, photoID, time.Now())
	if err != nil {
		return fmt.Errorf("添加图片到相册失败: %w", err)
	}
	return nil
}

// RemovePhotoFromAlbum 从相册移除图片
func (s *DB) RemovePhotoFromAlbum(albumID int64, photoID int64, userID int64) error {
	// 验证相册属于当前用户
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM albums WHERE id = ? AND created_by = ?`,
		albumID, userID).Scan(&count); err != nil || count == 0 {
		return fmt.Errorf("相册不存在")
	}
	_, err := s.db.Exec(`DELETE FROM album_photos WHERE album_id = ? AND photo_id = ?`,
		albumID, photoID)
	return err
}

// ListAlbumPhotos 查询相册内图片（游标分页，按拍摄时间排序）
func (s *DB) ListAlbumPhotos(params storage.ListAlbumPhotosParams) (*storage.PhotoPage, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 30
	}

	var rows *sql.Rows
	var err error

	if params.Cursor == "" {
		rows, err = s.db.Query(`
			SELECT p.id, p.uuid, p.original_name, p.mime_type, p.size, p.width, p.height,
			       p.taken_at, p.uploaded_at, p.uploaded_by, p.deleted_at, p.deleted_by
			FROM photos p
			JOIN album_photos ap ON ap.photo_id = p.id
			WHERE ap.album_id = ? AND p.uploaded_by = ? AND p.deleted_at IS NULL
			ORDER BY p.taken_at DESC, p.id DESC
			LIMIT ?`, params.AlbumID, params.UserID, limit+1)
	} else {
		c, err2 := decodeCursor(params.Cursor)
		if err2 != nil {
			return nil, err2
		}
		rows, err = s.db.Query(`
			SELECT p.id, p.uuid, p.original_name, p.mime_type, p.size, p.width, p.height,
			       p.taken_at, p.uploaded_at, p.uploaded_by, p.deleted_at, p.deleted_by
			FROM photos p
			JOIN album_photos ap ON ap.photo_id = p.id
			WHERE ap.album_id = ? AND p.uploaded_by = ? AND p.deleted_at IS NULL
			  AND (p.taken_at < ? OR (p.taken_at = ? AND p.id < ?))
			ORDER BY p.taken_at DESC, p.id DESC
			LIMIT ?`, params.AlbumID, params.UserID, c.TakenAt, c.TakenAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, fmt.Errorf("查询相册图片失败: %w", err)
	}
	defer rows.Close()

	return collectPhotoPage(rows, limit)
}

// IsPhotoInAlbum 检查图片是否在相册中
func (s *DB) IsPhotoInAlbum(albumID int64, photoID int64) (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM album_photos WHERE album_id = ? AND photo_id = ?`,
		albumID, photoID).Scan(&count)
	return count > 0, err
}

// scanAlbum 从数据库行扫描 Album 对象
func scanAlbum(row interface {
	Scan(...interface{}) error
}) (*storage.Album, error) {
	var a storage.Album
	var coverPhotoID sql.NullInt64
	err := row.Scan(
		&a.ID, &a.Name, &a.Description, &coverPhotoID,
		&a.CreatedBy, &a.CreatedAt, &a.PhotoCount,
	)
	if err != nil {
		return nil, err
	}
	if coverPhotoID.Valid {
		a.CoverPhotoID = &coverPhotoID.Int64
	}
	return &a, nil
}
