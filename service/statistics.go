package service

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	statisticspb "e-commerse/rpc/statistics"
)

// LogFilePath 定义日志文件存储路径
const LogFilePath = "/var/log/e-commerce"

// 文件锁，防止并发操作日志文件
var logFileLock sync.Mutex

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

// UploadLog 实现RPCserver服务端方法，接收ack，返回日志内容
func (s *StatisticsServer) UploadLog(ctx context.Context, req *statisticspb.StatisticsRequest) (*statisticspb.StatisticsResponse, error) {
	// 加锁防止并发操作日志文件
	logFileLock.Lock()
	defer logFileLock.Unlock()

	// 记录请求信息
	log.Printf("收到日志请求，客户端ack: %s", req.Ack)

	// 日志文件名称
	logFileName := "goods-access.log"
	fullPath := filepath.Join(LogFilePath, logFileName)
	backupPath := filepath.Join("/tmp/e-commerce-logs", logFileName)

	// 确定实际使用的路径
	var actualPath string
	if _, err := os.Stat(fullPath); err == nil {
		actualPath = fullPath
	} else if _, err := os.Stat(backupPath); err == nil {
		actualPath = backupPath
	} else {
		// 日志文件不存在，返回空内容但不是错误
		log.Println("日志文件不存在，返回空内容")
		return &statisticspb.StatisticsResponse{
			Log: []byte{},
		}, nil
	}

	// 读取日志文件内容
	logData, err := ioutil.ReadFile(actualPath)
	if err != nil {
		log.Printf("读取日志文件失败: %v", err)
		return &statisticspb.StatisticsResponse{
			Log: []byte{},
		}, fmt.Errorf("读取日志文件失败: %v", err)
	}

	// 删除日志文件 - 读取后删除
	if err := os.Remove(actualPath); err != nil {
		log.Printf("删除日志文件失败: %v", err)
	}

	log.Printf("已读取并返回日志文件内容，大小: %d 字节", len(logData))

	// 返回日志内容
	return &statisticspb.StatisticsResponse{
		Log: logData,
	}, nil
}

// LogGoodsAccess 记录商品访问日志
func LogGoodsAccess(goodsID, tag string) error {
	// 加锁防止并发写入
	logFileLock.Lock()
	defer logFileLock.Unlock()

	// 构建日志文件路径
	logFileName := "goods-access.log"
	fullPath := filepath.Join(LogFilePath, logFileName)

	// 确保日志目录存在
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		// 如果无法创建主目录，使用备用目录
		fullPath = filepath.Join("/tmp/e-commerce-logs", logFileName)
		if err := os.MkdirAll("/tmp/e-commerce-logs", 0755); err != nil {
			return fmt.Errorf("无法创建日志目录: %v", err)
		}
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

	log.Printf("已记录商品访问: ID=%s, Tag=%s", goodsID, tag)
	return nil
}
