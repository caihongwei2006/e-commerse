package utils

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Claims 自定义JWT声明结构体
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

var (
	// 从环境变量获取JWT密钥，如果没有则使用默认值
	JWTSecret = []byte(getEnvOrDefault("JWT_SECRET", "e-commerce-secret-key-change-in-production"))
	jwtSecret = JWTSecret

	// 令牌有效期
	TokenExpireDuration = 24 * time.Hour

	// 刷新令牌有效期
	RefreshTokenExpireDuration = 7 * 24 * time.Hour
)

// getEnvOrDefault 获取环境变量，如不存在则使用默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GenerateToken(userID, username string, extraClaims map[string]interface{}) (string, int64, error) {
	// 计算过期时间
	now := time.Now()
	expireTime := now.Add(TokenExpireDuration)
	expireAt := expireTime.Unix()

	// 创建JWT声明
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "e-commerce-api",                         // 签发者
			Subject:   userID,                                   // 主题
			ExpiresAt: jwt.NewNumericDate(expireTime),           // 过期时间
			NotBefore: jwt.NewNumericDate(now),                  // 生效时间
			IssuedAt:  jwt.NewNumericDate(now),                  // 签发时间
			ID:        fmt.Sprintf("%s-%d", userID, now.Unix()), // JWT唯一标识
		},
	}

	// 添加额外的用户信息
	if email, ok := extraClaims["email"].(string); ok {
		claims.Email = email
	}
	if phone, ok := extraClaims["phone"].(string); ok {
		claims.Phone = phone
	}
	if avatar, ok := extraClaims["avatar"].(string); ok {
		claims.Avatar = avatar
	}
	if role, ok := extraClaims["role"].(string); ok {
		claims.Role = role
	}

	// 创建JWT对象并签名
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString(jwtSecret)

	return token, expireAt, err
}

// GenerateRefreshToken 生成刷新令牌
// 只包含最少的用户信息，有效期更长
func GenerateRefreshToken(userID string) (string, int64, error) {
	// 计算过期时间
	now := time.Now()
	expireTime := now.Add(RefreshTokenExpireDuration)
	expireAt := expireTime.Unix()

	// 创建最小化的JWT声明
	claims := jwt.RegisteredClaims{
		Issuer:    "e-commerce-api",
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(expireTime),
		IssuedAt:  jwt.NewNumericDate(now),
		ID:        fmt.Sprintf("refresh-%s-%d", userID, now.Unix()),
	}

	// 创建JWT对象并签名
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString(jwtSecret)

	return token, expireAt, err
}

// ParseToken 解析JWT令牌
// 返回声明信息和可能的错误
func ParseToken(tokenString string) (*Claims, error) {
	// 使用密钥解析令牌
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	// 处理解析错误
	if err != nil {
		return nil, err
	}

	// 类型断言获取声明
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ParseRefreshToken 解析刷新令牌
// 返回用户ID和可能的错误
func ParseRefreshToken(tokenString string) (string, error) {
	// 使用密钥解析令牌
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return claims.Subject, nil
	}

	return "", errors.New("invalid refresh token")
}

func ExtractUserIDFromToken(tokenString string) (string, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}
