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

// TruncateToTokens truncates the given text to a maximum number of tokens.
func (cm *ContextManager) TruncateToTokens(text string, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}
	ids, _, err := cm.codec.Encode(text)
	if err != nil || len(ids) <= maxTokens {
		return text
	}
	truncated, err := cm.codec.Decode(ids[:maxTokens])
	if err != nil {
		// Fallback simple approximation
		return text[:len(text)*maxTokens/len(ids)]
	}
	return truncated
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

// ContextAction represents the action taken or recommended for context management.
type ContextAction int

const (
	// ContextOK indicates context is within safe limits.
	ContextOK ContextAction = iota
	// ContextWarn indicates context is approaching the limit.
	ContextWarn
	// ContextTrimmed indicates the context has been trimmed to free up space.
	ContextTrimmed
	// ContextNeedsSummarize indicates the context needs to be summarized.
	ContextNeedsSummarize
	// ContextHardStop indicates context limit is critical and execution must stop.
	ContextHardStop
)

// ContextResult contains the decision made by ContextManager.
type ContextResult struct {
	Action   ContextAction
	Messages []api.Message
	Log      string
}

// ManageContext implements the escalating context management strategy.
// It checks the total token size (system prompt + messages) against thresholds
// and performs warnings, trimming, summarization requests, or hard stops.
func (cm *ContextManager) ManageContext(messages []api.Message, systemPrompt string, maxLimit int, warnPct, trimPct, summarizePct int) ContextResult {
	if maxLimit <= 0 {
		return ContextResult{Action: ContextOK, Messages: messages}
	}

	sysTokens := cm.CountTokens(systemPrompt)
	msgTokens := cm.CountMessagesTokens(messages)
	total := sysTokens + msgTokens

	hardStopThreshold := (maxLimit * 98) / 100
	summarizeThreshold := (maxLimit * summarizePct) / 100
	trimThreshold := (maxLimit * trimPct) / 100
	warnThreshold := (maxLimit * warnPct) / 100

	if total >= hardStopThreshold {
		return ContextResult{
			Action:   ContextHardStop,
			Messages: messages,
			Log:      fmt.Sprintf("Context size (%d) reached hard-stop threshold (98%% of %d). Execution paused.", total, maxLimit),
		}
	}

	if total >= summarizeThreshold {
		return ContextResult{
			Action:   ContextNeedsSummarize,
			Messages: messages,
			Log:      fmt.Sprintf("Context size (%d) reached summarize threshold (%d%% of %d). Summarization required.", total, summarizePct, maxLimit),
		}
	}

	if total >= trimThreshold {
		if len(messages) > 4 {
			trimmed := make([]api.Message, 4)
			copy(trimmed, messages[len(messages)-4:])
			newMsgTokens := cm.CountMessagesTokens(trimmed)
			newTotal := sysTokens + newMsgTokens
			return ContextResult{
				Action:   ContextTrimmed,
				Messages: trimmed,
				Log:      fmt.Sprintf("Context size (%d) exceeded trim threshold (%d%% of %d). Kept last 4 messages. New context size: %d.", total, trimPct, maxLimit, newTotal),
			}
		}
	}

	if total >= warnThreshold {
		return ContextResult{
			Action:   ContextWarn,
			Messages: messages,
			Log:      fmt.Sprintf("Context size (%d) reached warning threshold (%d%% of %d).", total, warnPct, maxLimit),
		}
	}

	return ContextResult{
		Action:   ContextOK,
		Messages: messages,
	}
}
