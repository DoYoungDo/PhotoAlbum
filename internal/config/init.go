package config

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	return runInitWizardWithReader(bufio.NewReader(os.Stdin), os.Stdout)
}

func runInitWizardWithReader(reader *bufio.Reader, out io.Writer) (*Config, error) {
	printf := func(format string, args ...any) {
		fmt.Fprintf(out, format, args...)
	}
	println := func(msg string) {
		fmt.Fprintln(out, msg)
	}
	println("未找到配置文件，开始初始化...")
	println("-----------------------------------")

	// 输入端口
	port, err := promptIntWithWriter(reader, out, "请输入服务端口", 8080, func(v int) error {
		if v <= 0 || v > 65535 {
			return fmt.Errorf("端口号必须在 1-65535 之间")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 输入存储路径
	storagePath, err := promptStringWithWriter(reader, out, "请输入图片存储路径", "./photos")
	if err != nil {
		return nil, err
	}

	// 自动创建存储目录
	storagePath = strings.TrimSpace(storagePath)
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}
	printf("存储目录已就绪: %s\n", storagePath)

	println("")
	println("接下来创建默认管理员用户")
	println("-----------------------------------")
	username, err := promptStringWithWriter(reader, out, "请输入默认用户名", "admin")
	if err != nil {
		return nil, err
	}
	var password string
	for {
		printf("请输入默认密码 [至少6位]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("读取输入失败: %w", err)
		}
		password = strings.TrimSpace(input)
		if len(password) < 6 {
			println("输入无效: 密码长度不能少于6位")
			continue
		}
		break
	}
	user, err := newUser(username, password)
	if err != nil {
		return nil, err
	}

	// 自动生成 JWT Secret
	jwtSecret, err := generateSecret(32)
	if err != nil {
		return nil, fmt.Errorf("生成 JWT Secret 失败: %w", err)
	}

	cfg := &Config{
		Port:        port,
		StoragePath: storagePath,
		JWTSecret:   jwtSecret,
		Users:       []User{user},
	}

	return cfg, nil
}

// promptString 提示用户输入字符串，支持默认值
func promptString(reader *bufio.Reader, prompt, defaultVal string) (string, error) {
	return promptStringWithWriter(reader, os.Stdout, prompt, defaultVal)
}

func promptStringWithWriter(reader *bufio.Reader, out io.Writer, prompt, defaultVal string) (string, error) {
	if defaultVal == "" {
		fmt.Fprintf(out, "%s: ", prompt)
	} else {
		fmt.Fprintf(out, "%s [默认: %s]: ", prompt, defaultVal)
	}
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
	return promptIntWithWriter(reader, os.Stdout, prompt, defaultVal, validate)
}

func promptIntWithWriter(reader *bufio.Reader, out io.Writer, prompt string, defaultVal int, validate func(int) error) (int, error) {
	for {
		fmt.Fprintf(out, "%s [默认: %d]: ", prompt, defaultVal)
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
			fmt.Fprintln(out, "请输入有效的整数")
			continue
		}
		if validate != nil {
			if err := validate(v); err != nil {
				fmt.Fprintf(out, "输入无效: %v\n", err)
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
