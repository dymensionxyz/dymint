package retry

import (
	"time"
)

// TODO: rethink this package. can probably use avast/retry-go instead

const (
	defaultBackoffInitialDelay = 200 * time.Millisecond
	defaultBackoffMaxDelay     = 30 * time.Second
	defaultBackoffFactor       = 2
)

// BackoffConfig is a configuration for a backoff, it's used to create new instances
type BackoffConfig struct {
	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay     time.Duration `json:"max_delay"`
	GrowthFactor float64       `json:"growth_factor"`
}

// Backoff creates a new Backoff instance with the configuration (starting at 0 attempts made so far)
func (c BackoffConfig) Backoff() Backoff {
	return Backoff{
		delay:        c.InitialDelay,
		maxDelay:     c.MaxDelay,
		growthFactor: c.GrowthFactor,
	}
}

type Backoff struct {
	delay        time.Duration
	maxDelay     time.Duration
	growthFactor float64
}

type BackoffOption func(*BackoffConfig)

func WithInitialDelay(d time.Duration) BackoffOption {
	return func(b *BackoffConfig) {
		b.InitialDelay = d
	}
}

// WithMaxDelay sets the maximum delay for the backoff. The delay will not exceed this value.
// Set 0 to disable the maximum delay.
func WithMaxDelay(d time.Duration) BackoffOption {
	return func(b *BackoffConfig) {
		b.MaxDelay = d
	}
}

// WithGrowthFactor sets the growth factor for the backoff. The delay will be multiplied by this factor on each call to Delay.
// The factor should be greater than 1.0
func WithGrowthFactor(x float64) BackoffOption {
	return func(b *BackoffConfig) {
		b.GrowthFactor = x
	}
}

func NewBackoffConfig(opts ...BackoffOption) BackoffConfig {
	ret := BackoffConfig{
		InitialDelay: defaultBackoffInitialDelay,
		MaxDelay:     defaultBackoffMaxDelay,
		GrowthFactor: defaultBackoffFactor,
	}
	for _, o := range opts {
		o(&ret)
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
