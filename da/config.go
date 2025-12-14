package da

import (
	"time"

	uretry "github.com/dymensionxyz/dymint/utils/retry"
)

const (
	// DefaultRetryDelay is the default delay between retry attempts
	DefaultRetryDelay = 3 * time.Second
	// DefaultRetryAttempts is the default number of retry attempts
	DefaultRetryAttempts = 5
	// DefaultTimeout is the default request timeout
	DefaultTimeout = 30 * time.Second
)

// DefaultSubmitBackoff is the default backoff configuration for DA submissions
var DefaultSubmitBackoff = uretry.NewBackoffConfig(
	uretry.WithInitialDelay(time.Second*6),
	uretry.WithMaxDelay(time.Second*6),
)

// BaseConfig contains common configuration fields for all DA clients.
// Embed this struct in DA-specific configs to get standard retry/timeout behavior.
type BaseConfig struct {
	Backoff       uretry.BackoffConfig `json:"backoff,omitempty"`
	RetryAttempts *int                 `json:"retry_attempts,omitempty"`
	RetryDelay    time.Duration        `json:"retry_delay,omitempty"`
	Timeout       time.Duration        `json:"timeout,omitempty"`
}

// SetDefaults sets default values for unset fields
func (c *BaseConfig) SetDefaults() {
	if c.RetryDelay == 0 {
		c.RetryDelay = DefaultRetryDelay
	}
	if c.Backoff == (uretry.BackoffConfig{}) {
		c.Backoff = DefaultSubmitBackoff
	}
	if c.RetryAttempts == nil {
		attempts := DefaultRetryAttempts
		c.RetryAttempts = &attempts
	}
	if c.Timeout == 0 {
		c.Timeout = DefaultTimeout
	}
}

// GetRetryAttempts returns retry attempts with a safe default
func (c *BaseConfig) GetRetryAttempts() int {
	if c.RetryAttempts == nil {
		return DefaultRetryAttempts
	}
	return *c.RetryAttempts
}
