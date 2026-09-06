package model

import (
	"time"

	"github.com/caoyingjunz/rainbow/pkg/db/model/rainbow"
)

func init() {
	register(&Access{})
}

type Access struct {
	rainbow.Model
	rainbow.UserModel

	AccessKey  string     `gorm:"type:varchar(255);uniqueIndex:idx_access_key" json:"access_key"`
	SecretKey  string     `json:"secret_key"`
	ExpireTime *time.Time `json:"expire_time"`

	// SecretEncrypted 标记 SecretKey 是否已加密存储（存量明文数据为 false）
	SecretEncrypted bool `json:"-"`
}

func (a *Access) TableName() string {
	return "accesses"
}
