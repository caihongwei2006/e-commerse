package main

import (
	"context"
	"e-commerse/router"
	"e-commerse/utils"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	startTime := time.Now()
	fmt.Printf("服务器启动于 %s\n", startTime)

	// 初始化各种依赖
	utils.InitConfig()
	utils.InitRedis()
	//起一个goroutine定时上传日志
	/*  go utils.UploadLogPeriodically(
		"/var/log/myapp/access.log", // 日志文件路径
		"localhost:50051",           // Python RPC服务地址
		30*time.Second,              // 上传间隔
	)
	*/
	// 初始化路由
	r := router.Router()
	if r == nil {
		log.Fatal("路由初始化失败")
	}

	// 定义服务器端口
	serverPort := ":8080" // 可从配置文件或环境变量获取
	if port := os.Getenv("SERVER_PORT"); port != "" {
		serverPort = ":" + port
	}

	// 创建服务器
	server := &http.Server{
		Addr:           serverPort,
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	duration := time.Since(startTime)
	fmt.Printf("服务器初始化完成，用时 %s\n", duration)

	// 优雅启动
	go func() {
		fmt.Printf("服务器开始监听端口%s\n", serverPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("正在关闭服务器...")

	// 创建一个5秒超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("服务器关闭出错:", err)
	}

	fmt.Println("服务器已成功关闭")
}
