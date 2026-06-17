package agent

import (
	"strings"
	"testing"

	"github.com/yonatanzilberman/lmhub/internal/config"
)

func TestBudgetManager_Allocate(t *testing.T) {
	cm, err := NewContextManager()
	if err != nil {
		t.Fatalf("failed to create context manager: %v", err)
	}

	cfg := &config.ContextBudgetConfig{
		ProjectContextMaxTokens: 20,
		MemoryMaxTokens:         20,
		RAGMaxTokens:            30,
		TotalMaxTokens:          50,
	}

	bm := NewBudgetManager(cm, cfg)

	// Case 1: All under budget
	projCtx := "Project settings: Go 1.21"           // ~5 tokens
	memory := "Fact: uses resty\nFact: uses bubbletea" // ~10 tokens
	rag := "RAG: SSE implementation details\n\nChunk 2 details" // ~12 tokens

	alloc := bm.Allocate(projCtx, memory, rag)
	if alloc.TotalInjected > cfg.TotalMaxTokens {
		t.Errorf("expected total tokens (%d) <= max (%d)", alloc.TotalInjected, cfg.TotalMaxTokens)
	}
	if alloc.ProjectContext != projCtx {
		t.Errorf("expected project context to remain intact, got: %s", alloc.ProjectContext)
	}
	if alloc.MemoryFacts != "Fact: uses resty\nFact: uses bubbletea" {
		t.Errorf("expected memory to remain intact, got: %s", alloc.MemoryFacts)
	}

	// Case 2: RAG over budget individually
	ragLong := strings.Repeat("Long RAG content chunk item data details. ", 15) // ~120 tokens
	alloc2 := bm.Allocate(projCtx, memory, ragLong)
	if alloc2.RAGTokens > cfg.RAGMaxTokens {
		t.Errorf("expected RAG tokens (%d) <= max RAG budget (%d)", alloc2.RAGTokens, cfg.RAGMaxTokens)
	}

	// Case 3: All over combined cap (TotalMaxTokens = 50)
	// Priority trimming: Project context -> Memory -> RAG
	projCtxLong := strings.Repeat("Project context description block data here. ", 5) // ~30 tokens -> will be truncated to 20
	memoryLong := "Fact 1: info\nFact 2: details\nFact 3: metadata\nFact 4: extra\nFact 5: more\nFact 6: database" // ~35 tokens -> will be truncated to 20
	ragLong2 := "Chunk 1: code\n\nChunk 2: rest\n\nChunk 3: build\n\nChunk 4: test" // ~25 tokens

	alloc3 := bm.Allocate(projCtxLong, memoryLong, ragLong2)

	if alloc3.TotalInjected > cfg.TotalMaxTokens {
		t.Errorf("expected total tokens (%d) <= total max (%d)", alloc3.TotalInjected, cfg.TotalMaxTokens)
	}

	// Project Context has highest priority (capped at 20 tokens)
	if alloc3.ProjectTokens > cfg.ProjectContextMaxTokens {
		t.Errorf("project tokens (%d) exceeded budget (%d)", alloc3.ProjectTokens, cfg.ProjectContextMaxTokens)
	}

	// Total was over 50, so RAG is trimmed first, then memory.
	// Let's verify that memory and project context are populated and total is exactly <= 50.
	if alloc3.ProjectTokens == 0 {
		t.Error("expected project context to be populated")
	}
}
