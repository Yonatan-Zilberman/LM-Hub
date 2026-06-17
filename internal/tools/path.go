package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PathInScope resolves the given path relative to scopeRoot, evaluates symlinks,
// and ensures the resolved path does not escape the scopeRoot directory.
// It returns the cleaned absolute path or an error if the path escapes scope.
func PathInScope(path, scopeRoot string) (string, error) {
	if scopeRoot == "" {
		return "", fmt.Errorf("scope root is empty")
	}

	absScope, err := filepath.Abs(scopeRoot)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path of scope root: %w", err)
	}
	absScope = filepath.Clean(absScope)

	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath = filepath.Clean(filepath.Join(absScope, path))
	}

	// Try resolving symlinks on the path.
	// If the file/dir does not exist, filepath.EvalSymlinks will fail.
	// In that case, we resolve symlinks on the closest existing parent directory.
	evalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		dir := filepath.Dir(absPath)
		evalDir, err := filepath.EvalSymlinks(dir)
		if err == nil {
			evalPath = filepath.Join(evalDir, filepath.Base(absPath))
		} else {
			evalPath = absPath
		}
	}

	evalScope, err := filepath.EvalSymlinks(absScope)
	if err != nil {
		evalScope = absScope
	}

	// Check if the evaluated path is within evaluated scope root.
	rel, err := filepath.Rel(evalScope, evalPath)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative path: %w", err)
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes scope root: %s (resolved: %s, scope: %s)", path, evalPath, evalScope)
	}

	return evalPath, nil
}
