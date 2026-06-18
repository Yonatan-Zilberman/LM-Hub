package session

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/yonatanzilberman/lmhub/internal/api"
)

// InjectedContext captures the RAG, project context, and memory state at a given turn.
type InjectedContext struct {
	MessageIndex   int      `json:"message_index"`
	ProjectContext string   `json:"project_context"`
	MemoryFacts    []string `json:"memory_facts"`
	RAGChunks      []string `json:"rag_chunks"`
}

// Session contains a conversation history, mode, model, and context details.
type Session struct {
	ID               string            `json:"id"`
	Mode             string            `json:"mode"`
	ModelID          string            `json:"model_id"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	Messages         []api.Message     `json:"messages"`
	InjectedContexts []InjectedContext `json:"injected_contexts"`
}

// NewSession creates a new Session instance.
func NewSession(id, mode, modelID string) *Session {
	return &Session{
		ID:               id,
		Mode:             mode,
		ModelID:          modelID,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Messages:         make([]api.Message, 0),
		InjectedContexts: make([]InjectedContext, 0),
	}
}

// Save writes the session as JSON to the specified path.
func (s *Session) Save(path string) error {
	s.UpdatedAt = time.Now()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("session save: failed to create directories: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("session save: failed to marshal JSON: %w", err)
	}

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("session save: failed to write file: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("session save: failed to rename temp file: %w", err)
	}

	return nil
}

// Load loads a session from a JSON file.
func Load(path string) (*Session, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("session load: failed to open file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("session load: failed to read file: %w", err)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("session load: failed to unmarshal JSON: %w", err)
	}

	return &s, nil
}
