package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"photoalbum/internal/config"
	"photoalbum/internal/service"
	"photoalbum/internal/storage"
)

// maxUploadSize 单次上传最大 100MB
const maxUploadSize = 100 << 20

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "无效的请求体")
		return
	}

	cfg := &config.Config{Users: s.cfg.Users}
	if _, err := config.VerifyPassword(cfg, req.Username, req.Password); err != nil {
		writeError(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	token, err := s.generateToken(req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "生成令牌失败")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "登录成功"})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]string{"message": "已退出登录"})
}

func (s *Server) handleListPhotos(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	page, err := s.photoService.GetTimeline(storage.ListPhotosParams{
		UserID: userID,
		Cursor: r.URL.Query().Get("cursor"),
		Limit:  30,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func readUploadedFile(file multipart.File) ([]byte, error) {
	defer file.Close()
	return io.ReadAll(file)
}

func parseClientLastModified(r *http.Request) time.Time {
	value := strings.TrimSpace(r.FormValue("client_last_modified_ms"))
	if value == "" {
		return time.Time{}
	}
	ms, err := strconv.ParseInt(value, 10, 64)
	if err != nil || ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

func (s *Server) handleUploadPhoto(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "文件过大，最大支持 100MB")
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "缺少 photo 文件字段")
		return
	}
	data, err := readUploadedFile(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取文件失败")
		return
	}

	result, err := s.photoService.Upload(service.UploadInput{
		Reader:       bytes.NewReader(data),
		OriginalName: header.Filename,
		Size:         int64(len(data)),
		UploadedBy:   userID,
		FileModTime: func() time.Time {
			if t := parseClientLastModified(r); !t.IsZero() {
				return t
			}
			return time.Now()
		}(),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, result.Photo)
}

func (s *Server) handleGetPhoto(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	id, err := parseInt64Param(r.PathValue("id"), "图片ID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	photo, err := s.photoService.GetPhoto(id, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if photo == nil {
		writeError(w, http.StatusNotFound, "图片不存在")
		return
	}
	writeJSON(w, http.StatusOK, photo)
}

func (s *Server) handleDeletePhoto(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	id, err := parseInt64Param(r.PathValue("id"), "图片ID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.photoService.DeletePhoto(id, userID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "已移入回收站"})
}

func (s *Server) handleRestorePhoto(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	id, err := parseInt64Param(r.PathValue("id"), "图片ID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.photoService.RestorePhoto(id, userID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "恢复成功"})
}

func (s *Server) handleListTrash(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	page, err := s.photoService.GetTrash(storage.ListPhotosParams{
		UserID: userID,
		Cursor: r.URL.Query().Get("cursor"),
		Limit:  30,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) handleEmptyTrash(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	if err := s.photoService.EmptyTrash(userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "回收站已清空"})
}

func (s *Server) handleServePhoto(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	uuid := strings.TrimSuffix(r.PathValue("uuid"), filepath.Ext(r.PathValue("uuid")))
	// 使用 Any 版本，允许回收站中的图片也能被访问
	photo, err := s.photoService.GetPhotoByUUIDAny(uuid, userID)
	if err != nil || photo == nil {
		writeError(w, http.StatusNotFound, "图片不存在")
		return
	}
	http.ServeFile(w, r, s.photoService.PhotoPath(photo))
}

func (s *Server) handleServeThumbnail(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	uuid := strings.TrimSuffix(r.PathValue("uuid"), filepath.Ext(r.PathValue("uuid")))
	// 使用 Any 版本，允许回收站中的图片缩略图也能被访问
	photo, err := s.photoService.GetPhotoByUUIDAny(uuid, userID)
	if err != nil || photo == nil {
		writeError(w, http.StatusNotFound, "图片不存在")
		return
	}
	http.ServeFile(w, r, s.photoService.ThumbnailPath(photo))
}
