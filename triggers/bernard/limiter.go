package bernard

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
	"io/ioutil"
	"sync"
	"time"
)

type rateLimiter struct {
	ctx         context.Context
	concurrency int
	rl          *rate.Limiter
	sem         *semaphore.Weighted
}

func (r *rateLimiter) Wait() {
	_ = r.rl.Wait(r.ctx)
}

func (r *rateLimiter) Semaphore() *semaphore.Weighted {
	return r.sem
}

func newRateLimiter() *rateLimiter {
	concurrency := 5

	return &rateLimiter{
		ctx:         context.Background(),
		concurrency: concurrency,
		rl:          rate.NewLimiter(rate.Every(time.Second/time.Duration(concurrency)), concurrency),
		sem:         semaphore.NewWeighted(int64(concurrency)),
	}
}

var (
	accounts = make(map[string]string)
	limiters = make(map[string]*rateLimiter)
	lock     = &sync.Mutex{}
)

func getRateLimiter(account string) (*rateLimiter, error) {
	lock.Lock()
	defer lock.Unlock()

	// lookup project id for the account
	project, ok := accounts[account]
	if !ok {
		// parse account project
		b, err := ioutil.ReadFile(account)
		if err != nil {
			return nil, fmt.Errorf("failed reading account: %v: %w", account, err)
		}

		// decode account
		acct := new(struct {
			ProjectID string `json:"project_id"`
		})

		if err := json.Unmarshal(b, acct); err != nil {
			return nil, fmt.Errorf("failed decoding account: %v: %w", account, err)
		}

		project = acct.ProjectID
		accounts[account] = project
	}

	// return existing limiter for the account
	if limiter, ok := limiters[project]; ok {
		return limiter, nil
	}

	// add limiter to map
	limiter := newRateLimiter()
	limiters[project] = limiter

	return limiter, nil
}
