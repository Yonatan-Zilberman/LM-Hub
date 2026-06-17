package api

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

// Client is the API client for communicating with LM Studio.
type Client struct {
	BaseURL string
	Resty   *resty.Client
}

// NewClient creates a new Client instance.
func NewClient(baseURL string, timeoutSeconds int) *Client {
	r := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(time.Duration(timeoutSeconds) * time.Second).
		SetHeader("Content-Type", "application/json")

	return &Client{
		BaseURL: baseURL,
		Resty:   r,
	}
}

// Ping checks if the LM Studio server is reachable.
func (c *Client) Ping(ctx context.Context) error {
	// Ping /v1/models as a simple health check
	resp, err := c.Resty.R().
		SetContext(ctx).
		Get("/v1/models")
	if err != nil {
		return fmt.Errorf("failed to ping LM Studio server: %w", err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("ping returned non-success status code: %d", resp.StatusCode())
	}

	return nil
}
