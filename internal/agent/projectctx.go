package agent

import (
	"fmt"
	"os"
	"path/filepath"
)

// LoadProjectContext reads .lmhub/context.md from the given project root.
// If the file does not exist, it returns an empty string and no error.
// If it exists, it reads the content and truncates it to fit the maxTokens budget.
func LoadProjectContext(projectRoot string, cm *ContextManager, maxTokens int) (string, error) {
	path := filepath.Join(projectRoot, ".lmhub", "context.md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read project context file: %w", err)
	}

	content := string(data)
	if maxTokens > 0 {
		content = cm.TruncateToTokens(content, maxTokens)
	}
	return content, nil
}
