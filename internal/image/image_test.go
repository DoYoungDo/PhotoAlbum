package image

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
	"time"
)

// createTestJPEG 创建测试用 JPEG 图片字节
func createTestJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

// createTestPNG 创建测试用 PNG 图片字节
func createTestPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

// --- DetectMimeType 测试 ---

func TestDetectMimeType(t *testing.T) {
	cases := []struct {
		filename string
		expected string
	}{
		{"photo.jpg", "image/jpeg"},
		{"photo.JPG", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"photo.png", "image/png"},
		{"photo.gif", "image/gif"},
		{"photo.webp", "image/webp"},
		{"document.pdf", "application/pdf"},
	}
	for _, c := range cases {
		got := DetectMimeType(c.filename)
		if got != c.expected {
			t.Errorf("DetectMimeType(%q) = %q, 期望 %q", c.filename, got, c.expected)
		}
	}
}

// --- ExtractMeta 测试 ---

func TestExtractMeta_JPEG(t *testing.T) {
	data := createTestJPEG(800, 600)
	r := bytes.NewReader(data)
	now := time.Now()

	meta, err := ExtractMeta(r, "test.jpg", now)
	if err != nil {
		t.Fatalf("提取元数据失败: %v", err)
	}
	if meta.Width != 800 || meta.Height != 600 {
		t.Errorf("尺寸不匹配: 期望 800x600，得到 %dx%d", meta.Width, meta.Height)
	}
	if meta.MimeType != "image/jpeg" {
		t.Errorf("MimeType 不匹配: %s", meta.MimeType)
	}
	// 没有 EXIF，应该使用 fallback 时间
	if meta.TakenAt.IsZero() {
		t.Error("TakenAt 不应该为零值")
	}
}

func TestExtractMeta_PNG(t *testing.T) {
	data := createTestPNG(400, 300)
	r := bytes.NewReader(data)

	meta, err := ExtractMeta(r, "test.png", time.Now())
	if err != nil {
		t.Fatalf("提取 PNG 元数据失败: %v", err)
	}
	if meta.Width != 400 || meta.Height != 300 {
		t.Errorf("PNG 尺寸不匹配: 期望 400x300，得到 %dx%d", meta.Width, meta.Height)
	}
}

func TestExtractMeta_UnsupportedFormat(t *testing.T) {
	r := bytes.NewReader([]byte("not an image"))
	_, err := ExtractMeta(r, "doc.pdf", time.Now())
	if err == nil {
		t.Error("不支持的格式应该返回错误")
	}
}

func TestExtractMeta_FallbackTime(t *testing.T) {
	data := createTestJPEG(100, 100)
	r := bytes.NewReader(data)
	fallback := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	meta, err := ExtractMeta(r, "test.jpg", fallback)
	if err != nil {
		t.Fatal(err)
	}
	// 无 EXIF，应该使用 fallback
	if !meta.TakenAt.Equal(fallback) {
		t.Errorf("期望使用 fallback 时间 %v，得到 %v", fallback, meta.TakenAt)
	}
}

// --- calcThumbnailSize 测试 ---

func TestCalcThumbnailSize_LandscapeNeedsResize(t *testing.T) {
	w, h := calcThumbnailSize(800, 600, 400)
	if w != 400 {
		t.Errorf("横向图片宽度应该为 400，得到 %d", w)
	}
	if h != 300 {
		t.Errorf("横向图片高度应该为 300，得到 %d", h)
	}
}

func TestCalcThumbnailSize_PortraitNeedsResize(t *testing.T) {
	w, h := calcThumbnailSize(600, 800, 400)
	if h != 400 {
		t.Errorf("纵向图片高度应该为 400，得到 %d", h)
	}
	if w != 300 {
		t.Errorf("纵向图片宽度应该为 300，得到 %d", w)
	}
}

func TestCalcThumbnailSize_SmallImageNoResize(t *testing.T) {
	w, h := calcThumbnailSize(200, 150, 400)
	if w != 200 || h != 150 {
		t.Errorf("小图不应该被放大，期望 200x150，得到 %dx%d", w, h)
	}
}

func TestCalcThumbnailSize_Square(t *testing.T) {
	w, h := calcThumbnailSize(600, 600, 400)
	if w != 400 || h != 400 {
		t.Errorf("正方形图片期望 400x400，得到 %dx%d", w, h)
	}
}

// --- GenerateThumbnail 测试 ---

func TestGenerateThumbnail_JPEG(t *testing.T) {
	data := createTestJPEG(800, 600)
	r := bytes.NewReader(data)
	destPath := t.TempDir() + "/thumb.jpg"

	if err := GenerateThumbnail(r, "image/jpeg", destPath); err != nil {
		t.Fatalf("生成 JPEG 缩略图失败: %v", err)
	}

	// 验证缩略图尺寸
	thumbFile := mustOpenFile(t, destPath)
	defer thumbFile.Close()
	cfg, _, err := image.DecodeConfig(thumbFile)
	if err != nil {
		t.Fatalf("解码缩略图失败: %v", err)
	}
	if cfg.Width != 400 || cfg.Height != 300 {
		t.Errorf("缩略图尺寸期望 400x300，得到 %dx%d", cfg.Width, cfg.Height)
	}
}

func TestGenerateThumbnail_PNG(t *testing.T) {
	data := createTestPNG(400, 300)
	r := bytes.NewReader(data)
	destPath := t.TempDir() + "/thumb.png"

	if err := GenerateThumbnail(r, "image/png", destPath); err != nil {
		t.Fatalf("生成 PNG 缩略图失败: %v", err)
	}
}

func TestGenerateThumbnail_SmallImage(t *testing.T) {
	// 小图不应该被放大
	data := createTestJPEG(200, 150)
	r := bytes.NewReader(data)
	destPath := t.TempDir() + "/thumb_small.jpg"

	if err := GenerateThumbnail(r, "image/jpeg", destPath); err != nil {
		t.Fatalf("生成小图缩略图失败: %v", err)
	}

	thumbFile := mustOpenFile(t, destPath)
	defer thumbFile.Close()
	cfg, _, _ := image.DecodeConfig(thumbFile)
	if cfg.Width != 200 || cfg.Height != 150 {
		t.Errorf("小图缩略图不应该被放大，期望 200x150，得到 %dx%d", cfg.Width, cfg.Height)
	}
}

func TestGenerateThumbnail_CreatesDirectory(t *testing.T) {
	data := createTestJPEG(100, 100)
	r := bytes.NewReader(data)
	destPath := t.TempDir() + "/subdir/thumb.jpg"

	if err := GenerateThumbnail(r, "image/jpeg", destPath); err != nil {
		t.Fatalf("应该自动创建目录: %v", err)
	}
}

func TestOrientedDimensions_Rotate90(t *testing.T) {
	w, h := orientedDimensions(800, 600, 6)
	if w != 600 || h != 800 {
		t.Fatalf("期望 600x800，得到 %dx%d", w, h)
	}
}

func TestApplyOrientation_Rotate90CW(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	red := color.RGBA{R: 255, A: 255}
	green := color.RGBA{G: 255, A: 255}
	blue := color.RGBA{B: 255, A: 255}
	img.Set(0, 0, red)
	img.Set(1, 0, green)
	img.Set(0, 2, blue)

	rotated := applyOrientation(img, 6)
	if rotated.Bounds().Dx() != 3 || rotated.Bounds().Dy() != 2 {
		t.Fatalf("顺时针旋转后期望 3x2，得到 %dx%d", rotated.Bounds().Dx(), rotated.Bounds().Dy())
	}
	if !colorEqual(rotated.At(2, 0), red) {
		t.Fatal("原左上角红色像素应移动到新图右上角")
	}
	if !colorEqual(rotated.At(2, 1), green) {
		t.Fatal("原右上角绿色像素应移动到新图右下角")
	}
	if !colorEqual(rotated.At(0, 0), blue) {
		t.Fatal("原左下角蓝色像素应移动到新图左上角")
	}
}

func TestApplyOrientation_None(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 5))
	result := applyOrientation(img, 1)
	if result.Bounds().Dx() != 4 || result.Bounds().Dy() != 5 {
		t.Fatalf("无方向变换时尺寸不应变化，得到 %dx%d", result.Bounds().Dx(), result.Bounds().Dy())
	}
}
