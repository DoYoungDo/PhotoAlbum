package sqlite

import (
	"testing"
	"time"

	"photoalbum/internal/storage"
)

func TestCreateAlbum_Success(t *testing.T) {
	db := newTestDB(t)
	album := &storage.Album{
		Name:      "旅行",
		CreatedBy: 1,
		CreatedAt: time.Now(),
	}
	if err := db.CreateAlbum(album); err != nil {
		t.Fatalf("创建相册失败: %v", err)
	}
	if album.ID == 0 {
		t.Error("创建后 ID 应该被填充")
	}
}

func TestGetAlbumByID_Success(t *testing.T) {
	db := newTestDB(t)
	album := &storage.Album{Name: "测试相册", CreatedBy: 1, CreatedAt: time.Now()}
	db.CreateAlbum(album)

	got, err := db.GetAlbumByID(album.ID, 1)
	if err != nil || got == nil {
		t.Fatalf("查询相册失败: %v", err)
	}
	if got.Name != "测试相册" {
		t.Errorf("相��名不匹配")
	}
}

func TestGetAlbumByID_WrongUser(t *testing.T) {
	db := newTestDB(t)
	album := &storage.Album{Name: "私有相册", CreatedBy: 1, CreatedAt: time.Now()}
	db.CreateAlbum(album)

	got, _ := db.GetAlbumByID(album.ID, 2)
	if got != nil {
		t.Error("不同用户不应该查到相册")
	}
}

func TestListAlbums_Success(t *testing.T) {
	db := newTestDB(t)
	db.CreateAlbum(&storage.Album{Name: "A", CreatedBy: 1, CreatedAt: time.Now()})
	db.CreateAlbum(&storage.Album{Name: "B", CreatedBy: 1, CreatedAt: time.Now()})
	db.CreateAlbum(&storage.Album{Name: "C", CreatedBy: 2, CreatedAt: time.Now()})

	albums, err := db.ListAlbums(1)
	if err != nil {
		t.Fatalf("查��失败: %v", err)
	}
	if len(albums) != 2 {
		t.Errorf("期望 2 个相册，得到 %d", len(albums))
	}
}

func TestUpdateAlbum_Success(t *testing.T) {
	db := newTestDB(t)
	album := &storage.Album{Name: "旧名字", CreatedBy: 1, CreatedAt: time.Now()}
	db.CreateAlbum(album)

	album.Name = "新名字"
	album.Description = "描述"
	if err := db.UpdateAlbum(album); err != nil {
		t.Fatalf("更新失败: %v", err)
	}

	got, _ := db.GetAlbumByID(album.ID, 1)
	if got.Name != "新名字" || got.Description != "描述" {
		t.Errorf("更新后数据不匹配")
	}
}

func TestDeleteAlbum_Success(t *testing.T) {
	db := newTestDB(t)
	album := &storage.Album{Name: "待删除", CreatedBy: 1, CreatedAt: time.Now()}
	db.CreateAlbum(album)

	if err := db.DeleteAlbum(album.ID, 1); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	got, _ := db.GetAlbumByID(album.ID, 1)
	if got != nil {
		t.Error("删除后不应该查到相册")
	}
}

func TestAddAndRemovePhotoFromAlbum(t *testing.T) {
	db := newTestDB(t)
	album := &storage.Album{Name: "相册", CreatedBy: 1, CreatedAt: time.Now()}
	db.CreateAlbum(album)
	p := makePhoto(1, time.Now())
	db.SavePhoto(p)

	// 添加
	if err := db.AddPhotoToAlbum(album.ID, p.ID, 1); err != nil {
		t.Fatalf("添加图片到相册失败: %v", err)
	}
	inAlbum, _ := db.IsPhotoInAlbum(album.ID, p.ID)
	if !inAlbum {
		t.Error("图片应该在相册中")
	}

	// 重复添加应该不报错（INSERT OR IGNORE）
	if err := db.AddPhotoToAlbum(album.ID, p.ID, 1); err != nil {
		t.Errorf("重复添加不应该报错: %v", err)
	}

	// 移除
	if err := db.RemovePhotoFromAlbum(album.ID, p.ID, 1); err != nil {
		t.Fatalf("移除图片失败: %v", err)
	}
	inAlbum, _ = db.IsPhotoInAlbum(album.ID, p.ID)
	if inAlbum {
		t.Error("移除后不应该在相册中")
	}
}

func TestListAlbumPhotos_Pagination(t *testing.T) {
	db := newTestDB(t)
	album := &storage.Album{Name: "相册", CreatedBy: 1, CreatedAt: time.Now()}
	db.CreateAlbum(album)

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		p := makePhoto(1, base.Add(time.Duration(i)*time.Hour))
		p.UUID = "album-photo-" + string(rune('a'+i))
		db.SavePhoto(p)
		db.AddPhotoToAlbum(album.ID, p.ID, 1)
	}

	page1, err := db.ListAlbumPhotos(storage.ListAlbumPhotosParams{
		AlbumID: album.ID, UserID: 1, Limit: 3,
	})
	if err != nil {
		t.Fatalf("查询相册图片失败: %v", err)
	}
	if len(page1.Photos) != 3 || !page1.HasMore {
		t.Errorf("第一页应该有 3 张且有更多，实际 %d 张 hasMore=%v", len(page1.Photos), page1.HasMore)
	}

	page2, _ := db.ListAlbumPhotos(storage.ListAlbumPhotosParams{
		AlbumID: album.ID, UserID: 1, Limit: 3, Cursor: page1.NextCursor,
	})
	if len(page2.Photos) != 2 || page2.HasMore {
		t.Errorf("第二页应该有 2 张且无更多，实际 %d 张 hasMore=%v", len(page2.Photos), page2.HasMore)
	}
}

func TestAlbumPhotoCount(t *testing.T) {
	db := newTestDB(t)
	album := &storage.Album{Name: "计数测试", CreatedBy: 1, CreatedAt: time.Now()}
	db.CreateAlbum(album)

	photos := make([]*storage.Photo, 3)
	for i := 0; i < 3; i++ {
		p := makePhoto(1, time.Now())
		p.UUID = "count-" + string(rune('a'+i))
		db.SavePhoto(p)
		db.AddPhotoToAlbum(album.ID, p.ID, 1)
		photos[i] = p
	}

	got, _ := db.GetAlbumByID(album.ID, 1)
	if got.PhotoCount != 3 {
		t.Errorf("期望 PhotoCount=3，得到 %d", got.PhotoCount)
	}

	// 软删除一张后计数应该减少
	db.SoftDeletePhoto(photos[0].ID, 1, 1)
	got, _ = db.GetAlbumByID(album.ID, 1)
	if got.PhotoCount != 2 {
		t.Errorf("软删除后期望 PhotoCount=2，得到 %d", got.PhotoCount)
	}

	// ListAlbums 也要验证
	albums, _ := db.ListAlbums(1)
	if len(albums) == 0 || albums[0].PhotoCount != 2 {
		t.Errorf("ListAlbums 软删除后期望 PhotoCount=2，得到 %d", albums[0].PhotoCount)
	}
}

func TestAlbumCoverUUID(t *testing.T) {
	db := newTestDB(t)
	album := &storage.Album{Name: "封面测试", CreatedBy: 1, CreatedAt: time.Now()}
	db.CreateAlbum(album)

	// 无图片时 CoverUUID 为空
	got, _ := db.GetAlbumByID(album.ID, 1)
	if got.CoverUUID != "" {
		t.Errorf("无图片时 CoverUUID 应为空，得到 %s", got.CoverUUID)
	}

	// 添加图片后 CoverUUID 应为该图片的 UUID
	p := makePhoto(1, time.Now())
	p.UUID = "cover-test-uuid"
	db.SavePhoto(p)
	db.AddPhotoToAlbum(album.ID, p.ID, 1)

	got, _ = db.GetAlbumByID(album.ID, 1)
	if got.CoverUUID != "cover-test-uuid" {
		t.Errorf("期望 CoverUUID=cover-test-uuid，得到 %s", got.CoverUUID)
	}

	// 软删除图片后 CoverUUID 应再次为空
	db.SoftDeletePhoto(p.ID, 1, 1)
	got, _ = db.GetAlbumByID(album.ID, 1)
	if got.CoverUUID != "" {
		t.Errorf("软删除后 CoverUUID 应为空，得到 %s", got.CoverUUID)
	}
}
