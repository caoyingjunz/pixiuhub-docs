package model

import (
	"time"

	"github.com/caoyingjunz/rainbow/pkg/db/model/rainbow"
)

func init() {
	register(&Audit{})
}

// Audit 操作审计日志
type Audit struct {
	rainbow.Model

	UserId     string    `gorm:"type:varchar(255);index:idx_audit_user" json:"user_id"`
	UserName   string    `gorm:"type:varchar(255)" json:"user_name"`
	Method     string    `gorm:"type:varchar(16)" json:"method"`
	Path       string    `gorm:"type:varchar(512);index:idx_audit_path" json:"path"`
	StatusCode int       `json:"status_code"`
	ClientIP   string    `gorm:"type:varchar(64)" json:"client_ip"`
	Duration   int64     `json:"duration_ms"` // 耗时（毫秒）
	CreatedAt  time.Time `json:"created_at"`
}

func (a *Audit) TableName() string {
	return "audits"
}
