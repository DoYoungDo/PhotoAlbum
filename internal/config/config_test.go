package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// 创建临时配置文件用于测试
func writeTempConfig(t *testing.T, dir string, cfg *Config) string {
	t.Helper()
	path := filepath.Join(dir, configFileName)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("序列化配置失败: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("写入临时配置失败: %v", err)
	}
	return path
}

func validConfig() *Config {
	return &Config{
		Port:        8080,
		StoragePath: "/tmp/photos",
		JWTSecret:   "testsecret",
		Users:       []User{},
	}
}

// --- loadFromPath 测试 ---

func TestLoadFromPath_Success(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, validConfig())
	path := filepath.Join(dir, configFileName)

	cfg, err := loadFromPath(path)
	if err != nil {
		t.Fatalf("期望加载成功，得到错误: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("期望 Port=8080，得到 %d", cfg.Port)
	}
	if cfg.StoragePath != "/tmp/photos" {
		t.Errorf("期望 StoragePath=/tmp/photos，得到 %s", cfg.StoragePath)
	}
}

func TestLoadFromPath_FileNotFound(t *testing.T) {
	_, err := loadFromPath("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("期望返回错误，但得到 nil")
	}
	if err != ErrConfigNotFound {
		t.Errorf("期望 ErrConfigNotFound，得到 %v", err)
	}
}

func TestLoadFromPath_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	if err := os.WriteFile(path, []byte("not valid json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := loadFromPath(path)
	if err == nil {
		t.Fatal("期望返回错误，但得到 nil")
	}
}

// --- validate 测试 ---

func TestValidate_InvalidPort(t *testing.T) {
	cases := []int{0, -1, 65536, 99999}
	for _, port := range cases {
		cfg := validConfig()
		cfg.Port = port
		if err := cfg.validate(); err == nil {
			t.Errorf("端口 %d 应该校验失败", port)
		}
	}
}

func TestValidate_EmptyStoragePath(t *testing.T) {
	cfg := validConfig()
	cfg.StoragePath = ""
	if err := cfg.validate(); err == nil {
		t.Error("空 StoragePath 应该校验失败")
	}
}

func TestValidate_EmptyJWTSecret(t *testing.T) {
	cfg := validConfig()
	cfg.JWTSecret = ""
	if err := cfg.validate(); err == nil {
		t.Error("空 JWTSecret 应该校验失败")
	}
}

func TestValidate_Valid(t *testing.T) {
	cfg := validConfig()
	if err := cfg.validate(); err != nil {
		t.Errorf("有效配置不应该校验失败: %v", err)
	}
}

// --- saveToPath 测试 ---

func TestSaveToPath_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	cfg := validConfig()

	if err := cfg.saveToPath(path); err != nil {
		t.Fatalf("保存失败: %v", err)
	}

	// 读回验证
	loaded, err := loadFromPath(path)
	if err != nil {
		t.Fatalf("读取刚保存的文件失败: %v", err)
	}
	if loaded.Port != cfg.Port {
		t.Errorf("Port 不一致: 期望 %d，得到 %d", cfg.Port, loaded.Port)
	}
	if loaded.StoragePath != cfg.StoragePath {
		t.Errorf("StoragePath 不一致")
	}
}

// --- generateSecret 测试 ---

func TestGenerateSecret_Length(t *testing.T) {
	secret, err := generateSecret(32)
	if err != nil {
		t.Fatalf("生成 secret 失败: %v", err)
	}
	// 32 字节 hex 编码后为 64 字符
	if len(secret) != 64 {
		t.Errorf("期望长度 64，得到 %d", len(secret))
	}
}

func TestGenerateSecret_Unique(t *testing.T) {
	s1, _ := generateSecret(32)
	s2, _ := generateSecret(32)
	if s1 == s2 {
		t.Error("两次生成的 secret 不应该相同")
	}
}
