package model

import (
	"time"

	"github.com/caoyingjunz/rainbow/pkg/db/model/rainbow"
)

func init() {
	register(&User{})
}

type User struct {
	rainbow.Model

	UserId string `gorm:"type:varchar(255);uniqueIndex:idx_user" json:"user_id"` // 用户ID
	Name   string `json:"name"`                                                  // 用户名
	Role   int    `json:"role"`                                                  // 用户角色，0 普通用户， 1 管理员

	// 账号密码登录（bcrypt 哈希存储，禁止明文）
	Password string `gorm:"type:varchar(255)" json:"-"`
	Status   int    `json:"status"` // 用户状态，0 正常，1 禁用

	UserType    int       `json:"user_type"`                                                                              // 0 个人版，1 专有版
	PaymentType int       `json:"payment_type"`                                                                           // 付费模式 0 按量付费， 1 包年包月
	ExpireTime  time.Time `gorm:"column:expire_time;type:datetime;default:current_timestamp;not null" json:"expire_time"` // 包年包月时到期时间
	RemainCount int       `json:"remain_count"`                                                                           // 按量付费时剩余次数
	EnableChart bool      `json:"enable_chart"`                                                                           // 启用 chart 仓库
	Email       string    `json:"email"`                                                                                  // 用户邮箱
}

// 用户状态
const (
	UserStatusNormal    = 0
	UserStatusForbidden = 1
)

// 用户角色
const (
	RoleNormal = 0
	RoleAdmin  = 1
)

func (t *User) TableName() string {
	return "users"
}
