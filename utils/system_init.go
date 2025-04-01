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
	DB    *gorm.DB
	Redis *redis.Client
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