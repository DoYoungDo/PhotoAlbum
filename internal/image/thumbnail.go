package image

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

const ThumbnailLongEdge = 400

// GenerateThumbnail 从 src 读取原图，生成缩略图写入 destPath
// 长边缩放到 ThumbnailLongEdge，保持比例，使用双线性插值
func GenerateThumbnail(src io.ReadSeeker, mimeType string, destPath string) error {
	// 解码原图
	orig, err := decodeImage(src, mimeType)
	if err != nil {
		return fmt.Errorf("解码图片失败: %w", err)
	}

	// 计算缩略图尺寸
	thumbW, thumbH := calcThumbnailSize(orig.Bounds().Dx(), orig.Bounds().Dy(), ThumbnailLongEdge)

	// 缩放
	thumb := image.NewRGBA(image.Rect(0, 0, thumbW, thumbH))
	draw.BiLinear.Scale(thumb, thumb.Bounds(), orig, orig.Bounds(), draw.Over, nil)

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("创建缩略图目录失败: %w", err)
	}

	// 写入目标文件
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建缩略图文件失败: %w", err)
	}
	defer f.Close()

	return encodeImage(f, thumb, mimeType)
}

// calcThumbnailSize 计算缩略图尺寸，长边不超过 maxEdge，保持比例
func calcThumbnailSize(w, h, maxEdge int) (int, int) {
	if w <= maxEdge && h <= maxEdge {
		return w, h
	}
	ratio := float64(maxEdge) / math.Max(float64(w), float64(h))
	return int(math.Round(float64(w) * ratio)), int(math.Round(float64(h) * ratio))
}

// decodeImage 根据 mimeType 解码图片
func decodeImage(r io.Reader, mimeType string) (image.Image, error) {
	switch mimeType {
	case "image/jpeg":
		return jpeg.Decode(r)
	case "image/png":
		return png.Decode(r)
	case "image/gif":
		return gif.Decode(r)
	case "image/webp":
		return webp.Decode(r)
	default:
		img, _, err := image.Decode(r)
		return img, err
	}
}

// encodeImage 根据 mimeType 编码图片到 writer
func encodeImage(w io.Writer, img image.Image, mimeType string) error {
	switch mimeType {
	case "image/png":
		return png.Encode(w, img)
	case "image/gif":
		return gif.Encode(w, img, nil)
	default:
		// JPEG 和 WebP 都输出为 JPEG 缩略图
		return jpeg.Encode(w, img, &jpeg.Options{Quality: 85})
	}
}
