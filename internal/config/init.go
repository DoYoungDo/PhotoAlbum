package config

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// LoadOrInit 加载配置文件，如果不存在则运行初始化向导
func LoadOrInit() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	cfg, err := loadFromPath(path)
	if err == nil {
		return cfg, nil
	}

	if !errors.Is(err, ErrConfigNotFound) {
		return nil, err
	}

	// 配置文件不存在，运行初始化向导
	cfg, err = runInitWizard()
	if err != nil {
		return nil, err
	}

	if err := cfg.saveToPath(path); err != nil {
		return nil, err
	}

	fmt.Printf("\n配置已保存到 %s\n", path)
	return cfg, nil
}

// runInitWizard 运行命令行初始化向导
func runInitWizard() (*Config, error) {
	fmt.Println("未找到配置文件，开始初始化...")
	fmt.Println("-----------------------------------")

	reader := bufio.NewReader(os.Stdin)

	// 输入端口
	port, err := promptInt(reader, "请输入服务端口", 8080, func(v int) error {
		if v <= 0 || v > 65535 {
			return fmt.Errorf("端口号必须在 1-65535 之间")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 输入存储路径
	storagePath, err := promptString(reader, "请输入图片存储路径", "./photos")
	if err != nil {
		return nil, err
	}

	// 自动创建存储目录
	storagePath = strings.TrimSpace(storagePath)
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}
	fmt.Printf("存储目录已就绪: %s\n", storagePath)

	// 自动生成 JWT Secret
	jwtSecret, err := generateSecret(32)
	if err != nil {
		return nil, fmt.Errorf("生成 JWT Secret 失败: %w", err)
	}

	cfg := &Config{
		Port:        port,
		StoragePath: storagePath,
		JWTSecret:   jwtSecret,
		Users:       []User{},
	}

	return cfg, nil
}

// promptString 提示用户输入字符串，支持默认值
func promptString(reader *bufio.Reader, prompt, defaultVal string) (string, error) {
	fmt.Printf("%s [默认: %s]: ", prompt, defaultVal)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("读取输入失败: %w", err)
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal, nil
	}
	return input, nil
}

// promptInt 提示用户输入整数，支持默认值和校验
func promptInt(reader *bufio.Reader, prompt string, defaultVal int, validate func(int) error) (int, error) {
	for {
		fmt.Printf("%s [默认: %d]: ", prompt, defaultVal)
		input, err := reader.ReadString('\n')
		if err != nil {
			return 0, fmt.Errorf("读取输入失败: %w", err)
		}
		input = strings.TrimSpace(input)
		if input == "" {
			return defaultVal, nil
		}
		v, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("请输入有效的整数")
			continue
		}
		if validate != nil {
			if err := validate(v); err != nil {
				fmt.Printf("输入无效: %v\n", err)
				continue
			}
		}
		return v, nil
	}
}

// generateSecret 生成指定字节长度的随机十六进制字符串
func generateSecret(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
