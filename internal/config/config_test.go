package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Verify standard defaults are populated
	assert.Equal(t, "http://localhost:1234", cfg.LMStudio.BaseURL)
	assert.Equal(t, 120, cfg.LMStudio.TimeoutSeconds)
	assert.True(t, cfg.LMStudio.Stream)
	assert.Equal(t, "qwen/qwen3.6-35b-a3b", cfg.ModeModels.Build)
	assert.True(t, cfg.RAG.Enabled)
	assert.Equal(t, 15, cfg.Agent.MaxIterations)
}
