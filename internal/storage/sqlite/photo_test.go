package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"photoalbum/internal/storage"
)

// newTestDB 创建测试用内存数据库
func newTestDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// makePhoto 创建测试用 Photo
func makePhoto(userID int64, takenAt time.Time) *storage.Photo {
	return &storage.Photo{
		UUID:         "uuid-" + takenAt.Format("20060102150405"),
		OriginalName: "test.jpg",
		MimeType:     "image/jpeg",
		Size:         1024,
		Width:        800,
		Height:       600,
		TakenAt:      takenAt,
		UploadedAt:   time.Now(),
		UploadedBy:   userID,
	}
}

// --- DB 初始化测试 ---

func TestNew_CreatesSchema(t *testing.T) {
	db := newTestDB(t)
	// 验证表存在
	tables := []string{"photos", "albums", "album_photos", "share_links"}
	for _, table := range tables {
		var name string
		err := db.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		if err != nil {
			t.Errorf("表 %s 不存在: %v", table, err)
		}
	}
}

func TestNew_InvalidPath(t *testing.T) {
	_, err := New("/nonexistent/path/test.db")
	if err == nil {
		t.Error("无效路径应该返回错误")
	}
}

// --- Photo 测试 ---

func TestSavePhoto_Success(t *testing.T) {
	db := newTestDB(t)
	p := makePhoto(1, time.Now())

	if err := db.SavePhoto(p); err != nil {
		t.Fatalf("保存图片失败: %v", err)
	}
	if p.ID == 0 {
		t.Error("保存后 ID 应该被填充")
	}
}

func TestSavePhoto_DuplicateUUID(t *testing.T) {
	db := newTestDB(t)
	p := makePhoto(1, time.Now())
	db.SavePhoto(p)

	p2 := makePhoto(1, time.Now())
	p2.UUID = p.UUID
	if err := db.SavePhoto(p2); err == nil {
		t.Error("重复 UUID 应该返回错误")
	}
}

func TestGetPhotoByID_Success(t *testing.T) {
	db := newTestDB(t)
	p := makePhoto(1, time.Now())
	db.SavePhoto(p)

	got, err := db.GetPhotoByID(p.ID, 1)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if got == nil || got.UUID != p.UUID {
		t.Errorf("查询结果不匹配")
	}
}

func TestGetPhotoByID_WrongUser(t *testing.T) {
	db := newTestDB(t)
	p := makePhoto(1, time.Now())
	db.SavePhoto(p)

	got, err := db.GetPhotoByID(p.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("不同用户不应该能查到图片")
	}
}

func TestGetPhotoByUUID_Success(t *testing.T) {
	db := newTestDB(t)
	p := makePhoto(1, time.Now())
	db.SavePhoto(p)

	got, err := db.GetPhotoByUUID(p.UUID, 1)
	if err != nil || got == nil {
		t.Fatalf("按 UUID 查询失败: %v", err)
	}
	if got.ID != p.ID {
		t.Errorf("期望 ID=%d，得到 %d", p.ID, got.ID)
	}
}

func TestGetPhotoByUUIDAny_IncludesDeleted(t *testing.T) {
	db := newTestDB(t)
	p := makePhoto(1, time.Now())
	db.SavePhoto(p)
	db.SoftDeletePhoto(p.ID, 1, 1)

	// 普通查询不到
	got, _ := db.GetPhotoByUUID(p.UUID, 1)
	if got != nil {
		t.Error("软删除后 GetPhotoByUUID 不应该返回")
	}

	// Any 查询得到
	got, err := db.GetPhotoByUUIDAny(p.UUID, 1)
	if err != nil || got == nil {
		t.Fatalf("GetPhotoByUUIDAny 应该返回软删除图片: %v", err)
	}
	if got.DeletedAt == nil {
		t.Error("DeletedAt 应该不为 nil")
	}
}

func TestListPhotos_Pagination(t *testing.T) {
	db := newTestDB(t)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// 插入 5 张图片
	for i := 0; i < 5; i++ {
		p := makePhoto(1, base.Add(time.Duration(i)*time.Hour))
		p.UUID = filepath.Join("uuid", string(rune('a'+i)))
		db.SavePhoto(p)
	}

	// 第一页，limit=3
	page1, err := db.ListPhotos(storage.ListPhotosParams{UserID: 1, Limit: 3})
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(page1.Photos) != 3 {
		t.Errorf("期望 3 张，得到 %d", len(page1.Photos))
	}
	if !page1.HasMore {
		t.Error("应该有更多")
	}

	// 第二页
	page2, err := db.ListPhotos(storage.ListPhotosParams{UserID: 1, Limit: 3, Cursor: page1.NextCursor})
	if err != nil {
		t.Fatalf("查询第二页失败: %v", err)
	}
	if len(page2.Photos) != 2 {
		t.Errorf("期望 2 张，得到 %d", len(page2.Photos))
	}
	if page2.HasMore {
		t.Error("不应该有更多")
	}
}

func TestListPhotos_UserIsolation(t *testing.T) {
	db := newTestDB(t)
	db.SavePhoto(makePhoto(1, time.Now()))
	db.SavePhoto(makePhoto(2, time.Now()))

	page, _ := db.ListPhotos(storage.ListPhotosParams{UserID: 1, Limit: 10})
	if len(page.Photos) != 1 {
		t.Errorf("用户隔离失败，期望 1 张，得到 %d", len(page.Photos))
	}
}

func TestSoftDeletePhoto_And_Restore(t *testing.T) {
	db := newTestDB(t)
	p := makePhoto(1, time.Now())
	db.SavePhoto(p)

	// 软删除
	if err := db.SoftDeletePhoto(p.ID, 1, 1); err != nil {
		t.Fatalf("软删除失败: %v", err)
	}

	// 正常查询应该查不到
	got, _ := db.GetPhotoByID(p.ID, 1)
	if got != nil {
		t.Error("软删除后不应该在正常查询中出现")
	}

	// 回收站应该能查到
	trashed, _ := db.ListTrashedPhotos(storage.ListPhotosParams{UserID: 1, Limit: 10})
	if len(trashed.Photos) != 1 {
		t.Error("软删除后应该在回收站中出现")
	}

	// 恢复
	if err := db.RestorePhoto(p.ID, 1); err != nil {
		t.Fatalf("恢复失败: %v", err)
	}
	got, _ = db.GetPhotoByID(p.ID, 1)
	if got == nil {
		t.Error("恢复后应该能正常查到")
	}
}

func TestHardDeleteTrashedPhotos(t *testing.T) {
	db := newTestDB(t)
	p1 := makePhoto(1, time.Now())
	p2 := &storage.Photo{UUID: "uuid-2", OriginalName: "b.jpg", MimeType: "image/jpeg", Size: 512, TakenAt: time.Now(), UploadedAt: time.Now(), UploadedBy: 1}
	db.SavePhoto(p1)
	db.SavePhoto(p2)
	db.SoftDeletePhoto(p1.ID, 1, 1)
	db.SoftDeletePhoto(p2.ID, 1, 1)

	uuids, err := db.HardDeleteTrashedPhotos(1)
	if err != nil {
		t.Fatalf("清空回收站失败: %v", err)
	}
	if len(uuids) != 2 {
		t.Errorf("期望 2 个 UUID，得到 %d", len(uuids))
	}

	trashed, _ := db.ListTrashedPhotos(storage.ListPhotosParams{UserID: 1, Limit: 10})
	if len(trashed.Photos) != 0 {
		t.Error("清空后回收站应该为空")
	}
}

func TestSavePhoto_ConcurrentWritesDoNotBusy(t *testing.T) {
	db := newTestDB(t)
	const n = 20

	var wg sync.WaitGroup
	errCh := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p := makePhoto(1, time.Now().Add(time.Duration(i)*time.Second))
			p.UUID = fmt.Sprintf("concurrent-%d", i)
			if err := db.SavePhoto(p); err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("并发写入不应失败，得到错误: %v", err)
	}

	page, err := db.ListPhotos(storage.ListPhotosParams{UserID: 1, Limit: 100})
	if err != nil {
		t.Fatalf("查询并发写入结果失败: %v", err)
	}
	if len(page.Photos) != n {
		t.Fatalf("期望保存 %d 张图片，实际 %d 张", n, len(page.Photos))
	}
}

// --- 游标编解码测试 ---

func TestCursorEncodeDecode(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	encoded := encodeCursor(now, 42)
	c, err := decodeCursor(encoded)
	if err != nil {
		t.Fatalf("解码失败: %v", err)
	}
	if c.ID != 42 {
		t.Errorf("期望 ID=42，得到 %d", c.ID)
	}
	if !c.TakenAt.Equal(now) {
		t.Errorf("时间不匹配")
	}
}

func TestDecodeCursor_Invalid(t *testing.T) {
	_, err := decodeCursor("not-valid-base64!!!")
	if err == nil {
		t.Error("无效游标应该返回错误")
	}
}

func TestListPhotos_PaginationManySameTimestamp(t *testing.T) {
	db := newTestDB(t)
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 200; i++ {
		p := makePhoto(1, base)
		p.UUID = fmt.Sprintf("same-ts-%03d", i)
		if err := db.SavePhoto(p); err != nil {
			t.Fatalf("保存第 %d 张图片失败: %v", i, err)
		}
	}

	count := 0
	cursor := ""
	for {
		page, err := db.ListPhotos(storage.ListPhotosParams{UserID: 1, Limit: 30, Cursor: cursor})
		if err != nil {
			t.Fatalf("分页查询失败: %v", err)
		}
		count += len(page.Photos)
		if !page.HasMore {
			break
		}
		if page.NextCursor == "" {
			t.Fatal("HasMore=true 时 NextCursor 不应为空")
		}
		cursor = page.NextCursor
	}

	if count != 200 {
		t.Fatalf("期望遍历 200 张图片，实际 %d 张", count)
	}
}

func TestListTrashedPhotos_PaginationUsesDeletedAtCursor(t *testing.T) {
	db := newTestDB(t)
	baseTakenAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 80; i++ {
		p := makePhoto(1, baseTakenAt.Add(time.Duration(i)*time.Hour))
		p.UUID = fmt.Sprintf("trash-%03d", i)
		if err := db.SavePhoto(p); err != nil {
			t.Fatalf("保存第 %d 张图片失败: %v", i, err)
		}
		if err := db.SoftDeletePhoto(p.ID, 1, 1); err != nil {
			t.Fatalf("软删除第 %d 张图片失败: %v", i, err)
		}
	}

	count := 0
	cursor := ""
	for {
		page, err := db.ListTrashedPhotos(storage.ListPhotosParams{UserID: 1, Limit: 30, Cursor: cursor})
		if err != nil {
			t.Fatalf("回收站分页查询失败: %v", err)
		}
		count += len(page.Photos)
		if !page.HasMore {
			break
		}
		if page.NextCursor == "" {
			t.Fatal("HasMore=true 时 NextCursor 不应为空")
		}
		cursor = page.NextCursor
	}

	if count != 80 {
		t.Fatalf("期望遍历 80 张回收站图片，实际 %d 张", count)
	}
}

// 确保测试文件不依赖 os 包报错
var _ = os.DevNull
