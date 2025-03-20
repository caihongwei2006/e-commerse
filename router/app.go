package router

import (
	"e-commerse/service"

	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	r := gin.Default()
	r.GET("/recommend", service.Recommend)
	r.GET("/userinfo", service.GetUserInfo)
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
