package service

import (
	"fmt"
	"strings"
	"time"

	"photoalbum/internal/storage"
)

// AlbumService 相册业务逻辑
type AlbumService struct {
	repo storage.Repository
}

// NewAlbumService 创建相册服务
func NewAlbumService(repo storage.Repository) *AlbumService {
	return &AlbumService{repo: repo}
}

// CreateAlbum 创建相册
func (s *AlbumService) CreateAlbum(name, description string, userID int64) (*storage.Album, error) {
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

// GetAlbum 获取单个相册
func (s *AlbumService) GetAlbum(id int64, userID int64) (*storage.Album, error) {
	return s.repo.GetAlbumByID(id, userID)
}

// ListAlbums 获取用户所有相册
func (s *AlbumService) ListAlbums(userID int64) ([]*storage.Album, error) {
	return s.repo.ListAlbums(userID)
}

// UpdateAlbum 更新相册信息
func (s *AlbumService) UpdateAlbum(id int64, name, description string, coverPhotoID *int64, userID int64) (*storage.Album, error) {
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

// DeleteAlbum 删除相册
func (s *AlbumService) DeleteAlbum(id int64, userID int64) error {
	return s.repo.DeleteAlbum(id, userID)
}

// AddPhoto 将图片添加到相册
func (s *AlbumService) AddPhoto(albumID int64, photoID int64, userID int64) error {
	return s.repo.AddPhotoToAlbum(albumID, photoID, userID)
}

// RemovePhoto 从相册移除图片
func (s *AlbumService) RemovePhoto(albumID int64, photoID int64, userID int64) error {
	return s.repo.RemovePhotoFromAlbum(albumID, photoID, userID)
}

// ListAlbumPhotos 获取相册内图片（游标分页）
func (s *AlbumService) ListAlbumPhotos(params storage.ListAlbumPhotosParams) (*storage.PhotoPage, error) {
	// 验证相册存在且属于当前用户
	album, err := s.repo.GetAlbumByID(params.AlbumID, params.UserID)
	if err != nil {
		return nil, err
	}
	if album == nil {
		return nil, fmt.Errorf("相册不存在")
	}
	return s.repo.ListAlbumPhotos(params)
}

// GetCoverPhoto 获取相册封面图片
// 若 CoverPhotoID 有值则返回指定图片，否则返回相册中最新添加的图片
func (s *AlbumService) GetCoverPhoto(album *storage.Album) (*storage.Photo, error) {
	if album.CoverPhotoID != nil {
		return s.repo.GetPhotoByID(*album.CoverPhotoID, album.CreatedBy)
	}
	// 取相册第一张（按拍摄时间倒序）
	page, err := s.repo.ListAlbumPhotos(storage.ListAlbumPhotosParams{
		AlbumID: album.ID,
		UserID:  album.CreatedBy,
		Limit:   1,
	})
	if err != nil || len(page.Photos) == 0 {
		return nil, err
	}
	return page.Photos[0], nil
}

// GetAlbumDownloadEntries 获取整个相册的下载条目，按当前相册中的未删除图片构建。
func (s *AlbumService) GetAlbumDownloadEntries(albumID int64, userID int64, photoService *PhotoService) (string, []DownloadEntry, error) {
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
			Path:     photoService.PhotoPath(photo),
			MimeType: photo.MimeType,
		})
	}

	albumName := strings.TrimSpace(album.Name)
	if albumName == "" {
		albumName = "album"
	}
	return albumName, entries, nil
}
