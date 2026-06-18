package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yonatanzilberman/lmhub/internal/api"
	"github.com/yonatanzilberman/lmhub/internal/config"
)

// MemoryManager coordinates memory storage, retrieval, and injection.
type MemoryManager struct {
	projectStore *Store
	globalStore  *Store
	projectPath  string
	cfg          *config.MemoryConfig
	client       *api.Client
}

// NewMemoryManager creates a new MemoryManager instance.
func NewMemoryManager(projectStore *Store, globalStore *Store, projectPath string, cfg *config.MemoryConfig, client *api.Client) *MemoryManager {
	return &MemoryManager{
		projectStore: projectStore,
		globalStore:  globalStore,
		projectPath:  projectPath,
		cfg:          cfg,
		client:       client,
	}
}

// GenerateID creates a simple random hex string for fact IDs.
func GenerateID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp if random fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// AddFact adds a new fact to the appropriate store based on scope.
func (mm *MemoryManager) AddFact(scope string, content string, source string, confidence float64) error {
	if !mm.cfg.Enabled {
		return nil
	}

	fact := &MemoryFact{
		ID:         GenerateID(),
		Scope:      scope,
		Content:    content,
		Source:     source,
		Confidence: confidence,
		CreatedAt:  time.Now(),
		LastUsed:   time.Now(),
		UseCount:   0,
	}

	if scope == "global" {
		// Enforce max global facts
		existing, err := mm.globalStore.ListByScope("global")
		if err == nil && len(existing) >= mm.cfg.MaxFactsGlobal {
			// Evict the oldest/least used fact
			sort.Slice(existing, func(i, j int) bool {
				if existing[i].UseCount == existing[j].UseCount {
					return existing[i].CreatedAt.Before(existing[j].CreatedAt)
				}
				return existing[i].UseCount < existing[j].UseCount
			})
			_ = mm.globalStore.Delete(existing[0].ID)
		}
		return mm.globalStore.Put(fact)
	}

	// Enforce max project facts
	projScope := mm.getProjectScope()
	existing, err := mm.projectStore.ListByScope(projScope)
	if err == nil && len(existing) >= mm.cfg.MaxFactsPerProject {
		// Evict the oldest/least used fact
		sort.Slice(existing, func(i, j int) bool {
			if existing[i].UseCount == existing[j].UseCount {
				return existing[i].CreatedAt.Before(existing[j].CreatedAt)
			}
			return existing[i].UseCount < existing[j].UseCount
		})
		_ = mm.projectStore.Delete(existing[0].ID)
	}
	return mm.projectStore.Put(fact)
}

// ForgetFact deletes a fact from both stores.
func (mm *MemoryManager) ForgetFact(id string) error {
	// Try to delete from global store
	err := mm.globalStore.Delete(id)
	if err != nil {
		return err
	}
	// Try to delete from project store
	return mm.projectStore.Delete(id)
}

// ClearProject deletes all facts associated with the current project scope.
func (mm *MemoryManager) ClearProject() error {
	return mm.projectStore.DeleteByScope(mm.getProjectScope())
}

// ClearGlobal deletes all facts in the global scope.
func (mm *MemoryManager) ClearGlobal() error {
	return mm.globalStore.DeleteByScope("global")
}

// ListFacts returns all facts (project and global) for the current workspace.
func (mm *MemoryManager) ListFacts() ([]*MemoryFact, error) {
	var allFacts []*MemoryFact

	// Load global facts
	globals, err := mm.globalStore.ListByScope("global")
	if err != nil {
		return nil, fmt.Errorf("failed to list global facts: %w", err)
	}
	allFacts = append(allFacts, globals...)

	// Load project facts
	projects, err := mm.projectStore.ListByScope(mm.getProjectScope())
	if err != nil {
		return nil, fmt.Errorf("failed to list project facts: %w", err)
	}
	allFacts = append(allFacts, projects...)

	// Sort facts by CreatedAt descending by default for display listing
	sort.Slice(allFacts, func(i, j int) bool {
		return allFacts[i].CreatedAt.After(allFacts[j].CreatedAt)
	})

	return allFacts, nil
}

// getProjectScope constructs the scope identifier for the current project.
func (mm *MemoryManager) getProjectScope() string {
	return fmt.Sprintf("project:%s", mm.projectPath)
}

// InjectFacts retrieves relevant facts and returns them formatted as a newline-separated string.
// It also increments the UseCount and updates LastUsed for injected facts.
func (mm *MemoryManager) InjectFacts() string {
	if !mm.cfg.Enabled {
		return ""
	}

	facts, err := mm.ListFacts()
	if err != nil || len(facts) == 0 {
		return ""
	}

	// Sort facts by UseCount ascending (least important first, most important last)
	// so that if budget manager truncates from the start, the least used ones are dropped first.
	sort.Slice(facts, func(i, j int) bool {
		if facts[i].UseCount == facts[j].UseCount {
			return facts[i].CreatedAt.Before(facts[j].CreatedAt)
		}
		return facts[i].UseCount < facts[j].UseCount
	})

	var formatted []string
	for _, f := range facts {
		// Format: "- [Source] Content"
		sourceBadge := f.Source
		if f.Source == "extracted" {
			sourceBadge = "auto"
		}
		formatted = append(formatted, fmt.Sprintf("- [%s] %s", sourceBadge, f.Content))

		// Update usage statistics
		f.UseCount++
		f.LastUsed = time.Now()
		if f.Scope == "global" {
			_ = mm.globalStore.Put(f)
		} else {
			_ = mm.projectStore.Put(f)
		}
	}

	return strings.Join(formatted, "\n")
}

// ExtractAndStore parses facts from conversation history and stores those above the threshold.
func (mm *MemoryManager) ExtractAndStore(ctx context.Context, modelID string, history []api.Message) error {
	if !mm.cfg.Enabled || !mm.cfg.AutoExtract || mm.client == nil || len(history) == 0 {
		return nil
	}

	ext := NewExtractor(mm.client)
	facts, err := ext.ExtractFacts(ctx, modelID, history)
	if err != nil {
		return fmt.Errorf("failed to extract facts: %w", err)
	}

	projScope := mm.getProjectScope()
	for _, f := range facts {
		if f.Confidence >= mm.cfg.AutoExtractThreshold {
			_ = mm.AddFact(projScope, f.Content, "extracted", f.Confidence)
		}
	}

	return nil
}
