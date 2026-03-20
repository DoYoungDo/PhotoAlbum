package image

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

// SupportedMimeTypes 支持的图片 MIME 类型
var SupportedMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// Meta 图片元数据
type Meta struct {
	Width    int
	Height   int
	MimeType string
	TakenAt  time.Time // 拍摄时间
	EXIF     *EXIFData
}

// EXIFData 从图片中提取的 EXIF 信息
type EXIFData struct {
	Make         string    `json:"make,omitempty"`
	Model        string    `json:"model,omitempty"`
	Orientation  int       `json:"orientation,omitempty"`
	TakenAt      time.Time `json:"taken_at,omitempty"`
	Width        int       `json:"width,omitempty"`
	Height       int       `json:"height,omitempty"`
	FNumber      string    `json:"f_number,omitempty"`
	ExposureTime string    `json:"exposure_time,omitempty"`
	ISOSpeed     int       `json:"iso_speed,omitempty"`
	FocalLength  string    `json:"focal_length,omitempty"`
	Latitude     float64   `json:"latitude,omitempty"`
	Longitude    float64   `json:"longitude,omitempty"`
	HasGPS       bool      `json:"has_gps,omitempty"`
}

// ExtractMeta 从 ReadSeeker 中提取图片元数据
// fileCreateTime 作为没有 EXIF 时的后备时间
func ExtractMeta(rs io.ReadSeeker, filename string, fileCreateTime time.Time) (*Meta, error) {
	mimeType := DetectMimeType(filename)
	if !SupportedMimeTypes[mimeType] {
		return nil, fmt.Errorf("不支持的图片格式: %s", mimeType)
	}

	// 尝试读取 EXIF
	var exifData *EXIFData
	takenAt := fileCreateTime

	if mimeType == "image/jpeg" {
		if ed, t, err := parseEXIF(rs); err == nil {
			exifData = ed
			if !t.IsZero() {
				takenAt = t
			}
		}
		// 重置读取位置
		if _, err := rs.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek 失败: %w", err)
		}
	}

	// 解码图片尺寸
	cfg, _, err := image.DecodeConfig(rs)
	if err != nil {
		return nil, fmt.Errorf("解析图片尺寸失败: %w", err)
	}

	width, height := cfg.Width, cfg.Height
	if exifData != nil {
		width, height = orientedDimensions(width, height, exifData.Orientation)
		exifData.Width = width
		exifData.Height = height
	}

	return &Meta{
		Width:    width,
		Height:   height,
		MimeType: mimeType,
		TakenAt:  takenAt,
		EXIF:     exifData,
	}, nil
}

// parseEXIF 从 JPEG 中解析 EXIF 数据
func parseEXIF(rs io.ReadSeeker) (*EXIFData, time.Time, error) {
	x, err := exif.Decode(rs)
	if err != nil {
		return nil, time.Time{}, err
	}

	data := &EXIFData{}
	var takenAt time.Time

	// 拍摄时间
	if t, err := x.DateTime(); err == nil {
		takenAt = t
		data.TakenAt = t
	}

	// 相机品牌
	if tag, err := x.Get(exif.Make); err == nil {
		data.Make, _ = tag.StringVal()
	}

	// 相机型号
	if tag, err := x.Get(exif.Model); err == nil {
		data.Model, _ = tag.StringVal()
	}

	// 方向信息，常见手机照片会依赖该字段决定显示方向
	if tag, err := x.Get(exif.Orientation); err == nil {
		if v, err := tag.Int(0); err == nil {
			data.Orientation = v
		}
	}

	// 光圈
	if tag, err := x.Get(exif.FNumber); err == nil {
		data.FNumber = tag.String()
	}

	// 曝光时间
	if tag, err := x.Get(exif.ExposureTime); err == nil {
		data.ExposureTime = tag.String()
	}

	// ISO
	if tag, err := x.Get(exif.ISOSpeedRatings); err == nil {
		if v, err := tag.Int(0); err == nil {
			data.ISOSpeed = v
		}
	}

	// 焦距
	if tag, err := x.Get(exif.FocalLength); err == nil {
		data.FocalLength = tag.String()
	}

	// GPS
	if lat, lon, err := x.LatLong(); err == nil {
		data.Latitude = lat
		data.Longitude = lon
		data.HasGPS = true
	}

	return data, takenAt, nil
}

func orientedDimensions(width, height, orientation int) (int, int) {
	switch orientation {
	case 5, 6, 7, 8:
		return height, width
	default:
		return width, height
	}
}

// DetectMimeType 根据文件名后缀检测 MIME 类型
func DetectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	// mime.TypeByExtension 在不同平台行为可能不一致，手动补充常见类型
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		t := mime.TypeByExtension(ext)
		if t == "" {
			return "application/octet-stream"
		}
		return t
	}
}
