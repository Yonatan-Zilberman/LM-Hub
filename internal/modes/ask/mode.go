package ask

import (
	"context"
	"fmt"
	"strings"

	"github.com/yonatanzilberman/lmhub/internal/agent"
	"github.com/yonatanzilberman/lmhub/internal/api"
	"github.com/yonatanzilberman/lmhub/internal/config"
	"github.com/yonatanzilberman/lmhub/internal/modelmanager"
)

// AskMode handles stateful conversation chat loop.
type AskMode struct {
	client         *api.Client
	modelManager   *modelmanager.Manager
	contextManager *agent.ContextManager
	budgetManager  *agent.BudgetManager
	cfg            *config.Config
	history        []api.Message
	systemPrompt   string
}

// NewAskMode creates a new AskMode instance.
func NewAskMode(
	client *api.Client,
	mm *modelmanager.Manager,
	cm *agent.ContextManager,
	bm *agent.BudgetManager,
	cfg *config.Config,
) *AskMode {
	return &AskMode{
		client:         client,
		modelManager:   mm,
		contextManager: cm,
		budgetManager:  bm,
		cfg:            cfg,
		history:        make([]api.Message, 0),
	}
}

// Reset clears the conversation history.
func (am *AskMode) Reset() {
	am.history = make([]api.Message, 0)
}

// History returns the conversation history.
func (am *AskMode) History() []api.Message {
	return am.history
}

// SetHistory replaces the history.
func (am *AskMode) SetHistory(hist []api.Message) {
	am.history = hist
}

// AddMessage adds a message to history.
func (am *AskMode) AddMessage(msg api.Message) {
	am.history = append(am.history, msg)
}

// SendUserMessage sends a user message and yields response stream chunks.
// It automatically manages context and renders the system prompt.
func (am *AskMode) SendUserMessage(ctx context.Context, modelID string, text string, cwd, osName, shell, projectContext, memoryFacts string, temp float64, maxToks int) (<-chan api.StreamChunk, string, error) {
	// Add user message to history
	am.history = append(am.history, api.Message{
		Role:    "user",
		Content: text,
	})

	// Allocate budget and render prompt
	allocation := am.budgetManager.Allocate(projectContext, memoryFacts, "")
	am.systemPrompt = agent.RenderAskPrompt(cwd, osName, shell, allocation.ProjectContext, allocation.MemoryFacts)

	// Manage context size
	metrics := am.modelManager.Metrics().Get()
	limit := metrics.ContextLimit
	if limit == 0 {
		limit = 128000 // Safe default
	}

	result := am.contextManager.ManageContext(
		am.history,
		am.systemPrompt,
		limit,
		am.cfg.Agent.ContextWarnPct,
		am.cfg.Agent.ContextTrimPct,
		am.cfg.Agent.ContextSummarizePct,
	)

	// Respond to the context management result
	if result.Action == agent.ContextHardStop {
		// Remove the last user message since it failed to send due to hard stop
		if len(am.history) > 0 {
			am.history = am.history[:len(am.history)-1]
		}
		return nil, result.Log, fmt.Errorf("context limit reached (hard-stop): %s", result.Log)
	}

	if result.Action == agent.ContextTrimmed {
		am.history = result.Messages
	}

	// Prepare payload messages (inject system prompt at the beginning)
	reqMessages := []api.Message{
		{
			Role:    "system",
			Content: am.systemPrompt,
		},
	}
	reqMessages = append(reqMessages, am.history...)

	// Create request
	req := api.ChatRequest{
		Model:       modelID,
		Messages:    reqMessages,
		Temperature: temp,
		MaxTokens:   maxToks,
		TopP:        0.95,
	}

	// Stream chat response
	stream, err := am.client.ChatCompletionStream(ctx, req)
	if err != nil {
		return nil, result.Log, fmt.Errorf("failed to start streaming response: %w", err)
	}

	// Create a wrapper channel that appends the response to history once finished
	outChan := make(chan api.StreamChunk, 100)
	go func() {
		defer close(outChan)
		var sb strings.Builder

		for chunk := range stream {
			if chunk.Error != nil {
				outChan <- chunk
				return
			}

			if chunk.Content != "" {
				sb.WriteString(chunk.Content)
				outChan <- chunk
			}

			if chunk.Done {
				// Append assistant response to history
				am.history = append(am.history, api.Message{
					Role:    "assistant",
					Content: sb.String(),
				})

				// Update metrics with final usage/tokens if available
				totalLen := am.contextManager.CountMessagesTokens(am.history) + am.contextManager.CountTokens(am.systemPrompt)
				am.modelManager.Metrics().UpdateTelemetry(modelID, limit, totalLen, metrics.RAMUsedGB)

				outChan <- chunk
			}
		}
	}()

	return outChan, result.Log, nil
}

