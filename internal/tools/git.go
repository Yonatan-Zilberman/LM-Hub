// Package tools implements agent tools for LM Hub.
package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GitStatusTool checks the status of the git repository.
type GitStatusTool struct {
	scopeRoot string
}

// NewGitStatusTool creates a new git_status tool.
func NewGitStatusTool(scopeRoot string) *GitStatusTool {
	return &GitStatusTool{scopeRoot: scopeRoot}
}

// Name returns the tool name.
func (t *GitStatusTool) Name() string { return "git_status" }

// Description returns the tool description.
func (t *GitStatusTool) Description() string {
	return "Show the working tree status of the git repository."
}

// Schema returns the JSON schema for arguments.
func (t *GitStatusTool) Schema() ToolSchema {
	return ToolSchema{
		Type:       "object",
		Properties: map[string]interface{}{},
	}
}

// Permission returns Safe level.
func (t *GitStatusTool) Permission() PermissionLevel { return Safe }

// Undoable returns false.
func (t *GitStatusTool) Undoable() bool { return false }

// Snapshot is a no-op.
func (t *GitStatusTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, nil
}

// Execute returns porcelain status of the repo.
func (t *GitStatusTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	repo, err := git.PlainOpen(t.scopeRoot)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("open git repo: %w", err).Error()}, nil
	}

	w, err := repo.Worktree()
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("get worktree: %w", err).Error()}, nil
	}

	status, err := w.Status()
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("git status: %w", err).Error()}, nil
	}

	if status.IsClean() {
		return ToolResult{Content: "nothing to commit, working tree clean"}, nil
	}

	return ToolResult{Content: status.String()}, nil
}

// GitDiffTool retrieves repository diffs.
type GitDiffTool struct {
	scopeRoot string
}

// NewGitDiffTool creates a new git_diff tool.
func NewGitDiffTool(scopeRoot string) *GitDiffTool {
	return &GitDiffTool{scopeRoot: scopeRoot}
}

// Name returns the tool name.
func (t *GitDiffTool) Name() string { return "git_diff" }

// Description returns the tool description.
func (t *GitDiffTool) Description() string {
	return "Show changes between commits, commit and working tree, etc."
}

// Schema returns the JSON schema.
func (t *GitDiffTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"staged": map[string]interface{}{"type": "boolean", "description": "Show staged changes only"},
			"file":   map[string]interface{}{"type": "string", "description": "Optional file path to filter diff"},
		},
	}
}

// Permission returns Safe.
func (t *GitDiffTool) Permission() PermissionLevel { return Safe }

// Undoable returns false.
func (t *GitDiffTool) Undoable() bool { return false }

// Snapshot is a no-op.
func (t *GitDiffTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, nil
}

// Execute performs diff. It uses system git binary if available for standard format output.
func (t *GitDiffTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	staged, _ := args["staged"].(bool)
	fileFilter, _ := args["file"].(string)

	// Since go-git lacks a straightforward unified diff output for worktree changes,
	// we check for system git command and use it if possible.
	gitPath, err := exec.LookPath("git")
	if err == nil {
		gitArgs := []string{"diff"}
		if staged {
			gitArgs = append(gitArgs, "--cached")
		}
		if fileFilter != "" {
			gitArgs = append(gitArgs, "--", fileFilter)
		}
		cmd := exec.CommandContext(ctx, gitPath, gitArgs...)
		cmd.Dir = t.scopeRoot
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		runErr := cmd.Run()
		if runErr != nil && stderr.Len() > 0 {
			return ToolResult{IsError: true, Content: stderr.String()}, nil
		}
		return ToolResult{Content: stdout.String()}, nil
	}

	// Fallback to basic go-git status reporting
	repo, err := git.PlainOpen(t.scopeRoot)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("open repo: %w", err).Error()}, nil
	}
	w, err := repo.Worktree()
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("worktree: %w", err).Error()}, nil
	}
	status, err := w.Status()
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("status: %w", err).Error()}, nil
	}

	var sb strings.Builder
	sb.WriteString("Git diff fallback (file list only):\n")
	for path, fStat := range status {
		if fileFilter != "" && path != fileFilter {
			continue
		}
		if staged {
			if fStat.Staging == git.Modified || fStat.Staging == git.Added || fStat.Staging == git.Deleted {
				sb.WriteString(fmt.Sprintf("Staged: %c %s\n", fStat.Staging, path))
			}
		} else {
			if fStat.Worktree == git.Modified || fStat.Worktree == git.Deleted || fStat.Worktree == git.Untracked {
				sb.WriteString(fmt.Sprintf("Modified: %c %s\n", fStat.Worktree, path))
			}
		}
	}
	return ToolResult{Content: sb.String()}, nil
}

// GitAddTool stages files.
type GitAddTool struct {
	scopeRoot string
}

// NewGitAddTool creates a new git_add tool.
func NewGitAddTool(scopeRoot string) *GitAddTool {
	return &GitAddTool{scopeRoot: scopeRoot}
}

// Name returns the tool name.
func (t *GitAddTool) Name() string { return "git_add" }

// Description returns the tool description.
func (t *GitAddTool) Description() string {
	return "Add file contents to the staging index."
}

// Schema returns the JSON schema.
func (t *GitAddTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"paths": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "List of paths to stage",
			},
		},
		Required: []string{"paths"},
	}
}

// Permission returns Warn.
func (t *GitAddTool) Permission() PermissionLevel { return Warn }

// Undoable returns true.
func (t *GitAddTool) Undoable() bool { return true }

// Snapshot records the action inverse (restore --staged).
func (t *GitAddTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	paths := getSliceStrings(args["paths"])
	inversePaths := make([]interface{}, len(paths))
	for i, p := range paths {
		inversePaths[i] = p
	}

	return UndoRecord{
		ToolName:  t.Name(),
		Args:      args,
		InverseOp: "git_restore_staged",
		InverseArgs: map[string]interface{}{
			"paths": inversePaths,
		},
		Timestamp:   time.Now(),
		Description: fmt.Sprintf("Staged %d file(s): %s", len(paths), strings.Join(paths, ", ")),
	}, nil
}

// Execute stages the specified files.
func (t *GitAddTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	paths := getSliceStrings(args["paths"])
	repo, err := git.PlainOpen(t.scopeRoot)
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("open repo: %w", err).Error()}, nil
	}
	w, err := repo.Worktree()
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("worktree: %w", err).Error()}, nil
	}

	var added []string
	for _, p := range paths {
		resolved, err := PathInScope(p, t.scopeRoot)
		if err != nil {
			return ToolResult{IsError: true, Content: fmt.Sprintf("path scope error: %s", err.Error())}, nil
		}
		evalScope, err := filepath.EvalSymlinks(t.scopeRoot)
		if err != nil {
			evalScope = t.scopeRoot
		}
		rel, err := filepath.Rel(evalScope, resolved)
		if err != nil {
			return ToolResult{IsError: true, Content: fmt.Errorf("rel path: %w", err).Error()}, nil
		}
		_, err = w.Add(rel)
		if err != nil {
			return ToolResult{IsError: true, Content: fmt.Errorf("add %s: %w", rel, err).Error()}, nil
		}
		added = append(added, rel)
	}

	return ToolResult{
		Content: fmt.Sprintf("Successfully staged files: %s", strings.Join(added, ", ")),
	}, nil
}

// GitRestoreStagedTool is a helper tool used to unstage files.
type GitRestoreStagedTool struct {
	scopeRoot string
}

// NewGitRestoreStagedTool creates a new git_restore_staged tool.
func NewGitRestoreStagedTool(scopeRoot string) *GitRestoreStagedTool {
	return &GitRestoreStagedTool{scopeRoot: scopeRoot}
}

// Name returns the name.
func (t *GitRestoreStagedTool) Name() string { return "git_restore_staged" }

// Description returns the description.
func (t *GitRestoreStagedTool) Description() string {
	return "Unstages files (helper for undo git_add)."
}

// Schema returns the schema.
func (t *GitRestoreStagedTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"paths": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
		},
		Required: []string{"paths"},
	}
}

// Permission returns Safe.
func (t *GitRestoreStagedTool) Permission() PermissionLevel { return Safe }

// Undoable returns false.
func (t *GitRestoreStagedTool) Undoable() bool { return false }

// Snapshot is a no-op.
func (t *GitRestoreStagedTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, nil
}

// Execute resets staging index for specified files.
func (t *GitRestoreStagedTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	paths := getSliceStrings(args["paths"])
	gitPath, err := exec.LookPath("git")
	if err == nil {
		gitArgs := []string{"restore", "--staged"}
		for _, p := range paths {
			gitArgs = append(gitArgs, p)
		}
		cmd := exec.CommandContext(ctx, gitPath, gitArgs...)
		cmd.Dir = t.scopeRoot
		output, err := cmd.CombinedOutput()
		if err != nil {
			return ToolResult{IsError: true, Content: fmt.Sprintf("git restore failed: %v\nOutput: %s", err, string(output))}, nil
		}
		return ToolResult{Content: "Successfully unstaged files."}, nil
	}

	// Go-git fallback (requires full staging area reset to HEAD)
	repo, err := git.PlainOpen(t.scopeRoot)
	if err != nil {
		return ToolResult{IsError: true, Content: err.Error()}, nil
	}
	w, err := repo.Worktree()
	if err != nil {
		return ToolResult{IsError: true, Content: err.Error()}, nil
	}
	head, err := repo.Head()
	if err != nil {
		// No commits yet, just reset index to empty
		return ToolResult{Content: "No HEAD commit to reset staging area, consider manual revert."}, nil
	}
	err = w.Reset(&git.ResetOptions{
		Commit: head.Hash(),
		Mode:   git.MixedReset,
	})
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("go-git reset: %w", err).Error()}, nil
	}
	return ToolResult{Content: "Unstaged working tree changes."}, nil
}

// GitCommitTool commits staged changes.
type GitCommitTool struct {
	scopeRoot string
}

// NewGitCommitTool creates a new git_commit tool.
func NewGitCommitTool(scopeRoot string) *GitCommitTool {
	return &GitCommitTool{scopeRoot: scopeRoot}
}

// Name returns the name.
func (t *GitCommitTool) Name() string { return "git_commit" }

// Description returns the description.
func (t *GitCommitTool) Description() string {
	return "Record staged changes to the repository history."
}

// Schema returns the schema.
func (t *GitCommitTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"message": map[string]interface{}{"type": "string", "description": "The commit message"},
		},
		Required: []string{"message"},
	}
}

// Permission returns Warn.
func (t *GitCommitTool) Permission() PermissionLevel { return Warn }

// Undoable returns true.
func (t *GitCommitTool) Undoable() bool { return true }

// Snapshot records git_reset_commit inverse.
func (t *GitCommitTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{
		ToolName:    t.Name(),
		Args:        args,
		InverseOp:   "git_reset_commit",
		InverseArgs: map[string]interface{}{},
		Timestamp:   time.Now(),
		Description: fmt.Sprintf("Committed: %s", args["message"]),
	}, nil
}

// Execute performs git commit.
func (t *GitCommitTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	msg, _ := args["message"].(string)
	repo, err := git.PlainOpen(t.scopeRoot)
	if err != nil {
		return ToolResult{IsError: true, Content: err.Error()}, nil
	}
	w, err := repo.Worktree()
	if err != nil {
		return ToolResult{IsError: true, Content: err.Error()}, nil
	}

	commitHash, err := w.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "LMHub Agent",
			Email: "agent@lmhub.local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("git commit: %w", err).Error()}, nil
	}

	return ToolResult{
		Content: fmt.Sprintf("Successfully committed. Commit Hash: %s", commitHash.String()),
	}, nil
}

// GitResetCommitTool is a helper tool to reset the last commit (undo git_commit).
type GitResetCommitTool struct {
	scopeRoot string
}

// NewGitResetCommitTool creates a new git_reset_commit tool.
func NewGitResetCommitTool(scopeRoot string) *GitResetCommitTool {
	return &GitResetCommitTool{scopeRoot: scopeRoot}
}

// Name returns name.
func (t *GitResetCommitTool) Name() string { return "git_reset_commit" }

// Description returns description.
func (t *GitResetCommitTool) Description() string {
	return "Resets the HEAD commit to HEAD~1 while keeping changes (helper for undo git_commit)."
}

// Schema returns schema.
func (t *GitResetCommitTool) Schema() ToolSchema {
	return ToolSchema{
		Type:       "object",
		Properties: map[string]interface{}{},
	}
}

// Permission returns Safe.
func (t *GitResetCommitTool) Permission() PermissionLevel { return Safe }

// Undoable returns false.
func (t *GitResetCommitTool) Undoable() bool { return false }

// Snapshot is a no-op.
func (t *GitResetCommitTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, nil
}

// Execute performs git reset --soft HEAD~1.
func (t *GitResetCommitTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	repo, err := git.PlainOpen(t.scopeRoot)
	if err != nil {
		return ToolResult{IsError: true, Content: err.Error()}, nil
	}

	head, err := repo.Head()
	if err != nil {
		return ToolResult{IsError: true, Content: err.Error()}, nil
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return ToolResult{IsError: true, Content: err.Error()}, nil
	}

	if len(commit.ParentHashes) == 0 {
		return ToolResult{IsError: true, Content: "No parent commit exists to reset to."}, nil
	}

	w, err := repo.Worktree()
	if err != nil {
		return ToolResult{IsError: true, Content: err.Error()}, nil
	}

	parentHash := commit.ParentHashes[0]
	err = w.Reset(&git.ResetOptions{
		Commit: parentHash,
		Mode:   git.SoftReset,
	})
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Errorf("reset to parent: %w", err).Error()}, nil
	}

	return ToolResult{Content: "Successfully reset last commit (soft). changes are kept in index."}, nil
}

// GitLogTool shows git logs.
type GitLogTool struct {
	scopeRoot string
}

// NewGitLogTool creates a new git_log tool.
func NewGitLogTool(scopeRoot string) *GitLogTool {
	return &GitLogTool{scopeRoot: scopeRoot}
}

// Name returns the name.
func (t *GitLogTool) Name() string { return "git_log" }

// Description returns the description.
func (t *GitLogTool) Description() string {
	return "Show commit logs."
}

// Schema returns the schema.
func (t *GitLogTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"n":       map[string]interface{}{"type": "integer", "description": "Number of commits to show"},
			"oneline": map[string]interface{}{"type": "boolean", "description": "Display one line per commit"},
		},
	}
}

// Permission returns Safe.
func (t *GitLogTool) Permission() PermissionLevel { return Safe }

// Undoable returns false.
func (t *GitLogTool) Undoable() bool { return false }

// Snapshot is a no-op.
func (t *GitLogTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, nil
}

// Execute retrieves logs.
func (t *GitLogTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	var limit int
	if nVal, ok := args["n"]; ok {
		if nFloat, ok := nVal.(float64); ok {
			limit = int(nFloat)
		} else if nInt, ok := nVal.(int); ok {
			limit = nInt
		}
	}
	oneline, _ := args["oneline"].(bool)

	repo, err := git.PlainOpen(t.scopeRoot)
	if err != nil {
		return ToolResult{IsError: true, Content: err.Error()}, nil
	}

	cIter, err := repo.Log(&git.LogOptions{})
	if err != nil {
		return ToolResult{IsError: true, Content: err.Error()}, nil
	}
	defer cIter.Close()

	var sb strings.Builder
	count := 0
	err = cIter.ForEach(func(c *object.Commit) error {
		if limit > 0 && count >= limit {
			return filepath.SkipAll
		}
		count++

		if oneline {
			msg := strings.Split(c.Message, "\n")[0]
			sb.WriteString(fmt.Sprintf("%s %s\n", c.Hash.String()[:8], msg))
		} else {
			sb.WriteString(fmt.Sprintf("commit %s\nAuthor: %s <%s>\nDate:   %s\n\n    %s\n\n",
				c.Hash.String(), c.Author.Name, c.Author.Email, c.Author.When.Format(time.RFC1123),
				strings.ReplaceAll(c.Message, "\n", "\n    ")))
		}
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return ToolResult{IsError: true, Content: err.Error()}, nil
	}

	if sb.Len() == 0 {
		return ToolResult{Content: "No commits found."}, nil
	}

	return ToolResult{Content: sb.String()}, nil
}

// GitBranchTool manages branches.
type GitBranchTool struct {
	scopeRoot string
}

// NewGitBranchTool creates a new git_branch tool.
func NewGitBranchTool(scopeRoot string) *GitBranchTool {
	return &GitBranchTool{scopeRoot: scopeRoot}
}

// Name returns the name.
func (t *GitBranchTool) Name() string { return "git_branch" }

// Description returns the description.
func (t *GitBranchTool) Description() string {
	return "List, create, checkout, or delete branches."
}

// Schema returns the schema.
func (t *GitBranchTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []interface{}{"list", "create", "switch", "delete"},
				"description": "Action to perform",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Branch name required for create, switch, delete",
			},
		},
		Required: []string{"action"},
	}
}

// Permission returns default permission level.
func (t *GitBranchTool) Permission() PermissionLevel { return Safe }

// Undoable returns default.
func (t *GitBranchTool) Undoable() bool { return true }

// Snapshot records branch delete inverse for branch create.
func (t *GitBranchTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	action, _ := args["action"].(string)
	name, _ := args["name"].(string)

	if action == "create" && name != "" {
		return UndoRecord{
			ToolName:  t.Name(),
			Args:      args,
			InverseOp: "git_branch",
			InverseArgs: map[string]interface{}{
				"action": "delete",
				"name":   name,
			},
			Timestamp:   time.Now(),
			Description: fmt.Sprintf("Created branch %s", name),
		}, nil
	}
	return UndoRecord{}, nil
}

// Execute performs branch actions.
func (t *GitBranchTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	action, _ := args["action"].(string)
	name, _ := args["name"].(string)

	repo, err := git.PlainOpen(t.scopeRoot)
	if err != nil {
		return ToolResult{IsError: true, Content: err.Error()}, nil
	}

	switch action {
	case "list":
		bIter, err := repo.Branches()
		if err != nil {
			return ToolResult{IsError: true, Content: err.Error()}, nil
		}
		defer bIter.Close()

		head, err := repo.Head()
		var activeHash plumbing.Hash
		if err == nil {
			activeHash = head.Hash()
		}

		var sb strings.Builder
		err = bIter.ForEach(func(ref *plumbing.Reference) error {
			prefix := "  "
			if ref.Hash() == activeHash {
				prefix = "* "
			}
			sb.WriteString(fmt.Sprintf("%s%s\n", prefix, ref.Name().Short()))
			return nil
		})
		if err != nil {
			return ToolResult{IsError: true, Content: err.Error()}, nil
		}
		return ToolResult{Content: sb.String()}, nil

	case "create":
		if name == "" {
			return ToolResult{IsError: true, Content: "branch name is required for create"}, nil
		}
		head, err := repo.Head()
		if err != nil {
			return ToolResult{IsError: true, Content: fmt.Errorf("get head: %w", err).Error()}, nil
		}
		refName := plumbing.NewBranchReferenceName(name)
		ref := plumbing.NewHashReference(refName, head.Hash())
		err = repo.Storer.SetReference(ref)
		if err != nil {
			return ToolResult{IsError: true, Content: fmt.Errorf("create branch reference: %w", err).Error()}, nil
		}
		return ToolResult{Content: fmt.Sprintf("Successfully created branch %s", name)}, nil

	case "switch":
		if name == "" {
			return ToolResult{IsError: true, Content: "branch name is required for switch"}, nil
		}
		w, err := repo.Worktree()
		if err != nil {
			return ToolResult{IsError: true, Content: err.Error()}, nil
		}
		err = w.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(name),
		})
		if err != nil {
			return ToolResult{IsError: true, Content: fmt.Errorf("checkout branch: %w", err).Error()}, nil
		}
		return ToolResult{Content: fmt.Sprintf("Successfully checked out branch %s", name)}, nil

	case "delete":
		if name == "" {
			return ToolResult{IsError: true, Content: "branch name is required for delete"}, nil
		}
		refName := plumbing.NewBranchReferenceName(name)
		err = repo.Storer.RemoveReference(refName)
		if err != nil {
			return ToolResult{IsError: true, Content: fmt.Errorf("delete branch reference: %w", err).Error()}, nil
		}
		return ToolResult{Content: fmt.Sprintf("Successfully deleted branch %s", name)}, nil

	default:
		return ToolResult{IsError: true, Content: fmt.Sprintf("unknown branch action: %s", action)}, nil
	}
}

// GitStashTool manages git stash using command line execution.
type GitStashTool struct {
	scopeRoot string
}

// NewGitStashTool creates a new git_stash tool.
func NewGitStashTool(scopeRoot string) *GitStashTool {
	return &GitStashTool{scopeRoot: scopeRoot}
}

// Name returns the name.
func (t *GitStashTool) Name() string { return "git_stash" }

// Description returns the description.
func (t *GitStashTool) Description() string {
	return "Stash, pop, or list working directory changes."
}

// Schema returns the schema.
func (t *GitStashTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []interface{}{"push", "pop", "list"},
				"description": "Stash action to perform",
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Optional message when pushing to stash",
			},
		},
		Required: []string{"action"},
	}
}

// Permission returns Warn level.
func (t *GitStashTool) Permission() PermissionLevel { return Warn }

// Undoable returns false.
func (t *GitStashTool) Undoable() bool { return false }

// Snapshot is a no-op.
func (t *GitStashTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, nil
}

// Execute performs git stash via system command.
func (t *GitStashTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	action, _ := args["action"].(string)
	message, _ := args["message"].(string)

	gitPath, err := exec.LookPath("git")
	if err != nil {
		return ToolResult{IsError: true, Content: "git executable not found in PATH"}, nil
	}

	gitArgs := []string{"stash"}
	switch action {
	case "push":
		gitArgs = append(gitArgs, "push")
		if message != "" {
			gitArgs = append(gitArgs, "-m", message)
		}
	case "pop":
		gitArgs = append(gitArgs, "pop")
	case "list":
		gitArgs = append(gitArgs, "list")
	default:
		return ToolResult{IsError: true, Content: fmt.Sprintf("invalid stash action: %s", action)}, nil
	}

	cmd := exec.CommandContext(ctx, gitPath, gitArgs...)
	cmd.Dir = t.scopeRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ToolResult{IsError: true, Content: fmt.Sprintf("git stash failed: %v\nOutput: %s", err, string(output))}, nil
	}

	return ToolResult{Content: string(output)}, nil
}

// getSliceStrings safely converts interface value to string slice.
func getSliceStrings(val interface{}) []string {
	if val == nil {
		return nil
	}
	if sliceOfStr, ok := val.([]string); ok {
		return sliceOfStr
	}
	if sliceOfIface, ok := val.([]interface{}); ok {
		res := make([]string, len(sliceOfIface))
		for i, item := range sliceOfIface {
			res[i], _ = item.(string)
		}
		return res
	}
	return nil
}
