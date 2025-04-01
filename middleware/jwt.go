package middleware

import (
	"e-commerse/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// JWT 认证中间件
func JWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取令牌
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "请求未携带令牌",
			})
			c.Abort()
			return
		}

		// 分割Bearer和令牌内容
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "令牌格式错误",
			})
			c.Abort()
			return
		}

		// 获取令牌内容
		tokenString := parts[1]

		// 检查令牌是否在黑名单中
		if utils.IsTokenBlacklisted(tokenString) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "令牌已被吊销",
			})
			c.Abort()
			return
		}

		// 解析令牌
		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "无效的令牌",
				"error":   err.Error(),
			})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("phone", claims.Phone)
		c.Set("avatar", claims.Avatar)
		c.Set("role", claims.Role)
		c.Set("token", tokenString)

		c.Next()
	}
}
