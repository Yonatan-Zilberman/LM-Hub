package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/yonatanzilberman/lmhub/internal/api"
)

// Extractor uses the LM Studio API to analyze conversation history and extract technical facts.
type Extractor struct {
	client *api.Client
}

// NewExtractor creates a new Extractor instance.
func NewExtractor(client *api.Client) *Extractor {
	return &Extractor{client: client}
}

// ExtractedFact holds raw fact details returned by the LLM.
type ExtractedFact struct {
	Content    string  `json:"content"`
	Confidence float64 `json:"confidence"`
}

// ExtractFacts parses conversation history and returns a list of candidate MemoryFacts.
func (e *Extractor) ExtractFacts(ctx context.Context, modelID string, history []api.Message) ([]ExtractedFact, error) {
	if len(history) == 0 {
		return nil, nil
	}

	// Limit history to last 15 messages to prevent overloading the model context
	lookback := history
	if len(lookback) > 15 {
		lookback = lookback[len(lookback)-15:]
	}

	// Build dialogue transcript
	var transcript strings.Builder
	for _, msg := range lookback {
		roleName := "User"
		if msg.Role == "assistant" {
			roleName = "Assistant"
		} else if msg.Role == "system" {
			continue // skip system prompt
		}
		transcript.WriteString(fmt.Sprintf("%s: %s\n\n", roleName, msg.Content))
	}

	systemInstruction := `You are an expert technical fact extractor.
Analyze the following conversation between a user and an AI assistant.
Extract up to 5 concrete, reusable technical facts about the project or workspace that were mentioned or discovered.
Only extract facts that are persistent and will be useful in future sessions.
Do not extract opinions, temporary/transitional information, or code blocks.
Response format must be a valid JSON array of objects. Do not include markdown code block fences or explanations.
JSON schema:
[
  {
    "content": "Description of the fact (e.g., 'PostgreSQL runs on port 5433 locally')",
    "confidence": 0.0-1.0
  }
]
`

	req := api.ChatRequest{
		Model: modelID,
		Messages: []api.Message{
			{Role: "system", Content: systemInstruction},
			{Role: "user", Content: fmt.Sprintf("Here is the conversation transcript:\n\n%s", transcript.String())},
		},
		Temperature: 0.1, // low temperature for structured output
	}

	resp, err := e.client.ChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("extraction chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("extraction returned empty choices")
	}

	rawContent := resp.Choices[0].Message.Content
	return parseExtractedFacts(rawContent)
}

// parseExtractedFacts cleans code blocks and parses the JSON array.
func parseExtractedFacts(raw string) ([]ExtractedFact, error) {
	trimmed := strings.TrimSpace(raw)

	// Regex to extract JSON block if wrapped in markdown code fences
	re := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
	if matches := re.FindStringSubmatch(trimmed); len(matches) > 1 {
		trimmed = strings.TrimSpace(matches[1])
	}

	if !strings.HasPrefix(trimmed, "[") {
		// Try to find the first index of "[" and last index of "]"
		startIdx := strings.Index(trimmed, "[")
		endIdx := strings.LastIndex(trimmed, "]")
		if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
			trimmed = trimmed[startIdx : endIdx+1]
		} else {
			return nil, fmt.Errorf("response does not contain a JSON array: %s", raw)
		}
	}

	var facts []ExtractedFact
	if err := json.Unmarshal([]byte(trimmed), &facts); err != nil {
		return nil, fmt.Errorf("failed to parse extracted facts JSON: %w (raw response: %s)", err, raw)
	}

	return facts, nil
}
