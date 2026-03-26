package media

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPosterPath(t *testing.T) {
	got := PosterPath("/tmp/storage", "video-1")
	want := filepath.Join("/tmp/storage", ".posters", "video-1.jpg")
	if got != want {
		t.Fatalf("期望 %s，得到 %s", want, got)
	}
}

func TestGeneratePosterWithRunner_Success(t *testing.T) {
	dir := t.TempDir()
	poster := filepath.Join(dir, ".posters", "demo.jpg")
	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if err := os.MkdirAll(filepath.Dir(poster), 0755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(poster, []byte("jpg"), 0644); err != nil {
			return nil, err
		}
		return []byte("ok"), nil
	}

	if err := generatePosterWithRunner("demo.mp4", poster, runner); err != nil {
		t.Fatalf("期望成功，得到错误: %v", err)
	}
	if _, err := os.Stat(poster); err != nil {
		t.Fatalf("poster 文件应存在: %v", err)
	}
}

func TestGeneratePosterWithRunner_FFmpegUnavailable(t *testing.T) {
	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, exec.ErrNotFound
	}

	err := generatePosterWithRunner("demo.mp4", filepath.Join(t.TempDir(), "p.jpg"), runner)
	if err == nil || !strings.Contains(err.Error(), "ffmpeg") {
		t.Fatalf("期望 ffmpeg 缺失错误，得到 %v", err)
	}
}
