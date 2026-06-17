package safety

import (
	"fmt"
	"os"
	"strings"

	"github.com/yonatanzilberman/lmhub/internal/tools"
)

// Classifier evaluates tools and parameters to determine execution safety tiers.
type Classifier struct {
	blocklist []string
}

// NewClassifier creates a new tool execution classifier.
func NewClassifier(blocklist []string) *Classifier {
	return &Classifier{blocklist: blocklist}
}

// Classify inspects the tool name and arguments to determine its effective PermissionLevel.
func (c *Classifier) Classify(t tools.Tool, args map[string]interface{}) tools.PermissionLevel {
	if t.Name() == "run_command" {
		cmdStr, _ := args["cmd"].(string)
		for _, pattern := range c.blocklist {
			if pattern != "" && strings.Contains(cmdStr, pattern) {
				return tools.Dangerous
			}
		}
	}

	if t.Name() == "git_branch" {
		action, _ := args["action"].(string)
		switch action {
		case "delete":
			return tools.Dangerous
		case "create", "switch":
			return tools.Warn
		default:
			return tools.Safe
		}
	}

	if t.Name() == "docker_compose" {
		action, _ := args["action"].(string)
		if action == "down" {
			return tools.Dangerous
		}
		return tools.Warn
	}

	return t.Permission()
}

// FileSizeGuard checks if the file at path exists and exceeds the configured max bytes.
// It returns an error if the file is larger than maxBytes.
func FileSizeGuard(path string, maxBytes int64) error {
	if maxBytes <= 0 {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File does not exist yet, size is 0
		}
		return fmt.Errorf("stat file: %w", err)
	}

	if info.Size() > maxBytes {
		return fmt.Errorf("file size %d bytes exceeds limit of %d bytes", info.Size(), maxBytes)
	}

	return nil
}
