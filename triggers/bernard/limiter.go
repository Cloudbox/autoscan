package bernard

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

const (
	// how many requests can be sent per second by all drives using the same account file
	requestLimit = 8
	// how many drives can run at once (at the trigger level), e.g. 2 triggers, with 5 drives each.
	syncLimit = 5
)

type rateLimiter struct {
	ctx context.Context
	rl  *rate.Limiter
	sem *semaphore.Weighted
}

func (r *rateLimiter) Wait() {
	_ = r.rl.Wait(r.ctx)
}

func (r *rateLimiter) Acquire(n int64) error {
	return r.sem.Acquire(r.ctx, n)
}

func (r *rateLimiter) Release(n int64) {
	r.sem.Release(n)
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		ctx: context.Background(),
		rl:  rate.NewLimiter(rate.Every(time.Second/time.Duration(requestLimit)), requestLimit),
		sem: semaphore.NewWeighted(int64(syncLimit)),
	}
}

var (
	limiters = make(map[string]*rateLimiter)
	lock     = &sync.Mutex{}
)

func getRateLimiter(account string) (*rateLimiter, error) {
	lock.Lock()
	defer lock.Unlock()

	// return existing limiter for the account
	if limiter, ok := limiters[account]; ok {
		return limiter, nil
	}

	// add limiter to map
	limiter := newRateLimiter()
	limiters[account] = limiter

	return limiter, nil
}
