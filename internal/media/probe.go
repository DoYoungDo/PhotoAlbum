package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var ErrProbeUnavailable = errors.New("ffprobe 不可用")
var ErrInvalidVideo = errors.New("无效的视频文件")

type VideoMeta struct {
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	DurationMS int64  `json:"duration_ms"`
	FormatName string `json:"format_name"`
	CodecName  string `json:"codec_name"`
}

type commandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

type ffprobeOutput struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	} `json:"streams"`
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
	} `json:"format"`
}

func ProbeVideo(path string) (*VideoMeta, error) {
	return probeVideoWithRunner(path, execRunner)
}

func probeVideoWithRunner(path string, runner commandRunner) (*VideoMeta, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := runner(ctx, "ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		path,
	)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("%w: 请先安装 ffprobe", ErrProbeUnavailable)
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidVideo, err)
	}

	var data ffprobeOutput
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, fmt.Errorf("%w: 解析 ffprobe 输出失败", ErrInvalidVideo)
	}

	meta := &VideoMeta{FormatName: data.Format.FormatName}
	for _, stream := range data.Streams {
		if stream.CodecType != "video" {
			continue
		}
		meta.Width = stream.Width
		meta.Height = stream.Height
		meta.CodecName = stream.CodecName
		break
	}

	if strings.TrimSpace(data.Format.Duration) != "" {
		d, err := strconv.ParseFloat(data.Format.Duration, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: duration 字段无效", ErrInvalidVideo)
		}
		meta.DurationMS = int64(d * 1000)
	}

	if meta.Width <= 0 || meta.Height <= 0 || meta.DurationMS <= 0 {
		return nil, fmt.Errorf("%w: 缺少必要的视频元数据", ErrInvalidVideo)
	}

	if meta.FormatName == "" {
		meta.FormatName = "mp4"
	}

	return meta, nil
}

func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}
