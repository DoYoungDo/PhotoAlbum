package service

import (
	"fmt"
	"time"

	"photoalbum/internal/media"
	"photoalbum/internal/storage"
)

type RegisterUploadedVideoInput struct {
	UUID         string
	OriginalName string
	MimeType     string
	Size         int64
	UploadedBy   int64
	TakenAt      time.Time
	Meta         *media.VideoMeta
}

func (s *PhotoService) RegisterUploadedVideo(input RegisterUploadedVideoInput) (*storage.Photo, error) {
	if input.UUID == "" {
		return nil, fmt.Errorf("UUID 不能为空")
	}
	if input.Meta == nil {
		return nil, fmt.Errorf("视频元数据不能为空")
	}
	if input.MimeType == "" {
		input.MimeType = "video/mp4"
	}
	if input.TakenAt.IsZero() {
		input.TakenAt = time.Now()
	}

	photo := &storage.Photo{
		UUID:         input.UUID,
		OriginalName: input.OriginalName,
		MediaKind:    storage.MediaKindVideo,
		MimeType:     input.MimeType,
		Size:         input.Size,
		Width:        input.Meta.Width,
		Height:       input.Meta.Height,
		DurationMS:   input.Meta.DurationMS,
		TakenAt:      input.TakenAt,
		UploadedAt:   time.Now(),
		UploadedBy:   input.UploadedBy,
	}

	if err := s.repo.SavePhoto(photo); err != nil {
		return nil, fmt.Errorf("保存视频记录失败: %w", err)
	}
	return photo, nil
}
