package twitterclient

import (
	"context"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WangWilly/xSync/pkgs/commonpkg/utils"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

////////////////////////////////////////////////////////////////////////////////

// rateLimitManager manages rate limiting for multiple API endpoints
type rateLimitManager struct {
	pathLimitsMap *utils.SyncMap[string, *xRateLimit]
	pathCondsMap  *utils.SyncMap[string, *sync.Cond]
	nonBlocking   bool
}

// newRateLimiter creates a new rate limiter
func newRateLimiter(nonBlocking bool) *rateLimitManager {
	return &rateLimitManager{
		pathLimitsMap: utils.NewSyncMap[string, *xRateLimit](),
		pathCondsMap:  utils.NewSyncMap[string, *sync.Cond](),
		nonBlocking:   nonBlocking,
	}
}

////////////////////////////////////////////////////////////////////////////////

// check verifies if a request can proceed without hitting rate limits
func (rlMgr *rateLimitManager) check(ctx context.Context, url *url.URL) error {
	if !rlMgr.shouldWork(url) {
		return nil
	}

	path := url.Path
	pathCond, _ := rlMgr.pathCondsMap.LoadOrStore(path, sync.NewCond(&sync.Mutex{}))
	pathCond.L.Lock()
	defer pathCond.L.Unlock()

	pathLimit, loaded := rlMgr.pathLimitsMap.LoadOrStore(path, &xRateLimit{})
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
	for pathLimit != nil && !pathLimit.Ready {
		pathCond.Wait()
		pathLimit, loaded = rlMgr.pathLimitsMap.LoadOrStore(path, &xRateLimit{})
		if !loaded {
			// 上个请求失败了，从它身上继承初始化速率限制的重任
			return nil
		}
	}

	// limiter 为 nil 意味着不对此路径做速率限制
	if pathLimit != nil {
		return pathLimit.safePreRequest(ctx, rlMgr.nonBlocking)
	}
	return nil
}

// reset resets the rate limit information after a request
func (rlMgr *rateLimitManager) reset(url *url.URL, resp *resty.Response) {
	if !rlMgr.shouldWork(url) {
		return
	}

	path := url.Path
	pathCond, ok := rlMgr.pathCondsMap.Load(path)
	if !ok {
		return // BeforeRequest 从未调用的情况下调用了 OnError/OnRetry
	}
	pathCond.L.Lock()
	defer pathCond.L.Unlock()

	pathLimit, ok := rlMgr.pathLimitsMap.Load(path)
	if !ok {
		return
	}
	if pathLimit == nil || pathLimit.Ready {
		return
	}

	if resp == nil || resp.RawResponse == nil {
		// 将此路径设为首次请求前的状态
		rlMgr.pathLimitsMap.Delete(path)
		pathCond.Signal()
		return
	}

	// 请求成功，或发生了错误/触发了重试条件但有能力更新速率限制
	rateLimit := newRateLimit(resp)
	rlMgr.pathLimitsMap.Store(path, rateLimit)
	pathCond.Broadcast()
}

// shouldWork determines if rate limiting should be applied to the given URL
func (rlMgr *rateLimitManager) shouldWork(url *url.URL) bool {
	return !strings.HasSuffix(url.Host, "twimg.com")
}

// wouldBlock checks if a request to the given path would block due to rate limiting
func (rlMgr *rateLimitManager) wouldBlock(path string) bool {
	if pathLimit, ok := rlMgr.pathLimitsMap.Load(path); ok && pathLimit != nil {
		return pathLimit.safeWouldBlock()
	}
	return false
}

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

// newRateLimit creates a rate limit from HTTP response headers
func newRateLimit(resp *resty.Response) *xRateLimit {
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

////////////////////////////////////////////////////////////////////////////////

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

// wouldBlock checks if making a request would trigger rate limiting (internal, not thread-safe)
func (rl *xRateLimit) wouldBlock() bool {
	threshold := max(2*rl.Limit/100, 1)
	return rl.Remaining <= threshold && time.Now().Before(rl.ResetTime)
}
