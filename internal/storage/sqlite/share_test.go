package sqlite

import (
	"testing"
	"time"

	"photoalbum/internal/storage"
)

func makeShareLink(userID int64, targetID int64, linkType string) *storage.ShareLink {
	return &storage.ShareLink{
		Token:     "token-" + time.Now().Format("150405.000000000"),
		Type:      linkType,
		TargetID:  targetID,
		CreatedBy: userID,
		CreatedAt: time.Now(),
	}
}

func TestCreateShareLink_Success(t *testing.T) {
	db := newTestDB(t)
	link := makeShareLink(1, 10, storage.ShareTypePhoto)

	if err := db.CreateShareLink(link); err != nil {
		t.Fatalf("创建分享链接失败: %v", err)
	}
	if link.ID == 0 {
		t.Error("创建后 ID 应该被填充")
	}
}

func TestGetShareLinkByToken_Success(t *testing.T) {
	db := newTestDB(t)
	link := makeShareLink(1, 10, storage.ShareTypePhoto)
	db.CreateShareLink(link)

	got, err := db.GetShareLinkByToken(link.Token)
	if err != nil || got == nil {
		t.Fatalf("查询分享链接失败: %v", err)
	}
	if got.TargetID != 10 {
		t.Errorf("TargetID 不匹配")
	}
}

func TestGetShareLinkByToken_NotFound(t *testing.T) {
	db := newTestDB(t)
	got, err := db.GetShareLinkByToken("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("不存在的 token 应该返回 nil")
	}
}

func TestGetShareLinkByToken_Expired(t *testing.T) {
	db := newTestDB(t)
	link := makeShareLink(1, 10, storage.ShareTypePhoto)
	past := time.Now().Add(-time.Hour)
	link.ExpiresAt = &past
	db.CreateShareLink(link)

	got, _ := db.GetShareLinkByToken(link.Token)
	if got != nil {
		t.Error("过期的分享链接不应该被返回")
	}
}

func TestGetShareLinkByToken_NotExpired(t *testing.T) {
	db := newTestDB(t)
	link := makeShareLink(1, 10, storage.ShareTypeAlbum)
	future := time.Now().Add(24 * time.Hour)
	link.ExpiresAt = &future
	db.CreateShareLink(link)

	got, _ := db.GetShareLinkByToken(link.Token)
	if got == nil {
		t.Error("未过期的分享链接应该被返回")
	}
}

func TestGetShareLinkByToken_NeverExpires(t *testing.T) {
	db := newTestDB(t)
	link := makeShareLink(1, 10, storage.ShareTypePhoto)
	// ExpiresAt 为 nil，永不过期
	db.CreateShareLink(link)

	got, _ := db.GetShareLinkByToken(link.Token)
	if got == nil {
		t.Error("永不过期的分享链接应该被返回")
	}
}

func TestListShareLinks_Success(t *testing.T) {
	db := newTestDB(t)
	db.CreateShareLink(makeShareLink(1, 1, storage.ShareTypePhoto))
	db.CreateShareLink(makeShareLink(1, 2, storage.ShareTypeAlbum))
	db.CreateShareLink(makeShareLink(2, 3, storage.ShareTypePhoto))

	links, err := db.ListShareLinks(1)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(links) != 2 {
		t.Errorf("期望 2 条，得到 %d", len(links))
	}
}

func TestDeleteShareLink_Success(t *testing.T) {
	db := newTestDB(t)
	link := makeShareLink(1, 10, storage.ShareTypePhoto)
	db.CreateShareLink(link)

	if err := db.DeleteShareLink(link.ID, 1); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	got, _ := db.GetShareLinkByToken(link.Token)
	if got != nil {
		t.Error("删除后不应该查到")
	}
}

func TestDeleteShareLink_WrongUser(t *testing.T) {
	db := newTestDB(t)
	link := makeShareLink(1, 10, storage.ShareTypePhoto)
	db.CreateShareLink(link)

	if err := db.DeleteShareLink(link.ID, 2); err == nil {
		t.Error("不同用户不应该能删除分享链接")
	}
}
