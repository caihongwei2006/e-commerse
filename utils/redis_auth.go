package utils

import (
	"context"
	"encoding/json"
	"time"
)

const (
	// 用户缓存前缀
	UserCachePrefix = "user:cache:"

	// 用户令牌黑名单前缀
	TokenBlacklistPrefix = "token:blacklist:"

	// 用户缓存过期时间（1小时）
	UserCacheExpiration = time.Hour
)

// UserCache 用户缓存结构体
type UserCache struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
	Role     string `json:"role"`
}

// CacheUserInfo 缓存用户信息到Redis
func CacheUserInfo(userID string, userInfo UserCache) error {
	data, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}

	key := UserCachePrefix + userID
	return RedisClient.Set(context.Background(), key, string(data), UserCacheExpiration).Err()
}

// GetCachedUserInfo 从Redis获取缓存的用户信息
func GetCachedUserInfo(userID string) (*UserCache, error) {
	key := UserCachePrefix + userID
	data, err := RedisClient.Get(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}

	var userInfo UserCache
	if err := json.Unmarshal([]byte(data), &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// AddTokenToBlacklist 将令牌加入黑名单,避免
// 参数:
//   - token: JWT令牌
//   - expiration: 令牌原本的过期时间
func AddTokenToBlacklist(token string, expiration time.Duration) error {
	key := TokenBlacklistPrefix + token
	return RedisClient.Set(context.Background(), key, "1", expiration).Err()
}

// IsTokenBlacklisted 检查令牌是否在黑名单中
func IsTokenBlacklisted(token string) bool {
	key := TokenBlacklistPrefix + token
	exists, err := RedisClient.Exists(context.Background(), key).Result()
	return err == nil && exists > 0
}
