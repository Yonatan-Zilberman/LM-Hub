package build

import (
	"sync"
	"time"

	"github.com/yonatanzilberman/lmhub/internal/modes/plan"
	"github.com/yonatanzilberman/lmhub/internal/tools"
)

// CommandRecord logs shell commands executed during a build session.
type CommandRecord struct {
	Cmd      string        `json:"cmd"`
	ExitCode int           `json:"exit_code"`
	Duration time.Duration `json:"duration"`
	Stdout   string        `json:"stdout,omitempty"`
	Stderr   string        `json:"stderr,omitempty"`
}

// ToolCallRecord logs individual tool calls executed during a build session.
type ToolCallRecord struct {
	ToolName  string                 `json:"tool_name"`
	Args      map[string]interface{} `json:"args"`
	Result    string                 `json:"result"`
	Duration  time.Duration          `json:"duration"`
	Undoable  bool                   `json:"undoable"`
	Timestamp time.Time              `json:"timestamp"`
}

// BuildSession encapsulates state for the duration of a Build Mode task.
type BuildSession struct {
	mu              sync.RWMutex
	ScopeRoot       string            `json:"scope_root"`
	FilesModified   []string          `json:"files_modified"`
	CommandsRun     []CommandRecord   `json:"commands_run"`
	ToolCallHistory []ToolCallRecord  `json:"tool_call_history"`
	Iteration       int               `json:"iteration"`
	MaxIterations   int               `json:"max_iterations"`
	StartedAt       time.Time         `json:"started_at"`
	UndoStack       *tools.UndoStack  `json:"-"`
	PlanRef         *plan.Plan        `json:"-"`
	CurrentStep     int               `json:"current_step"`
}

// NewBuildSession creates a new initialized build session.
func NewBuildSession(scopeRoot string, maxIterations int) *BuildSession {
	return &BuildSession{
		ScopeRoot:     scopeRoot,
		MaxIterations: maxIterations,
		StartedAt:     time.Now(),
		UndoStack:     tools.NewUndoStack(),
		CurrentStep:   -1,
	}
}

// AddFileModified appends a file path to the list of modified files if not already tracked.
func (s *BuildSession) AddFileModified(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, f := range s.FilesModified {
		if f == path {
			return
		}
	}
	s.FilesModified = append(s.FilesModified, path)
}

// AddCommandRun appends a new CommandRecord to the list.
func (s *BuildSession) AddCommandRun(cmd string, exitCode int, duration time.Duration, stdout, stderr string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.CommandsRun = append(s.CommandsRun, CommandRecord{
		Cmd:      cmd,
		ExitCode: exitCode,
		Duration: duration,
		Stdout:   stdout,
		Stderr:   stderr,
	})
}

// AddToolCall appends a new ToolCallRecord to the list.
func (s *BuildSession) AddToolCall(name string, args map[string]interface{}, result string, duration time.Duration, undoable bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ToolCallHistory = append(s.ToolCallHistory, ToolCallRecord{
		ToolName:  name,
		Args:      args,
		Result:    result,
		Duration:  duration,
		Undoable:  undoable,
		Timestamp: time.Now(),
	})
}

// IncrementIteration increments and returns the new iteration count.
func (s *BuildSession) IncrementIteration() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Iteration++
	return s.Iteration
}

// GetIteration returns the current iteration count.
func (s *BuildSession) GetIteration() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Iteration
}

// GetFilesModified returns a copy of the list of modified files.
func (s *BuildSession) GetFilesModified() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copyList := make([]string, len(s.FilesModified))
	copy(copyList, s.FilesModified)
	return copyList
}

// GetCommandsRun returns a copy of the list of commands run.
func (s *BuildSession) GetCommandsRun() []CommandRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copyList := make([]CommandRecord, len(s.CommandsRun))
	copy(copyList, s.CommandsRun)
	return copyList
}

// GetToolCallHistory returns a copy of the tool call history.
func (s *BuildSession) GetToolCallHistory() []ToolCallRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copyList := make([]ToolCallRecord, len(s.ToolCallHistory))
	copy(copyList, s.ToolCallHistory)
	return copyList
}

// SetPlan sets the loaded plan for this session.
func (s *BuildSession) SetPlan(p *plan.Plan) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PlanRef = p
	s.CurrentStep = 0
}

// GetPlan returns the loaded plan.
func (s *BuildSession) GetPlan() *plan.Plan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.PlanRef
}

// SetCurrentStep updates the active plan step index.
func (s *BuildSession) SetCurrentStep(idx int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentStep = idx
}

// GetCurrentStep returns the active plan step index.
func (s *BuildSession) GetCurrentStep() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CurrentStep
}
