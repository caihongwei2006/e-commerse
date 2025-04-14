package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// 日志文件存储路径
	LogFilePath = "/var/log/e-commerce"
	// 备用日志路径
	BackupLogPath = "/tmp/e-commerce-logs"
	// 商品访问日志文件名
	GoodsAccessLogFile = "goods-access.log"
)

// 文件锁，防止并发操作日志文件
var logFileLock sync.Mutex

// InitLogDirectory 初始化日志目录
func InitLogDirectory() {
	if err := os.MkdirAll(LogFilePath, 0755); err != nil {
		log.Printf("无法创建日志目录 %s: %v", LogFilePath, err)
		// 使用备用目录
		if err := os.MkdirAll(BackupLogPath, 0755); err != nil {
			log.Printf("无法创建备用日志目录 %s: %v", BackupLogPath, err)
		}
	}
}

// GetLogPath 获取可用的日志文件路径
func GetLogPath(filename string) string {
	primaryPath := filepath.Join(LogFilePath, filename)
	if _, err := os.Stat(filepath.Dir(primaryPath)); err == nil {
		return primaryPath
	}

	// 使用备用路径
	backupPath := filepath.Join(BackupLogPath, filename)
	if _, err := os.Stat(filepath.Dir(backupPath)); err != nil {
		// 创建备用目录
		os.MkdirAll(filepath.Dir(backupPath), 0755)
	}

	return backupPath
}

// WriteAccessLog 写入访问日志
func WriteAccessLog(goodsID, tag string) error {
	logFileLock.Lock()
	defer logFileLock.Unlock()

	logPath := GetLogPath(GoodsAccessLogFile)

	// 构建日志内容
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("tag:%s time:%s id:%s\n", tag, timestamp, goodsID)

	// 打开日志文件
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("无法打开日志文件: %v", err)
	}
	defer f.Close()

	// 写入日志内容
	if _, err := f.WriteString(logEntry); err != nil {
		return fmt.Errorf("无法写入日志: %v", err)
	}

	return nil
}

// ReadAndClearLog 读取并清除日志文件
func ReadAndClearLog(filename string) ([]byte, error) {
	logFileLock.Lock()
	defer logFileLock.Unlock()

	// 查找主路径和备用路径
	primaryPath := filepath.Join(LogFilePath, filename)
	backupPath := filepath.Join(BackupLogPath, filename)

	var logPath string
	if _, err := os.Stat(primaryPath); err == nil {
		logPath = primaryPath
	} else if _, err := os.Stat(backupPath); err == nil {
		logPath = backupPath
	} else {
		return []byte{}, nil // 文件不存在，返回空内容
	}

	// 读取日志内容
	content, err := ioutil.ReadFile(logPath)
	if err != nil {
		return nil, fmt.Errorf("读取日志文件失败: %v", err)
	}

	// 删除日志文件
	if err := os.Remove(logPath); err != nil {
		log.Printf("警告: 无法删除日志文件 %s: %v", logPath, err)
	}

	return content, nil
}

// ParseLogEntry 解析日志条目
func ParseLogEntry(line string) map[string]string {
	result := make(map[string]string)
	parts := strings.Split(line, " ")

	for _, part := range parts {
		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			if len(kv) == 2 {
				result[kv[0]] = kv[1]
			}
		}
	}

	return result
}
