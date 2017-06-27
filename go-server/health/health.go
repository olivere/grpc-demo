package health

import (
	"net/http"
	"sync"
)

var (
	mu              sync.RWMutex
	healthzStatus   = http.StatusOK
	readinessStatus = http.StatusOK
)

// HealthzStatus returns the current health status as a HTTP status code.
func HealthzStatus() int {
	mu.RLock()
	defer mu.RUnlock()
	return healthzStatus
}

// ReadinessStatus returns the current readiness status as a HTTP status code.
func ReadinessStatus() int {
	mu.RLock()
	defer mu.RUnlock()
	return readinessStatus
}

// SetHealtzStatus sets the health status as a HTTP status code.
func SetHealtzStatus(status int) {
	mu.Lock()
	healthzStatus = status
	mu.Unlock()
}

// SetReadinessStatus sets the readiness status as a HTTP status code.
func SetReadinessStatus(status int) {
	mu.Lock()
	readinessStatus = status
	mu.Unlock()
}

// HealthzHandler returns the current health status.
func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(HealthzStatus())
}

// ReadinessHandler returns the current readiness status.
func ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(ReadinessStatus())
}

// ToggleHealthzStatusHandler toggles the current health status between
// http.StatusOK and http.StatusServiceUnavailable.
func ToggleHealthzStatusHandler(w http.ResponseWriter, r *http.Request) {
	switch HealthzStatus() {
	case http.StatusOK:
		SetHealtzStatus(http.StatusServiceUnavailable)
	case http.StatusServiceUnavailable:
		SetHealtzStatus(http.StatusOK)
	}
	w.WriteHeader(http.StatusOK)
}
