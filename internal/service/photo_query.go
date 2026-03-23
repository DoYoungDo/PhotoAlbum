package service

import (
	"fmt"
	"os"
	"path/filepath"

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
		return fmt.Errorf("图片不在回收站中")
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
