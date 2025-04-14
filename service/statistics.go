package service

import (
	"context"
	"e-commerse/utils"
	"log"

	statisticspb "e-commerse/rpc/statistics"
)

// StatisticsServer 实现统计服务接口
type StatisticsServer struct {
	statisticspb.UnimplementedStatisticsServer
}

// NewStatisticsServer 创建一个新的统计服务实例
func NewStatisticsServer() *StatisticsServer {
	return &StatisticsServer{}
}

// GetLog 实现RPC方法，接收请求字符串，返回日志内容并清除日志
func (s *StatisticsServer) GetLog(ctx context.Context, req *statisticspb.StatisticsRequest) (*statisticspb.StatisticsResponse, error) {
	// 记录请求信息
	log.Printf("收到日志请求，客户端请求: %s", req.Req)

	// 读取并清除日志文件
	logData, err := utils.ReadAndClearLog(utils.GoodsAccessLogFile)
	if err != nil {
		log.Printf("读取日志文件失败: %v", err)
		return &statisticspb.StatisticsResponse{
			Log: []byte{},
		}, err
	}

	log.Printf("已读取并返回日志文件内容，大小: %d 字节", len(logData))

	// 返回日志内容
	return &statisticspb.StatisticsResponse{
		Log: logData,
	}, nil
}

// LogGoodsAccess 记录商品访问日志 - 这个函数会被userservice.go中的GetGoods调用
func LogGoodsAccess(goodsID, tag string) error {
	return utils.WriteAccessLog(goodsID, tag)
}
