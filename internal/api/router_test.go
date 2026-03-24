package api

import (
	"bytes"
	"encoding/json"
	"errors"
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
	"photoalbum/internal/media"
	"photoalbum/internal/service"
	"photoalbum/internal/storage"
)

type stubRegistrar struct {
	register func(input service.RegisterUploadedVideoInput) (*storage.Photo, error)
}

func (s stubRegistrar) RegisterUploadedVideo(input service.RegisterUploadedVideoInput) (*storage.Photo, error) {
	return s.register(input)
}

func okRegistrar() stubRegistrar {
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
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("期望 501，得到 %d", w.Code)
	}
	if body := w.Body.String(); body == "" {
		t.Fatal("占位接口应该返回错误信息")
	}
}

func withProbeStub(t *testing.T, stub func(string) (*media.VideoMeta, error)) {
	t.Helper()
	old := probeVideoFunc
	probeVideoFunc = stub
	t.Cleanup(func() {
		probeVideoFunc = old
	})
}

func withPosterStub(t *testing.T, stub func(string, string) error) {
	t.Helper()
	old := generatePosterFunc
	generatePosterFunc = stub
	t.Cleanup(func() {
		generatePosterFunc = old
	})
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
	withProbeStub(t, func(path string) (*media.VideoMeta, error) {
		return &media.VideoMeta{Width: 1920, Height: 1080, DurationMS: 12345, FormatName: "mp4", CodecName: "h264"}, nil
	})
	withPosterStub(t, func(videoPath, posterPath string) error {
		if err := os.MkdirAll(filepath.Dir(posterPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(posterPath, []byte("jpg"), 0644)
	})
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
	if resp.Meta.DurationMS != 12345 || resp.Meta.Width != 1920 || resp.Meta.Height != 1080 {
		t.Fatalf("返回的元数据不正确: %+v", resp.Meta)
	}
	if resp.Photo.ID != 99 || resp.Photo.MediaKind != storage.MediaKindVideo || resp.Photo.DurationMS != 12345 {
		t.Fatalf("返回的媒体记录不正确: %+v", resp.Photo)
	}
	if resp.PosterPath == "" || resp.PosterError != "" {
		t.Fatalf("poster 返回不正确: path=%s err=%s", resp.PosterPath, resp.PosterError)
	}
	if _, err := os.Stat(resp.PosterPath); err != nil {
		t.Fatalf("poster 文件应存在: %v", err)
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

func TestUploadPlaceholder_ReturnsPosterErrorButKeepsSuccess(t *testing.T) {
	cfg := testConfig()
	cfg.StoragePath = t.TempDir()
	withProbeStub(t, func(path string) (*media.VideoMeta, error) {
		return &media.VideoMeta{Width: 1920, Height: 1080, DurationMS: 12345, FormatName: "mp4", CodecName: "h264"}, nil
	})
	withPosterStub(t, func(videoPath, posterPath string) error {
		return fmt.Errorf("ffmpeg 执行失败")
	})
	router := NewRouter(cfg, http.NotFoundHandler(), okRegistrar())
	req := uploadRequest(t, "/api/media/upload", "demo.mp4", mp4Sample(), true)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("期望 201，得到 %d", w.Code)
	}
	var resp struct {
		Path        string `json:"path"`
		PosterPath  string `json:"poster_path"`
		PosterError string `json:"poster_error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Path == "" {
		t.Fatal("视频文件路径不能为空")
	}
	if resp.PosterPath != "" || resp.PosterError == "" {
		t.Fatalf("poster 失败时返回不正确: path=%s err=%s", resp.PosterPath, resp.PosterError)
	}
}

func TestUploadPlaceholder_ReturnsProbeUnavailableError(t *testing.T) {
	cfg := testConfig()
	cfg.StoragePath = t.TempDir()
	withProbeStub(t, func(path string) (*media.VideoMeta, error) {
		return nil, fmt.Errorf("%w: 请先安装 ffprobe", media.ErrProbeUnavailable)
	})
	router := NewRouter(cfg, http.NotFoundHandler(), okRegistrar())
	req := uploadRequest(t, "/api/media/upload", "demo.mp4", mp4Sample(), true)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，得到 %d", w.Code)
	}
	entries, err := os.ReadDir(filepath.Join(cfg.StoragePath, tempMediaDirName))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("读取临时目录失败: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("探测失败后应清理临时文件，实际剩余 %d 个", len(entries))
	}
}

func TestUploadPlaceholder_ReturnsInvalidVideoError(t *testing.T) {
	cfg := testConfig()
	cfg.StoragePath = t.TempDir()
	withProbeStub(t, func(path string) (*media.VideoMeta, error) {
		return nil, fmt.Errorf("%w: 文件损坏", media.ErrInvalidVideo)
	})
	router := NewRouter(cfg, http.NotFoundHandler(), okRegistrar())
	req := uploadRequest(t, "/api/media/upload", "demo.mp4", mp4Sample(), true)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", w.Code)
	}
}

func TestUploadPlaceholder_CleansFileWhenRegisterFails(t *testing.T) {
	cfg := testConfig()
	cfg.StoragePath = t.TempDir()
	withProbeStub(t, func(path string) (*media.VideoMeta, error) {
		return &media.VideoMeta{Width: 1920, Height: 1080, DurationMS: 12345, FormatName: "mp4", CodecName: "h264"}, nil
	})
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
