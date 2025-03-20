package service

import (
	"context"
	"e-commerse/utils"
	"net/http"
	"time"

	recommendpb "e-commerse/rpc/proto/recommend"
	userpb "e-commerse/rpc/proto/user"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetUserInfo(c *gin.Context) {
	user := utils.UserBasic{}
	user.UserID = c.Query("id")
	if user.UserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "用户ID不能为空",
		})
		return
	}
	conn, err := grpc.Dial("8.152.221.3:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "无法连接用户服务",
			"error":   err.Error(),
		})
		return
	}
	defer conn.Close()

	// 创建gRPC客户端
	client := userpb.NewUserServiceClient(conn)

	// 设置请求上下文，加入超时控制
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 调用gRPC方法获取用户信息
	resp, err := client.GetUserById(ctx, &userpb.UserRequest{
		UserId: user.UserID,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取用户信息失败",
			"error":   err.Error(),
		})
		return
	}

	// 封装用户信息
	userInfo := map[string]interface{}{
		"id":         resp.Id,
		"name":       resp.Name,
		"email":      resp.Email,
		"phone":      resp.Phone,
		"avatar":     resp.Avatar,
		"gender":     resp.Gender,
		"address":    resp.Address,
		"created_at": time.Unix(resp.CreatedAt, 0).Format("2006-01-02 15:04:05"),
		"status":     resp.Status,
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取用户信息成功",
		"data":    userInfo,
	})
}

var upGrade = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func Recommend(c *gin.Context) {
	UserID := c.Query("id")
	if UserID == "" {
		c.JSON(400, gin.H{
			"message": "id is required",
		})
		return
	}
	conn, err := grpc.Dial("8.152.221.3:9090", grpc.WithInsecure())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "failed to connect to Java server",
			"error":   err.Error(),
		})
		return
	}
	defer conn.Close()
	client := recommendpb.NewRecommendServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	req := &recommendpb.RecommendRequest{
		UserId: UserID,
	}
	resp, err := client.GetRecommendations(ctx, req)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取推荐失败",
			"error":   err.Error(),
		})
		return
	}

	// 将gRPC响应转换为前端所需的格式
	recommendations := make([]map[string]interface{}, 0)
	for _, item := range resp.Items {
		if item == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "推荐数据为空,连接java失败",
			})
		}
		recommendations = append(recommendations, map[string]interface{}{
			"good_id":     1,
			"merchant_id": 3,
			"name":        "Apple Pen",
			"price":       114.5,
			"picture":     "https://img.com/asdf",
			"full_desc":   "This is an apple, this is a pen",
		})
	}
	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取推荐成功",
		"data":    recommendations,
	})
}
