package agent

import (
	"fmt"

	"github.com/tiktoken-go/tokenizer"
	"github.com/yonatanzilberman/lmhub/internal/api"
)

// ContextManager manages the conversation context window and token usage.
type ContextManager struct {
	codec tokenizer.Codec
}

// NewContextManager creates a new ContextManager instance.
func NewContextManager() (*ContextManager, error) {
	enc, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenizer: %w", err)
	}
	return &ContextManager{codec: enc}, nil
}

// CountTokens returns the approximate token count of the given text.
func (cm *ContextManager) CountTokens(text string) int {
	ids, _, err := cm.codec.Encode(text)
	if err != nil {
		// Fallback simple approximation: ~4 chars per token
		return len(text) / 4
	}
	return len(ids)
}

// CountMessagesTokens estimates the total tokens in the conversation messages.
func (cm *ContextManager) CountMessagesTokens(messages []api.Message) int {
	total := 0
	for _, m := range messages {
		// Base tokens per message metadata + content tokens
		total += 4 + cm.CountTokens(m.Content) + cm.CountTokens(m.Role)
	}
	return total
}

// ManageContext implements the escalating context management strategy.
// It trims the history if total tokens exceed the specified warning or trim thresholds.
// It returns a flag indicating if trimming occurred, and the managed messages.
func (cm *ContextManager) ManageContext(messages []api.Message, systemPrompt string, maxLimit int, warnPct, trimPct int) (bool, []api.Message, string) {
	if len(messages) <= 1 {
		return false, messages, ""
	}

	sysTokens := cm.CountTokens(systemPrompt)
	msgTokens := cm.CountMessagesTokens(messages)
	total := sysTokens + msgTokens

	warnThreshold := (maxLimit * warnPct) / 100
	trimThreshold := (maxLimit * trimPct) / 100

	if total < warnThreshold {
		return false, messages, ""
	}

	// If it exceeds the trim threshold (85%), drop older messages except system prompt and last 4 messages.
	if total >= trimThreshold {
		// Keep the first message (system prompt usually, but system prompt is separate anyway)
		// and the last 4 turns (e.g. 2 user messages and 2 assistant responses)
		keepCount := 4
		if len(messages) > keepCount {
			trimmed := make([]api.Message, keepCount)
			copy(trimmed, messages[len(messages)-keepCount:])
			msgLog := fmt.Sprintf("Context size (%d) exceeded trim threshold (%d). Kept last %d messages.", total, trimThreshold, keepCount)
			return true, trimmed, msgLog
		}
	}

	// 70% threshold warning
	return false, messages, fmt.Sprintf("Context size (%d) approaching limit (%d). Consider clearing context.", total, maxLimit)
}
