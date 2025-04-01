package service

import (
	"context"
	loginpb "e-commerse/rpc/login"
	"e-commerse/utils"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// 用户相关常量
const (
	// Redis缓存键前缀
	UserCachePrefix = "user:info:"
	// 用户缓存有效期
	UserCacheTTL = 24 * time.Hour
	// Token有效期
	TokenExpireDuration = 24 * time.Hour
)

// RegisterRequest 注册请求结构体
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
}

// LoginRequest 登录请求结构体
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserInfoCache 用于缓存的用户信息结构
type UserInfoCache struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
	Gender   int32  `json:"gender"`
}

// JWT自定义声明结构体
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.StandardClaims
}

// Register 用户注册
func Register(c *gin.Context) {
	// 解析请求参数
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 参数验证
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "用户名和密码不能为空",
		})
		return
	}

	// 连接gRPC服务器
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
	client := loginpb.NewUserServiceClient(conn)

	// 设置请求上下文，加入超时控制
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 调用gRPC方法注册用户
	resp, err := client.Register(ctx, &loginpb.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Phone:    req.Phone,
		Avatar:   req.Avatar,
	})
	if err != nil {
		c.JSON(500, gin.H{
			"code":    500,
			"message": "注册失败,可能未连接到Java代码",
		})
	}

	// 检查是否成功
	if !resp.Success {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    resp.Code,
			"message": resp.Message,
		})
		return
	}

	// 返回注册成功结果
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "注册成功",
		"data": map[string]interface{}{
			"user_id": resp.UserId,
		},
	})
}

func Login(c *gin.Context) {
	// 解析请求参数
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 参数验证
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "用户名和密码不能为空",
		})
		return
	}

	// 连接gRPC服务器进行身份验证
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
	client := loginpb.NewUserServiceClient(conn)

	// 设置请求上下文，加入超时控制
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 调用gRPC方法登录
	resp, err := client.Login(ctx, &loginpb.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "登录失败,可能未连接到Java代码",
		})
		return
	}
	// 检查登录是否成功
	if !resp.Success {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    resp.Code,
			"message": resp.Message,
		})
		return
	}

	// 登录成功，生成JWT令牌
	token, err := GenerateToken(resp.UserInfo.Id, resp.UserInfo.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "生成令牌失败",
			"error":   err.Error(),
		})
		return
	}

	// 将用户信息缓存到Redis，减少后续gRPC调用
	userCache := UserInfoCache{
		ID:       resp.UserInfo.Id,
		Username: resp.UserInfo.Username,
		Email:    resp.UserInfo.Email,
		Phone:    resp.UserInfo.Phone,
		Avatar:   resp.UserInfo.Avatar,
		Gender:   resp.UserInfo.Gender,
	}

	// 序列化用户信息
	userCacheJSON, _ := json.Marshal(userCache)

	// 存入Redis缓存
	err = utils.RedisClient.Set(
		context.Background(),
		UserCachePrefix+resp.UserInfo.Id,
		userCacheJSON,
		UserCacheTTL,
	).Err()

	// 设置用户在线状态
	utils.SetUserOnline(resp.UserInfo.Id)

	// 返回登录成功结果
	userInfo := map[string]interface{}{
		"id":       resp.UserInfo.Id,
		"username": resp.UserInfo.Username,
		"email":    resp.UserInfo.Email,
		"phone":    resp.UserInfo.Phone,
		"avatar":   resp.UserInfo.Avatar,
		"gender":   resp.UserInfo.Gender,
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "登录成功",
		"data": map[string]interface{}{
			"token":     token,
			"user_info": userInfo,
		},
	})
}

// GenerateToken 生成JWT令牌
// 参数: userID - 用户ID，username - 用户名
// 返回: 生成的JWT令牌字符串和可能的错误
func GenerateToken(userID, username string) (string, error) {
	// 设置JWT声明
	nowTime := time.Now()
	expireTime := nowTime.Add(TokenExpireDuration)

	claims := Claims{
		UserID:   userID,
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(), // 过期时间
			IssuedAt:  nowTime.Unix(),    // 签发时间
			Issuer:    "e-commerce-api",  // 签发人
		},
	}

	// 使用HS256算法创建令牌
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString([]byte(utils.JWTSecret))

	return token, err
}

// GetUserFromCache 从缓存获取用户信息
// 如果缓存未命中则返回error，调用方需要回退到gRPC获取
func GetUserFromCache(userID string) (UserInfoCache, error) {
	// 从Redis获取用户缓存
	data, err := utils.RedisClient.Get(context.Background(), UserCachePrefix+userID).Result()

	if err != nil {
		if err == redis.Nil {
			return UserInfoCache{}, errors.New("用户缓存不存在")
		}
		return UserInfoCache{}, err
	}

	// 解析用户信息
	var userInfo UserInfoCache
	if err := json.Unmarshal([]byte(data), &userInfo); err != nil {
		return UserInfoCache{}, err
	}

	return userInfo, nil
}
