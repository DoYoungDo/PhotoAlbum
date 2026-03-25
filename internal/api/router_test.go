package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"photoalbum/internal/config"
	"photoalbum/internal/service"
	"photoalbum/internal/storage"
)

type stubRegistrar struct {
	addPhoto                func(albumID int64, photoID int64, userID int64) error
	createAlbum             func(name, description string, userID int64) (*storage.Album, error)
	createShare             func(input service.CreateShareInput) (*storage.ShareLink, error)
	deleteAlbum             func(id int64, userID int64) error
	deleteShare             func(id int64, userID int64) error
	getAlbum                func(id int64, userID int64) (*storage.Album, error)
	getAlbumDownloadEntries func(albumID int64, userID int64) (string, []service.DownloadEntry, error)
	getShareByToken         func(token string) (*storage.ShareLink, error)
	listAlbums              func(userID int64) ([]*storage.Album, error)
	listShares              func(userID int64) ([]*storage.ShareLink, error)
	removePhoto             func(albumID int64, photoID int64, userID int64) error
	updateAlbum             func(id int64, name, description string, coverPhotoID *int64, userID int64) (*storage.Album, error)
	register                func(input service.RegisterUploadedVideoInput) (*storage.Photo, error)
	deletePhoto             func(id int64, userID int64) error
	emptyTrash              func(userID int64) error
	getDownloadEntries      func(photoIDs []int64, userID int64) ([]service.DownloadEntry, error)
	getAlbumMedia           func(params storage.ListAlbumPhotosParams) (*storage.PhotoPage, error)
	getPhoto                func(id int64, userID int64) (*storage.Photo, error)
	getByUUID               func(uuid string, userID int64) (*storage.Photo, error)
	getTrash                func(params storage.ListPhotosParams) (*storage.PhotoPage, error)
	getTimeline             func(params storage.ListPhotosParams) (*storage.PhotoPage, error)
	mediaPath               func(photo *storage.Photo) string
	permanentlyDeletePhoto  func(id int64, userID int64) error
	posterPath              func(photo *storage.Photo) string
	restorePhoto            func(id int64, userID int64) error
}

func (s stubRegistrar) RegisterUploadedVideo(input service.RegisterUploadedVideoInput) (*storage.Photo, error) {
	return s.register(input)
}

func (s stubRegistrar) AddPhoto(albumID int64, photoID int64, userID int64) error {
	return s.addPhoto(albumID, photoID, userID)
}

func (s stubRegistrar) CreateAlbum(name, description string, userID int64) (*storage.Album, error) {
	return s.createAlbum(name, description, userID)
}

func (s stubRegistrar) CreateShare(input service.CreateShareInput) (*storage.ShareLink, error) {
	return s.createShare(input)
}

func (s stubRegistrar) DeleteAlbum(id int64, userID int64) error {
	return s.deleteAlbum(id, userID)
}

func (s stubRegistrar) DeleteShare(id int64, userID int64) error {
	return s.deleteShare(id, userID)
}

func (s stubRegistrar) UpdateAlbum(id int64, name, description string, coverPhotoID *int64, userID int64) (*storage.Album, error) {
	return s.updateAlbum(id, name, description, coverPhotoID, userID)
}

func (s stubRegistrar) GetAlbum(id int64, userID int64) (*storage.Album, error) {
	return s.getAlbum(id, userID)
}

func (s stubRegistrar) GetShareByToken(token string) (*storage.ShareLink, error) {
	return s.getShareByToken(token)
}

func (s stubRegistrar) GetAlbumDownloadEntries(albumID int64, userID int64) (string, []service.DownloadEntry, error) {
	return s.getAlbumDownloadEntries(albumID, userID)
}

func (s stubRegistrar) ListAlbums(userID int64) ([]*storage.Album, error) {
	return s.listAlbums(userID)
}

func (s stubRegistrar) ListShares(userID int64) ([]*storage.ShareLink, error) {
	return s.listShares(userID)
}

func (s stubRegistrar) RemovePhoto(albumID int64, photoID int64, userID int64) error {
	return s.removePhoto(albumID, photoID, userID)
}

func (s stubRegistrar) DeletePhoto(id int64, userID int64) error {
	return s.deletePhoto(id, userID)
}

func (s stubRegistrar) EmptyTrash(userID int64) error {
	return s.emptyTrash(userID)
}

func (s stubRegistrar) GetDownloadEntries(photoIDs []int64, userID int64) ([]service.DownloadEntry, error) {
	return s.getDownloadEntries(photoIDs, userID)
}

func (s stubRegistrar) GetAlbumMedia(params storage.ListAlbumPhotosParams) (*storage.PhotoPage, error) {
	return s.getAlbumMedia(params)
}

func (s stubRegistrar) GetTrash(params storage.ListPhotosParams) (*storage.PhotoPage, error) {
	return s.getTrash(params)
}

func (s stubRegistrar) GetPhoto(id int64, userID int64) (*storage.Photo, error) {
	return s.getPhoto(id, userID)
}

func (s stubRegistrar) GetPhotoByUUIDAny(uuid string, userID int64) (*storage.Photo, error) {
	return s.getByUUID(uuid, userID)
}

func (s stubRegistrar) GetTimeline(params storage.ListPhotosParams) (*storage.PhotoPage, error) {
	return s.getTimeline(params)
}

func (s stubRegistrar) MediaPath(photo *storage.Photo) string {
	return s.mediaPath(photo)
}

func (s stubRegistrar) PosterPath(photo *storage.Photo) string {
	return s.posterPath(photo)
}

func (s stubRegistrar) PermanentlyDeletePhoto(id int64, userID int64) error {
	return s.permanentlyDeletePhoto(id, userID)
}

func (s stubRegistrar) RestorePhoto(id int64, userID int64) error {
	return s.restorePhoto(id, userID)
}

func okRegistrar() stubRegistrar {
	mediaFilePath := func(photo *storage.Photo) string {
		return filepath.Join(tTempStoragePath, photo.UUID+".mp4")
	}
	posterFilePath := func(photo *storage.Photo) string {
		return filepath.Join(tTempStoragePath, ".posters", photo.UUID+".jpg")
	}
	return stubRegistrar{addPhoto: func(albumID int64, photoID int64, userID int64) error {
		return nil
	}, createAlbum: func(name, description string, userID int64) (*storage.Album, error) {
		return &storage.Album{ID: 8, Name: name, Description: description, CreatedBy: userID, CreatedAt: time.Now()}, nil
	}, createShare: func(input service.CreateShareInput) (*storage.ShareLink, error) {
		return &storage.ShareLink{ID: 3, Token: "new-token", Type: input.Type, TargetID: input.TargetID, CreatedBy: input.UserID, ExpiresAt: input.ExpiresAt, CreatedAt: time.Now()}, nil
	}, deleteAlbum: func(id int64, userID int64) error {
		return nil
	}, deleteShare: func(id int64, userID int64) error {
		return nil
	}, getAlbum: func(id int64, userID int64) (*storage.Album, error) {
		coverID := int64(12)
		return &storage.Album{ID: id, Name: "旅行", Description: "相册描述", CreatedBy: userID, CoverPhotoID: &coverID, PhotoCount: 2}, nil
	}, getShareByToken: func(token string) (*storage.ShareLink, error) {
		if token == "missing" {
			return nil, nil
		}
		return &storage.ShareLink{ID: 4, Token: token, Type: storage.ShareTypePhoto, TargetID: 11, CreatedBy: 1, CreatedAt: time.Now()}, nil
	}, getAlbumDownloadEntries: func(albumID int64, userID int64) (string, []service.DownloadEntry, error) {
		return "旅行", []service.DownloadEntry{{FileName: "album.mp4", Path: mediaFilePath(&storage.Photo{UUID: "media-1"}), MimeType: "video/mp4"}}, nil
	}, listAlbums: func(userID int64) ([]*storage.Album, error) {
		coverID := int64(12)
		return []*storage.Album{
			{ID: 1, Name: "旅行", Description: "春游", CreatedBy: userID, CoverPhotoID: &coverID, PhotoCount: 2},
			{ID: 2, Name: "收藏", Description: "混合媒体", CreatedBy: userID, PhotoCount: 5},
		}, nil
	}, listShares: func(userID int64) ([]*storage.ShareLink, error) {
		return []*storage.ShareLink{
			{ID: 1, Token: "token-1", Type: storage.ShareTypePhoto, TargetID: 11, CreatedBy: userID, CreatedAt: time.Now()},
			{ID: 2, Token: "token-2", Type: storage.ShareTypeAlbum, TargetID: 8, CreatedBy: userID, CreatedAt: time.Now()},
		}, nil
	}, removePhoto: func(albumID int64, photoID int64, userID int64) error {
		return nil
	}, updateAlbum: func(id int64, name, description string, coverPhotoID *int64, userID int64) (*storage.Album, error) {
		return &storage.Album{ID: id, Name: name, Description: description, CreatedBy: userID, CoverPhotoID: coverPhotoID, CreatedAt: time.Now()}, nil
	}, register: func(input service.RegisterUploadedVideoInput) (*storage.Photo, error) {
		return &storage.Photo{
			ID:           99,
			UUID:         input.UUID,
			OriginalName: input.OriginalName,
			MediaKind:    storage.MediaKindVideo,
			MimeType:     input.MimeType,
			Size:         input.Size,
			Width:        input.Meta.Width,
			Height:       input.Meta.Height,
			DurationMS:   input.Meta.DurationMS,
			UploadedBy:   input.UploadedBy,
			TakenAt:      input.TakenAt,
			UploadedAt:   time.Now(),
		}, nil
	}, deletePhoto: func(id int64, userID int64) error {
		return nil
	}, emptyTrash: func(userID int64) error {
		return nil
	}, getDownloadEntries: func(photoIDs []int64, userID int64) ([]service.DownloadEntry, error) {
		entries := make([]service.DownloadEntry, 0, len(photoIDs))
		for _, id := range photoIDs {
			entries = append(entries, service.DownloadEntry{
				FileName: fmt.Sprintf("media-%d.mp4", id),
				Path:     mediaFilePath(&storage.Photo{UUID: fmt.Sprintf("media-%d", id)}),
				MimeType: "video/mp4",
			})
		}
		return entries, nil
	}, getAlbumMedia: func(params storage.ListAlbumPhotosParams) (*storage.PhotoPage, error) {
		return &storage.PhotoPage{Photos: []*storage.Photo{
			{
				ID:           11,
				UUID:         "album-image-1",
				OriginalName: "album.jpg",
				MediaKind:    storage.MediaKindImage,
				MimeType:     "image/jpeg",
				UploadedBy:   params.UserID,
			},
			{
				ID:           12,
				UUID:         "album-video-1",
				OriginalName: "album.mp4",
				MediaKind:    storage.MediaKindVideo,
				MimeType:     "video/mp4",
				UploadedBy:   params.UserID,
			},
		}, NextCursor: "", HasMore: false}, nil
	}, getPhoto: func(id int64, userID int64) (*storage.Photo, error) {
		return &storage.Photo{
			ID:           id,
			UUID:         fmt.Sprintf("media-%d", id),
			OriginalName: "demo.mp4",
			MediaKind:    storage.MediaKindVideo,
			MimeType:     "video/mp4",
			DurationMS:   12000,
			UploadedBy:   userID,
		}, nil
	}, getByUUID: func(uuid string, userID int64) (*storage.Photo, error) {
		return &storage.Photo{
			ID:           99,
			UUID:         uuid,
			OriginalName: "demo.mp4",
			MediaKind:    storage.MediaKindVideo,
			MimeType:     "video/mp4",
			UploadedBy:   userID,
		}, nil
	}, getTrash: func(params storage.ListPhotosParams) (*storage.PhotoPage, error) {
		return &storage.PhotoPage{Photos: []*storage.Photo{
			{
				ID:           3,
				UUID:         "trash-1",
				OriginalName: "trash.mp4",
				MediaKind:    storage.MediaKindVideo,
				MimeType:     "video/mp4",
				UploadedBy:   params.UserID,
			},
		}, NextCursor: "", HasMore: false}, nil
	}, getTimeline: func(params storage.ListPhotosParams) (*storage.PhotoPage, error) {
		return &storage.PhotoPage{Photos: []*storage.Photo{
			{
				ID:           1,
				UUID:         "image-1",
				OriginalName: "demo.jpg",
				MediaKind:    storage.MediaKindImage,
				MimeType:     "image/jpeg",
				UploadedBy:   params.UserID,
			},
			{
				ID:           2,
				UUID:         "video-1",
				OriginalName: "demo.mp4",
				MediaKind:    storage.MediaKindVideo,
				MimeType:     "video/mp4",
				DurationMS:   12000,
				UploadedBy:   params.UserID,
			},
		}, NextCursor: "", HasMore: false}, nil
	}, mediaPath: mediaFilePath, permanentlyDeletePhoto: func(id int64, userID int64) error {
		return nil
	}, posterPath: posterFilePath, restorePhoto: func(id int64, userID int64) error {
		return nil
	}}
}

func testConfig() *config.Config {
	return &config.Config{
		StoragePath: tTempStoragePath,
		JWTSecret:   "test-secret",
		Users: []config.User{
			{Username: "alice", PasswordHash: "ignored"},
		},
	}
}

const tTempStoragePath = "."

func testToken(t *testing.T, secret, username string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("生成测试 token 失败: %v", err)
	}
	return signed
}

func uploadRequest(t *testing.T, path string, filename string, payload []byte, withFile bool) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if withFile {
		part, err := writer.CreateFormFile("media", filename)
		if err != nil {
			t.Fatalf("创建表单文件失败: %v", err)
		}
		if _, err := part.Write(payload); err != nil {
			t.Fatalf("写入表单文件失败: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭 multipart writer 失败: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func mp4Sample() []byte {
	return append([]byte{
		0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70,
		0x69, 0x73, 0x6f, 0x6d, 0x00, 0x00, 0x02, 0x00,
		0x69, 0x73, 0x6f, 0x6d, 0x69, 0x73, 0x6f, 0x32,
	}, bytes.Repeat([]byte{0x00}, 1024)...)
}

func TestNewRouter_MediaPlaceholder(t *testing.T) {
	legacy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(testConfig(), legacy, okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	var page struct {
		Photos []struct {
			ID        int64  `json:"id"`
			MediaKind string `json:"media_kind"`
		} `json:"photos"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("解析媒体列表响应失败: %v", err)
	}
	if len(page.Photos) != 2 {
		t.Fatalf("期望 2 条媒体，得到 %d", len(page.Photos))
	}
	if page.Photos[1].MediaKind != storage.MediaKindVideo {
		t.Fatalf("期望第二条为视频，得到 %s", page.Photos[1].MediaKind)
	}
}

func TestMediaList_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestGetMedia_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestGetMedia_InvalidID(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/abc", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", w.Code)
	}
}

func TestGetMedia_Success(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/2", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	var photo struct {
		ID        int64  `json:"id"`
		MediaKind string `json:"media_kind"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &photo); err != nil {
		t.Fatalf("解析媒体详情响应失败: %v", err)
	}
	if photo.ID != 2 || photo.MediaKind != storage.MediaKindVideo {
		t.Fatalf("返回媒体详情不正确: %+v", photo)
	}
}

func TestGetMedia_NotFound(t *testing.T) {
	registrar := okRegistrar()
	registrar.getPhoto = func(id int64, userID int64) (*storage.Photo, error) {
		return nil, nil
	}
	router := NewRouter(testConfig(), http.NotFoundHandler(), registrar)
	req := httptest.NewRequest(http.MethodGet, "/api/media/99", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，得到 %d", w.Code)
	}
}

func TestDeleteMedia_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodDelete, "/api/media/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestDeleteMedia_InvalidID(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodDelete, "/api/media/abc", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", w.Code)
	}
}

func TestDeleteMedia_Success(t *testing.T) {
	called := false
	router := NewRouter(testConfig(), http.NotFoundHandler(), stubRegistrar{
		register:    okRegistrar().register,
		deletePhoto: func(id int64, userID int64) error { called = true; return nil },
		getPhoto:    okRegistrar().getPhoto,
		getByUUID:   okRegistrar().getByUUID,
		getTimeline: okRegistrar().getTimeline,
		mediaPath:   okRegistrar().mediaPath,
		posterPath:  okRegistrar().posterPath,
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/media/3", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if !called {
		t.Fatal("应调用删除逻辑")
	}
	var resp struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析删除响应失败: %v", err)
	}
	if resp.Message != "已移入回收站" {
		t.Fatalf("删除响应不正确: %+v", resp)
	}
}

func TestDeleteMedia_ReturnsServiceError(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), stubRegistrar{
		register:    okRegistrar().register,
		deletePhoto: func(id int64, userID int64) error { return fmt.Errorf("照片/视频不存在") },
		getPhoto:    okRegistrar().getPhoto,
		getByUUID:   okRegistrar().getByUUID,
		getTimeline: okRegistrar().getTimeline,
		mediaPath:   okRegistrar().mediaPath,
		posterPath:  okRegistrar().posterPath,
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/media/99", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", w.Code)
	}
}

func TestListAlbumMedia_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/albums/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestGetAlbumDetail_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/albums/1/detail", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestGetAlbumDetail_InvalidAlbumID(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/albums/abc/detail", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", w.Code)
	}
}

func TestGetAlbumDetail_Success(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/albums/5/detail", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	var album struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		PhotoCount  int    `json:"photo_count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &album); err != nil {
		t.Fatalf("解析相册详情响应失败: %v", err)
	}
	if album.ID != 5 || album.Name != "旅行" || album.PhotoCount != 2 {
		t.Fatalf("相册详情响应不正确: %+v", album)
	}
}

func TestListAlbumsMedia_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/albums", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestListAlbumsMedia_Success(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/albums", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	var albums []struct {
		ID         int64  `json:"id"`
		Name       string `json:"name"`
		PhotoCount int    `json:"photo_count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &albums); err != nil {
		t.Fatalf("解析相册列表响应失败: %v", err)
	}
	if len(albums) != 2 {
		t.Fatalf("期望 2 个相册，得到 %d", len(albums))
	}
	if albums[0].ID != 1 || albums[1].PhotoCount != 5 {
		t.Fatalf("相册列表响应不正确: %+v", albums)
	}
}

func TestCreateAlbumMedia_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodPost, "/api/media/albums", strings.NewReader(`{"name":"旅行"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestCreateAlbumMedia_Success(t *testing.T) {
	called := false
	router := NewRouter(testConfig(), http.NotFoundHandler(), stubRegistrar{
		addPhoto: func(albumID int64, photoID int64, userID int64) error { return nil },
		createAlbum: func(name, description string, userID int64) (*storage.Album, error) {
			called = true
			if name != "旅行" || description != "相册描述" {
				return nil, fmt.Errorf("unexpected payload: %s / %s", name, description)
			}
			return &storage.Album{ID: 8, Name: name, Description: description, CreatedBy: userID, CreatedAt: time.Now()}, nil
		},
		getAlbum:                okRegistrar().getAlbum,
		getAlbumDownloadEntries: okRegistrar().getAlbumDownloadEntries,
		listAlbums:              okRegistrar().listAlbums,
		removePhoto:             okRegistrar().removePhoto,
		register:                okRegistrar().register,
		deletePhoto:             okRegistrar().deletePhoto,
		emptyTrash:              okRegistrar().emptyTrash,
		getDownloadEntries:      okRegistrar().getDownloadEntries,
		getAlbumMedia:           okRegistrar().getAlbumMedia,
		getPhoto:                okRegistrar().getPhoto,
		getByUUID:               okRegistrar().getByUUID,
		getTrash:                okRegistrar().getTrash,
		getTimeline:             okRegistrar().getTimeline,
		mediaPath:               okRegistrar().mediaPath,
		permanentlyDeletePhoto:  okRegistrar().permanentlyDeletePhoto,
		posterPath:              okRegistrar().posterPath,
		restorePhoto:            okRegistrar().restorePhoto,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/media/albums", strings.NewReader(`{"name":"旅行","description":"相册描述"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("期望 201，得到 %d", w.Code)
	}
	if !called {
		t.Fatal("应调用新建相册逻辑")
	}
	var album struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &album); err != nil {
		t.Fatalf("解析新建相册响应失败: %v", err)
	}
	if album.ID != 8 || album.Name != "旅行" {
		t.Fatalf("新建相册响应不正确: %+v", album)
	}
}

func TestUpdateAlbumMedia_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodPut, "/api/media/albums/8", strings.NewReader(`{"name":"旅行 2"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestUpdateAlbumMedia_Success(t *testing.T) {
	called := false
	var gotCoverID *int64
	router := NewRouter(testConfig(), http.NotFoundHandler(), stubRegistrar{
		addPhoto:                func(albumID int64, photoID int64, userID int64) error { return nil },
		createAlbum:             okRegistrar().createAlbum,
		getAlbum:                okRegistrar().getAlbum,
		getAlbumDownloadEntries: okRegistrar().getAlbumDownloadEntries,
		listAlbums:              okRegistrar().listAlbums,
		removePhoto:             okRegistrar().removePhoto,
		updateAlbum: func(id int64, name, description string, coverPhotoID *int64, userID int64) (*storage.Album, error) {
			called = true
			gotCoverID = coverPhotoID
			return &storage.Album{ID: id, Name: name, Description: description, CreatedBy: userID, CoverPhotoID: coverPhotoID, CreatedAt: time.Now()}, nil
		},
		register:               okRegistrar().register,
		deletePhoto:            okRegistrar().deletePhoto,
		emptyTrash:             okRegistrar().emptyTrash,
		getDownloadEntries:     okRegistrar().getDownloadEntries,
		getAlbumMedia:          okRegistrar().getAlbumMedia,
		getPhoto:               okRegistrar().getPhoto,
		getByUUID:              okRegistrar().getByUUID,
		getTrash:               okRegistrar().getTrash,
		getTimeline:            okRegistrar().getTimeline,
		mediaPath:              okRegistrar().mediaPath,
		permanentlyDeletePhoto: okRegistrar().permanentlyDeletePhoto,
		posterPath:             okRegistrar().posterPath,
		restorePhoto:           okRegistrar().restorePhoto,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/media/albums/8", strings.NewReader(`{"name":"旅行 2","description":"更新描述","cover_photo_id":12}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if !called {
		t.Fatal("应调用更新相册逻辑")
	}
	if gotCoverID == nil || *gotCoverID != 12 {
		t.Fatalf("封面参数不正确: %+v", gotCoverID)
	}
	var album struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &album); err != nil {
		t.Fatalf("解析更新相册响应失败: %v", err)
	}
	if album.ID != 8 || album.Name != "旅行 2" {
		t.Fatalf("更新相册响应不正确: %+v", album)
	}
}

func TestDeleteAlbumMedia_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodDelete, "/api/media/albums/8", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestDeleteAlbumMedia_Success(t *testing.T) {
	called := false
	router := NewRouter(testConfig(), http.NotFoundHandler(), stubRegistrar{
		addPhoto:                func(albumID int64, photoID int64, userID int64) error { return nil },
		createAlbum:             okRegistrar().createAlbum,
		deleteAlbum:             func(id int64, userID int64) error { called = true; return nil },
		getAlbum:                okRegistrar().getAlbum,
		getAlbumDownloadEntries: okRegistrar().getAlbumDownloadEntries,
		listAlbums:              okRegistrar().listAlbums,
		removePhoto:             okRegistrar().removePhoto,
		updateAlbum:             okRegistrar().updateAlbum,
		register:                okRegistrar().register,
		deletePhoto:             okRegistrar().deletePhoto,
		emptyTrash:              okRegistrar().emptyTrash,
		getDownloadEntries:      okRegistrar().getDownloadEntries,
		getAlbumMedia:           okRegistrar().getAlbumMedia,
		getPhoto:                okRegistrar().getPhoto,
		getByUUID:               okRegistrar().getByUUID,
		getTrash:                okRegistrar().getTrash,
		getTimeline:             okRegistrar().getTimeline,
		mediaPath:               okRegistrar().mediaPath,
		permanentlyDeletePhoto:  okRegistrar().permanentlyDeletePhoto,
		posterPath:              okRegistrar().posterPath,
		restorePhoto:            okRegistrar().restorePhoto,
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/media/albums/8", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if !called {
		t.Fatal("应调用删除相册逻辑")
	}
	var resp struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析删除相册响应失败: %v", err)
	}
	if resp.Message != "相册已删除" {
		t.Fatalf("删除相册响应不正确: %+v", resp)
	}
}

func TestListSharesMedia_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/shares", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestListSharesMedia_Success(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/shares", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	var links []struct {
		ID    int64  `json:"id"`
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &links); err != nil {
		t.Fatalf("解析分享列表响应失败: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("期望 2 条分享，得到 %d", len(links))
	}
	if links[0].Token != "token-1" || links[1].Type != storage.ShareTypeAlbum {
		t.Fatalf("分享列表响应不正确: %+v", links)
	}
}

func TestCreateShareMedia_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodPost, "/api/media/shares", strings.NewReader(`{"type":"photo","target_id":11}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestCreateShareMedia_Success(t *testing.T) {
	called := false
	var gotInput service.CreateShareInput
	router := NewRouter(testConfig(), http.NotFoundHandler(), stubRegistrar{
		addPhoto:    okRegistrar().addPhoto,
		createAlbum: okRegistrar().createAlbum,
		createShare: func(input service.CreateShareInput) (*storage.ShareLink, error) {
			called = true
			gotInput = input
			return &storage.ShareLink{ID: 3, Token: "new-token", Type: input.Type, TargetID: input.TargetID, CreatedBy: input.UserID, ExpiresAt: input.ExpiresAt, CreatedAt: time.Now()}, nil
		},
		deleteAlbum:             okRegistrar().deleteAlbum,
		getAlbum:                okRegistrar().getAlbum,
		getAlbumDownloadEntries: okRegistrar().getAlbumDownloadEntries,
		listAlbums:              okRegistrar().listAlbums,
		listShares:              okRegistrar().listShares,
		removePhoto:             okRegistrar().removePhoto,
		updateAlbum:             okRegistrar().updateAlbum,
		register:                okRegistrar().register,
		deletePhoto:             okRegistrar().deletePhoto,
		emptyTrash:              okRegistrar().emptyTrash,
		getDownloadEntries:      okRegistrar().getDownloadEntries,
		getAlbumMedia:           okRegistrar().getAlbumMedia,
		getPhoto:                okRegistrar().getPhoto,
		getByUUID:               okRegistrar().getByUUID,
		getTrash:                okRegistrar().getTrash,
		getTimeline:             okRegistrar().getTimeline,
		mediaPath:               okRegistrar().mediaPath,
		permanentlyDeletePhoto:  okRegistrar().permanentlyDeletePhoto,
		posterPath:              okRegistrar().posterPath,
		restorePhoto:            okRegistrar().restorePhoto,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/media/shares", strings.NewReader(`{"type":"photo","target_id":11,"expires_in_days":7}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("期望 201，得到 %d", w.Code)
	}
	if !called {
		t.Fatal("应调用创建分享逻辑")
	}
	if gotInput.Type != storage.ShareTypePhoto || gotInput.TargetID != 11 || gotInput.ExpiresAt == nil {
		t.Fatalf("创建分享参数不正确: %+v", gotInput)
	}
	var link struct {
		ID    int64  `json:"id"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &link); err != nil {
		t.Fatalf("解析创建分享响应失败: %v", err)
	}
	if link.ID != 3 || link.Token != "new-token" {
		t.Fatalf("创建分享响应不正确: %+v", link)
	}
}

func TestDeleteShareMedia_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodDelete, "/api/media/shares/3", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestDeleteShareMedia_Success(t *testing.T) {
	called := false
	router := NewRouter(testConfig(), http.NotFoundHandler(), stubRegistrar{
		addPhoto:                okRegistrar().addPhoto,
		createAlbum:             okRegistrar().createAlbum,
		createShare:             okRegistrar().createShare,
		deleteAlbum:             okRegistrar().deleteAlbum,
		deleteShare:             func(id int64, userID int64) error { called = true; return nil },
		getAlbum:                okRegistrar().getAlbum,
		getAlbumDownloadEntries: okRegistrar().getAlbumDownloadEntries,
		listAlbums:              okRegistrar().listAlbums,
		listShares:              okRegistrar().listShares,
		removePhoto:             okRegistrar().removePhoto,
		updateAlbum:             okRegistrar().updateAlbum,
		register:                okRegistrar().register,
		deletePhoto:             okRegistrar().deletePhoto,
		emptyTrash:              okRegistrar().emptyTrash,
		getDownloadEntries:      okRegistrar().getDownloadEntries,
		getAlbumMedia:           okRegistrar().getAlbumMedia,
		getPhoto:                okRegistrar().getPhoto,
		getByUUID:               okRegistrar().getByUUID,
		getTrash:                okRegistrar().getTrash,
		getTimeline:             okRegistrar().getTimeline,
		mediaPath:               okRegistrar().mediaPath,
		permanentlyDeletePhoto:  okRegistrar().permanentlyDeletePhoto,
		posterPath:              okRegistrar().posterPath,
		restorePhoto:            okRegistrar().restorePhoto,
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/media/shares/3", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if !called {
		t.Fatal("应调用删除分享逻辑")
	}
	var resp struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析删除分享响应失败: %v", err)
	}
	if resp.Message != "分享链接已删除" {
		t.Fatalf("删除分享响应不正确: %+v", resp)
	}
}

func TestGetShareByToken_NotFound(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/s/missing", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，得到 %d", w.Code)
	}
}

func TestGetShareByToken_Success(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/s/token-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	var link struct {
		Token string `json:"token"`
		Type  string `json:"type"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &link); err != nil {
		t.Fatalf("解析分享详情响应失败: %v", err)
	}
	if link.Token != "token-1" || link.Type != storage.ShareTypePhoto {
		t.Fatalf("分享详情响应不正确: %+v", link)
	}
}

func TestDownloadAlbumMedia_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/albums/1/download", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestDownloadAlbumMedia_Success(t *testing.T) {
	cfg := testConfig()
	storageDir := t.TempDir()
	cfg.StoragePath = storageDir
	mediaFile := filepath.Join(storageDir, "album-video.mp4")
	if err := os.WriteFile(mediaFile, []byte("album-video"), 0644); err != nil {
		t.Fatalf("创建测试相册媒体文件失败: %v", err)
	}
	router := NewRouter(cfg, http.NotFoundHandler(), stubRegistrar{
		addPhoto: func(albumID int64, photoID int64, userID int64) error { return nil },
		getAlbum: okRegistrar().getAlbum,
		getAlbumDownloadEntries: func(albumID int64, userID int64) (string, []service.DownloadEntry, error) {
			return "旅行/2026", []service.DownloadEntry{{FileName: "clip.mp4", Path: mediaFile, MimeType: "video/mp4"}}, nil
		},
		listAlbums:             okRegistrar().listAlbums,
		removePhoto:            okRegistrar().removePhoto,
		register:               okRegistrar().register,
		deletePhoto:            okRegistrar().deletePhoto,
		emptyTrash:             okRegistrar().emptyTrash,
		getDownloadEntries:     okRegistrar().getDownloadEntries,
		getAlbumMedia:          okRegistrar().getAlbumMedia,
		getPhoto:               okRegistrar().getPhoto,
		getByUUID:              okRegistrar().getByUUID,
		getTrash:               okRegistrar().getTrash,
		getTimeline:            okRegistrar().getTimeline,
		mediaPath:              okRegistrar().mediaPath,
		permanentlyDeletePhoto: okRegistrar().permanentlyDeletePhoto,
		posterPath:             okRegistrar().posterPath,
		restorePhoto:           okRegistrar().restorePhoto,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/media/albums/5/download", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/zip") {
		t.Fatalf("Content-Type 不正确: %s", ct)
	}
	disposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "旅行-2026.zip") {
		t.Fatalf("下载头不正确: %s", disposition)
	}
	zr, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if err != nil {
		t.Fatalf("解析 zip 失败: %v", err)
	}
	if len(zr.File) != 1 || zr.File[0].Name != "clip.mp4" {
		t.Fatalf("zip 条目不正确: %+v", zr.File)
	}
}

func TestListAlbumMedia_InvalidAlbumID(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/albums/abc", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", w.Code)
	}
}

func TestListAlbumMedia_Success(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/albums/5", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	var page struct {
		Photos []struct {
			ID        int64  `json:"id"`
			MediaKind string `json:"media_kind"`
		} `json:"photos"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("解析相册媒体响应失败: %v", err)
	}
	if len(page.Photos) != 2 {
		t.Fatalf("期望 2 条相册媒体，得到 %d", len(page.Photos))
	}
	if page.Photos[1].MediaKind != storage.MediaKindVideo {
		t.Fatalf("期望第二条为视频，得到 %s", page.Photos[1].MediaKind)
	}
}

func TestAddMediaToAlbum_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodPost, "/api/media/albums/1", strings.NewReader(`{"media_id":9}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestAddMediaToAlbum_UsesMediaID(t *testing.T) {
	called := false
	var gotAlbumID, gotMediaID int64
	router := NewRouter(testConfig(), http.NotFoundHandler(), stubRegistrar{
		addPhoto: func(albumID int64, photoID int64, userID int64) error {
			called = true
			gotAlbumID = albumID
			gotMediaID = photoID
			return nil
		},
		register:               okRegistrar().register,
		deletePhoto:            okRegistrar().deletePhoto,
		emptyTrash:             okRegistrar().emptyTrash,
		getDownloadEntries:     okRegistrar().getDownloadEntries,
		getAlbumMedia:          okRegistrar().getAlbumMedia,
		getPhoto:               okRegistrar().getPhoto,
		getByUUID:              okRegistrar().getByUUID,
		getTrash:               okRegistrar().getTrash,
		getTimeline:            okRegistrar().getTimeline,
		mediaPath:              okRegistrar().mediaPath,
		permanentlyDeletePhoto: okRegistrar().permanentlyDeletePhoto,
		posterPath:             okRegistrar().posterPath,
		restorePhoto:           okRegistrar().restorePhoto,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/media/albums/3", strings.NewReader(`{"media_id":9}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if !called {
		t.Fatal("应调用添加到相册逻辑")
	}
	if gotAlbumID != 3 || gotMediaID != 9 {
		t.Fatalf("透传参数不正确: album=%d media=%d", gotAlbumID, gotMediaID)
	}
}

func TestAddMediaToAlbum_AcceptsLegacyPhotoID(t *testing.T) {
	called := false
	router := NewRouter(testConfig(), http.NotFoundHandler(), stubRegistrar{
		addPhoto: func(albumID int64, photoID int64, userID int64) error {
			called = true
			if albumID != 3 || photoID != 7 {
				return fmt.Errorf("unexpected params: album=%d photo=%d", albumID, photoID)
			}
			return nil
		},
		register:               okRegistrar().register,
		deletePhoto:            okRegistrar().deletePhoto,
		emptyTrash:             okRegistrar().emptyTrash,
		getDownloadEntries:     okRegistrar().getDownloadEntries,
		getAlbumMedia:          okRegistrar().getAlbumMedia,
		getPhoto:               okRegistrar().getPhoto,
		getByUUID:              okRegistrar().getByUUID,
		getTrash:               okRegistrar().getTrash,
		getTimeline:            okRegistrar().getTimeline,
		mediaPath:              okRegistrar().mediaPath,
		permanentlyDeletePhoto: okRegistrar().permanentlyDeletePhoto,
		posterPath:             okRegistrar().posterPath,
		restorePhoto:           okRegistrar().restorePhoto,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/media/albums/3", strings.NewReader(`{"photo_id":7}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if !called {
		t.Fatal("应调用兼容 photo_id 的添加逻辑")
	}
}

func TestRemoveMediaFromAlbum_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodDelete, "/api/media/albums/1/9", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestRemoveMediaFromAlbum_InvalidMediaID(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodDelete, "/api/media/albums/1/abc", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", w.Code)
	}
}

func TestRemoveMediaFromAlbum_Success(t *testing.T) {
	called := false
	var gotAlbumID, gotMediaID int64
	router := NewRouter(testConfig(), http.NotFoundHandler(), stubRegistrar{
		addPhoto: func(albumID int64, photoID int64, userID int64) error {
			return nil
		},
		removePhoto: func(albumID int64, photoID int64, userID int64) error {
			called = true
			gotAlbumID = albumID
			gotMediaID = photoID
			return nil
		},
		register:               okRegistrar().register,
		deletePhoto:            okRegistrar().deletePhoto,
		emptyTrash:             okRegistrar().emptyTrash,
		getDownloadEntries:     okRegistrar().getDownloadEntries,
		getAlbumMedia:          okRegistrar().getAlbumMedia,
		getPhoto:               okRegistrar().getPhoto,
		getByUUID:              okRegistrar().getByUUID,
		getTrash:               okRegistrar().getTrash,
		getTimeline:            okRegistrar().getTimeline,
		mediaPath:              okRegistrar().mediaPath,
		permanentlyDeletePhoto: okRegistrar().permanentlyDeletePhoto,
		posterPath:             okRegistrar().posterPath,
		restorePhoto:           okRegistrar().restorePhoto,
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/media/albums/3/9", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if !called {
		t.Fatal("应调用从相册移除逻辑")
	}
	if gotAlbumID != 3 || gotMediaID != 9 {
		t.Fatalf("透传参数不正确: album=%d media=%d", gotAlbumID, gotMediaID)
	}
}

func TestListTrashMedia_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/trash", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestListTrashMedia_Success(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/trash", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	var page struct {
		Photos []struct {
			ID int64 `json:"id"`
		} `json:"photos"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("解析回收站响应失败: %v", err)
	}
	if len(page.Photos) != 1 || page.Photos[0].ID != 3 {
		t.Fatalf("回收站响应不正确: %+v", page)
	}
}

func TestRestoreMedia_Success(t *testing.T) {
	called := false
	router := NewRouter(testConfig(), http.NotFoundHandler(), stubRegistrar{
		register:               okRegistrar().register,
		deletePhoto:            okRegistrar().deletePhoto,
		emptyTrash:             okRegistrar().emptyTrash,
		getDownloadEntries:     okRegistrar().getDownloadEntries,
		getPhoto:               okRegistrar().getPhoto,
		getByUUID:              okRegistrar().getByUUID,
		getTrash:               okRegistrar().getTrash,
		getTimeline:            okRegistrar().getTimeline,
		mediaPath:              okRegistrar().mediaPath,
		permanentlyDeletePhoto: okRegistrar().permanentlyDeletePhoto,
		posterPath:             okRegistrar().posterPath,
		restorePhoto:           func(id int64, userID int64) error { called = true; return nil },
	})
	req := httptest.NewRequest(http.MethodPost, "/api/media/6/restore", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if !called {
		t.Fatal("应调用恢复逻辑")
	}
}

func TestHardDeleteMedia_Success(t *testing.T) {
	called := false
	router := NewRouter(testConfig(), http.NotFoundHandler(), stubRegistrar{
		register:               okRegistrar().register,
		deletePhoto:            okRegistrar().deletePhoto,
		emptyTrash:             okRegistrar().emptyTrash,
		getDownloadEntries:     okRegistrar().getDownloadEntries,
		getPhoto:               okRegistrar().getPhoto,
		getByUUID:              okRegistrar().getByUUID,
		getTrash:               okRegistrar().getTrash,
		getTimeline:            okRegistrar().getTimeline,
		mediaPath:              okRegistrar().mediaPath,
		permanentlyDeletePhoto: func(id int64, userID int64) error { called = true; return nil },
		posterPath:             okRegistrar().posterPath,
		restorePhoto:           okRegistrar().restorePhoto,
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/media/trash/6", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if !called {
		t.Fatal("应调用永久删除逻辑")
	}
}

func TestEmptyTrashMedia_Success(t *testing.T) {
	called := false
	router := NewRouter(testConfig(), http.NotFoundHandler(), stubRegistrar{
		register:               okRegistrar().register,
		deletePhoto:            okRegistrar().deletePhoto,
		emptyTrash:             func(userID int64) error { called = true; return nil },
		getDownloadEntries:     okRegistrar().getDownloadEntries,
		getPhoto:               okRegistrar().getPhoto,
		getByUUID:              okRegistrar().getByUUID,
		getTrash:               okRegistrar().getTrash,
		getTimeline:            okRegistrar().getTimeline,
		mediaPath:              okRegistrar().mediaPath,
		permanentlyDeletePhoto: okRegistrar().permanentlyDeletePhoto,
		posterPath:             okRegistrar().posterPath,
		restorePhoto:           okRegistrar().restorePhoto,
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/media/trash", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if !called {
		t.Fatal("应调用清空回收站逻辑")
	}
}

func TestDownloadMedia_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/1/download", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestDownloadMedia_InvalidID(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/api/media/abc/download", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", w.Code)
	}
}

func TestDownloadMedia_Success(t *testing.T) {
	cfg := testConfig()
	storageDir := t.TempDir()
	cfg.StoragePath = storageDir
	mediaFile := filepath.Join(storageDir, "media-2.mp4")
	if err := os.WriteFile(mediaFile, []byte("video-download"), 0644); err != nil {
		t.Fatalf("创建测试媒体文件失败: %v", err)
	}
	router := NewRouter(cfg, http.NotFoundHandler(), stubRegistrar{
		register: okRegistrar().register,
		getPhoto: func(id int64, userID int64) (*storage.Photo, error) {
			return &storage.Photo{ID: id, UUID: "media-2", OriginalName: "demo video.mp4", MediaKind: storage.MediaKindVideo, MimeType: "video/mp4", UploadedBy: userID}, nil
		},
		getByUUID:   okRegistrar().getByUUID,
		getTimeline: okRegistrar().getTimeline,
		mediaPath:   func(photo *storage.Photo) string { return mediaFile },
		posterPath:  okRegistrar().posterPath,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/media/2/download", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if body := w.Body.String(); body != "video-download" {
		t.Fatalf("下载内容不正确: %s", body)
	}
	disposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "attachment;") || !strings.Contains(disposition, "demo video.mp4") {
		t.Fatalf("下载头不正确: %s", disposition)
	}
	if ct := w.Header().Get("Content-Type"); ct != "video/mp4" {
		t.Fatalf("Content-Type 不正确: %s", ct)
	}
}

func TestDownloadMedia_NotFound(t *testing.T) {
	registrar := okRegistrar()
	registrar.getPhoto = func(id int64, userID int64) (*storage.Photo, error) {
		return nil, nil
	}
	router := NewRouter(testConfig(), http.NotFoundHandler(), registrar)
	req := httptest.NewRequest(http.MethodGet, "/api/media/99/download", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，得到 %d", w.Code)
	}
}

func TestDownloadMediaBatch_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodPost, "/api/media/download", strings.NewReader(`{"media_ids":[1]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestDownloadMediaBatch_UsesMediaIDsAndReturnsZip(t *testing.T) {
	cfg := testConfig()
	storageDir := t.TempDir()
	cfg.StoragePath = storageDir
	firstFile := filepath.Join(storageDir, "media-1.mp4")
	secondFile := filepath.Join(storageDir, "media-2.mp4")
	if err := os.WriteFile(firstFile, []byte("video-one"), 0644); err != nil {
		t.Fatalf("创建测试媒体文件失败: %v", err)
	}
	if err := os.WriteFile(secondFile, []byte("video-two"), 0644); err != nil {
		t.Fatalf("创建测试媒体文件失败: %v", err)
	}

	var gotIDs []int64
	router := NewRouter(cfg, http.NotFoundHandler(), stubRegistrar{
		register:    okRegistrar().register,
		deletePhoto: okRegistrar().deletePhoto,
		getDownloadEntries: func(photoIDs []int64, userID int64) ([]service.DownloadEntry, error) {
			gotIDs = append([]int64(nil), photoIDs...)
			return []service.DownloadEntry{
				{FileName: "clip-a.mp4", Path: firstFile, MimeType: "video/mp4"},
				{FileName: "clip-b.mp4", Path: secondFile, MimeType: "video/mp4"},
			}, nil
		},
		getPhoto:    okRegistrar().getPhoto,
		getByUUID:   okRegistrar().getByUUID,
		getTimeline: okRegistrar().getTimeline,
		mediaPath:   okRegistrar().mediaPath,
		posterPath:  okRegistrar().posterPath,
	})
	body := bytes.NewBufferString(`{"media_ids":[7,9]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/media/download", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if len(gotIDs) != 2 || gotIDs[0] != 7 || gotIDs[1] != 9 {
		t.Fatalf("media_ids 透传不正确: %+v", gotIDs)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/zip") {
		t.Fatalf("Content-Type 不正确: %s", ct)
	}

	zr, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if err != nil {
		t.Fatalf("解析 zip 失败: %v", err)
	}
	if len(zr.File) != 2 {
		t.Fatalf("期望 2 个文件，得到 %d", len(zr.File))
	}
	file, err := zr.File[0].Open()
	if err != nil {
		t.Fatalf("打开 zip 条目失败: %v", err)
	}
	data, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil {
		t.Fatalf("读取 zip 条目失败: %v", err)
	}
	if string(data) != "video-one" {
		t.Fatalf("zip 内容不正确: %s", string(data))
	}
}

func TestDownloadMediaBatch_AcceptsLegacyPhotoIDs(t *testing.T) {
	called := false
	router := NewRouter(testConfig(), http.NotFoundHandler(), stubRegistrar{
		register:    okRegistrar().register,
		deletePhoto: okRegistrar().deletePhoto,
		getDownloadEntries: func(photoIDs []int64, userID int64) ([]service.DownloadEntry, error) {
			called = true
			if len(photoIDs) != 1 || photoIDs[0] != 5 {
				return nil, fmt.Errorf("unexpected ids: %+v", photoIDs)
			}
			return []service.DownloadEntry{}, nil
		},
		getPhoto:    okRegistrar().getPhoto,
		getByUUID:   okRegistrar().getByUUID,
		getTimeline: okRegistrar().getTimeline,
		mediaPath:   okRegistrar().mediaPath,
		posterPath:  okRegistrar().posterPath,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/media/download", strings.NewReader(`{"photo_ids":[5]}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, testConfig().JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if !called {
		t.Fatal("应调用批量下载逻辑")
	}
}

func TestNewRouter_FallsBackToLegacyHandler(t *testing.T) {
	legacy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			t.Fatalf("期望回退到 /login，得到 %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusTeapot)
	})

	router := NewRouter(testConfig(), legacy, okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusTeapot {
		t.Fatalf("期望 418，得到 %d", w.Code)
	}
}

func TestUploadPlaceholder_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := uploadRequest(t, "/api/media/upload", "demo.mp4", mp4Sample(), true)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestUploadPlaceholder_RequiresMediaField(t *testing.T) {
	cfg := testConfig()
	router := NewRouter(cfg, http.NotFoundHandler(), okRegistrar())
	req := uploadRequest(t, "/api/media/upload", "", nil, false)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", w.Code)
	}
}

func TestUploadPlaceholder_RejectsNonMP4Extension(t *testing.T) {
	cfg := testConfig()
	cfg.StoragePath = t.TempDir()
	router := NewRouter(cfg, http.NotFoundHandler(), okRegistrar())
	req := uploadRequest(t, "/api/media/upload", "demo.mov", mp4Sample(), true)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", w.Code)
	}
}

func TestUploadPlaceholder_SavesFinalFileAndRecordAfterValidation(t *testing.T) {
	cfg := testConfig()
	cfg.StoragePath = t.TempDir()
	router := NewRouter(cfg, http.NotFoundHandler(), okRegistrar())
	req := uploadRequest(t, "/api/media/upload", "demo.mp4", mp4Sample(), true)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("期望 201，得到 %d", w.Code)
	}

	var resp struct {
		Message     string `json:"message"`
		Filename    string `json:"filename"`
		Path        string `json:"path"`
		PosterPath  string `json:"poster_path"`
		PosterError string `json:"poster_error"`
		ProbeError  string `json:"probe_error"`
		Meta        struct {
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			DurationMS int64  `json:"duration_ms"`
			FormatName string `json:"format_name"`
			CodecName  string `json:"codec_name"`
		} `json:"meta"`
		Photo struct {
			ID         int64  `json:"id"`
			UUID       string `json:"uuid"`
			MediaKind  string `json:"media_kind"`
			DurationMS int64  `json:"duration_ms"`
		} `json:"photo"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Filename != "demo.mp4" {
		t.Fatalf("期望文件名 demo.mp4，得到 %s", resp.Filename)
	}
	if resp.Path == "" {
		t.Fatal("响应中应返回最终文件路径")
	}
	if resp.Meta.DurationMS != 0 || resp.Meta.Width != 0 || resp.Meta.Height != 0 || resp.Meta.FormatName != "mp4" {
		t.Fatalf("返回的元数据不正确: %+v", resp.Meta)
	}
	if resp.Photo.ID != 99 || resp.Photo.MediaKind != storage.MediaKindVideo || resp.Photo.DurationMS != 0 {
		t.Fatalf("返回的媒体记录不正确: %+v", resp.Photo)
	}
	if resp.PosterPath != "" || resp.PosterError != "" {
		t.Fatalf("poster 返回不正确: path=%s err=%s", resp.PosterPath, resp.PosterError)
	}
	if resp.ProbeError != "" {
		t.Fatalf("正常探测时不应返回 probe 错误: %s", resp.ProbeError)
	}
	if filepath.Dir(resp.Path) != cfg.StoragePath {
		t.Fatalf("最终文件目录不正确: %s", resp.Path)
	}
	if filepath.Base(resp.Path) != resp.Photo.UUID+".mp4" {
		t.Fatalf("最终文件名应与 UUID 对应，得到 %s", filepath.Base(resp.Path))
	}
	data, err := os.ReadFile(resp.Path)
	if err != nil {
		t.Fatalf("读取临时文件失败: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("临时保存的文件内容不能为空")
	}
}

func TestUploadPlaceholder_CleansFileWhenRegisterFails(t *testing.T) {
	cfg := testConfig()
	cfg.StoragePath = t.TempDir()
	router := NewRouter(cfg, http.NotFoundHandler(), stubRegistrar{register: func(input service.RegisterUploadedVideoInput) (*storage.Photo, error) {
		return nil, fmt.Errorf("保存视频记录失败")
	}})
	req := uploadRequest(t, "/api/media/upload", "demo.mp4", mp4Sample(), true)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，得到 %d", w.Code)
	}
	entries, err := os.ReadDir(cfg.StoragePath)
	if err != nil {
		t.Fatalf("读取存储目录失败: %v", err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".mp4") {
			t.Fatalf("注册失败后不应残留视频文件: %s", entry.Name())
		}
	}
}

func TestServeMediaFile_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler(), okRegistrar())
	req := httptest.NewRequest(http.MethodGet, "/media/files/video-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestServeMediaFile_Success(t *testing.T) {
	cfg := testConfig()
	storageDir := t.TempDir()
	cfg.StoragePath = storageDir
	mediaFile := filepath.Join(storageDir, "video-1.mp4")
	if err := os.WriteFile(mediaFile, []byte("video"), 0644); err != nil {
		t.Fatalf("创建测试视频失败: %v", err)
	}
	router := NewRouter(cfg, http.NotFoundHandler(), stubRegistrar{
		register: okRegistrar().register,
		getByUUID: func(uuid string, userID int64) (*storage.Photo, error) {
			return &storage.Photo{UUID: uuid, OriginalName: "demo.mp4", MediaKind: storage.MediaKindVideo, MimeType: "video/mp4", UploadedBy: userID}, nil
		},
		mediaPath:  func(photo *storage.Photo) string { return mediaFile },
		posterPath: func(photo *storage.Photo) string { return filepath.Join(storageDir, ".posters", photo.UUID+".jpg") },
	})
	req := httptest.NewRequest(http.MethodGet, "/media/files/video-1", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if body := w.Body.String(); body != "video" {
		t.Fatalf("返回内容不正确: %s", body)
	}
}

func TestServePoster_Success(t *testing.T) {
	cfg := testConfig()
	storageDir := t.TempDir()
	cfg.StoragePath = storageDir
	posterFile := filepath.Join(storageDir, ".posters", "video-1.jpg")
	if err := os.MkdirAll(filepath.Dir(posterFile), 0755); err != nil {
		t.Fatalf("创建 poster 目录失败: %v", err)
	}
	if err := os.WriteFile(posterFile, []byte("jpg"), 0644); err != nil {
		t.Fatalf("创建测试 poster 失败: %v", err)
	}
	router := NewRouter(cfg, http.NotFoundHandler(), stubRegistrar{
		register: okRegistrar().register,
		getByUUID: func(uuid string, userID int64) (*storage.Photo, error) {
			return &storage.Photo{UUID: uuid, OriginalName: "demo.mp4", MediaKind: storage.MediaKindVideo, MimeType: "video/mp4", UploadedBy: userID}, nil
		},
		mediaPath:  func(photo *storage.Photo) string { return filepath.Join(storageDir, photo.UUID+".mp4") },
		posterPath: func(photo *storage.Photo) string { return posterFile },
	})
	req := httptest.NewRequest(http.MethodGet, "/media/posters/video-1", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
	if body := w.Body.String(); body != "jpg" {
		t.Fatalf("返回内容不正确: %s", body)
	}
}

func TestServePoster_Returns404WhenMissing(t *testing.T) {
	cfg := testConfig()
	storageDir := t.TempDir()
	cfg.StoragePath = storageDir
	router := NewRouter(cfg, http.NotFoundHandler(), stubRegistrar{
		register: okRegistrar().register,
		getByUUID: func(uuid string, userID int64) (*storage.Photo, error) {
			return &storage.Photo{UUID: uuid, OriginalName: "demo.mp4", MediaKind: storage.MediaKindVideo, MimeType: "video/mp4", UploadedBy: userID}, nil
		},
		mediaPath:  func(photo *storage.Photo) string { return filepath.Join(storageDir, photo.UUID+".mp4") },
		posterPath: func(photo *storage.Photo) string { return filepath.Join(storageDir, ".posters", photo.UUID+".jpg") },
	})
	req := httptest.NewRequest(http.MethodGet, "/media/posters/video-1", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，得到 %d", w.Code)
	}
}
