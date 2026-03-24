package api

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	register    func(input service.RegisterUploadedVideoInput) (*storage.Photo, error)
	getPhoto    func(id int64, userID int64) (*storage.Photo, error)
	getByUUID   func(uuid string, userID int64) (*storage.Photo, error)
	getTimeline func(params storage.ListPhotosParams) (*storage.PhotoPage, error)
	mediaPath   func(photo *storage.Photo) string
	posterPath  func(photo *storage.Photo) string
}

func (s stubRegistrar) RegisterUploadedVideo(input service.RegisterUploadedVideoInput) (*storage.Photo, error) {
	return s.register(input)
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

func okRegistrar() stubRegistrar {
	mediaFilePath := func(photo *storage.Photo) string {
		return filepath.Join(tTempStoragePath, photo.UUID+".mp4")
	}
	posterFilePath := func(photo *storage.Photo) string {
		return filepath.Join(tTempStoragePath, ".posters", photo.UUID+".jpg")
	}
	return stubRegistrar{register: func(input service.RegisterUploadedVideoInput) (*storage.Photo, error) {
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
	}, mediaPath: mediaFilePath, posterPath: posterFilePath}
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
