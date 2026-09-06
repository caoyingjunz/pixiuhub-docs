package passwordutil

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// EncryptPassword bcrypt 加密密码（用于账号密码登录）
func EncryptPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("加密密码失败: %v", err)
	}
	return string(hash), nil
}

// ValidatePassword 校验密码是否匹配 bcrypt 哈希
func ValidatePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
