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

// WithMaxDelay sets the maximum delay for the backoff. The delay will not exceed this value.
// Set 0 to disable the maximum delay.
func WithMaxDelay(d time.Duration) Option {
	return func(b *Backoff) {
		b.maxDelay = d
	}
}

// WithGrowthFactor sets the growth factor for the backoff. The delay will be multiplied by this factor on each call to Delay.
// The factor should be greater than 1.0
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

// Delay returns the current delay. The subsequent delay will be increased by the growth factor up to the maximum.
func (b *Backoff) Delay() time.Duration {
	ret := b.delay
	b.delay = time.Duration(float64(b.delay) * b.growthFactor)
	if b.maxDelay != 0 {
		b.delay = min(b.delay, b.maxDelay)
	}
	return ret
}

// Sleep sleeps for the current delay. The subsequent delay will be increased by the growth factor up to the maximum.
func (b *Backoff) Sleep() {
	time.Sleep(b.Delay())
}
