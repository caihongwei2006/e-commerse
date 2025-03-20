package utils

import (
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type UserBasic struct {
	gorm.Model
	Username   string
	Password   string
	Email      string
	Phone      string
	ClientPost string
	UserID     string
}

var (
	DB          *gorm.DB
	Redis       *redis.Client
	RedisClient *redis.Client // 导出的Redis客户端，用于其他包直接使用
)

func InitConfig() {
	viper.SetConfigName("app")
	viper.AddConfigPath("config")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("config file error!!!!!", err)
	}
	fmt.Println("config file success!!!!!!!")
	fmt.Println("config ", viper.Get("mysql"))
	fmt.Println("config mysql", viper.Get("mysql"))
}

func InitRedis() {
	Redis = redis.NewClient(&redis.Options{
		Addr:         viper.GetString("redis.addr"),
		Password:     viper.GetString("redis.password"),
		DB:           viper.GetInt("redis.db"),
		PoolSize:     viper.GetInt("redis.poolsize"),
		MinIdleConns: viper.GetInt("redis.minidleconns"),
	})
	pong, err := Redis.Ping(Redis.Context()).Result()
	if err != nil {
		fmt.Println("redis connect failed", err)
	} else {
		fmt.Println("redis connect success", pong)
	}

	// 设置全局可访问的Redis客户端
	RedisClient = Redis
}

const (
	PublishKey = "websocket"

	// Redis键前缀常量
	UserOnlinePrefix     = "user:online:"     // 用户在线状态前缀
	UserSessionPrefix    = "user:sessions:"   // 用户会话列表前缀
	ChatMessagePrefix    = "chat:messages:"   // 聊天消息前缀
	RoomInfoPrefix       = "room:"            // 聊天室信息前缀
	UserUnreadPrefix     = "user:unread:"     // 用户未读消息计数前缀
	UserLastActivePrefix = "user:lastactive:" // 用户最后活跃时间前缀
)
