package service

import (
	"testing"
	"time"

	"photoalbum/internal/storage"
)

func newTestAlbumService() (*AlbumService, *mockRepo) {
	repo := newMockRepo()
	return NewAlbumService(repo), repo
}

func TestCreateAlbum_Success(t *testing.T) {
	svc, _ := newTestAlbumService()
	album, err := svc.CreateAlbum("旅行", "2024年旅行照片", 1)
	if err != nil {
		t.Fatalf("创建相册失败: %v", err)
	}
	if album.ID == 0 {
		t.Error("创建后 ID 应该被填充")
	}
	if album.Name != "旅行" {
		t.Errorf("相册名不匹配: %s", album.Name)
	}
}

func TestCreateAlbum_EmptyName(t *testing.T) {
	svc, _ := newTestAlbumService()
	_, err := svc.CreateAlbum("", "", 1)
	if err == nil {
		t.Error("空名称应该返回错误")
	}
}

func TestListAlbums_UserIsolation(t *testing.T) {
	svc, _ := newTestAlbumService()
	svc.CreateAlbum("用户1的相册", "", 1)
	svc.CreateAlbum("用户2的相册", "", 2)

	albums, _ := svc.ListAlbums(1)
	if len(albums) != 1 {
		t.Errorf("用户隔离失败，期望 1 个相册，得到 %d", len(albums))
	}
}

func TestUpdateAlbum_Success(t *testing.T) {
	svc, _ := newTestAlbumService()
	album, _ := svc.CreateAlbum("旧名字", "", 1)

	updated, err := svc.UpdateAlbum(album.ID, "新名字", "新描述", nil, 1)
	if err != nil {
		t.Fatalf("更新失败: %v", err)
	}
	if updated.Name != "新名字" {
		t.Errorf("名字未更新: %s", updated.Name)
	}
}

func TestUpdateAlbum_NotFound(t *testing.T) {
	svc, _ := newTestAlbumService()
	_, err := svc.UpdateAlbum(999, "名字", "", nil, 1)
	if err == nil {
		t.Error("不存在的相册应该返回错误")
	}
}

func TestDeleteAlbum_Success(t *testing.T) {
	svc, _ := newTestAlbumService()
	album, _ := svc.CreateAlbum("待删除", "", 1)

	if err := svc.DeleteAlbum(album.ID, 1); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	got, _ := svc.GetAlbum(album.ID, 1)
	if got != nil {
		t.Error("删除后不应该能查到相册")
	}
}

func TestAddAndRemovePhoto(t *testing.T) {
	svc, repo := newTestAlbumService()
	album, _ := svc.CreateAlbum("相册", "", 1)

	// 先插入一张图片到 mock repo
	photo := &storage.Photo{UUID: "test-uuid", OriginalName: "a.jpg", MimeType: "image/jpeg", TakenAt: time.Now(), UploadedAt: time.Now(), UploadedBy: 1}
	repo.SavePhoto(photo)

	// 添加
	if err := svc.AddPhoto(album.ID, photo.ID, 1); err != nil {
		t.Fatalf("添加图片失败: %v", err)
	}

	page, _ := svc.ListAlbumPhotos(storage.ListAlbumPhotosParams{
		AlbumID: album.ID, UserID: 1, Limit: 10,
	})
	if len(page.Photos) != 1 {
		t.Errorf("期望 1 张，得到 %d", len(page.Photos))
	}

	// 移除
	if err := svc.RemovePhoto(album.ID, photo.ID, 1); err != nil {
		t.Fatalf("移除图片失败: %v", err)
	}
	page, _ = svc.ListAlbumPhotos(storage.ListAlbumPhotosParams{
		AlbumID: album.ID, UserID: 1, Limit: 10,
	})
	if len(page.Photos) != 0 {
		t.Error("移除后相册应该为空")
	}
}

func TestListAlbumPhotos_WrongAlbum(t *testing.T) {
	svc, _ := newTestAlbumService()
	_, err := svc.ListAlbumPhotos(storage.ListAlbumPhotosParams{
		AlbumID: 999, UserID: 1, Limit: 10,
	})
	if err == nil {
		t.Error("不存在的相册应该返回错误")
	}
}
