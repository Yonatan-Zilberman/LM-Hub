package modelmanager

import (
	"context"
	"time"

	"github.com/yonatanzilberman/lmhub/internal/api"
)

// Watcher periodically polls the loaded model's telemetry from LM Studio.
type Watcher struct {
	client   *api.Client
	metrics  *Metrics
	interval time.Duration
}

// NewWatcher creates a new Watcher instance.
func NewWatcher(client *api.Client, metrics *Metrics, intervalMs int) *Watcher {
	return &Watcher{
		client:   client,
		metrics:  metrics,
		interval: time.Duration(intervalMs) * time.Millisecond,
	}
}

// Start runs the telemetry polling loop in the background.
// It stops when the context is cancelled.
func (w *Watcher) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

func (w *Watcher) poll(ctx context.Context) {
	info, err := w.client.GetLoadedModelInfo(ctx)
	if err != nil {
		// Suppress or log error. We don't crash to satisfy offline-first graceful degradation
		return
	}

	if info == nil {
		// No model loaded
		w.metrics.UpdateTelemetry("", 0, 0, 0.0)
		return
	}

	// Update telemetry. Note: tokens_used is tracked client-side,
	// but we can preserve the current metrics value of tokens_used
	// by reading it first.
	currentMetrics := w.metrics.Get()
	
	w.metrics.UpdateTelemetry(
		info.ModelID,
		info.ContextLength,
		currentMetrics.TokensUsed, // Maintain current client-side token count
		info.RAMUsedGB,
	)
}
