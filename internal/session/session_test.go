package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yonatanzilberman/lmhub/internal/api"
)

func TestSessionSaveLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lmhub-session-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	s := NewSession("test-session-123", "ask", "qwen-35b")
	s.Messages = append(s.Messages, api.Message{
		Role:    "user",
		Content: "Hello",
	})
	s.InjectedContexts = append(s.InjectedContexts, InjectedContext{
		MessageIndex:   0,
		ProjectContext: "Proj ctx",
		MemoryFacts:    []string{"fact 1"},
		RAGChunks:      []string{"chunk 1"},
	})

	filePath := filepath.Join(tempDir, "ask-session.json")
	err = s.Save(filePath)
	assert.NoError(t, err)

	loaded, err := Load(filePath)
	assert.NoError(t, err)
	assert.Equal(t, s.ID, loaded.ID)
	assert.Equal(t, s.Mode, loaded.Mode)
	assert.Equal(t, s.ModelID, loaded.ModelID)
	assert.Len(t, loaded.Messages, 1)
	assert.Equal(t, "user", loaded.Messages[0].Role)
	assert.Equal(t, "Hello", loaded.Messages[0].Content)
	assert.Len(t, loaded.InjectedContexts, 1)
	assert.Equal(t, "Proj ctx", loaded.InjectedContexts[0].ProjectContext)
}

func TestSessionListCleanup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lmhub-session-list-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	s1 := NewSession("s1", "ask", "qwen")
	err = s1.Save(filepath.Join(tempDir, "s1.json"))
	assert.NoError(t, err)

	// Sleep for a short duration to ensure s2 is strictly newer
	time.Sleep(10 * time.Millisecond)

	s2 := NewSession("s2", "plan", "llama")
	err = s2.Save(filepath.Join(tempDir, "s2.json"))
	assert.NoError(t, err)

	infos, err := List(tempDir)
	assert.NoError(t, err)
	assert.Len(t, infos, 2)
	assert.Equal(t, "s2", infos[0].ID) // s2 is newer
	assert.Equal(t, "s1", infos[1].ID)

	err = CleanupOld(tempDir, 1)
	assert.NoError(t, err)

	infos, err = List(tempDir)
	assert.NoError(t, err)
	assert.Len(t, infos, 1)
	assert.Equal(t, "s2", infos[0].ID)
}
