package main

import (
	"honoka-chan/config"
	_ "honoka-chan/internal/handler"
	"honoka-chan/internal/middleware"
	"honoka-chan/internal/router"
	"honoka-chan/internal/startup"
	"honoka-chan/pkg/db"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化配置
	if err := config.InitConfig(); err != nil {
		log.Fatalf("初始化配置失败: %v", err)
	}

	// 初始化数据库表和用户数据
	startup.StartUp()

	// 处理系统信号，确保程序退出时关闭数据库
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChan
		log.Println("正在退出...")
		db.MainEng.Close()
		db.UserEng.Close()
		os.Exit(0)
	}()

	// Gin
	gin.SetMode(gin.ReleaseMode)

	// Router
	r := gin.New()

	// Logger
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{
			"/agreement/all",
			"/integration/appReport/initialize",
			"/report/ge/app",
			"/v1/account/reportRole",
		},
	}))
	r.Use(middleware.Recovery())

	// SIF
	router.SifRouter(r)

	if err := r.Run(":" + config.Conf.Settings.ListenPort); err != nil {
		log.Fatalf("HTTP 服务启动失败: %v", err)
	}
}
