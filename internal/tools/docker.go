// Package tools implements agent tools for LM Hub.
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// helper to create docker client from socket path or env
func getDockerCli(socketPath string) (*client.Client, error) {
	var opts []client.Opt
	if socketPath != "" {
		if strings.HasPrefix(socketPath, "unix://") || strings.HasPrefix(socketPath, "npipe://") {
			opts = append(opts, client.WithHost(socketPath))
		} else {
			opts = append(opts, client.WithHost("unix://"+socketPath))
		}
	} else {
		opts = append(opts, client.FromEnv)
	}
	opts = append(opts, client.WithAPIVersionNegotiation())
	return client.NewClientWithOpts(opts...)
}

// DockerPSTool lists Docker containers.
type DockerPSTool struct {
	scopeRoot  string
	socketPath string
}

// NewDockerPSTool creates a new docker_ps tool.
func NewDockerPSTool(scopeRoot, socketPath string) *DockerPSTool {
	return &DockerPSTool{scopeRoot: scopeRoot, socketPath: socketPath}
}

// Name returns the name.
func (t *DockerPSTool) Name() string { return "docker_ps" }

// Description returns description.
func (t *DockerPSTool) Description() string {
	return "List docker containers."
}

// Schema returns schema.
func (t *DockerPSTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"all": map[string]interface{}{"type": "boolean", "description": "Show all containers (default shows running only)"},
		},
	}
}

// Permission returns Safe.
func (t *DockerPSTool) Permission() PermissionLevel { return Safe }

// Undoable returns false.
func (t *DockerPSTool) Undoable() bool { return false }

// Snapshot is a no-op.
func (t *DockerPSTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, nil
}

// Execute lists containers.
func (t *DockerPSTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	all, _ := args["all"].(bool)
	cli, err := getDockerCli(t.socketPath)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("connect to docker: %w", err).Error()}, nil
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: all})
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("list containers: %w", err).Error()}, nil
	}

	if len(containers) == 0 {
		return ToolResult{Content: "No containers found."}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-12s  %-20s  %-20s  %-15s  %-15s\n", "CONTAINER ID", "IMAGE", "COMMAND", "STATUS", "NAMES"))
	for _, c := range containers {
		id := c.ID
		if len(id) > 12 {
			id = id[:12]
		}
		img := c.Image
		if len(img) > 20 {
			img = img[:20]
		}
		cmd := c.Command
		if len(cmd) > 20 {
			cmd = cmd[:20]
		}
		names := strings.Join(c.Names, ", ")
		if len(names) > 15 {
			names = names[:15]
		}
		sb.WriteString(fmt.Sprintf("%-12s  %-20s  %-20s  %-15s  %-15s\n", id, img, cmd, c.Status, names))
	}

	return ToolResult{Content: sb.String()}, nil
}

// DockerLogsTool retrieves logs from a container.
type DockerLogsTool struct {
	scopeRoot  string
	socketPath string
}

// NewDockerLogsTool creates a new docker_logs tool.
func NewDockerLogsTool(scopeRoot, socketPath string) *DockerLogsTool {
	return &DockerLogsTool{scopeRoot: scopeRoot, socketPath: socketPath}
}

// Name returns the name.
func (t *DockerLogsTool) Name() string { return "docker_logs" }

// Description returns the description.
func (t *DockerLogsTool) Description() string {
	return "Retrieve logs of a docker container."
}

// Schema returns schema.
func (t *DockerLogsTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"container": map[string]interface{}{"type": "string", "description": "Container ID or name"},
			"tail":      map[string]interface{}{"type": "string", "description": "Number of lines to show from the end (default 'all')"},
		},
		Required: []string{"container"},
	}
}

// Permission returns Safe.
func (t *DockerLogsTool) Permission() PermissionLevel { return Safe }

// Undoable returns false.
func (t *DockerLogsTool) Undoable() bool { return false }

// Snapshot is a no-op.
func (t *DockerLogsTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, nil
}

// Execute retrieves container logs.
func (t *DockerLogsTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	containerID, _ := args["container"].(string)
	tail, _ := args["tail"].(string)
	if tail == "" {
		tail = "all"
	}

	cli, err := getDockerCli(t.socketPath)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("connect to docker: %w", err).Error()}, nil
	}
	defer cli.Close()

	reader, err := cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
	})
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("get logs: %w", err).Error()}, nil
	}
	defer reader.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("read logs: %w", err).Error()}, nil
	}

	return ToolResult{Content: buf.String()}, nil
}

// DockerExecTool runs commands in containers.
type DockerExecTool struct {
	scopeRoot  string
	socketPath string
}

// NewDockerExecTool creates a new docker_exec tool.
func NewDockerExecTool(scopeRoot, socketPath string) *DockerExecTool {
	return &DockerExecTool{scopeRoot: scopeRoot, socketPath: socketPath}
}

// Name returns name.
func (t *DockerExecTool) Name() string { return "docker_exec" }

// Description returns description.
func (t *DockerExecTool) Description() string {
	return "Run a command inside a running docker container."
}

// Schema returns schema.
func (t *DockerExecTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"container": map[string]interface{}{"type": "string", "description": "Container ID or name"},
			"cmd":       map[string]interface{}{"type": "string", "description": "The command to run"},
		},
		Required: []string{"container", "cmd"},
	}
}

// Permission returns Warn.
func (t *DockerExecTool) Permission() PermissionLevel { return Warn }

// Undoable returns false.
func (t *DockerExecTool) Undoable() bool { return false }

// Snapshot is a no-op.
func (t *DockerExecTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, nil
}

// Execute runs command in container.
func (t *DockerExecTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	containerID, _ := args["container"].(string)
	cmdStr, _ := args["cmd"].(string)

	cli, err := getDockerCli(t.socketPath)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("connect to docker: %w", err).Error()}, nil
	}
	defer cli.Close()

	execCfg := container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"sh", "-c", cmdStr},
	}
	response, err := cli.ContainerExecCreate(ctx, containerID, execCfg)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("create exec instance: %w", err).Error()}, nil
	}

	resp, err := cli.ContainerExecAttach(ctx, response.ID, container.ExecStartOptions{})
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("attach exec instance: %w", err).Error()}, nil
	}
	defer resp.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Reader)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("read exec output: %w", err).Error()}, nil
	}

	return ToolResult{Content: buf.String()}, nil
}

// DockerBuildTool builds an image.
type DockerBuildTool struct {
	scopeRoot  string
	socketPath string
}

// NewDockerBuildTool creates a new docker_build tool.
func NewDockerBuildTool(scopeRoot, socketPath string) *DockerBuildTool {
	return &DockerBuildTool{scopeRoot: scopeRoot, socketPath: socketPath}
}

// Name returns the name.
func (t *DockerBuildTool) Name() string { return "docker_build" }

// Description returns description.
func (t *DockerBuildTool) Description() string {
	return "Build a Docker image from a Dockerfile in the workspace directory."
}

// Schema returns schema.
func (t *DockerBuildTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"context":    map[string]interface{}{"type": "string", "description": "Subdirectory path for build context"},
			"tag":        map[string]interface{}{"type": "string", "description": "Tag name for the built image"},
			"dockerfile": map[string]interface{}{"type": "string", "description": "Optional name of the Dockerfile (defaults to Dockerfile)"},
		},
		Required: []string{"context", "tag"},
	}
}

// Permission returns Warn.
func (t *DockerBuildTool) Permission() PermissionLevel { return Warn }

// Undoable returns false.
func (t *DockerBuildTool) Undoable() bool { return false }

// Snapshot is a no-op.
func (t *DockerBuildTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, nil
}

// Execute builds a docker image. It shells out to system docker for cleaner logging and caching.
func (t *DockerBuildTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	ctxDir, _ := args["context"].(string)
	tag, _ := args["tag"].(string)
	dockerfile, _ := args["dockerfile"].(string)

	resolvedCtx, err := PathInScope(ctxDir, t.scopeRoot)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Sprintf("context path scope error: %s", err.Error())}, nil
	}

	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return ToolResult{IsError: true, Content: "docker executable not found in PATH"}, nil
	}

	buildArgs := []string{"build", "-t", tag}
	if dockerfile != "" {
		resolvedDF, err := PathInScope(dockerfile, t.scopeRoot)
		if err != nil {
			return ToolResult{IsError: true, Content: fmt.Sprintf("dockerfile path scope error: %s", err.Error())}, nil
		}
		buildArgs = append(buildArgs, "-f", resolvedDF)
	}
	buildArgs = append(buildArgs, resolvedCtx)

	cmd := exec.CommandContext(ctx, dockerPath, buildArgs...)
	cmd.Dir = t.scopeRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if runErr != nil {
		return ToolResult{
			IsError: true,
			Content: fmt.Sprintf("build failed: %v\nError:\n%s\nOutput:\n%s", runErr, stderr.String(), stdout.String()),
		}, nil
	}

	return ToolResult{
		Content: fmt.Sprintf("Successfully built image: %s\nOutput:\n%s", tag, stdout.String()),
	}, nil
}

// DockerComposeTool manages compose files.
type DockerComposeTool struct {
	scopeRoot string
}

// NewDockerComposeTool creates a new docker_compose tool.
func NewDockerComposeTool(scopeRoot string) *DockerComposeTool {
	return &DockerComposeTool{scopeRoot: scopeRoot}
}

// Name returns the name.
func (t *DockerComposeTool) Name() string { return "docker_compose" }

// Description returns description.
func (t *DockerComposeTool) Description() string {
	return "Manage docker-compose services (up, down, restart, logs)."
}

// Schema returns schema.
func (t *DockerComposeTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"action":   map[string]interface{}{"type": "string", "enum": []interface{}{"up", "down", "restart", "logs"}, "description": "Compose action"},
			"file":     map[string]interface{}{"type": "string", "description": "Optional compose file path (defaults to docker-compose.yml)"},
			"services": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Optional subset of services"},
		},
		Required: []string{"action"},
	}
}

// Permission returns Warn (down is escalated dynamically to Dangerous in safety classifier).
func (t *DockerComposeTool) Permission() PermissionLevel { return Warn }

// Undoable returns false.
func (t *DockerComposeTool) Undoable() bool { return false }

// Snapshot is a no-op.
func (t *DockerComposeTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, nil
}

// Execute runs compose command.
func (t *DockerComposeTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	action, _ := args["action"].(string)
	file, _ := args["file"].(string)
	servicesVal := getSliceStrings(args["services"])

	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return ToolResult{IsError: true, Content: "docker command not found in PATH"}, nil
	}

	composeArgs := []string{"compose"}
	if file != "" {
		resolvedFile, err := PathInScope(file, t.scopeRoot)
		if err != nil {
			return ToolResult{IsError: true, Content: fmt.Sprintf("compose file scope error: %s", err.Error())}, nil
		}
		composeArgs = append(composeArgs, "-f", resolvedFile)
	}

	switch action {
	case "up":
		composeArgs = append(composeArgs, "up", "-d")
	case "down":
		composeArgs = append(composeArgs, "down")
	case "restart":
		composeArgs = append(composeArgs, "restart")
	case "logs":
		composeArgs = append(composeArgs, "logs")
	default:
		return ToolResult{IsError: true, Content: fmt.Sprintf("invalid compose action: %s", action)}, nil
	}

	if len(servicesVal) > 0 {
		composeArgs = append(composeArgs, servicesVal...)
	}

	cmd := exec.CommandContext(ctx, dockerPath, composeArgs...)
	cmd.Dir = t.scopeRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if runErr != nil {
		return ToolResult{
			IsError: true,
			Content: fmt.Sprintf("docker compose failed: %v\nError:\n%s\nOutput:\n%s", runErr, stderr.String(), stdout.String()),
		}, nil
	}

	return ToolResult{
		Content: fmt.Sprintf("Compose %s execution successful.\nOutput:\n%s", action, stdout.String()),
	}, nil
}

// DockerPullTool pulls an image.
type DockerPullTool struct {
	scopeRoot  string
	socketPath string
}

// NewDockerPullTool creates a new docker_pull tool.
func NewDockerPullTool(scopeRoot, socketPath string) *DockerPullTool {
	return &DockerPullTool{scopeRoot: scopeRoot, socketPath: socketPath}
}

// Name returns name.
func (t *DockerPullTool) Name() string { return "docker_pull" }

// Description returns description.
func (t *DockerPullTool) Description() string {
	return "Pull a docker image from registry."
}

// Schema returns schema.
func (t *DockerPullTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"image": map[string]interface{}{"type": "string", "description": "The image to pull (e.g. ubuntu:latest)"},
		},
		Required: []string{"image"},
	}
}

// Permission returns Warn.
func (t *DockerPullTool) Permission() PermissionLevel { return Warn }

// Undoable returns false.
func (t *DockerPullTool) Undoable() bool { return false }

// Snapshot is aop.
func (t *DockerPullTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, nil
}

// Execute pulls docker image.
func (t *DockerPullTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	imgStr, _ := args["image"].(string)

	cli, err := getDockerCli(t.socketPath)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("connect to docker: %w", err).Error()}, nil
	}
	defer cli.Close()

	reader, err := cli.ImagePull(ctx, imgStr, image.PullOptions{})
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("pull image: %w", err).Error()}, nil
	}
	defer reader.Close()

	// Wait for pull to complete, optionally streaming response.
	// Since we are returning content, let's parse the last message or output.
	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("read pull response: %w", err).Error()}, nil
	}

	// Pull output can be verbose, clean up if needed or return last status line.
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var statusLine string
	if len(lines) > 0 {
		// Try to parse the last JSON object which contains the status
		var statusMsg struct {
			Status string `json:"status"`
			ID     string `json:"id"`
		}
		lastLine := lines[len(lines)-1]
		if json.Unmarshal([]byte(lastLine), &statusMsg) == nil {
			statusLine = fmt.Sprintf("%s %s", statusMsg.Status, statusMsg.ID)
		} else {
			statusLine = lastLine
		}
	}

	return ToolResult{
		Content: fmt.Sprintf("Successfully pulled image %s.\nStatus: %s", imgStr, statusLine),
	}, nil
}
