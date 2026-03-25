package service

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"photoalbum/internal/storage"
)

// GetTimeline 获取时间线图片（游标分页）
func (s *PhotoService) GetTimeline(params storage.ListPhotosParams) (*storage.PhotoPage, error) {
	return s.repo.ListPhotos(params)
}

// GetTrash 获取回收站图片（游标分页）
func (s *PhotoService) GetTrash(params storage.ListPhotosParams) (*storage.PhotoPage, error) {
	return s.repo.ListTrashedPhotos(params)
}

// GetAlbumMedia 获取相册内媒体（游标分页）。
func (s *PhotoService) GetAlbumMedia(params storage.ListAlbumPhotosParams) (*storage.PhotoPage, error) {
	return s.repo.ListAlbumPhotos(params)
}

// GetAlbum 获取单个相册。
func (s *PhotoService) GetAlbum(id int64, userID int64) (*storage.Album, error) {
	return s.repo.GetAlbumByID(id, userID)
}

// ListAlbums 获取用户所有相册。
func (s *PhotoService) ListAlbums(userID int64) ([]*storage.Album, error) {
	return s.repo.ListAlbums(userID)
}

// CreateAlbum 创建相册。
func (s *PhotoService) CreateAlbum(name, description string, userID int64) (*storage.Album, error) {
	if name == "" {
		return nil, fmt.Errorf("相册名称不能为空")
	}
	album := &storage.Album{
		Name:        name,
		Description: description,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
	}
	if err := s.repo.CreateAlbum(album); err != nil {
		return nil, err
	}
	return album, nil
}

// UpdateAlbum 更新相册。
func (s *PhotoService) UpdateAlbum(id int64, name, description string, coverPhotoID *int64, userID int64) (*storage.Album, error) {
	album, err := s.repo.GetAlbumByID(id, userID)
	if err != nil {
		return nil, err
	}
	if album == nil {
		return nil, fmt.Errorf("相册不存在")
	}
	if name != "" {
		album.Name = name
	}
	album.Description = description
	album.CoverPhotoID = coverPhotoID
	if err := s.repo.UpdateAlbum(album); err != nil {
		return nil, err
	}
	return album, nil
}

// GetAlbumDownloadEntries 获取相册下载条目。
func (s *PhotoService) GetAlbumDownloadEntries(albumID int64, userID int64) (string, []DownloadEntry, error) {
	album, err := s.repo.GetAlbumByID(albumID, userID)
	if err != nil {
		return "", nil, err
	}
	if album == nil {
		return "", nil, fmt.Errorf("相册不存在")
	}

	page, err := s.repo.ListAlbumPhotos(storage.ListAlbumPhotosParams{
		AlbumID: albumID,
		UserID:  userID,
		Limit:   10000,
	})
	if err != nil {
		return "", nil, err
	}

	entries := make([]DownloadEntry, 0, len(page.Photos))
	usedNames := map[string]int{}
	for _, photo := range page.Photos {
		name := makeUniqueDownloadName(photo.OriginalName, usedNames)
		entries = append(entries, DownloadEntry{
			FileName: name,
			Path:     s.PhotoPath(photo),
			MimeType: photo.MimeType,
		})
	}

	albumName := album.Name
	if albumName == "" {
		albumName = "album"
	}
	return albumName, entries, nil
}

// AddPhoto 将媒体添加到相册。
func (s *PhotoService) AddPhoto(albumID int64, photoID int64, userID int64) error {
	return s.repo.AddPhotoToAlbum(albumID, photoID, userID)
}

// RemovePhoto 将媒体从相册移除。
func (s *PhotoService) RemovePhoto(albumID int64, photoID int64, userID int64) error {
	return s.repo.RemovePhotoFromAlbum(albumID, photoID, userID)
}

// GetPhoto 获取单张图片
func (s *PhotoService) GetPhoto(id int64, userID int64) (*storage.Photo, error) {
	return s.repo.GetPhotoByID(id, userID)
}

// GetPhotoByUUID ��过 UUID 获取单张图片（不含软删除）
func (s *PhotoService) GetPhotoByUUID(uuid string, userID int64) (*storage.Photo, error) {
	return s.repo.GetPhotoByUUID(uuid, userID)
}

// GetPhotoByUUIDAny 通过 UUID 获取图片，包含软删除（用于文件服务）
func (s *PhotoService) GetPhotoByUUIDAny(uuid string, userID int64) (*storage.Photo, error) {
	return s.repo.GetPhotoByUUIDAny(uuid, userID)
}

// DeletePhoto 软删除图片（移入回收站）
func (s *PhotoService) DeletePhoto(id int64, userID int64) error {
	return s.repo.SoftDeletePhoto(id, userID, userID)
}

// RestorePhoto 从回收站恢复图片
func (s *PhotoService) RestorePhoto(id int64, userID int64) error {
	return s.repo.RestorePhoto(id, userID)
}

// PermanentlyDeletePhoto 彻底删除单张回收站图片，并清理磁盘文件。
func (s *PhotoService) PermanentlyDeletePhoto(id int64, userID int64) error {
	photo, err := s.repo.GetPhotoByIDAny(id, userID)
	if err != nil {
		return err
	}
	if photo == nil || photo.DeletedAt == nil {
		return fmt.Errorf("照片/视频不在回收站中")
	}
	if err := s.repo.HardDeletePhoto(id, userID); err != nil {
		return err
	}

	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp"} {
		origPath := filepath.Join(s.storagePath, photo.UUID+ext)
		thumbPath := filepath.Join(s.storagePath, ".thumbnails", photo.UUID+ext)
		_ = os.Remove(origPath)
		_ = os.Remove(thumbPath)
	}
	return nil
}

// EmptyTrash 清空回收站，同时删除磁盘文件
func (s *PhotoService) EmptyTrash(userID int64) error {
	uuids, err := s.repo.HardDeleteTrashedPhotos(userID)
	if err != nil {
		return fmt.Errorf("清空回收站失败: %w", err)
	}

	// 删除磁盘上的原图和缩略图
	for _, u := range uuids {
		// 原图：尝试常见后缀（UUID 存储时不含扩展名，需要遍历）
		for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp"} {
			origPath := filepath.Join(s.storagePath, u+ext)
			thumbPath := filepath.Join(s.storagePath, ".thumbnails", u+ext)
			os.Remove(origPath)
			os.Remove(thumbPath)
		}
	}
	return nil
}
