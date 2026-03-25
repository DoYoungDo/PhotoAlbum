package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"photoalbum/internal/config"
	"photoalbum/internal/service"
	"photoalbum/internal/storage/sqlite"
)

// newTestServer 创建用于测试的 Server，使用 SQLite 临时数据库
func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	repo, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	// alice 用户，密码 password123（bcrypt hash 预生成）
	cfg := &config.Config{
		Port:        8080,
		StoragePath: dir,
		JWTSecret:   "test-secret",
		Users: []config.User{
			{Username: "alice", PasswordHash: "$2a$10$m2CWsTFrqFNGPW/bGg4UluO.WX/e.rgEkX4yxHJI.VABfOyGA8BA2"},
		},
	}

	photoSvc := service.NewPhotoService(repo, dir)
	return New(cfg, photoSvc, nil) // nil FS：测试中回退到本地文件系统
}

// --- 页面测试 ---

func TestStaticFile(t *testing.T) {
	if _, err := os.Stat(filepath.Join("web", "static", "app.css")); err != nil {
		t.Skipf("静态文件不存在: %v", err)
	}
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/static/app.css", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
}
