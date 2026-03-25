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
