package router

import (
	"e-commerse/middleware"
	"e-commerse/service"

	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	r := gin.Default()

	// 公共路由，无需鉴权
	r.POST("/login", service.Login)
	r.POST("/register", service.Register)

	// 受保护的路由，需通过JWT中间件进行认证
	authGroup := r.Group("/")
	authGroup.Use(middleware.JWT())
	{
		authGroup.GET("/recommend", service.Recommend)         // 获取推荐商品
		authGroup.GET("/userinfo", service.GetUserInfo)        // 查看本人信息
		authGroup.POST("/send", service.SendMsg)               // 发送消息
		authGroup.GET("/messages/:contace_id", service.GetMsg) // 获取历史消息
		authGroup.GET("/ws", service.WebSocketHandler)         // WebSocket连接
		authGroup.GET("/unread", service.GetUnreadMessages)    // 获取未读消息计数
		authGroup.GET("/goods/:id", service.GetGoods)          // 获取商品信息
		authGroup.PUT("/goods", service.CreateGoods)           // 创建商品

	}

	return r
}
