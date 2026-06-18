// Package api integrates with LM Studio APIs.
package api

import (
	"context"
	"fmt"
)

// CreateEmbeddings requests embeddings from LM Studio for the provided text inputs.
// It uses the configured embedding model name.
func (c *Client) CreateEmbeddings(ctx context.Context, model string, input []string) (*EmbeddingsResponse, error) {
	if len(input) == 0 {
		return &EmbeddingsResponse{}, nil
	}

	req := EmbeddingsRequest{
		Model: model,
		Input: input,
	}

	var respBody EmbeddingsResponse
	resp, err := c.Resty.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&respBody).
		Post("/v1/embeddings")

	if err != nil {
		return nil, fmt.Errorf("create embeddings API error: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("create embeddings API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return &respBody, nil
}
