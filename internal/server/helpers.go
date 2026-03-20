package server

import (
	"fmt"
	"net/http"
	"strconv"

	"photoalbum/internal/config"
)

func (s *Server) userIDByUsername(username string) (int64, error) {
	for i, u := range s.cfg.Users {
		if u.Username == username {
			return int64(i + 1), nil
		}
	}
	return 0, fmt.Errorf("用户不存在")
}

func (s *Server) currentUserID(username string) (int64, error) {
	return s.userIDByUsername(username)
}

// mustUserID 获取当前用户 ID，失败则写 401 并返回 0，调用方应检查返回值
func (s *Server) mustUserID(w http.ResponseWriter, r *http.Request) int64 {
	userID, err := s.currentUserID(currentUsername(r))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "用户不存在")
		return 0
	}
	return userID
}

func parseInt64Param(value string, name string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("无效的 %s", name)
	}
	return id, nil
}

func usernameExists(users []config.User, username string) bool {
	for _, u := range users {
		if u.Username == username {
			return true
		}
	}
	return false
}
