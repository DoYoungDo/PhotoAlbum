package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"image"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"photoalbum/internal/config"
	"photoalbum/internal/service"
	"photoalbum/internal/storage/sqlite"
)

func createTestJPEGBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

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
	albumSvc := service.NewAlbumService(repo)
	shareSvc := service.NewShareService(repo)
	return New(cfg, photoSvc, albumSvc, shareSvc, nil) // nil FS：测试中回退到本地文件系统
}

// withAuth 生成带有认证 cookie 的请求构造函数
func withAuth(t *testing.T, s *Server) func(method, path string, body []byte) *http.Request {
	t.Helper()
	token, err := s.generateToken("alice")
	if err != nil {
		t.Fatalf("生成 token 失败: %v", err)
	}
	return func(method, path string, body []byte) *http.Request {
		var req *http.Request
		if body != nil {
			req = httptest.NewRequest(method, path, bytes.NewReader(body))
		} else {
			req = httptest.NewRequest(method, path, nil)
		}
		req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
		return req
	}
}

// --- JWT 测试 ---

func TestGenerateAndParseToken(t *testing.T) {
	s := newTestServer(t)
	token, err := s.generateToken("alice")
	if err != nil {
		t.Fatalf("生成 token 失败: %v", err)
	}
	username, err := s.parseToken(token)
	if err != nil {
		t.Fatalf("解析 token 失败: %v", err)
	}
	if username != "alice" {
		t.Fatalf("期望 alice，得到 %s", username)
	}
}

func TestParseToken_Invalid(t *testing.T) {
	s := newTestServer(t)
	if _, err := s.parseToken("invalid-token"); err == nil {
		t.Fatal("无效 token 应该返回错误")
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	s := newTestServer(t)
	s2 := *s
	s2.cfg = &config.Config{JWTSecret: "other-secret"}
	token, _ := s2.generateToken("alice")
	if _, err := s.parseToken(token); err == nil {
		t.Fatal("不同 secret 签发的 token 应该解析失败")
	}
}

// --- Auth 中间件测试 ---

func TestAuthMiddleware_RedirectsHTMLRequests(t *testing.T) {
	s := newTestServer(t)
	// 页面路径（非 /api/）未登录应该跳转到 /login
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("期望 303，得到 %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/login" {
		t.Fatalf("期望跳转到 /login，得到 %s", loc)
	}
}

func TestAuthMiddleware_Returns401ForAPI(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/photos", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestAuthMiddleware_AllowsValidToken(t *testing.T) {
	s := newTestServer(t)
	newReq := withAuth(t, s)
	req := newReq(http.MethodGet, "/api/photos", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("有效 token 期望 200，得到 %d", w.Code)
	}
}

// --- Login / Logout 测试 ---

func TestLogin_Success(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d，body=%s", w.Code, w.Body.String())
	}
	found := false
	for _, c := range w.Result().Cookies() {
		if c.Name == authCookieName {
			found = true
		}
	}
	if !found {
		t.Fatal("登录后应该返回认证 cookie")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "wrongpass"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	s := newTestServer(t)
	body, _ := json.Marshal(map[string]string{"username": "nobody", "password": "pass"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestLogout_ClearsCookie(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
}

// --- 页面测试 ---

func TestLoginPage_Returns200(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
}

func TestIndexPage_WithAuth(t *testing.T) {
	s := newTestServer(t)
	newReq := withAuth(t, s)
	req := newReq(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
}

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

// --- Photo API 测试 ---

func TestListPhotos_AuthRequired(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/photos", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("期望 401，得到 %d", w.Code)
	}
}

func TestListPhotos_Empty(t *testing.T) {
	s := newTestServer(t)
	newReq := withAuth(t, s)
	req := newReq(http.MethodGet, "/api/photos", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d，body=%s", w.Code, w.Body.String())
	}
}

func TestParseClientLastModified(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/photos/upload", nil)
	req.Form = map[string][]string{
		"client_last_modified_ms": {"1710403200000"},
	}
	got := parseClientLastModified(req)
	if got.IsZero() {
		t.Fatal("期望解析出有效时间")
	}
	if got.UnixMilli() != 1710403200000 {
		t.Fatalf("期望 1710403200000，得到 %d", got.UnixMilli())
	}
}

func TestHandleUploadPhoto_UsesClientLastModifiedFallback(t *testing.T) {
	s := newTestServer(t)
	newReq := withAuth(t, s)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("photo", "test.jpg")
	if err != nil {
		t.Fatalf("创建 multipart 失败: %v", err)
	}
	_, _ = part.Write(createTestJPEGBytes(100, 100))
	clientMS := time.Date(2024, 3, 14, 12, 0, 0, 0, time.UTC).UnixMilli()
	_ = w.WriteField("client_last_modified_ms", strconv.FormatInt(clientMS, 10))
	_ = w.Close()

	req := newReq(http.MethodPost, "/api/photos/upload", body.Bytes())
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("期望 201，得到 %d，body=%s", rec.Code, rec.Body.String())
	}

	// 读取返回的图片对象，验证 taken_at 使用了客户端时间
	var photo struct {
		TakenAt string `json:"taken_at"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &photo); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	parsed, err := time.Parse(time.RFC3339Nano, photo.TakenAt)
	if err != nil {
		t.Fatalf("解析 taken_at 失败: %v", err)
	}
	if parsed.UnixMilli() != clientMS {
		t.Fatalf("期望 taken_at=%d，得到 %d", clientMS, parsed.UnixMilli())
	}

	// 上传后缩略图在后台 goroutine 生成，稍等片刻避免 TempDir 清理与后台写入竞争。
	time.Sleep(50 * time.Millisecond)
}

func TestDownloadPhoto_Success(t *testing.T) {
	s := newTestServer(t)
	newReq := withAuth(t, s)

	// 先上传一张图片
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("photo", "下载测试.jpg")
	if err != nil {
		t.Fatalf("创建 multipart 失败: %v", err)
	}
	imgBytes := createTestJPEGBytes(64, 64)
	_, _ = part.Write(imgBytes)
	_ = mw.WriteField("client_last_modified_ms", strconv.FormatInt(time.Now().UnixMilli(), 10))
	_ = mw.Close()

	uploadReq := newReq(http.MethodPost, "/api/photos/upload", body.Bytes())
	uploadReq.Header.Set("Content-Type", mw.FormDataContentType())
	uploadRec := httptest.NewRecorder()
	s.ServeHTTP(uploadRec, uploadReq)
	if uploadRec.Code != http.StatusCreated {
		t.Fatalf("上传期望 201，得到 %d，body=%s", uploadRec.Code, uploadRec.Body.String())
	}

	var photo struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(uploadRec.Body.Bytes(), &photo); err != nil {
		t.Fatalf("解析上传响应失败: %v", err)
	}

	// 再下载图片
	downloadReq := newReq(http.MethodGet, "/api/photos/"+strconv.FormatInt(photo.ID, 10)+"/download", nil)
	downloadRec := httptest.NewRecorder()
	s.ServeHTTP(downloadRec, downloadReq)
	if downloadRec.Code != http.StatusOK {
		t.Fatalf("下载期望 200，得到 %d，body=%s", downloadRec.Code, downloadRec.Body.String())
	}
	if got := downloadRec.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment;") || !strings.Contains(got, "filename*") {
		t.Fatalf("Content-Disposition 不正确: %q", got)
	}
	if got := downloadRec.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Fatalf("Content-Type 期望 image/jpeg，得到 %q", got)
	}
	if len(downloadRec.Body.Bytes()) == 0 {
		t.Fatal("下载内容不能为空")
	}

	time.Sleep(50 * time.Millisecond)
}

func TestDownloadPhoto_NotFound(t *testing.T) {
	s := newTestServer(t)
	newReq := withAuth(t, s)
	req := newReq(http.MethodGet, "/api/photos/9999/download", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，得到 %d", w.Code)
	}
}

func TestDownloadPhotos_ZipSuccessAndDeduplicateNames(t *testing.T) {
	s := newTestServer(t)
	newReq := withAuth(t, s)

	uploadSameName := func() int64 {
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		part, err := mw.CreateFormFile("photo", "same.jpg")
		if err != nil {
			t.Fatalf("创建 multipart 失败: %v", err)
		}
		_, _ = part.Write(createTestJPEGBytes(32, 32))
		_ = mw.WriteField("client_last_modified_ms", strconv.FormatInt(time.Now().UnixMilli(), 10))
		_ = mw.Close()

		req := newReq(http.MethodPost, "/api/photos/upload", body.Bytes())
		req.Header.Set("Content-Type", mw.FormDataContentType())
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("上传期望 201，得到 %d，body=%s", rec.Code, rec.Body.String())
		}
		var photo struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &photo); err != nil {
			t.Fatalf("解析上传响应失败: %v", err)
		}
		return photo.ID
	}

	id1 := uploadSameName()
	id2 := uploadSameName()

	body, _ := json.Marshal(map[string]any{"photo_ids": []int64{id1, id2}})
	req := newReq(http.MethodPost, "/api/photos/download", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("批量下载期望 200，得到 %d，body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/zip" {
		t.Fatalf("Content-Type 期望 application/zip，得到 %q", got)
	}

	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("解析 zip 失败: %v", err)
	}
	if len(zr.File) != 2 {
		t.Fatalf("zip 中文件数期望 2，得到 %d", len(zr.File))
	}
	names := []string{zr.File[0].Name, zr.File[1].Name}
	if !(containsString(names, "same.jpg") && containsString(names, "same (2).jpg")) {
		t.Fatalf("zip 内文件名不正确: %+v", names)
	}

	time.Sleep(50 * time.Millisecond)
}

func TestDownloadPhotos_EmptyRequest(t *testing.T) {
	s := newTestServer(t)
	newReq := withAuth(t, s)
	body, _ := json.Marshal(map[string]any{"photo_ids": []int64{}})
	req := newReq(http.MethodPost, "/api/photos/download", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", w.Code)
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func TestDeletePhoto_NotFound(t *testing.T) {
	s := newTestServer(t)
	newReq := withAuth(t, s)
	req := newReq(http.MethodDelete, "/api/photos/9999", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", w.Code)
	}
}

// --- Album API 测试 ---

func TestCreateAlbum_API(t *testing.T) {
	s := newTestServer(t)
	newReq := withAuth(t, s)
	body, _ := json.Marshal(map[string]string{"name": "测试相册", "description": "描述"})
	req := newReq(http.MethodPost, "/api/albums", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("期望 201，得到 %d，body=%s", w.Code, w.Body.String())
	}
}

func TestListAlbums_Empty(t *testing.T) {
	s := newTestServer(t)
	newReq := withAuth(t, s)
	req := newReq(http.MethodGet, "/api/albums", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", w.Code)
	}
}

func TestGetAlbum_NotFound(t *testing.T) {
	s := newTestServer(t)
	newReq := withAuth(t, s)
	req := newReq(http.MethodGet, "/api/albums/9999", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，得到 %d", w.Code)
	}
}

func TestDownloadAlbum_ZipSuccess(t *testing.T) {
	s := newTestServer(t)
	newReq := withAuth(t, s)

	// 创建相册
	body, _ := json.Marshal(map[string]string{"name": "旅行相册", "description": "test"})
	createReq := newReq(http.MethodPost, "/api/albums", body)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	s.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("创建相册期望 201，得到 %d，body=%s", createRec.Code, createRec.Body.String())
	}
	var album struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &album); err != nil {
		t.Fatalf("解析相册响应失败: %v", err)
	}

	uploadSameName := func() int64 {
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		part, err := mw.CreateFormFile("photo", "trip.jpg")
		if err != nil {
			t.Fatalf("创建 multipart 失败: %v", err)
		}
		_, _ = part.Write(createTestJPEGBytes(32, 32))
		_ = mw.Close()
		req := newReq(http.MethodPost, "/api/photos/upload", body.Bytes())
		req.Header.Set("Content-Type", mw.FormDataContentType())
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("上传期望 201，得到 %d，body=%s", rec.Code, rec.Body.String())
		}
		var photo struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &photo); err != nil {
			t.Fatalf("解析图片响应失败: %v", err)
		}
		return photo.ID
	}

	id1 := uploadSameName()
	id2 := uploadSameName()

	for _, pid := range []int64{id1, id2} {
		addBody, _ := json.Marshal(map[string]int64{"photo_id": pid})
		addReq := newReq(http.MethodPost, "/api/albums/"+strconv.FormatInt(album.ID, 10)+"/photos", addBody)
		addReq.Header.Set("Content-Type", "application/json")
		addRec := httptest.NewRecorder()
		s.ServeHTTP(addRec, addReq)
		if addRec.Code != http.StatusOK {
			t.Fatalf("添加到相册期望 200，得到 %d，body=%s", addRec.Code, addRec.Body.String())
		}
	}

	downloadReq := newReq(http.MethodGet, "/api/albums/"+strconv.FormatInt(album.ID, 10)+"/download", nil)
	downloadRec := httptest.NewRecorder()
	s.ServeHTTP(downloadRec, downloadReq)
	if downloadRec.Code != http.StatusOK {
		t.Fatalf("下载相册期望 200，得到 %d，body=%s", downloadRec.Code, downloadRec.Body.String())
	}
	if got := downloadRec.Header().Get("Content-Type"); got != "application/zip" {
		t.Fatalf("Content-Type 期望 application/zip，得到 %q", got)
	}
	zr, err := zip.NewReader(bytes.NewReader(downloadRec.Body.Bytes()), int64(downloadRec.Body.Len()))
	if err != nil {
		t.Fatalf("解析 zip 失败: %v", err)
	}
	if len(zr.File) != 2 {
		t.Fatalf("zip 中文件数期望 2，得到 %d", len(zr.File))
	}
	names := []string{zr.File[0].Name, zr.File[1].Name}
	if !(containsString(names, "trip.jpg") && containsString(names, "trip (2).jpg")) {
		t.Fatalf("zip 内文件名不正确: %+v", names)
	}

	time.Sleep(50 * time.Millisecond)
}

// --- Share API 测试 ---

func TestCreateShare_API(t *testing.T) {
	s := newTestServer(t)
	newReq := withAuth(t, s)
	body, _ := json.Marshal(map[string]any{"type": "photo", "target_id": int64(1)})
	req := newReq(http.MethodPost, "/api/shares", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("期望 201，得到 %d，body=%s", w.Code, w.Body.String())
	}
}

func TestDownloadSharedPhoto_Success(t *testing.T) {
	s := newTestServer(t)
	newReq := withAuth(t, s)

	// 上传图片
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("photo", "shared.jpg")
	if err != nil {
		t.Fatalf("创建 multipart 失败: %v", err)
	}
	_, _ = part.Write(createTestJPEGBytes(40, 40))
	_ = mw.Close()
	uploadReq := newReq(http.MethodPost, "/api/photos/upload", body.Bytes())
	uploadReq.Header.Set("Content-Type", mw.FormDataContentType())
	uploadRec := httptest.NewRecorder()
	s.ServeHTTP(uploadRec, uploadReq)
	if uploadRec.Code != http.StatusCreated {
		t.Fatalf("上传期望 201，得到 %d，body=%s", uploadRec.Code, uploadRec.Body.String())
	}
	var photo struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(uploadRec.Body.Bytes(), &photo); err != nil {
		t.Fatalf("解析图片响应失败: %v", err)
	}

	// 创建分享链接
	shareBody, _ := json.Marshal(map[string]any{"type": "photo", "target_id": photo.ID})
	shareReq := newReq(http.MethodPost, "/api/shares", shareBody)
	shareReq.Header.Set("Content-Type", "application/json")
	shareRec := httptest.NewRecorder()
	s.ServeHTTP(shareRec, shareReq)
	if shareRec.Code != http.StatusCreated {
		t.Fatalf("创建分享期望 201，得到 %d，body=%s", shareRec.Code, shareRec.Body.String())
	}
	var link struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(shareRec.Body.Bytes(), &link); err != nil {
		t.Fatalf("解析分享响应失败: %v", err)
	}

	// 匿名下载分享图片
	downloadReq := httptest.NewRequest(http.MethodGet, "/s/"+link.Token+"/download", nil)
	downloadRec := httptest.NewRecorder()
	s.ServeHTTP(downloadRec, downloadReq)
	if downloadRec.Code != http.StatusOK {
		t.Fatalf("分享下载期望 200，得到 %d，body=%s", downloadRec.Code, downloadRec.Body.String())
	}
	if got := downloadRec.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment;") {
		t.Fatalf("Content-Disposition 不正确: %q", got)
	}
	if len(downloadRec.Body.Bytes()) == 0 {
		t.Fatal("下载内容不能为空")
	}

	time.Sleep(50 * time.Millisecond)
}

func TestDownloadSharedPhoto_NotFound(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/s/invalid-token/download", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，得到 %d", w.Code)
	}
}

func TestGetShare_NotFound(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/s/nonexistent-token", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("期望 404，得到 %d", w.Code)
	}
}
