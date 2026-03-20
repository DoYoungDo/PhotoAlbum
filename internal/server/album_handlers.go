package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"photoalbum/internal/storage"
)

type albumRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	CoverPhotoID *int64 `json:"cover_photo_id"`
}

type albumPhotoRequest struct {
	PhotoID int64 `json:"photo_id"`
}

func (s *Server) handleListAlbums(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	albums, err := s.albumService.ListAlbums(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, albums)
}

func (s *Server) handleCreateAlbum(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	var req albumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	album, err := s.albumService.CreateAlbum(req.Name, req.Description, userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, album)
}

func (s *Server) handleGetAlbum(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	id, err := parseInt64Param(r.PathValue("id"), "相册ID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	album, err := s.albumService.GetAlbum(id, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if album == nil {
		writeError(w, http.StatusNotFound, "相册不存在")
		return
	}
	writeJSON(w, http.StatusOK, album)
}

func sanitizeZipName(name string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "", "<", "-", ">", "-", "|", "-")
	cleaned := strings.TrimSpace(replacer.Replace(name))
	if cleaned == "" {
		cleaned = "album"
	}
	return cleaned + ".zip"
}

func (s *Server) handleDownloadAlbum(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	id, err := parseInt64Param(r.PathValue("id"), "相册ID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	albumName, entries, err := s.albumService.GetAlbumDownloadEntries(id, userID, s.photoService)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", contentDispositionAttachment(sanitizeZipName(albumName)))
	if err := writeZipResponse(w, entries); err != nil {
		writeError(w, http.StatusInternalServerError, "打包下载失败")
		return
	}
}

func (s *Server) handleUpdateAlbum(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	id, err := parseInt64Param(r.PathValue("id"), "相册ID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req albumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	album, err := s.albumService.UpdateAlbum(id, req.Name, req.Description, req.CoverPhotoID, userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, album)
}

func (s *Server) handleDeleteAlbum(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	id, err := parseInt64Param(r.PathValue("id"), "相册ID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.albumService.DeleteAlbum(id, userID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "相册已删除"})
}

func (s *Server) handleListAlbumPhotos(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	albumID, err := parseInt64Param(r.PathValue("id"), "相册ID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	page, err := s.albumService.ListAlbumPhotos(storage.ListAlbumPhotosParams{
		AlbumID: albumID,
		UserID:  userID,
		Cursor:  r.URL.Query().Get("cursor"),
		Limit:   30,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) handleAddPhotoToAlbum(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	albumID, err := parseInt64Param(r.PathValue("id"), "相册ID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req albumPhotoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	if err := s.albumService.AddPhoto(albumID, req.PhotoID, userID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "已添加到相册"})
}

func (s *Server) handleRemovePhotoFromAlbum(w http.ResponseWriter, r *http.Request) {
	userID := s.mustUserID(w, r)
	if userID == 0 {
		return
	}
	albumID, err := parseInt64Param(r.PathValue("id"), "相册ID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	photoID, err := parseInt64Param(r.PathValue("photoId"), "图片ID")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.albumService.RemovePhoto(albumID, photoID, userID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "已从相册移除"})
}
