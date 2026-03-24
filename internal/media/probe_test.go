package media

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func TestProbeVideoWithRunner_Success(t *testing.T) {
	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte(`{
			"streams": [{"codec_type":"video","codec_name":"h264","width":1920,"height":1080}],
			"format": {"format_name":"mov,mp4,m4a,3gp,3g2,mj2","duration":"12.345"}
		}`), nil
	}

	meta, err := probeVideoWithRunner("demo.mp4", runner)
	if err != nil {
		t.Fatalf("期望成功，得到错误: %v", err)
	}
	if meta.Width != 1920 || meta.Height != 1080 {
		t.Fatalf("分辨率不正确: %+v", meta)
	}
	if meta.DurationMS != 12345 {
		t.Fatalf("期望时长 12345，得到 %d", meta.DurationMS)
	}
	if meta.CodecName != "h264" {
		t.Fatalf("期望编码 h264，得到 %s", meta.CodecName)
	}
}

func TestProbeVideoWithRunner_FFprobeUnavailable(t *testing.T) {
	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, exec.ErrNotFound
	}

	_, err := probeVideoWithRunner("demo.mp4", runner)
	if !errors.Is(err, ErrProbeUnavailable) {
		t.Fatalf("期望 ErrProbeUnavailable，得到 %v", err)
	}
}

func TestProbeVideoWithRunner_InvalidDuration(t *testing.T) {
	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte(`{
			"streams": [{"codec_type":"video","codec_name":"h264","width":1280,"height":720}],
			"format": {"format_name":"mp4","duration":"oops"}
		}`), nil
	}

	_, err := probeVideoWithRunner("demo.mp4", runner)
	if !errors.Is(err, ErrInvalidVideo) {
		t.Fatalf("期望 ErrInvalidVideo，得到 %v", err)
	}
}

func TestProbeVideoWithRunner_MissingVideoMeta(t *testing.T) {
	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte(`{
			"streams": [{"codec_type":"audio","codec_name":"aac"}],
			"format": {"format_name":"mp4","duration":"3.2"}
		}`), nil
	}

	_, err := probeVideoWithRunner("demo.mp4", runner)
	if !errors.Is(err, ErrInvalidVideo) {
		t.Fatalf("期望 ErrInvalidVideo，得到 %v", err)
	}
}
