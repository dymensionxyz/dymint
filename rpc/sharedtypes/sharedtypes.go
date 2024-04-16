package sharedtypes

import "sync"

// TODO: can remove ishealthy state?

// HealthStatus is a struct that holds the health status of the node.
// Should be safe for concurrent access.
type HealthStatus struct {
	IsHealthy bool  `json:"is_healthy"`
	Error     error `json:"error,omitempty"`
	mutex     sync.RWMutex
}

// Set sets the health status of the node. err is nil if the node is healthy.
func (h *HealthStatus) Set(err error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.IsHealthy = err == nil
	h.Error = err
}

// Get returns the health status of the node.
func (h *HealthStatus) Get() (isHealthy bool, err error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	return h.IsHealthy, h.Error
}
