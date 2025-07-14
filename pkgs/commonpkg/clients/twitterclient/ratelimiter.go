package twitterclient

import (
	"context"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

////////////////////////////////////////////////////////////////////////////////
// Rate Limiting Structures and Logic
////////////////////////////////////////////////////////////////////////////////

// xRateLimit represents Twitter API rate limit information for a specific endpoint
type xRateLimit struct {
	ResetTime time.Time
	Remaining int
	Limit     int
	Ready     bool
	Url       string
	Mtx       sync.Mutex
}

// wouldBlock checks if making a request would trigger rate limiting (internal, not thread-safe)
func (rl *xRateLimit) wouldBlock() bool {
	threshold := max(2*rl.Limit/100, 1)
	return rl.Remaining <= threshold && time.Now().Before(rl.ResetTime)
}

// safeWouldBlock checks if making a request would trigger rate limiting (thread-safe)
func (rl *xRateLimit) safeWouldBlock() bool {
	rl.Mtx.Lock()
	defer rl.Mtx.Unlock()
	return rl.wouldBlock()
}

// safePreRequest handles rate limiting logic before making a request
func (rl *xRateLimit) safePreRequest(ctx context.Context, nonBlocking bool) error {
	rl.Mtx.Lock()
	defer rl.Mtx.Unlock()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if time.Now().After(rl.ResetTime) {
		log.
			WithFields(log.Fields{"path": rl.Url}).
			Debugf("[RateLimiter] rate limit is expired")
		rl.Ready = false // 后续的请求等待本次请求完成更新速率限制
		return nil
	}

	if !rl.wouldBlock() {
		rl.Remaining--
		return nil
	}

	if nonBlocking {
		return ErrWouldBlock
	}

	insurance := 5 * time.Second
	log.
		WithFields(log.Fields{
			"path":  rl.Url,
			"until": rl.ResetTime.Add(insurance),
		}).
		Warnln("[RateLimiter] start sleeping")

	select {
	case <-time.After(time.Until(rl.ResetTime) + insurance):
		rl.Ready = false
	case <-ctx.Done():
	}
	return nil
}

// makeRateLimit creates a rate limit from HTTP response headers
func makeRateLimit(resp *resty.Response) *xRateLimit {
	header := resp.Header()
	limit := header.Get("X-Rate-Limit-Limit")
	if limit == "" {
		return nil // 没有速率限制信息
	}
	remaining := header.Get("X-Rate-Limit-Remaining")
	if remaining == "" {
		return nil // 没有速率限制信息
	}
	resetTime := header.Get("X-Rate-Limit-Reset")
	if resetTime == "" {
		return nil // 没有速率限制信息
	}

	resetTimeNum, err := strconv.ParseInt(resetTime, 10, 64)
	if err != nil {
		return nil
	}
	remainingNum, err := strconv.Atoi(remaining)
	if err != nil {
		return nil
	}
	limitNum, err := strconv.Atoi(limit)
	if err != nil {
		return nil
	}

	u, _ := url.Parse(resp.Request.URL)
	urlPath := filepath.Join(u.Host, u.Path)

	resetTimeTime := time.Unix(resetTimeNum, 0)
	return &xRateLimit{
		ResetTime: resetTimeTime,
		Remaining: remainingNum,
		Limit:     limitNum,
		Ready:     true,
		Url:       urlPath,
	}
}

// rateLimiter manages rate limiting for multiple API endpoints
type rateLimiter struct {
	limits      sync.Map
	conds       sync.Map
	nonBlocking bool
}

// newRateLimiter creates a new rate limiter
func newRateLimiter(nonBlocking bool) *rateLimiter {
	return &rateLimiter{nonBlocking: nonBlocking}
}

// check verifies if a request can proceed without hitting rate limits
func (rl *rateLimiter) check(ctx context.Context, url *url.URL) error {
	if !rl.shouldWork(url) {
		return nil
	}

	path := url.Path
	maybeCond, _ := rl.conds.LoadOrStore(path, sync.NewCond(&sync.Mutex{}))
	cond := maybeCond.(*sync.Cond)
	cond.L.Lock()
	defer cond.L.Unlock()

	maybeLimit, loaded := rl.limits.LoadOrStore(path, &xRateLimit{})
	limit := maybeLimit.(*xRateLimit)
	if !loaded {
		// 首次遇见某路径时直接请求初始化它，后续请求等待这次请求使 limit 就绪
		return nil
	}

	/*
		同一时刻仅允许一个未就绪的请求通过检查，其余在这里阻塞，等待前者将速率限制就绪
		未就绪的情况：
		1. 首次请求
		2. 休眠后，速率限制过期
	*/
	for limit != nil && !limit.Ready {
		cond.Wait()
		maybeLimit, loaded := rl.limits.LoadOrStore(path, &xRateLimit{})
		if !loaded {
			// 上个请求失败了，从它身上继承初始化速率限制的重任
			return nil
		}
		limit = maybeLimit.(*xRateLimit)
	}

	// limiter 为 nil 意味着不对此路径做速率限制
	if limit != nil {
		return limit.safePreRequest(ctx, rl.nonBlocking)
	}
	return nil
}

// reset resets the rate limit information after a request
func (rl *rateLimiter) reset(url *url.URL, resp *resty.Response) {
	if !rl.shouldWork(url) {
		return
	}

	path := url.Path
	maybeCond, ok := rl.conds.Load(path)
	if !ok {
		return // BeforeRequest 从未调用的情况下调用了 OnError/OnRetry
	}
	cond := maybeCond.(*sync.Cond)
	cond.L.Lock()
	defer cond.L.Unlock()

	maybeLimit, ok := rl.limits.Load(path)
	if !ok {
		return
	}
	limit := maybeLimit.(*xRateLimit)
	if limit == nil || limit.Ready {
		return
	}

	if resp == nil || resp.RawResponse == nil {
		// 将此路径设为首次请求前的状态
		rl.limits.Delete(path)
		cond.Signal()
		return
	}

	// 请求成功，或发生了错误/触发了重试条件但有能力更新速率限制
	rateLimit := makeRateLimit(resp)
	rl.limits.Store(path, rateLimit)
	cond.Broadcast()
}

// shouldWork determines if rate limiting should be applied to the given URL
func (rl *rateLimiter) shouldWork(url *url.URL) bool {
	return !strings.HasSuffix(url.Host, "twimg.com")
}

// wouldBlock checks if a request to the given path would block due to rate limiting
func (rl *rateLimiter) wouldBlock(path string) bool {
	if v, ok := rl.limits.Load(path); ok {
		return v.(*xRateLimit) != nil && v.(*xRateLimit).safeWouldBlock()
	}
	return false
}
