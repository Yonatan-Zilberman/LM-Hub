package memory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
	"github.com/yonatanzilberman/lmhub/internal/api"
	"github.com/yonatanzilberman/lmhub/internal/config"
)

func setupTestDBs(t *testing.T) (*Store, *Store) {
	tempDir := t.TempDir()

	projDBPath := filepath.Join(tempDir, "proj_memory.db")
	projDB, err := bbolt.Open(projDBPath, 0600, nil)
	require.NoError(t, err)
	t.Cleanup(func() { projDB.Close() })

	globalDBPath := filepath.Join(tempDir, "global_memory.db")
	globalDB, err := bbolt.Open(globalDBPath, 0600, nil)
	require.NoError(t, err)
	t.Cleanup(func() { globalDB.Close() })

	projStore, err := NewStore(projDB)
	require.NoError(t, err)

	globalStore, err := NewStore(globalDB)
	require.NoError(t, err)

	return projStore, globalStore
}

func TestStore_PutGetDelete(t *testing.T) {
	projStore, _ := setupTestDBs(t)

	fact := &MemoryFact{
		ID:         "test-id",
		Scope:      "project:/path",
		Content:    "Postgres runs on port 5433",
		Source:     "user",
		Confidence: 1.0,
		CreatedAt:  time.Now(),
		LastUsed:   time.Now(),
		UseCount:   1,
	}

	// Test Put
	err := projStore.Put(fact)
	require.NoError(t, err)

	// Test Get
	retrieved, err := projStore.Get("test-id")
	require.NoError(t, err)
	assert.Equal(t, fact.Content, retrieved.Content)
	assert.Equal(t, fact.Scope, retrieved.Scope)

	// Test ListByScope
	list, err := projStore.ListByScope("project:/path")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "test-id", list[0].ID)

	// Test Delete
	err = projStore.Delete("test-id")
	require.NoError(t, err)

	_, err = projStore.Get("test-id")
	assert.Error(t, err)
}

func TestMemoryManager_EvictionLimits(t *testing.T) {
	projStore, globalStore := setupTestDBs(t)

	cfg := &config.MemoryConfig{
		Enabled:              true,
		AutoExtract:          true,
		AutoExtractThreshold: 0.8,
		MaxFactsPerProject:   3, // small limit to test eviction
		MaxFactsGlobal:       2,
	}

	mm := NewMemoryManager(projStore, globalStore, "/path/to/project", cfg, nil)

	// Add 4 project facts (max 3)
	require.NoError(t, mm.AddFact("project:/path/to/project", "fact 1", "user", 1.0))
	time.Sleep(10 * time.Millisecond) // ensure distinct timestamps
	require.NoError(t, mm.AddFact("project:/path/to/project", "fact 2", "user", 1.0))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, mm.AddFact("project:/path/to/project", "fact 3", "user", 1.0))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, mm.AddFact("project:/path/to/project", "fact 4", "user", 1.0))

	// Verify one fact was evicted
	facts, err := mm.ListFacts()
	require.NoError(t, err)
	assert.NotEmpty(t, facts)
	
	// We have 4 added, 1 evicted, so total remaining project facts = 3
	projFacts, err := projStore.ListByScope(mm.getProjectScope())
	require.NoError(t, err)
	assert.Len(t, projFacts, 3)

	// Fact 1 should be evicted since it was oldest and has UseCount 0
	foundFact1 := false
	for _, f := range projFacts {
		if f.Content == "fact 1" {
			foundFact1 = true
		}
	}
	assert.False(t, foundFact1, "Oldest fact should have been evicted")

	// Add 3 global facts (max 2)
	require.NoError(t, mm.AddFact("global", "global 1", "user", 1.0))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, mm.AddFact("global", "global 2", "user", 1.0))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, mm.AddFact("global", "global 3", "user", 1.0))

	globalFacts, err := globalStore.ListByScope("global")
	require.NoError(t, err)
	assert.Len(t, globalFacts, 2)
	
	foundGlobal1 := false
	for _, f := range globalFacts {
		if f.Content == "global 1" {
			foundGlobal1 = true
		}
	}
	assert.False(t, foundGlobal1, "Oldest global fact should have been evicted")
}

func TestMemoryManager_InjectFacts(t *testing.T) {
	projStore, globalStore := setupTestDBs(t)

	cfg := &config.MemoryConfig{
		Enabled:              true,
		MaxFactsPerProject:   10,
		MaxFactsGlobal:       10,
	}

	mm := NewMemoryManager(projStore, globalStore, "/path/to/project", cfg, nil)

	// Add facts
	require.NoError(t, mm.AddFact("project:/path/to/project", "proj fact", "user", 1.0))
	require.NoError(t, mm.AddFact("global", "global fact", "extracted", 0.9))

	injected := mm.InjectFacts()
	assert.Contains(t, injected, "- [user] proj fact")
	assert.Contains(t, injected, "- [auto] global fact")

	// Verify UseCount incremented
	facts, err := mm.ListFacts()
	require.NoError(t, err)
	for _, f := range facts {
		assert.Equal(t, 1, f.UseCount)
	}
}

func TestExtractor_parseExtractedFacts(t *testing.T) {
	// Fenced JSON
	fenced := "```json\n[\n  {\"content\": \"Using Goose for migrations\", \"confidence\": 0.95}\n]\n```"
	facts, err := parseExtractedFacts(fenced)
	require.NoError(t, err)
	require.Len(t, facts, 1)
	assert.Equal(t, "Using Goose for migrations", facts[0].Content)
	assert.Equal(t, 0.95, facts[0].Confidence)

	// Plain JSON array
	plain := "[\n  {\"content\": \"Postgres port is 5433\", \"confidence\": 0.9}\n]"
	facts, err = parseExtractedFacts(plain)
	require.NoError(t, err)
	require.Len(t, facts, 1)
	assert.Equal(t, "Postgres port is 5433", facts[0].Content)

	// JSON surrounded by prose
	prose := "Here is the extracted fact:\n[\n  {\"content\": \"Redis runs locally\", \"confidence\": 0.8}\n]\nHope this helps!"
	facts, err = parseExtractedFacts(prose)
	require.NoError(t, err)
	require.Len(t, facts, 1)
	assert.Equal(t, "Redis runs locally", facts[0].Content)
}

func TestExtractor_ExtractFacts_LiveMock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/v1/chat/completions", req.URL.Path)
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{
			"choices": [
				{
					"message": {
						"role": "assistant",
						"content": "[{\"content\": \"User uses custom port 9999\", \"confidence\": 0.98}]"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client := api.NewClient(server.URL, 5)
	extractor := NewExtractor(client)

	history := []api.Message{
		{Role: "user", Content: "Make sure database runs on port 9999"},
		{Role: "assistant", Content: "Okay, I will configure database on port 9999"},
	}

	facts, err := extractor.ExtractFacts(context.Background(), "test-model", history)
	require.NoError(t, err)
	require.Len(t, facts, 1)
	assert.Equal(t, "User uses custom port 9999", facts[0].Content)
	assert.Equal(t, 0.98, facts[0].Confidence)
}
