package image

import (
	"os"
	"testing"
)

// mustOpenFile 打开文件，失败则 fatal
func mustOpenFile(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("打开文件失败 %s: %v", path, err)
	}
	return f
}
