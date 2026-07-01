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
	// mode_models are all empty by default — use whatever model is loaded in LM Studio
	assert.Equal(t, "", cfg.ModeModels.Ask)
	assert.Equal(t, "", cfg.ModeModels.Plan)
	assert.Equal(t, "", cfg.ModeModels.Build)
	assert.True(t, cfg.RAG.Enabled)
	assert.Equal(t, 15, cfg.Agent.MaxIterations)
}

func TestDefaultConfigNoMaxTokens(t *testing.T) {
	cfg := DefaultConfig()

	// max_tokens is removed from InferenceConfig — context size comes from LM Studio
	assert.Equal(t, 0.7, cfg.Inference.Temperature)
	assert.Equal(t, 0.95, cfg.Inference.TopP)
}
