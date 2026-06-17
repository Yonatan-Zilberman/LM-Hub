package tools

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRepo creates a temporary directory and initializes a basic Git repository
// with a first commit so HEAD is available.
func setupTestRepo(t *testing.T) (string, *git.Repository) {
	tempDir, err := ioutil.TempDir("", "lmhub-git-test-*")
	require.NoError(t, err)

	repo, err := git.PlainInit(tempDir, false)
	require.NoError(t, err)

	w, err := repo.Worktree()
	require.NoError(t, err)

	// Create an initial file and commit it so HEAD reference exists
	testFile := filepath.Join(tempDir, "init.txt")
	err = ioutil.WriteFile(testFile, []byte("initial content"), 0644)
	require.NoError(t, err)

	_, err = w.Add("init.txt")
	require.NoError(t, err)

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
		},
	})
	require.NoError(t, err)

	return tempDir, repo
}

// TestGitStatus verifies status reporting on clean and modified workspaces.
func TestGitStatus(t *testing.T) {
	tempDir, _ := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	tool := NewGitStatusTool(tempDir)
	ctx := context.Background()

	// Initial check: clean workspace
	res, err := tool.Execute(ctx, map[string]interface{}{})
	assert.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, "nothing to commit, working tree clean", res.Content)

	// Modify worktree
	err = ioutil.WriteFile(filepath.Join(tempDir, "init.txt"), []byte("modified content"), 0644)
	require.NoError(t, err)

	res, err = tool.Execute(ctx, map[string]interface{}{})
	assert.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Contains(t, res.Content, "M init.txt")
}

// TestGitAddAndUndo tests staging files and unstaging via undo helpers.
func TestGitAddAndUndo(t *testing.T) {
	tempDir, _ := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	addTool := NewGitAddTool(tempDir)
	restoreTool := NewGitRestoreStagedTool(tempDir)

	// Create new untracked file
	err := ioutil.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("testing git add"), 0644)
	require.NoError(t, err)

	// Stage it
	res, err := addTool.Execute(ctx, map[string]interface{}{
		"paths": []interface{}{"test.txt"},
	})
	assert.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Contains(t, res.Content, "Successfully staged files: test.txt")

	// Get undo record from snapshot
	rec, err := addTool.Snapshot(ctx, map[string]interface{}{
		"paths": []interface{}{"test.txt"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "git_restore_staged", rec.InverseOp)

	// Unstage it via the inverse operation (mimics undo)
	resUndo, err := restoreTool.Execute(ctx, rec.InverseArgs)
	assert.NoError(t, err)
	assert.False(t, resUndo.IsError)

	// Confirm it's back to untracked (i.e. unstaged)
	statusTool := NewGitStatusTool(tempDir)
	resStatus, err := statusTool.Execute(ctx, map[string]interface{}{})
	assert.NoError(t, err)
	assert.Contains(t, resStatus.Content, "?? test.txt")
}

// TestGitCommitAndUndo tests committing staged files and reverting via undo.
func TestGitCommitAndUndo(t *testing.T) {
	tempDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	w, err := repo.Worktree()
	require.NoError(t, err)

	err = ioutil.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("testing commit"), 0644)
	require.NoError(t, err)
	_, err = w.Add("test.txt")
	require.NoError(t, err)

	commitTool := NewGitCommitTool(tempDir)
	resetTool := NewGitResetCommitTool(tempDir)

	// Snapshot commit
	args := map[string]interface{}{"message": "Adds test.txt"}
	rec, err := commitTool.Snapshot(ctx, args)
	assert.NoError(t, err)
	assert.Equal(t, "git_reset_commit", rec.InverseOp)

	// Execute commit
	res, err := commitTool.Execute(ctx, args)
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	// Verify commit exists on HEAD
	head, err := repo.Head()
	require.NoError(t, err)
	commit, err := repo.CommitObject(head.Hash())
	require.NoError(t, err)
	assert.Equal(t, "Adds test.txt", commit.Message)

	// Revert the commit via git_reset_commit (undo)
	resUndo, err := resetTool.Execute(ctx, rec.InverseArgs)
	assert.NoError(t, err)
	assert.False(t, resUndo.IsError)

	// Verify HEAD has reverted to parent commit
	newHead, err := repo.Head()
	require.NoError(t, err)
	assert.NotEqual(t, head.Hash(), newHead.Hash())

	newCommit, err := repo.CommitObject(newHead.Hash())
	require.NoError(t, err)
	assert.Equal(t, "Initial commit", newCommit.Message)
}

// TestGitLog tests commit log format.
func TestGitLog(t *testing.T) {
	tempDir, _ := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	logTool := NewGitLogTool(tempDir)

	res, err := logTool.Execute(ctx, map[string]interface{}{
		"n":       1,
		"oneline": true,
	})
	assert.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Contains(t, res.Content, "Initial commit")
}

// TestGitBranch verifies branch listings, creation, switching, and deletion.
func TestGitBranch(t *testing.T) {
	tempDir, repo := setupTestRepo(t)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	branchTool := NewGitBranchTool(tempDir)

	// Create branch
	res, err := branchTool.Execute(ctx, map[string]interface{}{
		"action": "create",
		"name":   "feature-x",
	})
	assert.NoError(t, err)
	assert.False(t, res.IsError)

	// List branches
	resList, err := branchTool.Execute(ctx, map[string]interface{}{
		"action": "list",
	})
	assert.NoError(t, err)
	assert.Contains(t, resList.Content, "feature-x")
	assert.Contains(t, resList.Content, "* master") // default branch in plainInit is master or main

	// Switch branch
	resSwitch, err := branchTool.Execute(ctx, map[string]interface{}{
		"action": "switch",
		"name":   "feature-x",
	})
	assert.NoError(t, err)
	assert.False(t, resSwitch.IsError)

	// Confirm switched
	head, err := repo.Head()
	require.NoError(t, err)
	assert.Equal(t, plumbing.NewBranchReferenceName("feature-x"), head.Name())

	// Delete branch (checkout master first)
	_, _ = branchTool.Execute(ctx, map[string]interface{}{
		"action": "switch",
		"name":   "master",
	})
	resDel, err := branchTool.Execute(ctx, map[string]interface{}{
		"action": "delete",
		"name":   "feature-x",
	})
	assert.NoError(t, err)
	assert.False(t, resDel.IsError)

	// Confirm deleted from list
	resListAfter, _ := branchTool.Execute(ctx, map[string]interface{}{
		"action": "list",
	})
	assert.NotContains(t, resListAfter.Content, "feature-x")
}
