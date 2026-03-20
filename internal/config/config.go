package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = "config.json"

// configPathOverride 用于测试时覆盖配置文件路径
var configPathOverride string

// User 表示一个用户
type User struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
}

// Config 应用配置
type Config struct {
	Port        int    `json:"port"`
	StoragePath string `json:"storage_path"`
	JWTSecret   string `json:"jwt_secret"`
	Users       []User `json:"users"`
}

// configPath 返回配置文件的绝对路径（与可执行程序同级）
func configPath() (string, error) {
	if configPathOverride != "" {
		return configPathOverride, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("无法获取可执行文件路径: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), configFileName), nil
}

// Load 加载配置文件，文件不存在时返回错误
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	return loadFromPath(path)
}

// loadFromPath 从指定路径加载配置文件
func loadFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save 将配置保存到文件
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	return c.saveToPath(path)
}

// saveToPath 将配置保存到指定路径
func (c *Config) saveToPath(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	return nil
}

// validate 校验配置合法性
func (c *Config) validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("无效的端口号: %d", c.Port)
	}
	if c.StoragePath == "" {
		return fmt.Errorf("storage_path 不能为空")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("jwt_secret 不能为空")
	}
	return nil
}

// ErrConfigNotFound 配置文件不存在错误
var ErrConfigNotFound = fmt.Errorf("配置文件不存在")
