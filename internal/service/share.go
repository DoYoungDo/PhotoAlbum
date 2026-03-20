package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"photoalbum/internal/storage"
)

// ShareService 分享链接业务逻辑
type ShareService struct {
	repo storage.Repository
}

// NewShareService 创建分享服务
func NewShareService(repo storage.Repository) *ShareService {
	return &ShareService{repo: repo}
}

// CreateShareInput 创建分享链接的输入
type CreateShareInput struct {
	Type      string // "photo" 或 "album"
	TargetID  int64
	UserID    int64
	ExpiresAt *time.Time // nil 表示永不过期
}

// CreateShare 创建分享链接
func (s *ShareService) CreateShare(input CreateShareInput) (*storage.ShareLink, error) {
	if input.Type != storage.ShareTypePhoto && input.Type != storage.ShareTypeAlbum {
		return nil, fmt.Errorf("无效的分享类型: %s", input.Type)
	}

	token, err := generateToken(16)
	if err != nil {
		return nil, fmt.Errorf("生成 token 失败: %w", err)
	}

	link := &storage.ShareLink{
		Token:     token,
		Type:      input.Type,
		TargetID:  input.TargetID,
		CreatedBy: input.UserID,
		ExpiresAt: input.ExpiresAt,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateShareLink(link); err != nil {
		return nil, err
	}
	return link, nil
}

// GetShareByToken 通过 token 获取分享链接（已过期返回 nil）
func (s *ShareService) GetShareByToken(token string) (*storage.ShareLink, error) {
	return s.repo.GetShareLinkByToken(token)
}

// ListShares 获取用户所有分享链接
func (s *ShareService) ListShares(userID int64) ([]*storage.ShareLink, error) {
	return s.repo.ListShareLinks(userID)
}

// DeleteShare 删除分享���接
func (s *ShareService) DeleteShare(id int64, userID int64) error {
	return s.repo.DeleteShareLink(id, userID)
}

// generateToken 生成指定字节长度的随机 hex token
func generateToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
