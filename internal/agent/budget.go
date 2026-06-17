package agent

import (
	"strings"

	"github.com/yonatanzilberman/lmhub/internal/config"
)

// BudgetManager enforces context window allocation limits and priorities.
type BudgetManager struct {
	contextManager *ContextManager
	cfg            *config.ContextBudgetConfig
}

// NewBudgetManager creates a new BudgetManager.
func NewBudgetManager(cm *ContextManager, cfg *config.ContextBudgetConfig) *BudgetManager {
	return &BudgetManager{
		contextManager: cm,
		cfg:            cfg,
	}
}

// BudgetAllocation details how the context budget has been allocated and consumed.
type BudgetAllocation struct {
	ProjectContext string
	MemoryFacts    string
	RAGChunks      string
	ProjectTokens  int
	MemoryTokens   int
	RAGTokens      int
	TotalInjected  int
}

// Allocate fills sources in priority order:
// 1. Project context (highest priority)
// 2. Memory facts
// 3. RAG chunks (trimmed first if over budget)
func (bm *BudgetManager) Allocate(projectCtx, memoryFacts, ragChunks string) BudgetAllocation {
	cm := bm.contextManager

	// 1. Apply individual limits first
	allocatedProj := cm.TruncateToTokens(projectCtx, bm.cfg.ProjectContextMaxTokens)
	projTokens := cm.CountTokens(allocatedProj)

	// Memory limits: trim from start (LRU oldest first)
	allocatedMem := trimNewlineSeparated(memoryFacts, bm.cfg.MemoryMaxTokens, true, cm)
	memTokens := cm.CountTokens(allocatedMem)

	// RAG limits: trim from end (lowest score first)
	allocatedRAG := trimDoubleNewlineSeparated(ragChunks, bm.cfg.RAGMaxTokens, false, cm)
	ragTokens := cm.CountTokens(allocatedRAG)

	total := projTokens + memTokens + ragTokens

	// 2. If total exceeds maximum limit, trim by priority: RAG -> Memory -> Project Context
	if total > bm.cfg.TotalMaxTokens {
		// Trim RAG first
		ragAllowed := bm.cfg.TotalMaxTokens - projTokens - memTokens
		if ragAllowed <= 0 {
			allocatedRAG = ""
			ragTokens = 0
		} else {
			allocatedRAG = trimDoubleNewlineSeparated(allocatedRAG, ragAllowed, false, cm)
			ragTokens = cm.CountTokens(allocatedRAG)
		}
		total = projTokens + memTokens + ragTokens

		// If still over, trim Memory
		if total > bm.cfg.TotalMaxTokens {
			memAllowed := bm.cfg.TotalMaxTokens - projTokens
			if memAllowed <= 0 {
				allocatedMem = ""
				memTokens = 0
			} else {
				allocatedMem = trimNewlineSeparated(allocatedMem, memAllowed, true, cm)
				memTokens = cm.CountTokens(allocatedMem)
			}
			total = projTokens + memTokens + ragTokens

			// If still over, truncate Project Context
			if total > bm.cfg.TotalMaxTokens {
				allocatedProj = cm.TruncateToTokens(allocatedProj, bm.cfg.TotalMaxTokens)
				projTokens = cm.CountTokens(allocatedProj)
				total = projTokens + memTokens + ragTokens
			}
		}
	}

	return BudgetAllocation{
		ProjectContext: allocatedProj,
		MemoryFacts:    allocatedMem,
		RAGChunks:      allocatedRAG,
		ProjectTokens:  projTokens,
		MemoryTokens:   memTokens,
		RAGTokens:      ragTokens,
		TotalInjected:  total,
	}
}

// trimNewlineSeparated trims a newline-separated list of facts to fit maxTokens.
func trimNewlineSeparated(text string, maxTokens int, trimFromStart bool, cm *ContextManager) string {
	if text == "" || maxTokens <= 0 {
		return ""
	}
	lines := strings.Split(text, "\n")
	var clean []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			clean = append(clean, l)
		}
	}
	if len(clean) == 0 {
		return ""
	}

	joined := strings.Join(clean, "\n")
	if cm.CountTokens(joined) <= maxTokens {
		return joined
	}

	for len(clean) > 0 {
		if trimFromStart {
			clean = clean[1:]
		} else {
			clean = clean[:len(clean)-1]
		}
		joined = strings.Join(clean, "\n")
		if cm.CountTokens(joined) <= maxTokens {
			return joined
		}
	}
	return ""
}

// trimDoubleNewlineSeparated trims double-newline separated chunks of text.
func trimDoubleNewlineSeparated(text string, maxTokens int, trimFromStart bool, cm *ContextManager) string {
	if text == "" || maxTokens <= 0 {
		return ""
	}
	chunks := strings.Split(text, "\n\n")
	separator := "\n\n"
	if len(chunks) <= 1 {
		chunks = strings.Split(text, "\n")
		separator = "\n"
	}

	var clean []string
	for _, c := range chunks {
		if strings.TrimSpace(c) != "" {
			clean = append(clean, c)
		}
	}
	if len(clean) == 0 {
		return ""
	}

	joined := strings.Join(clean, separator)
	if cm.CountTokens(joined) <= maxTokens {
		return joined
	}

	for len(clean) > 0 {
		if trimFromStart {
			clean = clean[1:]
		} else {
			clean = clean[:len(clean)-1]
		}
		joined = strings.Join(clean, separator)
		if cm.CountTokens(joined) <= maxTokens {
			return joined
		}
	}
	return ""
}
