package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/caoyingjunz/rainbow/pkg/db/model"
)

const (
	auditQueueSize    = 2048
	auditWorkers      = 2
	auditWriteTimeout = 3 * time.Second
)

var auditQueue chan *model.Audit

// Audit 操作审计中间件：异步记录非 GET 请求（用户、方法、路径、状态码、耗时）
func Audit(db *gorm.DB) gin.HandlerFunc {
	if db != nil {
		auditQueue = make(chan *model.Audit, auditQueueSize)
		startAuditWorkers(db)
	}

	return func(c *gin.Context) {
		if auditQueue == nil {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		// 仅审计写操作
		if c.Request.Method == "GET" || c.Request.Method == "OPTIONS" || c.Request.Method == "HEAD" {
			return
		}

		userId := ""
		userName := ""
		if claims := GetLoginUser(c); claims != nil {
			userId = claims.UserId
			userName = claims.Name
		}

		audit := &model.Audit{
			UserId:     userId,
			UserName:   userName,
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			StatusCode: c.Writer.Status(),
			ClientIP:   c.ClientIP(),
			Duration:   time.Since(start).Milliseconds(),
			CreatedAt:  time.Now(),
		}

		// 队列满直接丢弃，不阻塞请求
		select {
		case auditQueue <- audit:
		default:
			klog.Warningf("审计队列已满，丢弃审计记录: %s %s", audit.Method, audit.Path)
		}
	}
}

func startAuditWorkers(db *gorm.DB) {
	for i := 0; i < auditWorkers; i++ {
		go func() {
			for audit := range auditQueue {
				writeAudit(db, audit)
			}
		}()
	}
}

func writeAudit(db *gorm.DB, audit *model.Audit) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := db.Create(audit).Error; err != nil {
			klog.Errorf("写入审计日志失败: %v", err)
		}
	}()

	select {
	case <-done:
	case <-time.After(auditWriteTimeout):
		klog.Errorf("写入审计日志超时")
	}
}
