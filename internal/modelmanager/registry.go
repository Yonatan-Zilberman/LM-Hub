package modelmanager

import (
	"context"
	"fmt"
	"sync"

	"github.com/yonatanzilberman/lmhub/internal/api"
)

// Registry manages the cached list of models available from LM Studio.
type Registry struct {
	mu     sync.RWMutex
	client *api.Client
	models []api.ModelInfo
}

// NewRegistry creates a new Registry instance.
func NewRegistry(client *api.Client) *Registry {
	return &Registry{
		client: client,
		models: make([]api.ModelInfo, 0),
	}
}

// Refresh fetches the latest list of models from LM Studio and updates the cache.
func (r *Registry) Refresh(ctx context.Context) error {
	models, err := r.client.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh registry: %w", err)
	}

	r.mu.Lock()
	r.models = models
	r.mu.Unlock()
	return nil
}

// List returns a copy of the currently cached models.
func (r *Registry) List() []api.ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copied := make([]api.ModelInfo, len(r.models))
	copy(copied, r.models)
	return copied
}

// Get finds a model by its key/identifier.
func (r *Registry) Get(key string) (api.ModelInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.models {
		if m.Key == key {
			return m, true
		}
	}
	return api.ModelInfo{}, false
}
