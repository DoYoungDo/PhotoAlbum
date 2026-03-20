package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"photoalbum/internal/config"
	"photoalbum/internal/service"
	"photoalbum/internal/storage"
)

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
	userID, err := s.currentUserID(currentUsername(r))
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
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

func (s *Server) handleUploadPhoto(w http.ResponseWriter, r *http.Request) {
	userID, err := s.currentUserID(currentUsername(r))
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "解析表单失败")
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "缺少 photo 文件")
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
		FileModTime:  time.Now(),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, result.Photo)
}

func (s *Server) handleGetPhoto(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.currentUserID(currentUsername(r))
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
	userID, _ := s.currentUserID(currentUsername(r))
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
	userID, _ := s.currentUserID(currentUsername(r))
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
	userID, _ := s.currentUserID(currentUsername(r))
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
	userID, _ := s.currentUserID(currentUsername(r))
	if err := s.photoService.EmptyTrash(userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "回收站已清空"})
}

func (s *Server) servePhotoFile(w http.ResponseWriter, photo *storage.Photo, thumbnail bool) {
	var path string
	if thumbnail {
		path = s.photoService.ThumbnailPath(photo)
	} else {
		path = s.photoService.PhotoPath(photo)
	}
	http.ServeFile(w, nil, path)
}

func (s *Server) handleServePhoto(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.currentUserID(currentUsername(r))
	uuid := strings.TrimSuffix(r.PathValue("uuid"), filepath.Ext(r.PathValue("uuid")))
	photo, err := s.photoService.GetPhotoByUUID(uuid, userID)
	if err != nil || photo == nil {
		writeError(w, http.StatusNotFound, "图片不存在")
		return
	}
	http.ServeFile(w, r, s.photoService.PhotoPath(photo))
}

func (s *Server) handleServeThumbnail(w http.ResponseWriter, r *http.Request) {
	userID, _ := s.currentUserID(currentUsername(r))
	uuid := strings.TrimSuffix(r.PathValue("uuid"), filepath.Ext(r.PathValue("uuid")))
	photo, err := s.photoService.GetPhotoByUUID(uuid, userID)
	if err != nil || photo == nil {
		writeError(w, http.StatusNotFound, "图片不存在")
		return
	}
	http.ServeFile(w, r, s.photoService.ThumbnailPath(photo))
}
