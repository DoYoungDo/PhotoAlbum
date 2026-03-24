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
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"photoalbum/internal/config"
	"photoalbum/internal/media"
)

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

	router := NewRouter(testConfig(), legacy)
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

func TestNewRouter_FallsBackToLegacyHandler(t *testing.T) {
	legacy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			t.Fatalf("期望回退到 /login，得到 %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusTeapot)
	})

	router := NewRouter(testConfig(), legacy)
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusTeapot {
		t.Fatalf("期望 418，得到 %d", w.Code)
	}
}

func TestUploadPlaceholder_RequiresAuth(t *testing.T) {
	router := NewRouter(testConfig(), http.NotFoundHandler())
	req := uploadRequest(t, "/api/media/upload", "demo.mp4", mp4Sample(), true)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestUploadPlaceholder_RequiresMediaField(t *testing.T) {
	cfg := testConfig()
	router := NewRouter(cfg, http.NotFoundHandler())
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
	router := NewRouter(cfg, http.NotFoundHandler())
	req := uploadRequest(t, "/api/media/upload", "demo.mov", mp4Sample(), true)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", w.Code)
	}
}

func TestUploadPlaceholder_SavesTempFileAfterValidation(t *testing.T) {
	cfg := testConfig()
	cfg.StoragePath = t.TempDir()
	withProbeStub(t, func(path string) (*media.VideoMeta, error) {
		return &media.VideoMeta{Width: 1920, Height: 1080, DurationMS: 12345, FormatName: "mp4", CodecName: "h264"}, nil
	})
	router := NewRouter(cfg, http.NotFoundHandler())
	req := uploadRequest(t, "/api/media/upload", "demo.mp4", mp4Sample(), true)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("期望 201，得到 %d", w.Code)
	}

	var resp struct {
		Message  string `json:"message"`
		Filename string `json:"filename"`
		Path     string `json:"path"`
		Meta     struct {
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			DurationMS int64  `json:"duration_ms"`
			FormatName string `json:"format_name"`
			CodecName  string `json:"codec_name"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if resp.Filename != "demo.mp4" {
		t.Fatalf("期望文件名 demo.mp4，得到 %s", resp.Filename)
	}
	if resp.Path == "" {
		t.Fatal("响应中应返回临时文件路径")
	}
	if resp.Meta.DurationMS != 12345 || resp.Meta.Width != 1920 || resp.Meta.Height != 1080 {
		t.Fatalf("返回的元数据不正确: %+v", resp.Meta)
	}
	if filepath.Dir(resp.Path) != filepath.Join(cfg.StoragePath, tempMediaDirName) {
		t.Fatalf("临时文件目录不正确: %s", resp.Path)
	}
	data, err := os.ReadFile(resp.Path)
	if err != nil {
		t.Fatalf("读取临时文件失败: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("临时保存的文件内容不能为空")
	}
}

func TestUploadPlaceholder_ReturnsProbeUnavailableError(t *testing.T) {
	cfg := testConfig()
	cfg.StoragePath = t.TempDir()
	withProbeStub(t, func(path string) (*media.VideoMeta, error) {
		return nil, fmt.Errorf("%w: 请先安装 ffprobe", media.ErrProbeUnavailable)
	})
	router := NewRouter(cfg, http.NotFoundHandler())
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
	router := NewRouter(cfg, http.NotFoundHandler())
	req := uploadRequest(t, "/api/media/upload", "demo.mp4", mp4Sample(), true)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: testToken(t, cfg.JWTSecret, "alice")})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", w.Code)
	}
}
