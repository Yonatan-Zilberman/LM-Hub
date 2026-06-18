package platform

import (
	"os"
	"path/filepath"
)

// WindowsPlatform implements Platform for Windows.
type WindowsPlatform struct{}

// NewWindowsPlatform creates a WindowsPlatform instance.
func NewWindowsPlatform() *WindowsPlatform {
	return &WindowsPlatform{}
}

// DefaultShell returns the default shell for Windows (powershell).
func (w *WindowsPlatform) DefaultShell() string {
	return "powershell"
}

// ConfigDir returns the Windows APPDATA path.
func (w *WindowsPlatform) ConfigDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "AppData", "Roaming", "lmhub"), nil
	}
	return filepath.Join(appData, "lmhub"), nil
}

// DataDir returns the Windows LOCALAPPDATA path.
func (w *WindowsPlatform) DataDir() (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "AppData", "Local", "lmhub"), nil
	}
	return filepath.Join(localAppData, "lmhub"), nil
}

// DockerSocket returns the Windows default docker socket pipe.
func (w *WindowsPlatform) DockerSocket() string {
	return `\\.\pipe\docker_engine`
}

// ShellArgs returns the executor and arguments for running a command in Windows.
func (w *WindowsPlatform) ShellArgs(cmd string) (string, []string) {
	return "powershell.exe", []string{"-Command", cmd}
}
