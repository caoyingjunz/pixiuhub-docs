package loginlimit

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/caoyingjunz/rainbow/pkg/util/lru"
)

const (
	// 全局限流：所有 IP 合计
	globalRate  = rate.Limit(20)
	globalBurst = 40
	// 每 IP 限流：每分钟 5 次登录尝试（突发 3）
	ipRate  = rate.Limit(5.0 / 60.0)
	ipBurst = 3
	// 用户名维度：连续失败 8 次锁定 2 分钟
	maxFailures  = 8
	lockDuration = 2 * time.Minute
	// 锁定期间每分钟仅 1 次探测（让用户知道仍在锁定，而不是无限期无响应）
	lockedProbeRate  = rate.Limit(1.0 / 60.0)
	lockedProbeBurst = 1
	// bcrypt 校验并发上限（防止爆破打满 CPU）
	maxConcurrentVerify = 4
	acquireWait         = 50 * time.Millisecond
	// 缓存容量与过期策略
	ipLimiterCap      = 8192
	probeLimiterCap   = 4096
	maxFailureEntries = 4096
	entryTTL          = 30 * time.Minute
	purgeInterval     = 5 * time.Minute
)

var (
	ipLimiters    = lru.NewLRUCache(ipLimiterCap)
	probeLimiters = lru.NewLRUCache(probeLimiterCap)
	verifySem     = make(chan struct{}, maxConcurrentVerify)
	globalLimiter = rate.NewLimiter(globalRate, globalBurst)

	failureMu sync.Mutex
	failures  = map[string]*failureState{}
	sweeper   sync.Once
)

type failureState struct {
	count       int
	lockedUntil time.Time
	lastSeen    time.Time
}

// AllowRequest 按 IP 判定登录请求是否放行（全局限流 + 每 IP 限流）
func AllowRequest(ip string) bool {
	ensureSweeper()
	if !globalLimiter.Allow() {
		return false
	}
	ipLimiter, _ := getRateLimiter(ipLimiters, "ip:"+ip, ipRate, ipBurst)
	return ipLimiter.Allow()
}

// AllowUserAttempt 用户名维度锁定探测：未锁定直接放行；锁定期间仅允许每分钟 1 次尝试
func AllowUserAttempt(name string) bool {
	ensureSweeper()
	key := normalizeUser(name)

	failureMu.Lock()
	defer failureMu.Unlock()

	st, ok := failures[key]
	if !ok {
		return true
	}
	st.lastSeen = time.Now()
	if time.Now().Before(st.lockedUntil) {
		probeLimiter, _ := getRateLimiter(probeLimiters, "locked-probe:"+key, lockedProbeRate, lockedProbeBurst)
		return probeLimiter.Allow()
	}
	return true
}

// RecordUserFailure 记录一次登录失败，连续失败达上限则锁定
func RecordUserFailure(name string) {
	key := normalizeUser(name)
	now := time.Now()

	failureMu.Lock()
	defer failureMu.Unlock()

	st, ok := failures[key]
	if !ok {
		st = &failureState{lastSeen: now}
		failures[key] = st
	}
	// 锁定过期后计数归零
	if now.After(st.lockedUntil) && st.count >= maxFailures {
		st.count = 0
	}
	st.count++
	st.lastSeen = now
	if st.count >= maxFailures {
		st.lockedUntil = now.Add(lockDuration)
	}
}

// ClearUserFailures 登录成功后清除失败记录
func ClearUserFailures(name string) {
	key := normalizeUser(name)

	failureMu.Lock()
	defer failureMu.Unlock()
	delete(failures, key)
}

// AcquireVerify 获取 bcrypt 校验并发闸门，防止爆破打满 CPU
func AcquireVerify() bool {
	select {
	case verifySem <- struct{}{}:
		return true
	default:
		timer := time.NewTimer(acquireWait)
		defer timer.Stop()
		select {
		case verifySem <- struct{}{}:
			return true
		case <-timer.C:
			return false
		}
	}
}

// ReleaseVerify 释放 bcrypt 校验闸门
func ReleaseVerify() {
	select {
	case <-verifySem:
	default:
	}
}

func normalizeUser(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func getRateLimiter(cache *lru.LRUCache, key string, limit rate.Limit, burst int) (*rate.Limiter, error) {
	if v := cache.Get(key); v != nil {
		if l, ok := v.(*rate.Limiter); ok {
			return l, nil
		}
		return nil, fmt.Errorf("invalid limiter cache value")
	}
	l := rate.NewLimiter(limit, burst)
	cache.Add(key, l)
	return l, nil
}

// ensureSweeper 后台周期清理过期的失败记录
func ensureSweeper() {
	sweeper.Do(func() {
		go func() {
			ticker := time.NewTicker(purgeInterval)
			defer ticker.Stop()
			for range ticker.C {
				purgeExpiredFailures()
			}
		}()
	})
}

func purgeExpiredFailures() {
	now := time.Now()

	failureMu.Lock()
	defer failureMu.Unlock()

	for key, st := range failures {
		if now.Sub(st.lastSeen) > entryTTL {
			delete(failures, key)
		}
	}
	// 超容量时按 lastSeen 淘汰最旧条目
	for len(failures) > maxFailureEntries {
		var oldestKey string
		var oldest time.Time
		for key, st := range failures {
			if oldestKey == "" || st.lastSeen.Before(oldest) {
				oldestKey = key
				oldest = st.lastSeen
			}
		}
		if oldestKey == "" {
			break
		}
		delete(failures, oldestKey)
	}
}
