package server

import (
	"encoding/json"
	"net/http"
	"time"

	"photoalbum/internal/service"
	"photoalbum/internal/storage"
)

type shareRequest struct {
	Type      string `json:"type"`
	TargetID  int64  `json:"target_id"`
	ExpiresIn *int64 `json:"expires_in_days,omitempty"`
}

func (s *Server) handleListShares(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	links, err := s.shareService.ListShares(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, links)
}

func (s *Server) handleCreateShare(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	var req shareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(*req.ExpiresIn) * 24 * time.Hour)
		expiresAt = &t
	}
	link, err := s.shareService.CreateShare(service.CreateShareInput{
		Type:      req.Type,
		TargetID:  req.TargetID,
		UserID:    userID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, link)
}

func (s *Server) handleDeleteShare(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	id, err := parseInt64Param(r.PathValue("id"), "分享ID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.shareService.DeleteShare(id, userID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "分享链接已删除"})
}

func (s *Server) handleGetShare(w http.ResponseWriter, r *http.Request) {
	link, err := s.shareService.GetShareByToken(r.PathValue("token"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if link == nil {
		writeError(w, http.StatusNotFound, "分享链接不存在或已过期")
		return
	}
	writeJSON(w, http.StatusOK, link)
}

func (s *Server) handleServeSharedMedia(w http.ResponseWriter, r *http.Request) {
	link, err := s.shareService.GetShareByToken(r.PathValue("token"))
	if err != nil || link == nil {
		writeError(w, http.StatusNotFound, "分享链接不存在或已过期")
		return
	}
	if link.Type != "photo" {
		writeError(w, http.StatusBadRequest, "当前分享不支持该媒体访问")
		return
	}
	photo, err := s.photoService.GetPhoto(link.TargetID, link.CreatedBy)
	if err != nil || photo == nil {
		writeError(w, http.StatusNotFound, "图片不存在")
		return
	}
	http.ServeFile(w, r, s.photoService.PhotoPath(photo))
}

func (s *Server) handleDownloadSharedPhoto(w http.ResponseWriter, r *http.Request) {
	link, err := s.shareService.GetShareByToken(r.PathValue("token"))
	if err != nil || link == nil {
		writeError(w, http.StatusNotFound, "分享链接不存在或已过期")
		return
	}
	if link.Type != storage.ShareTypePhoto {
		writeError(w, http.StatusBadRequest, "当前分享不支持下载")
		return
	}
	photo, err := s.photoService.GetPhoto(link.TargetID, link.CreatedBy)
	if err != nil || photo == nil {
		writeError(w, http.StatusNotFound, "图片不存在")
		return
	}
	w.Header().Set("Content-Type", photo.MimeType)
	w.Header().Set("Content-Disposition", contentDispositionAttachment(photo.OriginalName))
	http.ServeFile(w, r, s.photoService.PhotoPath(photo))
}

// handleGetSharePhotos 获取分享相册中的图片列表（无需登录）
func (s *Server) handleGetSharePhotos(w http.ResponseWriter, r *http.Request) {
	link, err := s.shareService.GetShareByToken(r.PathValue("token"))
	if err != nil || link == nil {
		writeError(w, http.StatusNotFound, "分享链接不存在或已过期")
		return
	}
	if link.Type != "album" {
		writeError(w, http.StatusBadRequest, "当前分享不是相册类型")
		return
	}
	page, err := s.albumService.ListAlbumPhotos(storage.ListAlbumPhotosParams{
		AlbumID: link.TargetID,
		UserID:  link.CreatedBy,
		Cursor:  r.URL.Query().Get("cursor"),
		Limit:   30,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}
