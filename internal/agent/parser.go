package agent

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"

	"github.com/yonatanzilberman/lmhub/internal/api"
)

// ParseMetrics tracks tool call parsing successes and failures.
type ParseMetrics struct {
	mu                  sync.Mutex
	TotalAttempts       int
	SuccessCount        int
	FailureCount        int
	ConsecutiveFailures int
}

// RecordSuccess registers a successful parsing attempt.
func (pm *ParseMetrics) RecordSuccess() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.TotalAttempts++
	pm.SuccessCount++
	pm.ConsecutiveFailures = 0
}

// RecordFailure registers a failed parsing attempt.
func (pm *ParseMetrics) RecordFailure() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.TotalAttempts++
	pm.FailureCount++
	pm.ConsecutiveFailures++
}

// ShouldWarn returns true if there have been 3 or more consecutive failures.
func (pm *ParseMetrics) ShouldWarn() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.ConsecutiveFailures >= 3
}

// Reset clears the metrics tracker.
func (pm *ParseMetrics) Reset() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.TotalAttempts = 0
	pm.SuccessCount = 0
	pm.FailureCount = 0
	pm.ConsecutiveFailures = 0
}

// GlobalParseMetrics is the package-level instance of ParseMetrics.
var GlobalParseMetrics = &ParseMetrics{}

// ToolCall represents a structured call to an agent tool.
type ToolCall struct {
	Name    string                 `json:"name"`
	Args    map[string]interface{} `json:"args"`
	RawText string                 `json:"-"`
}

// ParseError indicates that no tool calls could be parsed from the model response.
type ParseError struct {
	RawOutput string
}

func (e *ParseError) Error() string {
	return "failed to parse any tool calls from model response"
}

// ParseToolCall extracts tool calls from the model response using a 5-layer fallback strategy.
func ParseToolCall(content string, nativeCalls []api.ToolCall) ([]ToolCall, error) {
	// Layer 1: Native tool calls
	if len(nativeCalls) > 0 {
		var tcs []ToolCall
		for _, nc := range nativeCalls {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(nc.Function.Arguments), &args); err != nil {
				args = make(map[string]interface{})
			}
			tcs = append(tcs, ToolCall{
				Name: nc.Function.Name,
				Args: args,
			})
		}
		GlobalParseMetrics.RecordSuccess()
		return tcs, nil
	}

	// Layer 2: XML-style <tool_call>...</tool_call> tags
	xmlRegex := regexp.MustCompile(`(?s)<tool_call>(.*?)</tool_call>`)
	xmlMatches := xmlRegex.FindAllStringSubmatch(content, -1)
	if len(xmlMatches) > 0 {
		var tcs []ToolCall
		for _, match := range xmlMatches {
			raw := strings.TrimSpace(match[1])
			var tc ToolCall
			if err := json.Unmarshal([]byte(raw), &tc); err == nil && tc.Name != "" {
				tc.RawText = match[0]
				tcs = append(tcs, tc)
			}
		}
		if len(tcs) > 0 {
			GlobalParseMetrics.RecordSuccess()
			return tcs, nil
		}
	}

	// Layer 3: Markdown ```json code blocks
	mdRegex := regexp.MustCompile("(?s)```json\\s+(.*?)\\s+```")
	mdMatches := mdRegex.FindAllStringSubmatch(content, -1)
	if len(mdMatches) > 0 {
		var tcs []ToolCall
		for _, match := range mdMatches {
			raw := strings.TrimSpace(match[1])
			
			// Try parsing as array of tool calls
			var tcList []ToolCall
			if err := json.Unmarshal([]byte(raw), &tcList); err == nil {
				for _, t := range tcList {
					if t.Name != "" {
						t.RawText = match[0]
						tcs = append(tcs, t)
					}
				}
			} else {
				// Try parsing as single tool call
				var tc ToolCall
				if err := json.Unmarshal([]byte(raw), &tc); err == nil && tc.Name != "" {
					tc.RawText = match[0]
					tcs = append(tcs, tc)
				}
			}
		}
		if len(tcs) > 0 {
			GlobalParseMetrics.RecordSuccess()
			return tcs, nil
		}
	}

	// Layer 4: Regex/Brace matching extraction
	// Scan the text for JSON objects containing name and args
	var jsonObjs []string
	start := -1
	depth := 0
	for i := 0; i < len(content); i++ {
		if content[i] == '{' {
			if depth == 0 {
				start = i
			}
			depth++
		} else if content[i] == '}' {
			if depth > 0 {
				depth--
				if depth == 0 && start != -1 {
					jsonObjs = append(jsonObjs, content[start:i+1])
					start = -1
				}
			}
		}
	}

	var tcs []ToolCall
	for _, objStr := range jsonObjs {
		var tc ToolCall
		if err := json.Unmarshal([]byte(objStr), &tc); err == nil {
			if tc.Name != "" && tc.Args != nil {
				tc.RawText = objStr
				tcs = append(tcs, tc)
			}
		}
	}
	if len(tcs) > 0 {
		GlobalParseMetrics.RecordSuccess()
		return tcs, nil
	}

	// Layer 5: Failure
	GlobalParseMetrics.RecordFailure()
	return nil, &ParseError{RawOutput: content}
}

// ExtractThought parses and returns thoughts from the model response text,
// typically wrapped in <thought>...</thought> tags.
// If the tag is opened but not closed, it returns everything after the tag.
func ExtractThought(content string) string {
	re := regexp.MustCompile(`(?s)<thought>(.*?)</thought>`)
	match := re.FindStringSubmatch(content)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	idx := strings.Index(content, "<thought>")
	if idx != -1 {
		return strings.TrimSpace(content[idx+9:])
	}

	return ""
}
