package main

import (
	"sync"
	"sync/atomic"

	"golang.org/x/net/context"
	"golang.org/x/time/rate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/tap"
)

type Metrics struct {
	calls uint64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) Calls() uint64 {
	return atomic.LoadUint64(&m.calls)
}

func (m *Metrics) IncrementCalls(delta uint64) uint64 {
	return atomic.AddUint64(&m.calls, delta)
}

type TapHandler struct {
	tap.ServerInHandle

	metrics *Metrics

	ratesMu sync.RWMutex
	rates   map[string]*rate.Limiter
	qps     rate.Limit
	burst   int
}

func NewTapHandler(metrics *Metrics, qps rate.Limit, burst int) *TapHandler {
	return &TapHandler{
		metrics: metrics,
		rates:   make(map[string]*rate.Limiter),
		qps:     qps,
		burst:   burst,
	}
}

func (h *TapHandler) Handle(ctx context.Context, info *tap.Info) (context.Context, error) {
	h.metrics.IncrementCalls(1)

	// Rate limiter per user
	user, ok := getUserFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated,
			"client didn't pass a user")
	}

	h.ratesMu.Lock()
	if h.rates[user] == nil {
		h.rates[user] = rate.NewLimiter(h.qps, h.burst) // QPS, burst
	}
	if !h.rates[user].Allow() {
		h.ratesMu.Unlock()
		return nil, status.Error(codes.ResourceExhausted,
			"client exceeded rate limit")
	}
	h.ratesMu.Unlock()

	return ctx, nil
}
