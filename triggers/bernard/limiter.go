package bernard

import (
	"context"
	"golang.org/x/time/rate"
	"sync"
	"time"
)

type rateLimiter struct {
	ctx context.Context
	rl  *rate.Limiter
}

func (r *rateLimiter) Wait() {
	_ = r.rl.Wait(r.ctx)
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		ctx: context.Background(),
		rl:  rate.NewLimiter(rate.Every(time.Second/5), 5),
	}
}

var (
	limiters = make(map[string]*rateLimiter)
	lock     = &sync.Mutex{}
)

func getRateLimiter(account string) *rateLimiter {
	lock.Lock()
	defer lock.Unlock()

	// return existing limiter for the account
	if limiter, ok := limiters[account]; ok {
		return limiter
	}

	// add limiter to map
	limiter := newRateLimiter()
	limiters[account] = limiter

	return limiter
}
