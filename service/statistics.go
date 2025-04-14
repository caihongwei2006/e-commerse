package service

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	statisticspb "e-commerse/rpc/statistics"
)

// LogFilePath 定义日志文件存储路径
const LogFilePath = "/var/log/e-commerce"

// 确保日志目录存在
func init() {
	if err := os.MkdirAll(LogFilePath, 0755); err != nil {
		log.Printf("无法创建日志目录 %s: %v", LogFilePath, err)
		// 使用备用目录
		os.MkdirAll("/tmp/e-commerce-logs", 0755)
	}
}

// StatisticsServer 实现统计服务接口
type StatisticsServer struct {
	statisticspb.UnimplementedStatisticsServer
}

// NewStatisticsServer 创建一个新的统计服务实例
func NewStatisticsServer() *StatisticsServer {
	return &StatisticsServer{}
}

// UploadLog 处理日志上传请求
func (s *StatisticsServer) UploadLog(ctx context.Context, req *statisticspb.StatisticsRequest) (*statisticspb.StatisticsResponse, error) {
	// 生成唯一的日志文件名，使用时间戳
	timestamp := time.Now().Format("20060102-150405")
	logFileName := fmt.Sprintf("access-log-%s.log", timestamp)
	fullPath := filepath.Join(LogFilePath, logFileName)

	// 写入日志文件
	if err := ioutil.WriteFile(fullPath, req.Log, 0644); err != nil {
		log.Printf("写入日志文件失败: %v", err)

		// 尝试写入备用位置
		backupPath := filepath.Join("/tmp/e-commerce-logs", logFileName)
		if writeErr := ioutil.WriteFile(backupPath, req.Log, 0644); writeErr != nil {
			log.Printf("写入备用日志文件也失败: %v", writeErr)
			return &statisticspb.StatisticsResponse{
				Ack: "failed",
			}, err
		}
		fullPath = backupPath
	}

	// 处理日志内容，这里可以添加日志分析功能
	log.Printf("已收到并保存日志文件: %s, 大小: %d 字节", fullPath, len(req.Log))

	// 返回确认消息
	return &statisticspb.StatisticsResponse{
		Ack: "success",
	}, nil
}

// LogGoodsAccess 记录商品访问日志
func LogGoodsAccess(goodsID, tag string) error {
	// 构建日志文件路径
	logFileName := fmt.Sprintf("goods-access-%s.log", time.Now().Format("20060102"))
	fullPath := filepath.Join(LogFilePath, logFileName)

	// 确保日志目录存在
	if err := os.MkdirAll(LogFilePath, 0755); err != nil {
		// 如果无法创建主目录，使用备用目录
		fullPath = filepath.Join("/tmp/e-commerce-logs", logFileName)
		os.MkdirAll("/tmp/e-commerce-logs", 0755)
	}

	// 构建日志内容
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("tag:%s time:%s id:%s\n", tag, timestamp, goodsID)

	// 打开日志文件，如果不存在则创建
	f, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("无法打开日志文件: %v", err)
	}
	defer f.Close()

	// 写入日志条目
	if _, err := f.WriteString(logEntry); err != nil {
		return fmt.Errorf("无法写入日志条目: %v", err)
	}

	return nil
}
