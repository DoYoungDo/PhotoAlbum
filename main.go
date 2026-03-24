package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"photoalbum/internal/api"
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
	legacyApp := server.New(cfg, photoService, albumService, shareService, webFS)
	app := api.NewRouter(legacyApp)

	addr := fmt.Sprintf(":%d", cfg.Port)
	if host := preferredLANIP(); host != "" {
		fmt.Printf("HTTP 服务已启动: http://%s%s\n", host, addr)
	} else {
		fmt.Printf("HTTP 服务已启动: http://127.0.0.1%s\n", addr)
	}
	if err := http.ListenAndServe(addr, app); err != nil {
		fmt.Fprintf(os.Stderr, "错误: HTTP 服务启动失败: %v\n", err)
		os.Exit(1)
	}
}

func preferredLANIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	var fallback string
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP == nil || ipNet.IP.IsLoopback() {
			continue
		}

		ip := ipNet.IP.To4()
		if ip == nil || !isPrivateIPv4(ip) {
			continue
		}

		if ip[0] == 192 && ip[1] == 168 {
			return ip.String()
		}
		if fallback == "" {
			fallback = ip.String()
		}
	}

	return fallback
}

func isPrivateIPv4(ip net.IP) bool {
	return ip[0] == 10 ||
		(ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) ||
		(ip[0] == 192 && ip[1] == 168)
}
