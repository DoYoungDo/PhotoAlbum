package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"photoalbum/internal/config"
	"photoalbum/internal/server"
	"photoalbum/internal/service"
	"photoalbum/internal/storage/sqlite"
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

	dbPath := filepath.Join(cfg.StoragePath, "photoalbum.db")
	repo, err := sqlite.New(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 初始化数据库失败: %v\n", err)
		os.Exit(1)
	}
	defer repo.Close()

	photoService := service.NewPhotoService(repo, cfg.StoragePath)
	albumService := service.NewAlbumService(repo)
	shareService := service.NewShareService(repo)
	app := server.New(cfg, photoService, albumService, shareService)

	addr := fmt.Sprintf(":%d", cfg.Port)
	fmt.Printf("HTTP 服务已启动: http://127.0.0.1%s\n", addr)
	if err := http.ListenAndServe(addr, app); err != nil {
		fmt.Fprintf(os.Stderr, "错误: HTTP 服务启动失败: %v\n", err)
		os.Exit(1)
	}
}
