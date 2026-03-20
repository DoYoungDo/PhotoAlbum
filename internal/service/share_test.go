package service

import (
	"testing"
	"time"

	"photoalbum/internal/storage"
)

func newTestShareService() *ShareService {
	return NewShareService(newMockRepo())
}

func TestCreateShare_Photo(t *testing.T) {
	svc := newTestShareService()
	link, err := svc.CreateShare(CreateShareInput{
		Type:     storage.ShareTypePhoto,
		TargetID: 1,
		UserID:   1,
	})
	if err != nil {
		t.Fatalf("创建分享链接失败: %v", err)
	}
	if link.Token == "" {
		t.Error("Token 不能为空")
	}
	if link.ExpiresAt != nil {
		t.Error("未设置过期时间时 ExpiresAt 应为 nil")
	}
}

func TestCreateShare_WithExpiry(t *testing.T) {
	svc := newTestShareService()
	future := time.Now().Add(7 * 24 * time.Hour)
	link, err := svc.CreateShare(CreateShareInput{
		Type:      storage.ShareTypeAlbum,
		TargetID:  2,
		UserID:    1,
		ExpiresAt: &future,
	})
	if err != nil {
		t.Fatal(err)
	}
	if link.ExpiresAt == nil || !link.ExpiresAt.Equal(future) {
		t.Error("过期时间不匹配")
	}
}

func TestCreateShare_InvalidType(t *testing.T) {
	svc := newTestShareService()
	_, err := svc.CreateShare(CreateShareInput{
		Type:     "invalid",
		TargetID: 1,
		UserID:   1,
	})
	if err == nil {
		t.Error("无效类型应该返回错误")
	}
}

func TestGetShareByToken_Success(t *testing.T) {
	svc := newTestShareService()
	link, _ := svc.CreateShare(CreateShareInput{
		Type: storage.ShareTypePhoto, TargetID: 1, UserID: 1,
	})

	got, err := svc.GetShareByToken(link.Token)
	if err != nil || got == nil {
		t.Fatalf("查询分享链接失败: %v", err)
	}
	if got.TargetID != 1 {
		t.Errorf("TargetID 不匹配")
	}
}

func TestGetShareByToken_NotFound(t *testing.T) {
	svc := newTestShareService()
	got, _ := svc.GetShareByToken("nonexistent-token")
	if got != nil {
		t.Error("不存在的 token 应该返回 nil")
	}
}

func TestGenerateToken_Unique(t *testing.T) {
	t1, _ := generateToken(16)
	t2, _ := generateToken(16)
	if t1 == t2 {
		t.Error("两次生成的 token 不应该相同")
	}
}

func TestGenerateToken_Length(t *testing.T) {
	token, _ := generateToken(16)
	// 16 字节 hex 编码后为 32 字符
	if len(token) != 32 {
		t.Errorf("期望长度 32，得到 %d", len(token))
	}
}

func TestDeleteShare_Success(t *testing.T) {
	svc := newTestShareService()
	link, _ := svc.CreateShare(CreateShareInput{
		Type: storage.ShareTypePhoto, TargetID: 1, UserID: 1,
	})

	if err := svc.DeleteShare(link.ID, 1); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	got, _ := svc.GetShareByToken(link.Token)
	if got != nil {
		t.Error("删除后不应该查到分享链接")
	}
}

func TestListShares_UserIsolation(t *testing.T) {
	svc := newTestShareService()
	svc.CreateShare(CreateShareInput{Type: storage.ShareTypePhoto, TargetID: 1, UserID: 1})
	svc.CreateShare(CreateShareInput{Type: storage.ShareTypePhoto, TargetID: 2, UserID: 1})
	svc.CreateShare(CreateShareInput{Type: storage.ShareTypePhoto, TargetID: 3, UserID: 2})

	links, _ := svc.ListShares(1)
	if len(links) != 2 {
		t.Errorf("期望 2 条，得到 %d", len(links))
	}
}
