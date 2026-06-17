// Package tools implements agent tools for LM Hub.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// cacheEntry holds cached page fetch content.
type cacheEntry struct {
	content   string
	expiresAt time.Time
}

var (
	// fetchCache stores fetched HTML parsed content in memory.
	fetchCache = make(map[string]cacheEntry)
	// cacheMu protects fetchCache.
	cacheMu sync.Mutex
)

// WebSearchTool searches the web using DuckDuckGo.
type WebSearchTool struct {
	searchProvider string
	serperAPIKey   string
	baseURL        string
}

// NewWebSearchTool creates a new web_search tool.
func NewWebSearchTool(provider, serperKey string) *WebSearchTool {
	return &WebSearchTool{
		searchProvider: provider,
		serperAPIKey:   serperKey,
		baseURL:        "https://api.duckduckgo.com",
	}
}

// Name returns the name of the tool.
func (t *WebSearchTool) Name() string { return "web_search" }

// Description returns the description.
func (t *WebSearchTool) Description() string {
	return "Search the web using DuckDuckGo for info or answers."
}

// Schema returns the JSON schema.
func (t *WebSearchTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"query": map[string]interface{}{"type": "string", "description": "The search query"},
			"n":     map[string]interface{}{"type": "integer", "description": "Optional number of related topic summaries to return"},
		},
		Required: []string{"query"},
	}
}

// Permission returns Safe.
func (t *WebSearchTool) Permission() PermissionLevel { return Safe }

// Undoable returns false.
func (t *WebSearchTool) Undoable() bool { return false }

// Snapshot is a no-op.
func (t *WebSearchTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, nil
}

// Execute performs DuckDuckGo search.
func (t *WebSearchTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	query, _ := args["query"].(string)
	limit := 3
	if nVal, ok := args["n"]; ok {
		if nFloat, ok := nVal.(float64); ok && nFloat > 0 {
			limit = int(nFloat)
		} else if nInt, ok := nVal.(int); ok && nInt > 0 {
			limit = nInt
		}
	}

	baseURL := t.baseURL
	if baseURL == "" {
		baseURL = "https://api.duckduckgo.com"
	}
	apiURL := fmt.Sprintf("%s/?q=%s&format=json&no_html=1", baseURL, url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("create search request: %w", err).Error()}, nil
	}
	req.Header.Set("User-Agent", "LMHub/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("execute search: %w", err).Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ToolResult{IsError: true, Content: fmt.Sprintf("search returned HTTP status %s", resp.Status)}, nil
	}

	var data struct {
		AbstractText  string `json:"AbstractText"`
		AbstractURL   string `json:"AbstractURL"`
		RelatedTopics []struct {
			Text string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"RelatedTopics"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("decode search response: %w", err).Error()}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("DuckDuckGo Instant Answer for '%s':\n", query))
	if data.AbstractText != "" {
		sb.WriteString(fmt.Sprintf("\nAbstract: %s\nSource URL: %s\n", data.AbstractText, data.AbstractURL))
	}

	relatedCount := 0
	for _, topic := range data.RelatedTopics {
		if topic.Text != "" && topic.FirstURL != "" {
			if relatedCount == 0 {
				sb.WriteString("\nRelated Topics:\n")
			}
			sb.WriteString(fmt.Sprintf("- %s (URL: %s)\n", topic.Text, topic.FirstURL))
			relatedCount++
			if relatedCount >= limit {
				break
			}
		}
	}

	if data.AbstractText == "" && relatedCount == 0 {
		sb.WriteString("\nNo instant answers found. Try a different query or refine terms.")
	}

	return ToolResult{Content: sb.String()}, nil
}

// WebFetchTool fetches web pages and extracts structured text.
type WebFetchTool struct {
	timeoutSec      int
	cacheTTLMinutes int
}

// NewWebFetchTool creates a new web_fetch tool.
func NewWebFetchTool(timeoutSec, cacheTTLMinutes int) *WebFetchTool {
	if timeoutSec <= 0 {
		timeoutSec = 10
	}
	if cacheTTLMinutes <= 0 {
		cacheTTLMinutes = 60
	}
	return &WebFetchTool{
		timeoutSec:      timeoutSec,
		cacheTTLMinutes: cacheTTLMinutes,
	}
}

// Name returns the tool name.
func (t *WebFetchTool) Name() string { return "web_fetch" }

// Description returns description.
func (t *WebFetchTool) Description() string {
	return "Fetch a web page and parse its content into readable markdown."
}

// Schema returns schema.
func (t *WebFetchTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"url": map[string]interface{}{"type": "string", "description": "The URL of the webpage to fetch"},
		},
		Required: []string{"url"},
	}
}

// Permission returns Safe.
func (t *WebFetchTool) Permission() PermissionLevel { return Safe }

// Undoable returns false.
func (t *WebFetchTool) Undoable() bool { return false }

// Snapshot is a no-op.
func (t *WebFetchTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, nil
}

// Execute performs web fetching.
func (t *WebFetchTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	urlStr, _ := args["url"].(string)

	// Clean/validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return ToolResult{IsError: true, Content: "invalid URL scheme (must be http or https)"}, nil
	}

	// Check cache
	cacheMu.Lock()
	if cached, exists := fetchCache[urlStr]; exists && time.Now().Before(cached.expiresAt) {
		cacheMu.Unlock()
		return ToolResult{Content: cached.content}, nil
	}
	cacheMu.Unlock()

	// Perform HTTP fetch
	client := &http.Client{
		Timeout: time.Duration(t.timeoutSec) * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("create fetch request: %w", err).Error()}, nil
	}
	req.Header.Set("User-Agent", "LMHub/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("execute fetch: %w", err).Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ToolResult{IsError: true, Content: fmt.Sprintf("fetch returned HTTP status %s", resp.Status)}, nil
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("parse HTML document: %w", err).Error()}, nil
	}

	// Clean HTML document by removing script, style, nav, footer, etc.
	doc.Find("script, style, iframe, footer, nav, header, noscript").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Content from %s\n\n", urlStr))

	title := strings.TrimSpace(doc.Find("title").Text())
	if title != "" {
		sb.WriteString(fmt.Sprintf("## Title: %s\n\n", title))
	}

	// Parse main semantic blocks: h1, h2, h3, p, li
	doc.Find("h1, h2, h3, p, li").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text == "" {
			return
		}

		tagName := goquery.NodeName(s)
		switch tagName {
		case "h1":
			sb.WriteString(fmt.Sprintf("\n# %s\n\n", text))
		case "h2":
			sb.WriteString(fmt.Sprintf("\n## %s\n\n", text))
		case "h3":
			sb.WriteString(fmt.Sprintf("\n### %s\n\n", text))
		case "p":
			sb.WriteString(fmt.Sprintf("%s\n\n", text))
		case "li":
			sb.WriteString(fmt.Sprintf("- %s\n", text))
		}
	})

	content := strings.TrimSpace(sb.String())

	// Save cache
	cacheMu.Lock()
	fetchCache[urlStr] = cacheEntry{
		content:   content,
		expiresAt: time.Now().Add(time.Duration(t.cacheTTLMinutes) * time.Minute),
	}
	cacheMu.Unlock()

	return ToolResult{Content: content}, nil
}
