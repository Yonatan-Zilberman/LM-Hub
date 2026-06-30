package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// StreamChunk represents a single token or chunk streamed from the chat endpoint.
type StreamChunk struct {
	Content   string
	Done      bool
	Error     error
	TokSpeed  float64 // tokens/sec
	TTFTMs    int     // Time to first token in milliseconds
	UsageInfo *Usage  // Usage info if populated (usually at the end)
}

// ChatCompletion requests a non-streaming chat completion.
func (c *Client) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Stream = false

	resp, err := c.Resty.R().
		SetContext(ctx).
		SetBody(req).
		Post("/v1/chat/completions")
	if err != nil {
		return nil, fmt.Errorf("chat completion request failed: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("chat completion API returned error: status %d, body %s", resp.StatusCode(), resp.String())
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(resp.Body(), &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse chat completion response: %w", err)
	}

	return &chatResp, nil
}

// ChatCompletionStream requests a streaming chat completion.
// It returns a channel that yields StreamChunk objects as they arrive.
func (c *Client) ChatCompletionStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	req.Stream = true

	// Serialize request body
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chat request: %w", err)
	}

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Cache-Control", "no-cache")
	httpReq.Header.Set("Connection", "keep-alive")

	// Execute request using the configured HTTP client from Resty
	httpClient := c.Resty.GetClient()
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send streaming chat request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("streaming chat completions returned status %d: %s", resp.StatusCode, string(body))
	}

	outChan := make(chan StreamChunk, 100)

	go func() {
		defer resp.Body.Close()
		defer close(outChan)

		reader := bufio.NewReader(resp.Body)
		startTime := time.Now()
		var firstTokenTime time.Time
		tokenCount := 0
		var ttftMs int

		type lineResult struct {
			line string
			err  error
		}
		lineCh := make(chan lineResult, 1)
		go func() {
			defer close(lineCh)
			for {
				line, err := reader.ReadString('\n')
				select {
				case lineCh <- lineResult{line: line, err: err}:
					if err != nil {
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()

		for {
			select {
			case <-ctx.Done():
				outChan <- StreamChunk{Error: ctx.Err()}
				return
			case res, ok := <-lineCh:
				if !ok {
					return
				}
				if res.err != nil {
					if ctx.Err() != nil {
						outChan <- StreamChunk{Error: ctx.Err()}
						return
					}
					if res.err == io.EOF {
						return
					}
					outChan <- StreamChunk{Error: fmt.Errorf("error reading stream: %w", res.err)}
					return
				}

				line := strings.TrimSpace(res.line)
				if line == "" {
					continue
				}

				if !strings.HasPrefix(line, "data: ") {
					continue
				}

				dataStr := line[6:]
				if dataStr == "[DONE]" {
					outChan <- StreamChunk{Done: true}
					return
				}

				var chunk ChatResponse
				if err := json.Unmarshal([]byte(dataStr), &chunk); err != nil {
					outChan <- StreamChunk{Error: fmt.Errorf("failed to decode stream chunk: %w", err)}
					return
				}

				if tokenCount == 0 {
					firstTokenTime = time.Now()
					ttftMs = int(firstTokenTime.Sub(startTime).Milliseconds())
				}

				if len(chunk.Choices) > 0 {
					deltaContent := chunk.Choices[0].Delta.Content
					if deltaContent != "" {
						tokenCount++
						speed := 0.0
						if tokenCount > 1 && !firstTokenTime.IsZero() {
							elapsed := time.Since(firstTokenTime).Seconds()
							if elapsed > 0 {
								speed = float64(tokenCount) / elapsed
							}
						}

						var usageInfo *Usage
						if chunk.Usage.TotalTokens > 0 {
							usageInfo = &chunk.Usage
						}

						outChan <- StreamChunk{
							Content:   deltaContent,
							TokSpeed:  speed,
							TTFTMs:    ttftMs,
							UsageInfo: usageInfo,
						}
					}
				}
			}
		}
	}()

	return outChan, nil
}
