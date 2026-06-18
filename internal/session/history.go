package session

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Info holds basic session details for indexing/listing.
type Info struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Mode      string    `json:"mode"`
	ModelID   string    `json:"model_id"`
	UpdatedAt time.Time `json:"updated_at"`
}

// List scans a folder for session files and returns them sorted by last updated time (descending).
func List(dir string) ([]Info, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("session list: failed to read directory: %w", err)
	}

	var infos []Info
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		s, err := Load(path)
		if err != nil {
			// Skip corrupt files
			continue
		}

		infos = append(infos, Info{
			ID:        s.ID,
			Path:      path,
			Mode:      s.Mode,
			ModelID:   s.ModelID,
			UpdatedAt: s.UpdatedAt,
		})
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].UpdatedAt.After(infos[j].UpdatedAt)
	})

	return infos, nil
}

// Delete removes a session file by its ID or path.
func Delete(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("session delete: failed to delete file: %w", err)
	}
	return nil
}

// CleanupOld keeps only the newest maxHistory session files.
func CleanupOld(dir string, maxHistory int) error {
	if maxHistory <= 0 {
		return nil
	}

	infos, err := List(dir)
	if err != nil {
		return err
	}

	if len(infos) <= maxHistory {
		return nil
	}

	for i := maxHistory; i < len(infos); i++ {
		_ = os.Remove(infos[i].Path)
	}

	return nil
}
