package utils

import (
	"context"
	"io/ioutil"
	"log"
	"time"

	pb "e-commerse/rpc/statistics"

	"google.golang.org/grpc"
)

const (
	logPath    = "/var/log/myapp/access.log" // 日志存储路径
	serverAddr = "localhost:50051"           // Python RPC服务端地址
	interval   = 30 * time.Second            // 每30秒调用一次RPC上传日志
)

func UploadLogPeriodically(logPath, serverAddr string, interval time.Duration) {
	// 建立RPC连接
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Statistics service connection failed: %v\n", err)
	}
	defer conn.Close()

	client := pb.NewStatisticsClient(conn)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			uploadLog(logPath, client)
		}
	}
}

// uploadLog 单次日志上传逻辑，内部私有函数。
func uploadLog(logPath string, client pb.StatisticsClient) {
	data, err := ioutil.ReadFile(logPath)
	if err != nil {
		log.Printf("Read log error: %v\n", err)
		return
	}

	req := &pb.StatisticsRequest{Log: data}

	resp, err := client.UploadLog(context.Background(), req)
	if err != nil {
		log.Printf("Upload log error: %v\n", err)
		return
	}

	log.Printf("Log uploaded successfully, Ack: %s\n", resp.Ack)
}
