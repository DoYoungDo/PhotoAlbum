package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewRouter 创建新的 Gin 路由入口。
//
// 当前阶段仅为后续媒体能力预留 Gin 路由组，
// 其余现有功能继续回退到 legacy handler。
func NewRouter(legacy http.Handler) http.Handler {
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
	}

	legacyHandler := gin.WrapH(legacy)
	r.NoRoute(legacyHandler)
	r.NoMethod(legacyHandler)

	return r
}
