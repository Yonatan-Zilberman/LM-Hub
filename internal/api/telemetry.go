package api

import (
	"context"
	"fmt"
)

// GetLoadedModelInfo queries the available models and constructs telemetry information
// for the currently loaded model instance (if any).
func (c *Client) GetLoadedModelInfo(ctx context.Context) (*LoadedModelInfo, error) {
	models, err := c.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get loaded model info: %w", err)
	}

	for _, m := range models {
		if len(m.LoadedInstances) > 0 {
			inst := m.LoadedInstances[0]
			ctxLength := m.MaxContextLength
			
			// Try to get configured context length if available
			if config, ok := inst.Config["context_length"].(float64); ok {
				ctxLength = int(config)
			}

			ramUsed := float64(m.SizeBytes) / (1024 * 1024 * 1024)

			return &LoadedModelInfo{
				ModelID:       m.Key,
				ContextLength: ctxLength,
				RAMUsedGB:     ramUsed,
			}, nil
		}
	}

	return nil, nil // No loaded model
}
