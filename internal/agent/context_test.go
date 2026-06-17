package agent

import (
	"testing"

	"github.com/yonatanzilberman/lmhub/internal/api"
)

func TestContextManager_ManageContext(t *testing.T) {
	cm, err := NewContextManager()
	if err != nil {
		t.Fatalf("failed to create context manager: %v", err)
	}

	sysPrompt := "You are a helpful assistant." // ~6 tokens
	messages := []api.Message{
		{Role: "user", Content: "Hello assistant!"},              // ~4 tokens
		{Role: "assistant", Content: "Hello human!"},             // ~4 tokens
		{Role: "user", Content: "How is the weather today?"},     // ~6 tokens
		{Role: "assistant", Content: "It is sunny and bright!"},  // ~6 tokens
		{Role: "user", Content: "Tell me a joke."},               // ~5 tokens
		{Role: "assistant", Content: "Why did the chicken..."},   // ~5 tokens
	}

	// Case 1: Normal (Total limit = 1000, 36 tokens total, under warn threshold 70%)
	res1 := cm.ManageContext(messages, sysPrompt, 1000, 70, 85, 90)
	if res1.Action != ContextOK {
		t.Errorf("expected ContextOK, got %v", res1.Action)
	}
	if len(res1.Messages) != len(messages) {
		t.Errorf("expected no messages to be trimmed, got %d", len(res1.Messages))
	}

	// Case 2: Warn (Total limit = 80, warnPct = 70 => threshold 56. Total = 60. Above 56, below 68 (trim))
	res2 := cm.ManageContext(messages, sysPrompt, 80, 70, 85, 90)
	if res2.Action != ContextWarn {
		t.Errorf("expected ContextWarn, got %v (log: %s)", res2.Action, res2.Log)
	}

	// Case 3: Trim (Total limit = 75, trimPct = 80 => threshold 60. Total = 64. Above 60, below 67 (summarize))
	res3 := cm.ManageContext(messages, sysPrompt, 75, 50, 80, 90)
	if res3.Action != ContextTrimmed {
		t.Errorf("expected ContextTrimmed, got %v (log: %s)", res3.Action, res3.Log)
	}
	if len(res3.Messages) != 4 {
		t.Errorf("expected trimmed messages to keep last 4, got %d", len(res3.Messages))
	}

	// Case 4: Needs Summarize (Total limit = 70, summarizePct = 90 => threshold 63. Total = 64. Above 63, below 68 (hard stop))
	res4 := cm.ManageContext(messages, sysPrompt, 70, 50, 60, 90)
	if res4.Action != ContextNeedsSummarize {
		t.Errorf("expected ContextNeedsSummarize, got %v (log: %s)", res4.Action, res4.Log)
	}

	// Case 5: HardStop (Total limit = 60, hardStopThreshold = 98% => threshold 58. Total = 60. Above 58)
	res5 := cm.ManageContext(messages, sysPrompt, 60, 50, 60, 70)
	if res5.Action != ContextHardStop {
		t.Errorf("expected ContextHardStop, got %v (log: %s)", res5.Action, res5.Log)
	}
}
