package router

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/caoyingjunz/pixiulib/httputils"
	"github.com/caoyingjunz/pixiulib/strutil"
	"github.com/caoyingjunz/rainbow/pkg/util/lru"
	"github.com/caoyingjunz/rainbow/pkg/util/tokenutil"
	"github.com/gin-gonic/gin"
	"github.com/juju/ratelimit"
	"golang.org/x/time/rate"

	"github.com/caoyingjunz/rainbow/cmd/app/options"
	"github.com/caoyingjunz/rainbow/pkg/util/signatureutil"
)

// userContextKey 登录用户注入 gin context 的 key
const userContextKey = "login_user"

// GetLoginUser 从 gin context 获取登录用户 claims
func GetLoginUser(c *gin.Context) *tokenutil.LoginClaims {
	if v, ok := c.Get(userContextKey); ok {
		if claims, ok := v.(*tokenutil.LoginClaims); ok {
			return claims
		}
	}
	return nil
}

func NewMiddlewares(o *options.ServerOptions) {
	o.HttpEngine.Use(
		SignatureMiddleware(o),
		Authentication(o),
		UserRateLimiter(o),
		Limiter(o),
		Audit(o.GetDB()),
	)
}

func SignatureMiddleware(o *options.ServerOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isPublicPath(c.Request.URL.Path) {
			return
		}
		// 对 /api/v2 开头的 api 进行签名校验（携带 Bearer JWT 的浏览器请求跳过，走 JWT 认证）
		if strings.HasPrefix(c.Request.URL.String(), "/api/v2") &&
			!strings.HasPrefix(c.GetHeader("Authorization"), "Bearer ") {
			// 目前仅对 /api/v2 的资源进行签名校验
			if err := signatureutil.VerifySignature(c, o.Factory, o.ComponentConfig.Server.EncryptKey); err != nil {
				httputils.AbortFailedWithCode(c, http.StatusUnauthorized, err)
				return
			}
		}
	}
}

// Authentication 身份认证
func Authentication(o *options.ServerOptions) gin.HandlerFunc {
	cfg := o.ComponentConfig
	auth := cfg.Server.Auth

	return func(c *gin.Context) {
		// 安全修复：移除 debug 模式下的鉴权绕过，认证在任何运行模式下均强制生效
		if isPublicPath(c.Request.URL.Path) {
			return
		}

		// 支持两种认证方式:
		// 1. Authorization: Bearer JWT —— 浏览器账号密码登录态（优先识别）
		// 2. /api/v2 下的 AK/SK 签名 —— pixiuctl CLI 机器间认证（兼容保留）
		tokenStr := c.GetHeader("Authorization")
		if strings.HasPrefix(tokenStr, "Bearer ") {
			token := strings.TrimSpace(strings.TrimPrefix(tokenStr, "Bearer "))
			claims, err := tokenutil.ParseLoginToken(token, []byte(cfg.Server.JWTKey))
			if err != nil {
				httputils.AbortFailedWithCode(c, http.StatusUnauthorized, err)
				return
			}
			// 将登录用户写入 context，供后续业务使用
			c.Set(userContextKey, claims)
			return
		}

		// AK/SK 签名认证（pixiuctl CLI）
		if strings.HasPrefix(c.Request.URL.Path, "/api/v2") {
			accessKey := c.GetHeader("accessKey")
			if accessKey != auth.AccessKey {
				httputils.AbortFailedWithCode(c, http.StatusUnauthorized, fmt.Errorf("invalid Access Key"))
				return
			}

			timestamp := c.GetHeader("timestamp")
			if err := verifyTimeStamp(timestamp); err != nil {
				httputils.AbortFailedWithCode(c, http.StatusUnauthorized, err)
				return
			}

			signature := c.GetHeader("signature")
			if !verifySignature(accessKey, auth.SecretKey, signature, timestamp) {
				httputils.AbortFailedWithCode(c, http.StatusUnauthorized, fmt.Errorf("invalid Signature"))
				return
			}
			return
		}

		// 非 /api/v2 且无 JWT：未登录
		httputils.AbortFailedWithCode(c, http.StatusUnauthorized, fmt.Errorf("未登录"))
	}
}

func isPublicPath(path string) bool {
	if strings.HasPrefix(path, "/api/v2/pixiuctls") {
		return true
	}
	// 登录接口为公开路径（无需签名/登录）
	if strings.HasPrefix(path, "/api/v2/users/login") {
		return true
	}

	return false
}

func verifyTimeStamp(timestamp string) error {
	ts, err := strutil.ParseInt64(timestamp)
	if err != nil {
		return fmt.Errorf("invalid Timestamp %s %v", timestamp, err)
	}
	if time.Now().Unix()-ts > 60*5 {
		return fmt.Errorf("timestamp expired")
	}

	return nil
}

func verifySignature(accessKey, secretKey, signature, timestamp string) bool {
	// 构造签名字符串
	message := fmt.Sprintf("ak=%s&timestamp=%s", accessKey, timestamp)

	// 使用HMAC-SHA256算法生成签名
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(message))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	// 比较签名
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// Limiter 限速
func Limiter(o *options.ServerOptions) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(o.ComponentConfig.RateLimit.NormalRateLimit.MaxRequests), o.ComponentConfig.RateLimit.NormalRateLimit.MaxRequests)
	specialLimiter := rate.NewLimiter(rate.Limit(o.ComponentConfig.RateLimit.SpecialRateLimit.MaxRequests), o.ComponentConfig.RateLimit.SpecialRateLimit.MaxRequests)
	return func(c *gin.Context) {
		if isRateLimitedPath(c.Request.URL.Path, o.ComponentConfig.RateLimit.SpecialRateLimit.RateLimitedPath) {
			if !specialLimiter.Allow() {
				httputils.AbortFailedWithCode(c, http.StatusTooManyRequests, fmt.Errorf("too many requests"))
			}
		} else {
			if !limiter.Allow() {
				httputils.AbortFailedWithCode(c, http.StatusTooManyRequests, fmt.Errorf("too many requests"))
			}
		}
	}
}

// 检查请求路径是否在限速列表中
func isRateLimitedPath(path string, rateLimitedPaths []string) bool {
	for _, limitedPath := range rateLimitedPaths {
		if strings.Contains(path, limitedPath) {
			return true
		}
	}
	return false
}

func UserRateLimiter(o *options.ServerOptions) gin.HandlerFunc {

	cache := lru.NewLRUCache(o.ComponentConfig.RateLimit.UserRateLimit.Cap)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if !cache.Contains(clientIP) {
			cache.Add(clientIP, ratelimit.NewBucketWithQuantum(time.Second, int64(o.ComponentConfig.RateLimit.UserRateLimit.Capacity), int64(o.ComponentConfig.RateLimit.UserRateLimit.Quantum)))
			return
		}
		// 通过 ClientIP 取出 bucket
		val := cache.Get(clientIP)
		if val == nil {
			return
		}

		// 判断是否还有可用的 bucket
		bucket := val.(*ratelimit.Bucket)
		if bucket.TakeAvailable(1) == 0 {
			httputils.AbortFailedWithCode(c, http.StatusTooManyRequests, fmt.Errorf("too many requests"))
		}
	}
}
