package service

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"testing"
	"time"

	"photoalbum/internal/storage"
)

func createJPEGBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

func newTestPhotoService(t *testing.T) (*PhotoService, string) {
	t.Helper()
	dir := t.TempDir()
	repo := newMockRepo()
	svc := newPhotoServiceSync(repo, dir)
	return svc, dir
}

// --- Upload 测试 ---

func TestUpload_Success(t *testing.T) {
	svc, _ := newTestPhotoService(t)
	data := createJPEGBytes(800, 600)

	result, err := svc.Upload(UploadInput{
		Reader:       bytes.NewReader(data),
		OriginalName: "test.jpg",
		Size:         int64(len(data)),
		UploadedBy:   1,
		FileModTime:  time.Now(),
	})
	if err != nil {
		t.Fatalf("上传失败: %v", err)
	}
	if result.Photo.ID == 0 {
		t.Error("上传后 ID 应该被填充")
	}
	if result.Photo.UUID == "" {
		t.Error("UUID 不能为空")
	}
	if result.Photo.Width != 800 || result.Photo.Height != 600 {
		t.Errorf("尺寸不匹配: %dx%d", result.Photo.Width, result.Photo.Height)
	}
}

func TestUpload_FileWrittenToDisk(t *testing.T) {
	svc, dir := newTestPhotoService(t)
	data := createJPEGBytes(100, 100)

	result, err := svc.Upload(UploadInput{
		Reader:       bytes.NewReader(data),
		OriginalName: "photo.jpg",
		Size:         int64(len(data)),
		UploadedBy:   1,
		FileModTime:  time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// 验证文件已写入磁盘
	path := svc.PhotoPath(result.Photo)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("图片文件应该存在于 %s", path)
	}
	_ = dir
}

func TestUpload_UnsupportedFormat(t *testing.T) {
	svc, _ := newTestPhotoService(t)
	_, err := svc.Upload(UploadInput{
		Reader:       bytes.NewReader([]byte("not an image")),
		OriginalName: "file.pdf",
		Size:         100,
		UploadedBy:   1,
		FileModTime:  time.Now(),
	})
	if err == nil {
		t.Error("不支持的格式应该返回错误")
	}
}

func TestUpload_FallbackTime(t *testing.T) {
	svc, _ := newTestPhotoService(t)
	data := createJPEGBytes(100, 100)
	fallback := time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC)

	result, err := svc.Upload(UploadInput{
		Reader:       bytes.NewReader(data),
		OriginalName: "test.jpg",
		Size:         int64(len(data)),
		UploadedBy:   1,
		FileModTime:  fallback,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Photo.TakenAt.Equal(fallback) {
		t.Errorf("期望使用 fallback 时间 %v，得到 %v", fallback, result.Photo.TakenAt)
	}
}

// --- GetTimeline / GetTrash / DeletePhoto / RestorePhoto / EmptyTrash 测试 ---

func TestGetTimeline(t *testing.T) {
	svc, _ := newTestPhotoService(t)
	data := createJPEGBytes(100, 100)

	for i := 0; i < 3; i++ {
		svc.Upload(UploadInput{
			Reader:       bytes.NewReader(data),
			OriginalName: "test.jpg",
			Size:         int64(len(data)),
			UploadedBy:   1,
			FileModTime:  time.Now(),
		})
	}

	page, err := svc.GetTimeline(storage.ListPhotosParams{UserID: 1, Limit: 10})
	if err != nil {
		t.Fatalf("获取时间线失败: %v", err)
	}
	if len(page.Photos) != 3 {
		t.Errorf("期望 3 张，得到 %d", len(page.Photos))
	}
}

func TestDeleteAndRestorePhoto(t *testing.T) {
	svc, _ := newTestPhotoService(t)
	data := createJPEGBytes(100, 100)

	result, _ := svc.Upload(UploadInput{
		Reader:       bytes.NewReader(data),
		OriginalName: "test.jpg",
		Size:         int64(len(data)),
		UploadedBy:   1,
		FileModTime:  time.Now(),
	})

	// 删除
	if err := svc.DeletePhoto(result.Photo.ID, 1); err != nil {
		t.Fatalf("删除失败: %v", err)
	}

	// 时间线上应该消失
	page, _ := svc.GetTimeline(storage.ListPhotosParams{UserID: 1, Limit: 10})
	if len(page.Photos) != 0 {
		t.Error("删除后时间线应该为空")
	}

	// 回收站应该有
	trash, _ := svc.GetTrash(storage.ListPhotosParams{UserID: 1, Limit: 10})
	if len(trash.Photos) != 1 {
		t.Error("删除后回收站应该有 1 张")
	}

	// 恢复
	if err := svc.RestorePhoto(result.Photo.ID, 1); err != nil {
		t.Fatalf("恢复失败: %v", err)
	}
	page, _ = svc.GetTimeline(storage.ListPhotosParams{UserID: 1, Limit: 10})
	if len(page.Photos) != 1 {
		t.Error("恢复后时间线应该有 1 张")
	}
}

func TestEmptyTrash(t *testing.T) {
	svc, _ := newTestPhotoService(t)
	data := createJPEGBytes(100, 100)

	result, _ := svc.Upload(UploadInput{
		Reader:       bytes.NewReader(data),
		OriginalName: "test.jpg",
		Size:         int64(len(data)),
		UploadedBy:   1,
		FileModTime:  time.Now(),
	})
	svc.DeletePhoto(result.Photo.ID, 1)

	if err := svc.EmptyTrash(1); err != nil {
		t.Fatalf("清空回收站失败: %v", err)
	}

	trash, _ := svc.GetTrash(storage.ListPhotosParams{UserID: 1, Limit: 10})
	if len(trash.Photos) != 0 {
		t.Error("清空后回收站应该为空")
	}
}
