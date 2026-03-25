package server

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// maxUploadSize 单次上传最大 100MB
const maxUploadSize = 100 << 20

type photoDownloadRequest struct {
	PhotoIDs []int64 `json:"photo_ids"`
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

func contentDispositionAttachment(filename string) string {
	trimmed := strings.ReplaceAll(filename, "\"", "")
	trimmed = strings.ReplaceAll(trimmed, "\n", "")
	trimmed = strings.ReplaceAll(trimmed, "\r", "")
	if trimmed == "" {
		trimmed = "download"
	}
	return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", trimmed, url.PathEscape(trimmed))
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
		writeError(w, http.StatusNotFound, "照片/视频不存在")
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
		writeError(w, http.StatusNotFound, "照片/视频不存在")
		return
	}
	http.ServeFile(w, r, s.photoService.ThumbnailPath(photo))
}
