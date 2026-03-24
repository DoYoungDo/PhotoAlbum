package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

var probeVideoFunc = media.ProbeVideo
var generatePosterFunc = media.GeneratePoster

type videoRegistrar interface {
	RegisterUploadedVideo(input service.RegisterUploadedVideoInput) (*storage.Photo, error)
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
		media.GET("", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{
				"error": "媒体接口尚未实现",
			})
		})
		media.POST("/upload", authMiddleware(cfg), handleUploadPlaceholder(cfg, registrar))
	}

	legacyHandler := gin.WrapH(legacy)
	r.NoRoute(legacyHandler)
	r.NoMethod(legacyHandler)

	return r
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

		meta, err := probeVideoFunc(tempPath)
		if err != nil {
			_ = os.Remove(tempPath)
			status := http.StatusBadRequest
			if errors.Is(err, media.ErrProbeUnavailable) {
				status = http.StatusInternalServerError
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}

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

		posterPath := media.PosterPath(cfg.StoragePath, photo.UUID)
		posterError := ""
		if err := generatePosterFunc(finalPath, posterPath); err != nil {
			posterError = err.Error()
			posterPath = ""
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":      "视频上传成功",
			"filename":     file.Filename,
			"path":         finalPath,
			"poster_path":  posterPath,
			"poster_error": posterError,
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
