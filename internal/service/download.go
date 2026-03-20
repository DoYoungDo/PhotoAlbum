package service

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DownloadEntry 表示一个待下载的文件条目。
type DownloadEntry struct {
	FileName string
	Path     string
	MimeType string
}

// GetDownloadEntry 获取单张图片的下载条目。
func (s *PhotoService) GetDownloadEntry(photoID int64, userID int64) (*DownloadEntry, error) {
	photo, err := s.GetPhoto(photoID, userID)
	if err != nil {
		return nil, err
	}
	if photo == nil {
		return nil, fmt.Errorf("图片不存在")
	}
	return &DownloadEntry{
		FileName: photo.OriginalName,
		Path:     s.PhotoPath(photo),
		MimeType: photo.MimeType,
	}, nil
}

// GetDownloadEntries 获取多张图片的下载条目，并处理重名文件。
func (s *PhotoService) GetDownloadEntries(photoIDs []int64, userID int64) ([]DownloadEntry, error) {
	entries := make([]DownloadEntry, 0, len(photoIDs))
	usedNames := map[string]int{}

	for _, photoID := range photoIDs {
		entry, err := s.GetDownloadEntry(photoID, userID)
		if err != nil {
			return nil, err
		}
		entry.FileName = makeUniqueDownloadName(entry.FileName, usedNames)
		entries = append(entries, *entry)
	}

	return entries, nil
}

func makeUniqueDownloadName(name string, used map[string]int) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "download"
	}
	if used[base] == 0 {
		used[base] = 1
		return base
	}

	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	used[base]++
	return fmt.Sprintf("%s (%d)%s", stem, used[base], ext)
}
