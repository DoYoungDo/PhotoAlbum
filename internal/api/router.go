package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"photoalbum/internal/config"
	"photoalbum/internal/media"
	"photoalbum/internal/service"
	"photoalbum/internal/storage"
)

const authCookieName = "photoalbum_token"
const tempMediaDirName = ".media-upload-tmp"

type videoRegistrar interface {
	AddPhoto(albumID int64, photoID int64, userID int64) error
	CreateAlbum(name, description string, userID int64) (*storage.Album, error)
	CreateShare(input service.CreateShareInput) (*storage.ShareLink, error)
	DeleteAlbum(id int64, userID int64) error
	DeleteShare(id int64, userID int64) error
	GetAlbum(id int64, userID int64) (*storage.Album, error)
	GetAlbumDownloadEntries(albumID int64, userID int64) (string, []service.DownloadEntry, error)
	GetShareByToken(token string) (*storage.ShareLink, error)
	ListAlbums(userID int64) ([]*storage.Album, error)
	ListShares(userID int64) ([]*storage.ShareLink, error)
	RemovePhoto(albumID int64, photoID int64, userID int64) error
	UpdateAlbum(id int64, name, description string, coverPhotoID *int64, userID int64) (*storage.Album, error)
	RegisterUploadedVideo(input service.RegisterUploadedVideoInput) (*storage.Photo, error)
	DeletePhoto(id int64, userID int64) error
	EmptyTrash(userID int64) error
	GetPhoto(id int64, userID int64) (*storage.Photo, error)
	GetPhotoByUUIDAny(uuid string, userID int64) (*storage.Photo, error)
	GetDownloadEntries(photoIDs []int64, userID int64) ([]service.DownloadEntry, error)
	GetAlbumMedia(params storage.ListAlbumPhotosParams) (*storage.PhotoPage, error)
	GetTrash(params storage.ListPhotosParams) (*storage.PhotoPage, error)
	PermanentlyDeletePhoto(id int64, userID int64) error
	RestorePhoto(id int64, userID int64) error
	GetTimeline(params storage.ListPhotosParams) (*storage.PhotoPage, error)
	MediaPath(photo *storage.Photo) string
	PosterPath(photo *storage.Photo) string
	ThumbnailPath(photo *storage.Photo) string
}

type mediaDownloadRequest struct {
	MediaIDs []int64 `json:"media_ids"`
	PhotoIDs []int64 `json:"photo_ids"`
}

type albumMediaRequest struct {
	MediaID int64 `json:"media_id"`
	PhotoID int64 `json:"photo_id"`
}

type albumRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	CoverPhotoID *int64 `json:"cover_photo_id"`
}

type shareRequest struct {
	Type      string `json:"type"`
	TargetID  int64  `json:"target_id"`
	ExpiresIn *int64 `json:"expires_in_days,omitempty"`
}

type shareDetailResponse struct {
	*storage.ShareLink
	TargetUUID         string `json:"target_uuid,omitempty"`
	TargetOriginalName string `json:"target_original_name,omitempty"`
	TargetMediaKind    string `json:"target_media_kind,omitempty"`
	TargetMimeType     string `json:"target_mime_type,omitempty"`
}

type contextKey string

const userContextKey contextKey = "user"

// Claims JWT claims。
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// NewRouter 创建新的 Gin 路由入口。
//
// 当前阶段仅为后续媒体能力预留 Gin 路由组，
// 其余现有功能继续回退到 legacy handler。
func NewRouter(cfg *config.Config, legacy http.Handler, registrar videoRegistrar) http.Handler {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	media := r.Group("/api/media")
	{
		media.GET("/albums", authMiddleware(cfg), handleListAlbumsMedia(cfg, registrar))
		media.POST("/albums", authMiddleware(cfg), handleCreateAlbumMedia(cfg, registrar))
		media.GET("/albums/:id/detail", authMiddleware(cfg), handleGetAlbumDetail(cfg, registrar))
		media.GET("/albums/:id/download", authMiddleware(cfg), handleDownloadAlbumMedia(cfg, registrar))
		media.GET("/albums/:id", authMiddleware(cfg), handleListAlbumMedia(cfg, registrar))
		media.POST("/albums/:id", authMiddleware(cfg), handleAddMediaToAlbum(cfg, registrar))
		media.PUT("/albums/:id", authMiddleware(cfg), handleUpdateAlbumMedia(cfg, registrar))
		media.DELETE("/albums/:id", authMiddleware(cfg), handleDeleteAlbumMedia(cfg, registrar))
		media.DELETE("/albums/:id/:mediaId", authMiddleware(cfg), handleRemoveMediaFromAlbum(cfg, registrar))
		media.GET("/shares", authMiddleware(cfg), handleListSharesMedia(cfg, registrar))
		media.POST("/shares", authMiddleware(cfg), handleCreateShareMedia(cfg, registrar))
		media.DELETE("/shares/:id", authMiddleware(cfg), handleDeleteShareMedia(cfg, registrar))
		media.GET("", authMiddleware(cfg), handleListMedia(cfg, registrar))
		media.GET("/trash", authMiddleware(cfg), handleListTrashMedia(cfg, registrar))
		media.GET(":id", authMiddleware(cfg), handleGetMedia(cfg, registrar))
		media.GET(":id/download", authMiddleware(cfg), handleDownloadMedia(cfg, registrar))
		media.POST("/download", authMiddleware(cfg), handleDownloadMediaBatch(cfg, registrar))
		media.POST(":id/restore", authMiddleware(cfg), handleRestoreMedia(cfg, registrar))
		media.DELETE(":id", authMiddleware(cfg), handleDeleteMedia(cfg, registrar))
		media.DELETE("/trash", authMiddleware(cfg), handleEmptyTrashMedia(cfg, registrar))
		media.DELETE("/trash/:id", authMiddleware(cfg), handleHardDeleteMedia(cfg, registrar))
		media.POST("/upload", authMiddleware(cfg), handleUploadPlaceholder(cfg, registrar))
	}

	r.GET("/media/files/:uuid", authMiddleware(cfg), handleServeMediaFile(cfg, registrar))
	r.GET("/media/photos/:uuid", authMiddleware(cfg), handleServePhotoFile(cfg, registrar))
	r.GET("/media/thumbnails/:uuid", authMiddleware(cfg), handleServeThumbnailFile(cfg, registrar))
	r.GET("/media/posters/:uuid", authMiddleware(cfg), handleServePoster(cfg, registrar))
	r.GET("/media/s/:token/:uuid", handleServeSharedMediaFile(cfg, registrar))
	r.GET("/api/s/:token", handleGetShareByToken(cfg, registrar))
	r.GET("/api/s/:token/photos", handleGetSharedAlbumMedia(cfg, registrar))
	r.GET("/s/:token/download", handleDownloadSharedMedia(cfg, registrar))
	r.GET("/s/:token", handleSharePage())

	legacyHandler := gin.WrapH(legacy)
	r.NoRoute(legacyHandler)
	r.NoMethod(legacyHandler)

	return r
}

func handleGetMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "照片/视频ID 无效"})
			return
		}
		photo, err := registrar.GetPhoto(id, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if photo == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "照片/视频不存在"})
			return
		}
		c.JSON(http.StatusOK, photo)
	}
}

func handleDownloadMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "照片/视频ID 无效"})
			return
		}
		photo, err := registrar.GetPhoto(id, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if photo == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "照片/视频不存在"})
			return
		}

		c.Header("Content-Type", photo.MimeType)
		c.Header("Content-Disposition", contentDispositionAttachment(photo.OriginalName))
		c.File(registrar.MediaPath(photo))
	}
}

func handleDeleteMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "照片/视频ID 无效"})
			return
		}
		if err := registrar.DeletePhoto(id, userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "已移入回收站"})
	}
}

func handleRestoreMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "照片/视频ID 无效"})
			return
		}
		if err := registrar.RestorePhoto(id, userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "恢复成功"})
	}
}

func handleListTrashMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		page, err := registrar.GetTrash(storage.ListPhotosParams{
			UserID: userID,
			Cursor: c.Query("cursor"),
			Limit:  30,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, page)
	}
}

func handleListAlbumMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		albumID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || albumID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "相册ID 无效"})
			return
		}
		page, err := registrar.GetAlbumMedia(storage.ListAlbumPhotosParams{
			AlbumID: albumID,
			UserID:  userID,
			Cursor:  c.Query("cursor"),
			Limit:   30,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, page)
	}
}

func handleAddMediaToAlbum(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		albumID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || albumID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "相册ID 无效"})
			return
		}

		var req albumMediaRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
			return
		}
		mediaID := req.MediaID
		if mediaID <= 0 {
			mediaID = req.PhotoID
		}
		if mediaID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "media_id 不能为空"})
			return
		}
		if err := registrar.AddPhoto(albumID, mediaID, userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "已添加到相册"})
	}
}

func handleRemoveMediaFromAlbum(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		albumID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || albumID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "相册ID 无效"})
			return
		}
		mediaID, err := strconv.ParseInt(c.Param("mediaId"), 10, 64)
		if err != nil || mediaID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "照片/视频ID 无效"})
			return
		}
		if err := registrar.RemovePhoto(albumID, mediaID, userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "已从相册移除"})
	}
}

func handleGetAlbumDetail(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		albumID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || albumID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "相册ID 无效"})
			return
		}
		album, err := registrar.GetAlbum(albumID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if album == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "相册不存在"})
			return
		}
		c.JSON(http.StatusOK, album)
	}
}

func handleListAlbumsMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		albums, err := registrar.ListAlbums(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, albums)
	}
}

func handleCreateAlbumMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		var req albumRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
			return
		}
		album, err := registrar.CreateAlbum(req.Name, req.Description, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, album)
	}
}

func handleUpdateAlbumMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		albumID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || albumID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "相册ID 无效"})
			return
		}
		var req albumRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
			return
		}
		album, err := registrar.UpdateAlbum(albumID, req.Name, req.Description, req.CoverPhotoID, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, album)
	}
}

func handleDeleteAlbumMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		albumID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || albumID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "相册ID 无效"})
			return
		}
		if err := registrar.DeleteAlbum(albumID, userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "相册已删除"})
	}
}

func handleListSharesMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		links, err := registrar.ListShares(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, links)
	}
}

func handleCreateShareMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		var req shareRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
			return
		}
		var expiresAt *time.Time
		if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
			t := time.Now().Add(time.Duration(*req.ExpiresIn) * 24 * time.Hour)
			expiresAt = &t
		}
		link, err := registrar.CreateShare(service.CreateShareInput{
			Type:      req.Type,
			TargetID:  req.TargetID,
			UserID:    userID,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, link)
	}
}

func handleGetShareByToken(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		link, err := registrar.GetShareByToken(c.Param("token"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if link == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "分享链接不存在或已过期"})
			return
		}
		resp := shareDetailResponse{ShareLink: link}
		if link.Type == storage.ShareTypePhoto {
			photo, err := registrar.GetPhoto(link.TargetID, link.CreatedBy)
			if err == nil && photo != nil {
				resp.TargetUUID = photo.UUID
				resp.TargetOriginalName = photo.OriginalName
				resp.TargetMediaKind = photo.MediaKind
				resp.TargetMimeType = photo.MimeType
			}
		}
		c.JSON(http.StatusOK, resp)
	}
}

func handleGetSharedAlbumMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		link, err := registrar.GetShareByToken(c.Param("token"))
		if err != nil || link == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "分享链接不存在或已过期"})
			return
		}
		if link.Type != storage.ShareTypeAlbum {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前分享不是相册类型"})
			return
		}
		page, err := registrar.GetAlbumMedia(storage.ListAlbumPhotosParams{
			AlbumID: link.TargetID,
			UserID:  link.CreatedBy,
			Cursor:  c.Query("cursor"),
			Limit:   30,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, page)
	}
}

func handleServePhotoFile(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		uuid := strings.TrimSuffix(c.Param("uuid"), filepath.Ext(c.Param("uuid")))
		photo, err := registrar.GetPhotoByUUIDAny(uuid, userID)
		if err != nil || photo == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "照片/视频不存在"})
			return
		}
		c.File(registrar.MediaPath(photo))
	}
}

func handleServeThumbnailFile(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		uuid := strings.TrimSuffix(c.Param("uuid"), filepath.Ext(c.Param("uuid")))
		photo, err := registrar.GetPhotoByUUIDAny(uuid, userID)
		if err != nil || photo == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "照片/视频不存在"})
			return
		}
		c.File(registrar.ThumbnailPath(photo))
	}
}

func handleServeSharedMediaFile(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		link, err := registrar.GetShareByToken(c.Param("token"))
		if err != nil || link == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "分享链接不存在或已过期"})
			return
		}

		switch link.Type {
		case storage.ShareTypePhoto:
			photo, err := registrar.GetPhoto(link.TargetID, link.CreatedBy)
			if err != nil || photo == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "照片/视频不存在"})
				return
			}
			c.File(registrar.MediaPath(photo))
		case storage.ShareTypeAlbum:
			uuid := strings.TrimSuffix(c.Param("uuid"), filepath.Ext(c.Param("uuid")))
			page, err := registrar.GetAlbumMedia(storage.ListAlbumPhotosParams{AlbumID: link.TargetID, UserID: link.CreatedBy, Limit: 10000})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			for _, photo := range page.Photos {
				if photo.UUID == uuid {
					c.File(registrar.MediaPath(photo))
					return
				}
			}
			c.JSON(http.StatusNotFound, gin.H{"error": "照片/视频不存在"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前分享不支持该媒体访问"})
		}
	}
}

func handleDownloadSharedMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		link, err := registrar.GetShareByToken(c.Param("token"))
		if err != nil || link == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "分享链接不存在或已过期"})
			return
		}
		if link.Type != storage.ShareTypePhoto {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前分享不支持下载"})
			return
		}
		photo, err := registrar.GetPhoto(link.TargetID, link.CreatedBy)
		if err != nil || photo == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "照片/视频不存在"})
			return
		}
		c.Header("Content-Type", photo.MimeType)
		c.Header("Content-Disposition", contentDispositionAttachment(photo.OriginalName))
		c.File(registrar.MediaPath(photo))
	}
}

func handleDeleteShareMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "分享ID 无效"})
			return
		}
		if err := registrar.DeleteShare(id, userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "分享链接已删除"})
	}
}

func handleDownloadAlbumMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		albumID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || albumID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "相册ID 无效"})
			return
		}
		albumName, entries, err := registrar.GetAlbumDownloadEntries(albumID, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Header("Content-Type", "application/zip")
		c.Header("Content-Disposition", contentDispositionAttachment(sanitizeZipName(albumName)))
		if err := writeZipResponse(c.Writer, entries); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "打包下载失败"})
			return
		}
	}
}

func handleEmptyTrashMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		if err := registrar.EmptyTrash(userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "回收站已清空"})
	}
}

func handleHardDeleteMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "照片/视频ID 无效"})
			return
		}
		if err := registrar.PermanentlyDeletePhoto(id, userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "已永久删除"})
	}
}

func handleDownloadMediaBatch(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		var req mediaDownloadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体"})
			return
		}

		ids := req.MediaIDs
		if len(ids) == 0 {
			ids = req.PhotoIDs
		}
		if len(ids) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "media_ids 不能为空"})
			return
		}

		entries, err := registrar.GetDownloadEntries(ids, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		zipName := time.Now().Format("photoalbum-selection-20060102-150405.zip")
		c.Header("Content-Type", "application/zip")
		c.Header("Content-Disposition", contentDispositionAttachment(zipName))
		if err := writeZipResponse(c.Writer, entries); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "打包下载失败"})
			return
		}
	}
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

func handleListMedia(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		page, err := registrar.GetTimeline(storage.ListPhotosParams{
			UserID: userID,
			Cursor: c.Query("cursor"),
			Limit:  30,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, page)
	}
}

func handleServeMediaFile(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		uuid := strings.TrimSuffix(c.Param("uuid"), filepath.Ext(c.Param("uuid")))
		photo, err := registrar.GetPhotoByUUIDAny(uuid, userID)
		if err != nil || photo == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "媒体不存在"})
			return
		}
		c.File(registrar.MediaPath(photo))
	}
}

func handleServePoster(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体服务未配置"})
			return
		}
		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		uuid := strings.TrimSuffix(c.Param("uuid"), filepath.Ext(c.Param("uuid")))
		photo, err := registrar.GetPhotoByUUIDAny(uuid, userID)
		if err != nil || photo == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "媒体不存在"})
			return
		}
		posterPath := registrar.PosterPath(photo)
		if _, err := os.Stat(posterPath); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "poster 不存在"})
			return
		}
		c.File(posterPath)
	}
}

func handleUploadPlaceholder(cfg *config.Config, registrar videoRegistrar) gin.HandlerFunc {
	return func(c *gin.Context) {
		if registrar == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "媒体上传服务未配置"})
			return
		}
		file, err := c.FormFile("media")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 media 文件字段"})
			return
		}
		if file.Size <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "上传文件不能为空"})
			return
		}
		if !strings.EqualFold(filepath.Ext(file.Filename), ".mp4") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前仅支持 mp4 视频上传"})
			return
		}

		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "打开上传文件失败"})
			return
		}
		defer src.Close()

		header := make([]byte, 512)
		n, err := io.ReadFull(src, header)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "读取上传文件失败"})
			return
		}

		tempPath, err := saveUploadedMedia(cfg.StoragePath, file.Filename, io.MultiReader(bytes.NewReader(header[:n]), src))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		meta := &media.VideoMeta{FormatName: "mp4"}

		uuid := strings.TrimSuffix(filepath.Base(tempPath), filepath.Ext(tempPath))
		finalPath := filepath.Join(cfg.StoragePath, uuid+filepath.Ext(file.Filename))
		if err := os.Rename(tempPath, finalPath); err != nil {
			_ = os.Remove(tempPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "移动媒体文件失败"})
			return
		}

		userID, err := currentUserID(cfg, currentUsername(c))
		if err != nil {
			_ = os.Remove(finalPath)
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		photo, err := registrar.RegisterUploadedVideo(service.RegisterUploadedVideoInput{
			UUID:         uuid,
			OriginalName: file.Filename,
			MimeType:     "video/mp4",
			Size:         file.Size,
			UploadedBy:   userID,
			TakenAt:      time.Now(),
			Meta:         meta,
		})
		if err != nil {
			_ = os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":      "视频上传成功",
			"filename":     file.Filename,
			"path":         finalPath,
			"poster_path":  "",
			"poster_error": "",
			"probe_error":  "",
			"meta":         meta,
			"photo":        photo,
		})
	}
}

func saveUploadedMedia(storagePath, originalName string, src io.Reader) (string, error) {
	baseDir := filepath.Join(storagePath, tempMediaDirName)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", fmt.Errorf("创建临时媒体目录失败: %w", err)
	}

	fileID := uuid.NewString()
	tempPath := filepath.Join(baseDir, fileID+filepath.Ext(originalName))
	dst, err := os.Create(tempPath)
	if err != nil {
		return "", fmt.Errorf("创建临时媒体文件失败: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("保存临时媒体文件失败: %w", err)
	}

	return tempPath, nil
}

func authMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(authCookieName)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或登录已过期"})
			c.Abort()
			return
		}

		username, err := parseToken(cfg.JWTSecret, cookie)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或登录已过期"})
			c.Abort()
			return
		}
		if !usernameExists(cfg.Users, username) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
			c.Abort()
			return
		}

		c.Set(string(userContextKey), username)
		c.Next()
	}
}

func currentUsername(c *gin.Context) string {
	username, _ := c.Get(string(userContextKey))
	v, _ := username.(string)
	return v
}

func currentUserID(cfg *config.Config, username string) (int64, error) {
	for i, u := range cfg.Users {
		if u.Username == username {
			return int64(i + 1), nil
		}
	}
	return 0, errors.New("用户不存在")
}

func parseToken(secret, tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token")
	}
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return "", errors.New("token expired")
	}
	return claims.Username, nil
}

func usernameExists(users []config.User, username string) bool {
	for _, u := range users {
		if u.Username == username {
			return true
		}
	}
	return false
}
