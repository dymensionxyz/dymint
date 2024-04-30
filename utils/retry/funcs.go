package retry

import (
	"time"
)

const (
	defaultInitialDelay = 200 * time.Millisecond
	defaultMaxDelay     = 30 * time.Second
	defaultFactor       = 2
)

type Backoff struct {
	delay        time.Duration
	maxDelay     time.Duration
	growthFactor float64
}

type Option func(*Backoff)

func WithInitialDelay(d time.Duration) Option {
	return func(b *Backoff) {
		b.delay = d
	}
}

func WithMaxDelay(d time.Duration) Option {
	return func(b *Backoff) {
		b.maxDelay = d
	}
}

func WithGrowthFactor(x float64) Option {
	return func(b *Backoff) {
		b.growthFactor = x
	}
}

func NewBackoff(opts ...Option) *Backoff {
	ret := &Backoff{
		delay:        defaultInitialDelay,
		maxDelay:     defaultMaxDelay,
		growthFactor: defaultFactor,
	}
	for _, o := range opts {
		o(ret)
	}
	return ret
}

func (b *Backoff) Sleep() {
	time.Sleep(b.delay)
	b.delay *= time.Duration(b.growthFactor)
	b.delay = min(b.delay, b.maxDelay)
}
