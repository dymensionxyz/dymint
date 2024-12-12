package retry

import (
	"time"
)

const (
	defaultBackoffInitialDelay = 200 * time.Millisecond
	defaultBackoffMaxDelay     = 30 * time.Second
	defaultBackoffFactor       = 2
)


type BackoffConfig struct {
	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay     time.Duration `json:"max_delay"`
	GrowthFactor float64       `json:"growth_factor"`
}


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



func WithMaxDelay(d time.Duration) BackoffOption {
	return func(b *BackoffConfig) {
		b.MaxDelay = d
	}
}



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


func (b *Backoff) Delay() time.Duration {
	ret := b.delay
	b.delay = time.Duration(float64(b.delay) * b.growthFactor)
	if b.maxDelay != 0 {
		b.delay = min(b.delay, b.maxDelay)
	}
	return ret
}


func (b *Backoff) Sleep() {
	time.Sleep(b.Delay())
}
