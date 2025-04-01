package router //app.go

import (
	"e-commerse/service"

	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	r := gin.Default()
	r.GET("/recommend", service.Recommend)      //获取推荐商品
	r.GET("/userinfo", service.GetUserInfo)     //查看本人信息
	r.POST("/send", service.SendMsg)            // 发送消息
	r.GET("/messages", service.GetMsg)          // 获取历史消息
	r.GET("/ws", service.WebSocketHandler)      // WebSocket连接
	r.GET("/unread", service.GetUnreadMessages) // 获取未读消息计数
	r.GET("/goods", service.GetGoods)           // 获取商品信息
	r.PUT("/goods", service.CreateGoods)        // 创建商品
	r.POST("/login", service.Login)             //用户登录
	r.POST("/register", service.Register)       //用户注册
	//r.GET("/goods", service.GetGoodsInfo)
	//r.GET("/vendorinfo", service.GetVendorinfo)   要不要后续加上
	//r.POST("/login", service.POSTLogin)
	//r.POST("/register", service.POSTRegister)
	/*im := r.Group("/im")
	{
		// WebSocket连接
		im.GET("/ws", service.IMConnect)

		// 获取聊天历史
		im.GET("/history", service.GetChatHistory)

		// 聊天室管理
		im.POST("/room", service.CreateChatRoom)
		im.GET("/rooms", service.GetRoomList)
	}*/
	return r
}
