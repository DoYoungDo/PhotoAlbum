package server

import (
	"html/template"
	"net/http"
)

// pageTemplate 占位模板，第五阶段替换为完整 UI
const pageTemplate = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} - PhotoAlbum</title>
</head>
<body>
  <h1>{{.Title}}</h1>
  {{if .Message}}<p>{{.Message}}</p>{{end}}
</body>
</html>`

var tmpl = template.Must(template.New("page").Parse(pageTemplate))

type pageData struct {
	Title   string
	Message string
}

func renderPage(w http.ResponseWriter, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, pageData{Title: title, Message: message})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "时间线", "")
}

func (s *Server) handleAlbumsPage(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "相册", "")
}

func (s *Server) handleAlbumDetailPage(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "相册详情", "")
}

func (s *Server) handleTrashPage(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "回收站", "")
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "登录", "")
}

func (s *Server) handleSharePage(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "分享", "")
}
