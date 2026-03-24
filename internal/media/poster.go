package media

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var ErrPosterGeneratorUnavailable = errors.New("ffmpeg 不可用")

func PosterPath(storagePath, uuid string) string {
	return filepath.Join(storagePath, ".posters", uuid+".jpg")
}

func GeneratePoster(videoPath, posterPath string) error {
	return generatePosterWithRunner(videoPath, posterPath, execPosterRunner)
}

func generatePosterWithRunner(videoPath, posterPath string, runner commandRunner) error {
	if err := os.MkdirAll(filepath.Dir(posterPath), 0755); err != nil {
		return fmt.Errorf("创建 poster 目录失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_, err := runner(ctx, "ffmpeg",
		"-y",
		"-ss", "00:00:01",
		"-i", videoPath,
		"-frames:v", "1",
		"-q:v", "2",
		posterPath,
	)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("%w: 请先安装 ffmpeg", ErrPosterGeneratorUnavailable)
		}
		return fmt.Errorf("生成视频 poster 失败: %w", err)
	}
	return nil
}

func execPosterRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}
