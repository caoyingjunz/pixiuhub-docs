package tokenutil

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	jwt.RegisteredClaims

	UserId  string `json:"user_id"`
	Project string `json:"project"`
}

// LoginClaims 登录态 JWT Claims（含用户 ID、名称、角色）
type LoginClaims struct {
	jwt.RegisteredClaims

	UserId string `json:"user_id"`
	Name   string `json:"name"`
	Role   int    `json:"role"`
}

// GenerateToken 生成 token
func GenerateToken(userId string, project string, jwtKey []byte) (string, error) {
	nowTime := time.Now()
	expiresTime := nowTime.Add(5 * time.Minute)
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresTime), // 过期时间
			IssuedAt:  jwt.NewNumericDate(nowTime),     // 签发时间
			NotBefore: jwt.NewNumericDate(nowTime),     // 生效时间
		},
		UserId:  userId,
		Project: project,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// GenerateLoginToken 生成登录态 JWT（6 小时有效期）
func GenerateLoginToken(userId, name string, role int, jwtKey []byte) (string, error) {
	nowTime := time.Now()
	expiresTime := nowTime.Add(6 * time.Hour)
	claims := &LoginClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresTime), // 过期时间
			IssuedAt:  jwt.NewNumericDate(nowTime),     // 签发时间
			NotBefore: jwt.NewNumericDate(nowTime),     // 生效时间
		},
		UserId: userId,
		Name:   name,
		Role:   role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ParseToken(tokenStr string, jwtKey []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors == jwt.ValidationErrorExpired {
				return nil, fmt.Errorf("token 已经过期")
			}
		}
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("failed to parse token")
}

// ParseLoginToken 解析登录态 JWT
func ParseLoginToken(tokenStr string, jwtKey []byte) (*LoginClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &LoginClaims{}, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors == jwt.ValidationErrorExpired {
				return nil, fmt.Errorf("登录已过期，请重新登录")
			}
		}
		return nil, err
	}

	if claims, ok := token.Claims.(*LoginClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("failed to parse login token")
}
