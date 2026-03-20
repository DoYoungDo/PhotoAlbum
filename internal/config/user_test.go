package config

import (
	"path/filepath"
	"testing"
)

// --- AddUser 测试 ---

func TestAddUser_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	cfg := validConfig()
	if err := cfg.saveToPath(path); err != nil {
		t.Fatal(err)
	}

	// 临时覆盖 configPath
	origConfigPath := overrideConfigPath(path)
	defer origConfigPath()

	if err := AddUser("alice", "password123"); err != nil {
		t.Fatalf("添加用户失败: %v", err)
	}

	// 验证用户已保存
	loaded, err := loadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Users) != 1 {
		t.Fatalf("期望 1 个用户，得到 %d", len(loaded.Users))
	}
	if loaded.Users[0].Username != "alice" {
		t.Errorf("期望用户名 alice，得到 %s", loaded.Users[0].Username)
	}
	if loaded.Users[0].PasswordHash == "" {
		t.Error("密码哈希不能为空")
	}
}

func TestAddUser_DuplicateUsername(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	cfg := validConfig()
	if err := cfg.saveToPath(path); err != nil {
		t.Fatal(err)
	}

	origConfigPath := overrideConfigPath(path)
	defer origConfigPath()

	if err := AddUser("alice", "password123"); err != nil {
		t.Fatal(err)
	}
	if err := AddUser("alice", "anotherpass"); err == nil {
		t.Error("重复用户名应该返回错误")
	}
}

func TestAddUser_EmptyUsername(t *testing.T) {
	if err := AddUser("", "password123"); err == nil {
		t.Error("空用户名应该返回错误")
	}
}

func TestAddUser_ShortPassword(t *testing.T) {
	if err := AddUser("alice", "123"); err == nil {
		t.Error("短密码应该返回错误")
	}
}

// --- VerifyPassword 测试 ---

func TestVerifyPassword_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	cfg := validConfig()
	if err := cfg.saveToPath(path); err != nil {
		t.Fatal(err)
	}

	origConfigPath := overrideConfigPath(path)
	defer origConfigPath()

	if err := AddUser("bob", "mypassword"); err != nil {
		t.Fatal(err)
	}

	loaded, _ := loadFromPath(path)
	user, err := VerifyPassword(loaded, "bob", "mypassword")
	if err != nil {
		t.Fatalf("验证密码失败: %v", err)
	}
	if user.Username != "bob" {
		t.Errorf("期望用户名 bob，得到 %s", user.Username)
	}
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	cfg := validConfig()
	if err := cfg.saveToPath(path); err != nil {
		t.Fatal(err)
	}

	origConfigPath := overrideConfigPath(path)
	defer origConfigPath()

	if err := AddUser("bob", "mypassword"); err != nil {
		t.Fatal(err)
	}

	loaded, _ := loadFromPath(path)
	_, err := VerifyPassword(loaded, "bob", "wrongpassword")
	if err == nil {
		t.Error("错误密码应该返回错误")
	}
}

func TestVerifyPassword_UserNotFound(t *testing.T) {
	cfg := validConfig()
	_, err := VerifyPassword(cfg, "nonexistent", "password")
	if err == nil {
		t.Error("不存在的用户应该返回错误")
	}
}
