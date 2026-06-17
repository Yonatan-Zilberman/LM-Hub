package ask

import (
	"context"
	"fmt"
	"strings"

	"github.com/yonatanzilberman/lmhub/internal/agent"
	"github.com/yonatanzilberman/lmhub/internal/api"
	"github.com/yonatanzilberman/lmhub/internal/modelmanager"
)

// AskMode handles stateful conversation chat loop.
type AskMode struct {
	client         *api.Client
	modelManager   *modelmanager.Manager
	contextManager *agent.ContextManager
	history        []api.Message
	systemPrompt   string
}

// NewAskMode creates a new AskMode instance.
func NewAskMode(client *api.Client, mm *modelmanager.Manager, cm *agent.ContextManager) *AskMode {
	return &AskMode{
		client:         client,
		modelManager:   mm,
		contextManager: cm,
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

	// Render prompt
	am.systemPrompt = agent.RenderAskPrompt(cwd, osName, shell, projectContext, memoryFacts)

	// Manage context size
	metrics := am.modelManager.Metrics().Get()
	limit := metrics.ContextLimit
	if limit == 0 {
		limit = 128000 // Safe default
	}

	trimmed, newHist, logMsg := am.contextManager.ManageContext(am.history, am.systemPrompt, limit, 70, 85)
	if trimmed {
		am.history = newHist
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
		return nil, logMsg, fmt.Errorf("failed to start streaming response: %w", err)
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
				// or calculate roughly
				totalLen := am.contextManager.CountMessagesTokens(am.history) + am.contextManager.CountTokens(am.systemPrompt)
				am.modelManager.Metrics().UpdateTelemetry(modelID, limit, totalLen, metrics.RAMUsedGB)
				
				outChan <- chunk
			}
		}
	}()

	return outChan, logMsg, nil
}
