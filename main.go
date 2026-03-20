package main

import (
	"fmt"
	"os"

	"photoalbum/internal/config"
)

func main() {
	// 处理子命令
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "adduser":
			if err := config.RunAddUserWizard(); err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
			return
		default:
			fmt.Fprintf(os.Stderr, "未知命令: %s\n", os.Args[1])
			os.Exit(1)
		}
	}

	// 加载或初始化配置
	cfg, err := config.LoadOrInit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("配置加载成功，服务将运行在端口 %d\n", cfg.Port)
	fmt.Printf("图片存储路径: %s\n", cfg.StoragePath)
	// TODO: 启动 HTTP 服务
}
