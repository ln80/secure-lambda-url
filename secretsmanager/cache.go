package secretsmanager

import (
	"context"
	"sync"
	"time"
)

type BlackList map[string]struct{}

type secret struct {
	value     string
	createdAt time.Time
}

func (s secret) IsZero() bool {
	return s.value == "" && s.createdAt.IsZero()
}

var (
	zeroSecret = secret{}
)

type Janitor struct {
	current, previous, pending secret
	cacheMu                    sync.RWMutex

	bl BlackList

	interval time.Duration
	done     chan struct{}
	once     sync.Once
}

func NewJanitor(interval time.Duration) *Janitor {
	return &Janitor{
		interval: interval,
		done:     make(chan struct{}),
		bl:       make(BlackList),
	}
}

func (j *Janitor) Run(ctx context.Context, onCleanup func()) {
	cleanup := func() {
		j.setCache(zeroSecret, zeroSecret, zeroSecret)
		j.clearBlackList()
		if onCleanup != nil {
			onCleanup()
		}
	}

	go func() {
		ticker := time.NewTicker(j.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cleanup()
			case <-j.done:
				cleanup()
				return
			case <-ctx.Done():
				j.stop()
			}
		}
	}()
}

func (j *Janitor) getCache() (cur, prev, pen secret, found bool) {
	j.cacheMu.Lock()
	defer j.cacheMu.Unlock()

	cur, prev, pen, found = j.current, j.previous, j.pending, !j.current.IsZero()

	return
}

func (j *Janitor) setCache(cur, prev, pen secret) {
	j.cacheMu.Lock()
	defer j.cacheMu.Unlock()

	j.current, j.previous, j.pending = cur, prev, pen
}

func (j *Janitor) blackList(val string) {
	j.cacheMu.Lock()
	defer j.cacheMu.Unlock()

	j.bl[val] = struct{}{}
}

func (j *Janitor) isBlackListed(val string) bool {
	j.cacheMu.Lock()
	defer j.cacheMu.Unlock()

	_, ok := j.bl[val]
	return ok
}

func (j *Janitor) clearBlackList() {
	j.cacheMu.Lock()
	defer j.cacheMu.Unlock()

	j.bl = make(BlackList)
}

func (j *Janitor) stop() {
	j.once.Do(func() { close(j.done) })
}
