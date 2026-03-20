package server

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"photoalbum/internal/config"
	"photoalbum/internal/service"
)

// Server HTTP 服务器
type Server struct {
	cfg          *config.Config
	photoService *service.PhotoService
	albumService *service.AlbumService
	shareService *service.ShareService
	mux          *http.ServeMux
	staticFS     fs.FS // embed 或本地文件系统
}

// New 创建并初始化 Server
func New(
	cfg *config.Config,
	photoService *service.PhotoService,
	albumService *service.AlbumService,
	shareService *service.ShareService,
	staticFS fs.FS,
) *Server {
	s := &Server{
		cfg:          cfg,
		photoService: photoService,
		albumService: albumService,
		shareService: shareService,
		staticFS:     staticFS,
		mux:          http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// ServeHTTP 实现 http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// registerRoutes 注册所有路由
func (s *Server) registerRoutes() {
	// 静态资源：优先使用 embed FS，回退到本地文件系统
	var staticHandler http.Handler
	if s.staticFS != nil {
		sub, err := fs.Sub(s.staticFS, "web/static")
		if err == nil {
			staticHandler = http.FileServer(http.FS(sub))
		}
	}
	if staticHandler == nil {
		staticHandler = http.FileServer(http.Dir("web/static"))
	}
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", staticHandler))

	// 页面路由（返回 HTML，需要登录）
	s.mux.HandleFunc("GET /", s.auth(s.handleIndex))
	s.mux.HandleFunc("GET /albums", s.auth(s.handleAlbumsPage))
	s.mux.HandleFunc("GET /albums/{id}", s.auth(s.handleAlbumDetailPage))
	s.mux.HandleFunc("GET /trash", s.auth(s.handleTrashPage))

	// 登录/登出
	s.mux.HandleFunc("GET /login", s.handleLoginPage)
	s.mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/auth/logout", s.handleLogout)

	// 图片 API
	s.mux.HandleFunc("GET /api/photos", s.auth(s.handleListPhotos))
	s.mux.HandleFunc("POST /api/photos/upload", s.auth(s.handleUploadPhoto))
	s.mux.HandleFunc("POST /api/photos/download", s.auth(s.handleDownloadPhotos))
	s.mux.HandleFunc("GET /api/photos/{id}", s.auth(s.handleGetPhoto))
	s.mux.HandleFunc("GET /api/photos/{id}/download", s.auth(s.handleDownloadPhoto))
	s.mux.HandleFunc("DELETE /api/photos/{id}", s.auth(s.handleDeletePhoto))
	s.mux.HandleFunc("POST /api/photos/{id}/restore", s.auth(s.handleRestorePhoto))

	// 图片文件服务
	s.mux.HandleFunc("GET /media/photos/{uuid}", s.auth(s.handleServePhoto))
	s.mux.HandleFunc("GET /media/thumbnails/{uuid}", s.auth(s.handleServeThumbnail))

	// 回收站 API
	s.mux.HandleFunc("GET /api/trash", s.auth(s.handleListTrash))
	s.mux.HandleFunc("DELETE /api/trash", s.auth(s.handleEmptyTrash))

	// 相册 API
	s.mux.HandleFunc("GET /api/albums", s.auth(s.handleListAlbums))
	s.mux.HandleFunc("POST /api/albums", s.auth(s.handleCreateAlbum))
	s.mux.HandleFunc("GET /api/albums/{id}", s.auth(s.handleGetAlbum))
	s.mux.HandleFunc("GET /api/albums/{id}/download", s.auth(s.handleDownloadAlbum))
	s.mux.HandleFunc("PUT /api/albums/{id}", s.auth(s.handleUpdateAlbum))
	s.mux.HandleFunc("DELETE /api/albums/{id}", s.auth(s.handleDeleteAlbum))
	s.mux.HandleFunc("GET /api/albums/{id}/photos", s.auth(s.handleListAlbumPhotos))
	s.mux.HandleFunc("POST /api/albums/{id}/photos", s.auth(s.handleAddPhotoToAlbum))
	s.mux.HandleFunc("DELETE /api/albums/{id}/photos/{photoId}", s.auth(s.handleRemovePhotoFromAlbum))

	// 分享 API
	s.mux.HandleFunc("GET /api/shares", s.auth(s.handleListShares))
	s.mux.HandleFunc("POST /api/shares", s.auth(s.handleCreateShare))
	s.mux.HandleFunc("DELETE /api/shares/{id}", s.auth(s.handleDeleteShare))

	// ���享访问（无需登录）
	s.mux.HandleFunc("GET /s/{token}", s.handleSharePage)
	s.mux.HandleFunc("GET /s/{token}/download", s.handleDownloadSharedPhoto)
	s.mux.HandleFunc("GET /api/s/{token}", s.handleGetShare)
	s.mux.HandleFunc("GET /api/s/{token}/photos", s.handleGetSharePhotos)
	s.mux.HandleFunc("GET /media/s/{token}/{uuid}", s.handleServeSharedMedia)
}

// writeJSON 写入 JSON 响应
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError 写入错误响应
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
