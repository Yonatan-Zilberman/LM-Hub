package platform

// Platform abstracts OS-specific paths and shells.
type Platform interface {
	DefaultShell() string
	ConfigDir() (string, error)
	DataDir() (string, error)
	DockerSocket() string
}
