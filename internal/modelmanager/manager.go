package modelmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/yonatanzilberman/lmhub/internal/api"
)

// Manager coordinates the loading, unloading, and auto-switching of models.
type Manager struct {
	client   *api.Client
	registry *Registry
	metrics  *Metrics
}

// NewManager creates a new Manager instance.
func NewManager(client *api.Client, reg *Registry, metrics *Metrics) *Manager {
	return &Manager{
		client:   client,
		registry: reg,
		metrics:  metrics,
	}
}

// Registry returns the underlying model registry.
func (m *Manager) Registry() *Registry {
	return m.registry
}

// Metrics returns the underlying metrics tracker.
func (m *Manager) Metrics() *Metrics {
	return m.metrics
}

// EnsureModel checks the currently loaded model and handles the auto-switch flow:
// - If the correct model is loaded, it does nothing.
// - If a different model is loaded, it unloads it and loads the target model.
// - If no model is loaded, it loads the target model.
// Callback parameters allow sending non-blocking progress status back to the TUI.
func (m *Manager) EnsureModel(ctx context.Context, targetKey string, contextLength int, statusChan chan<- string) error {
	if targetKey == "" {
		return nil // No model pinned, use whatever is loaded
	}

	// 1. Refresh registry to get latest loaded states
	if statusChan != nil {
		statusChan <- "Checking available models..."
	}
	if err := m.registry.Refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh registry: %w", err)
	}

	// 2. Identify if target model is already loaded
	targetModel, exists := m.registry.Get(targetKey)
	if !exists {
		return fmt.Errorf("model %s is not downloaded or available in LM Studio", targetKey)
	}

	isLoaded := len(targetModel.LoadedInstances) > 0
	if isLoaded {
		// Target is loaded, verify if context length is close enough or if we need to reload
		// (For now, if it's loaded we proceed directly to save time)
		if statusChan != nil {
			statusChan <- fmt.Sprintf("Model %s is already loaded.", targetModel.DisplayName)
		}
		
		// Update initial metrics
		m.updateMetricsFromModel(targetModel)
		return nil
	}

	// 3. Unload any other loaded models
	models := m.registry.List()
	for _, model := range models {
		for _, inst := range model.LoadedInstances {
			if statusChan != nil {
				statusChan <- fmt.Sprintf("Unloading model: %s...", model.DisplayName)
			}
			_, err := m.client.UnloadModel(ctx, inst.ID)
			if err != nil {
				return fmt.Errorf("failed to unload model %s: %w", model.DisplayName, err)
			}
			// Small pause to allow server to settle
			time.Sleep(500 * time.Millisecond)
		}
	}

	// 4. Load the target model
	if statusChan != nil {
		statusChan <- fmt.Sprintf("Loading model: %s...", targetModel.DisplayName)
	}
	
	loadResp, err := m.client.LoadModel(ctx, targetKey, contextLength)
	if err != nil {
		return fmt.Errorf("failed to load model %s: %w", targetKey, err)
	}

	if statusChan != nil {
		statusChan <- fmt.Sprintf("Loaded in %.2fs. Warming up...", loadResp.LoadTimeSeconds)
	}

	// Refresh registry again to cache the loaded instance details
	if err := m.registry.Refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh registry post-load: %w", err)
	}

	// Retrieve updated info
	if updatedModel, found := m.registry.Get(targetKey); found {
		m.updateMetricsFromModel(updatedModel)
	} else {
		// Fallback baseline update
		m.metrics.UpdateTelemetry(targetKey, contextLength, 0, float64(targetModel.SizeBytes)/(1024*1024*1024))
	}

	return nil
}

// UnloadAll unloads all models currently loaded in LM Studio.
func (m *Manager) UnloadAll(ctx context.Context) error {
	if err := m.registry.Refresh(ctx); err != nil {
		return err
	}
	models := m.registry.List()
	for _, model := range models {
		for _, inst := range model.LoadedInstances {
			_, err := m.client.UnloadModel(ctx, inst.ID)
			if err != nil {
				return err
			}
		}
	}
	m.metrics.UpdateTelemetry("", 0, 0, 0)
	return nil
}

func (m *Manager) updateMetricsFromModel(model api.ModelInfo) {
	if len(model.LoadedInstances) > 0 {
		inst := model.LoadedInstances[0]
		ctxLen := model.MaxContextLength
		if configCtx, ok := inst.Config["context_length"].(float64); ok {
			ctxLen = int(configCtx)
		}
		ramGB := float64(model.SizeBytes) / (1024 * 1024 * 1024)
		m.metrics.UpdateTelemetry(model.Key, ctxLen, 0, ramGB)
	}
}
