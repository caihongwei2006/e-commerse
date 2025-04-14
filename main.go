package main

import (
	"context"
	"e-commerse/router"
	statisticspb "e-commerse/rpc/statistics"
	"e-commerse/service"
	"e-commerse/utils"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

// 注册统计服务到 gRPC 服务器
func registerStatisticsService(grpcServer *grpc.Server) {
	statisticsServer := service.NewStatisticsServer()
	statisticspb.RegisterStatisticsServer(grpcServer, statisticsServer)
	log.Println("Statistics服务注册成功")
}

// 启动 gRPC 服务器
func startGRPCServer() *grpc.Server {
	grpcPort := ":9090" // 可以从配置文件获取
	if port := os.Getenv("GRPC_PORT"); port != "" {
		grpcPort = ":" + port
	}
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("gRPC服务器监听端口失败: %v", err)
	}

	// 创建 gRPC 服务器
	grpcServer := grpc.NewServer()

	// 注册服务
	registerStatisticsService(grpcServer)
	// 在这里注册其他 gRPC 服务...

	// 启动 gRPC 服务器（非阻塞方式）
	go func() {
		log.Printf("gRPC服务器开始监听端口%s\n", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC服务器启动失败: %v", err)
		}
	}()

	// 返回服务器实例，以便在程序结束时优雅关闭
	return grpcServer
}

func main() {
	startTime := time.Now()
	fmt.Printf("服务器启动于 %s\n", startTime)

	// 初始化各种依赖
	utils.InitConfig()
	utils.InitRedis()

	// 启动 gRPC 服务器
	grpcServer := startGRPCServer()

	// 起一个goroutine定时上传日志
	/*  go utils.UploadLogPeriodically(
	        "/var/log/myapp/access.log", // 日志文件路径
	        "localhost:50051",           // Python RPC服务地址
	        30*time.Second,              // 上传间隔
	    )
	*/

	// 初始化HTTP路由
	r := router.Router()
	if r == nil {
		log.Fatal("路由初始化失败")
	}

	// 定义HTTP服务器端口
	serverPort := ":8080" // 可从配置文件或环境变量获取
	if port := os.Getenv("SERVER_PORT"); port != "" {
		serverPort = ":" + port
	}

	// 创建HTTP服务器
	server := &http.Server{
		Addr:           serverPort,
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	duration := time.Since(startTime)
	fmt.Printf("服务器初始化完成，用时 %s\n", duration)

	// 优雅启动HTTP服务器
	go func() {
		fmt.Printf("HTTP服务器开始监听端口%s\n", serverPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP服务器启动失败: %v", err)
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

	// 优雅关闭HTTP服务器
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("HTTP服务器关闭出错:", err)
	}

	// 优雅关闭gRPC服务器
	grpcServer.GracefulStop()

	fmt.Println("所有服务器已成功关闭")
}
