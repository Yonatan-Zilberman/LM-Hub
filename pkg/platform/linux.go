package platform

import (
	"os"
	"path/filepath"
)

// LinuxPlatform implements Platform for Linux.
type LinuxPlatform struct{}

// NewLinuxPlatform creates a LinuxPlatform instance.
func NewLinuxPlatform() *LinuxPlatform {
	return &LinuxPlatform{}
}

// DefaultShell returns the default shell for Linux (bash).
func (l *LinuxPlatform) DefaultShell() string {
	return "bash"
}

// ConfigDir returns the Linux config path, adhering to XDG specs.
func (l *LinuxPlatform) ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "lmhub"), nil
	}
	return filepath.Join(home, ".config", "lmhub"), nil
}

// DataDir returns the Linux data path, adhering to XDG specs.
func (l *LinuxPlatform) DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "lmhub"), nil
	}
	return filepath.Join(home, ".local", "share", "lmhub"), nil
}

// DockerSocket returns the Linux default docker socket path.
func (l *LinuxPlatform) DockerSocket() string {
	return "/var/run/docker.sock"
}

// ShellArgs returns the executor and arguments for running a command in Linux.
func (l *LinuxPlatform) ShellArgs(cmd string) (string, []string) {
	return "/bin/bash", []string{"-c", cmd}
}
