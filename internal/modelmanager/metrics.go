package modelmanager

import "sync"

// Metrics tracks the real-time inference metrics of the loaded model.
type Metrics struct {
	mu              sync.RWMutex
	ModelID         string
	TokensPerSecond float64
	TTFTMs          int
	TotalTokens     int
	TokensUsed      int
	ContextLimit    int
	RAMUsedGB       float64
}

// Get retrieves a copy of current metrics thread-safely.
func (m *Metrics) Get() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Metrics{
		ModelID:         m.ModelID,
		TokensPerSecond: m.TokensPerSecond,
		TTFTMs:          m.TTFTMs,
		TotalTokens:     m.TotalTokens,
		TokensUsed:      m.TokensUsed,
		ContextLimit:    m.ContextLimit,
		RAMUsedGB:       m.RAMUsedGB,
	}
}

// UpdateFromCompletion updates stream-oriented metrics.
func (m *Metrics) UpdateFromCompletion(tokSec float64, ttftMs int, generatedTokens int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TokensPerSecond = tokSec
	m.TTFTMs = ttftMs
	m.TotalTokens += generatedTokens
}

// UpdateTelemetry updates telemetry-oriented metrics.
func (m *Metrics) UpdateTelemetry(modelID string, ctxLimit int, tokensUsed int, ramGB float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ModelID = modelID
	m.ContextLimit = ctxLimit
	m.TokensUsed = tokensUsed
	m.RAMUsedGB = ramGB
}
