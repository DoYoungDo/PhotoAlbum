package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	imgpkg "photoalbum/internal/image"
	"photoalbum/internal/storage"
)

// PhotoService 图片业务逻辑
type PhotoService struct {
	repo          storage.Repository
	storagePath   string
	syncThumbnail bool // 测试用：同步生成缩略图
}

// NewPhotoService 创建图片服务
func NewPhotoService(repo storage.Repository, storagePath string) *PhotoService {
	return &PhotoService{
		repo:        repo,
		storagePath: storagePath,
	}
}

// newPhotoServiceSync 创建同步模式图片服务（仅用于测试）
func newPhotoServiceSync(repo storage.Repository, storagePath string) *PhotoService {
	return &PhotoService{
		repo:          repo,
		storagePath:   storagePath,
		syncThumbnail: true,
	}
}

// UploadInput 上传图片的输入参数
type UploadInput struct {
	Reader       io.ReadSeeker
	OriginalName string
	Size         int64
	UploadedBy   int64
	FileModTime  time.Time // 文件修改时间，作为 EXIF 缺失时的后备
}

// UploadResult 上传结果
type UploadResult struct {
	Photo *storage.Photo
}

// Upload 处理图片上传：提取元数据、存储文件、写数据库
func (s *PhotoService) Upload(input UploadInput) (*UploadResult, error) {
	// 1. 提取图片元数据（EXIF、尺寸、类型）
	fallbackTime := input.FileModTime
	if fallbackTime.IsZero() {
		fallbackTime = time.Now()
	}

	meta, err := imgpkg.ExtractMeta(input.Reader, input.OriginalName, fallbackTime)
	if err != nil {
		return nil, fmt.Errorf("解析图片失败: %w", err)
	}

	// 2. 生成 UUID 文件名
	photoUUID := uuid.New().String()
	ext := filepath.Ext(input.OriginalName)
	filename := photoUUID + ext

	// 3. 重置读取位置，写入磁盘
	if _, err := input.Reader.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek 失败: %w", err)
	}

	destPath := filepath.Join(s.storagePath, filename)
	if err := writeFile(destPath, input.Reader); err != nil {
		return nil, fmt.Errorf("保存图片文件失败: %w", err)
	}

	// 4. 写数据库
	photo := &storage.Photo{
		UUID:         photoUUID,
		OriginalName: input.OriginalName,
		MimeType:     meta.MimeType,
		Size:         input.Size,
		Width:        meta.Width,
		Height:       meta.Height,
		TakenAt:      meta.TakenAt,
		UploadedAt:   time.Now(),
		UploadedBy:   input.UploadedBy,
	}

	if err := s.repo.SavePhoto(photo); err != nil {
		// 数据库写入失败，清理已写入的文件
		os.Remove(destPath)
		return nil, fmt.Errorf("保存图片记录失败: %w", err)
	}

	// 5. 生成缩略图（生产环境异步，测试环境同步）
	if s.syncThumbnail {
		s.generateThumbnail(photo, destPath, meta.MimeType)
	} else {
		go s.generateThumbnail(photo, destPath, meta.MimeType)
	}

	return &UploadResult{Photo: photo}, nil
}

// generateThumbnail 生成缩略图（在后台 goroutine 中调用）
func (s *PhotoService) generateThumbnail(photo *storage.Photo, srcPath string, mimeType string) {
	thumbPath := s.ThumbnailPath(photo)

	f, err := os.Open(srcPath)
	if err != nil {
		return
	}
	defer f.Close()

	imgpkg.GenerateThumbnail(f, mimeType, thumbPath) //nolint:errcheck
}

// PhotoPath 返回图片原图的磁盘路径
func (s *PhotoService) PhotoPath(photo *storage.Photo) string {
	ext := filepath.Ext(photo.OriginalName)
	return filepath.Join(s.storagePath, photo.UUID+ext)
}

// ThumbnailPath 返回图片缩略图的磁盘路径
func (s *PhotoService) ThumbnailPath(photo *storage.Photo) string {
	ext := filepath.Ext(photo.OriginalName)
	return filepath.Join(s.storagePath, ".thumbnails", photo.UUID+ext)
}

// writeFile 将 reader 内容写入目标路径，目标目录若不存在则自动创建
func writeFile(destPath string, r io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}
