package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// ListModels returns all models available in LM Studio.
func (c *Client) ListModels(ctx context.Context) ([]ModelInfo, error) {
	resp, err := c.Resty.R().
		SetContext(ctx).
		Get("/api/v1/models")
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("list models API returned error: status %d, body %s", resp.StatusCode(), resp.String())
	}

	var listResp ModelListResponse
	if err := json.Unmarshal(resp.Body(), &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse models list response: %w", err)
	}

	return listResp.Models, nil
}

// LoadModel loads the model with the given key into memory.
func (c *Client) LoadModel(ctx context.Context, key string, contextLength int) (*LoadModelResponse, error) {
	req := LoadModelRequest{
		Model:         key,
		ContextLength: contextLength,
	}

	resp, err := c.Resty.R().
		SetContext(ctx).
		SetBody(req).
		Post("/api/v1/models/load")
	if err != nil {
		return nil, fmt.Errorf("failed to load model %s: %w", key, err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("load model API returned error: status %d, body %s", resp.StatusCode(), resp.String())
	}

	var loadResp LoadModelResponse
	if err := json.Unmarshal(resp.Body(), &loadResp); err != nil {
		return nil, fmt.Errorf("failed to parse load model response: %w", err)
	}

	return &loadResp, nil
}

// UnloadModel unloads the specified model instance from memory.
func (c *Client) UnloadModel(ctx context.Context, instanceID string) (*UnloadModelResponse, error) {
	req := UnloadModelRequest{
		InstanceID: instanceID,
	}

	resp, err := c.Resty.R().
		SetContext(ctx).
		SetBody(req).
		Post("/api/v1/models/unload")
	if err != nil {
		return nil, fmt.Errorf("failed to unload model instance %s: %w", instanceID, err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("unload model API returned error: status %d, body %s", resp.StatusCode(), resp.String())
	}

	var unloadResp UnloadModelResponse
	if err := json.Unmarshal(resp.Body(), &unloadResp); err != nil {
		return nil, fmt.Errorf("failed to parse unload model response: %w", err)
	}

	return &unloadResp, nil
}
