package platform

import (
	"os"
	"path/filepath"
)

// DarwinPlatform implements Platform for macOS.
type DarwinPlatform struct{}

// NewDarwinPlatform creates a DarwinPlatform instance.
func NewDarwinPlatform() *DarwinPlatform {
	return &DarwinPlatform{}
}

// DefaultShell returns the default shell for macOS (zsh).
func (d *DarwinPlatform) DefaultShell() string {
	return "zsh"
}

// ConfigDir returns the macOS config path: ~/.config/lmhub/
func (d *DarwinPlatform) ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "lmhub"), nil
}

// DataDir returns the macOS data path: ~/.local/share/lmhub/
func (d *DarwinPlatform) DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "lmhub"), nil
}

// DockerSocket returns the macOS default docker socket path.
func (d *DarwinPlatform) DockerSocket() string {
	return "/var/run/docker.sock"
}
