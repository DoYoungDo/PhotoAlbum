package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"photoalbum/internal/config"
)

type contextKey string

const userContextKey contextKey = "user"
const authCookieName = "photoalbum_token"

// Claims JWT claims
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// generateToken 为指定用户生成 JWT
func (s *Server) generateToken(username string) (string, error) {
	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

// parseToken 解析 JWT 并返回用户名
func (s *Server) parseToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}
	return claims.Username, nil
}

// auth 登录校验中间件，未登录则根据请求类型返回 401 或跳转登录页
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(authCookieName)
		if err != nil {
			s.handleAuthFailure(w, r)
			return
		}

		username, err := s.parseToken(cookie.Value)
		if err != nil {
			s.handleAuthFailure(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, username)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) handleAuthFailure(w http.ResponseWriter, r *http.Request) {
	if isPageRequest(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	writeError(w, http.StatusUnauthorized, "未登录或登录已过期")
}

// isPageRequest 判断是否为页面请求（非 API 路径）
// API 路径以 /api/ 开头，始终返回 JSON
func isPageRequest(r *http.Request) bool {
	path := r.URL.Path
	if len(path) >= 5 && path[:5] == "/api/" {
		return false
	}
	if len(path) >= 8 && path[:8] == "/media/" {
		return false
	}
	return true
}

// currentUsername 从请求上下文获取当前登录用户名
func currentUsername(r *http.Request) string {
	username, _ := r.Context().Value(userContextKey).(string)
	return username
}

// findUser 查找配置中的用户
func (s *Server) findUser(username string) *config.User {
	for _, u := range s.cfg.Users {
		if u.Username == username {
			user := u
			return &user
		}
	}
	return nil
}
