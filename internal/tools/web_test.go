package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWebSearchMock runs DuckDuckGo search mock server test.
func TestWebSearchMock(t *testing.T) {
	mockResponse := `{
		"AbstractText": "LM Studio is an app to run local LLMs.",
		"AbstractURL": "https://lmstudio.ai",
		"RelatedTopics": [
			{"Text": "Local LLM runner", "FirstURL": "https://example.com/llm"},
			{"Text": "Ollama alternative", "FirstURL": "https://example.com/alt"}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "LMHub/1.0", r.Header.Get("User-Agent"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	ctx := context.Background()
	searchTool := NewWebSearchTool("duckduckgo", "")
	searchTool.baseURL = server.URL

	res, err := searchTool.Execute(ctx, map[string]interface{}{
		"query": "LM Studio",
		"n":     2,
	})
	assert.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Contains(t, res.Content, "LM Studio is an app to run local LLMs.")
	assert.Contains(t, res.Content, "Source URL: https://lmstudio.ai")
	assert.Contains(t, res.Content, "Local LLM runner")
}

// TestWebFetchMock runs goquery parsing webpage mock test.
func TestWebFetchMock(t *testing.T) {
	mockHTML := `<html>
		<head><title>Test Page</title></head>
		<body>
			<nav>Header Nav</nav>
			<h1>Main Heading</h1>
			<p>First paragraph about LMHub.</p>
			<h2>Section Heading</h2>
			<p>Second paragraph details.</p>
			<ul>
				<li>First point</li>
				<li>Second point</li>
			</ul>
			<footer>Footer content to be removed</footer>
		</body>
	</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	ctx := context.Background()
	fetchTool := NewWebFetchTool(5, 1)

	// Fetch webpage
	res, err := fetchTool.Execute(ctx, map[string]interface{}{
		"url": server.URL,
	})
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	// Verify header, h1, h2, paragraphs and lists are present and navigation/footer are removed
	assert.Contains(t, res.Content, "Title: Test Page")
	assert.Contains(t, res.Content, "# Main Heading")
	assert.Contains(t, res.Content, "First paragraph about LMHub.")
	assert.Contains(t, res.Content, "## Section Heading")
	assert.Contains(t, res.Content, "Second paragraph details.")
	assert.Contains(t, res.Content, "- First point")
	assert.Contains(t, res.Content, "- Second point")
	assert.NotContains(t, res.Content, "Header Nav")
	assert.NotContains(t, res.Content, "Footer content")

	// Test cache hit
	// We change the server response to verify it returns cached contents
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	resCache, err := fetchTool.Execute(ctx, map[string]interface{}{
		"url": server.URL,
	})
	assert.NoError(t, err)
	assert.False(t, resCache.IsError)
	assert.Contains(t, resCache.Content, "Main Heading") // returns cached page
}
