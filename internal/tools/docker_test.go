package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// isDockerRunning attempts to connect to the docker daemon to see if it is running.
func isDockerRunning(socketPath string) bool {
	cli, err := getDockerCli(socketPath)
	if err != nil {
		return false
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = cli.Ping(ctx)
	return err == nil
}

// TestDockerGracefulOffline verifies that the Docker tools return error results
// gracefully instead of crashing when the Docker daemon is unreachable.
func TestDockerGracefulOffline(t *testing.T) {
	ctx := context.Background()
	dummySocket := "/var/run/nonexistent_docker_dummy.sock"

	psTool := NewDockerPSTool("/tmp", dummySocket)
	res, err := psTool.Execute(ctx, map[string]interface{}{})
	assert.NoError(t, err)
	assert.True(t, res.IsError)
	assert.Contains(t, strings.ToLower(res.Content), "docker daemon")

	logsTool := NewDockerLogsTool("/tmp", dummySocket)
	resLogs, err := logsTool.Execute(ctx, map[string]interface{}{
		"container": "dummy-container",
	})
	assert.NoError(t, err)
	assert.True(t, resLogs.IsError)

	execTool := NewDockerExecTool("/tmp", dummySocket)
	resExec, err := execTool.Execute(ctx, map[string]interface{}{
		"container": "dummy",
		"cmd":       "ls",
	})
	assert.NoError(t, err)
	assert.True(t, resExec.IsError)
}

// TestDockerLiveSuite runs actual container list calls if a live Docker daemon is running.
func TestDockerLiveSuite(t *testing.T) {
	socketPath := "" // Use default socket path from env
	if !isDockerRunning(socketPath) {
		t.Skip("Docker daemon is not running, skipping live Docker test suite")
	}

	ctx := context.Background()
	psTool := NewDockerPSTool("/tmp", socketPath)

	// Test docker_ps
	res, err := psTool.Execute(ctx, map[string]interface{}{
		"all": true,
	})
	assert.NoError(t, err)
	assert.False(t, res.IsError)
	assert.NotEmpty(t, res.Content)
}
