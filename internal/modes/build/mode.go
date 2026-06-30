package build

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/yonatanzilberman/lmhub/internal/agent"
	"github.com/yonatanzilberman/lmhub/internal/api"
	"github.com/yonatanzilberman/lmhub/internal/config"
	"github.com/yonatanzilberman/lmhub/internal/memory"
	"github.com/yonatanzilberman/lmhub/internal/modelmanager"
	"github.com/yonatanzilberman/lmhub/internal/modes/plan"
	"github.com/yonatanzilberman/lmhub/internal/safety"
	"github.com/yonatanzilberman/lmhub/internal/tools"
	"github.com/yonatanzilberman/lmhub/internal/rag"
)

// AgentStepMsg is sent to the Bubbletea application to update the UI on agent activity.
type AgentStepMsg struct {
	StepType  string                 // "thought", "tool_call", "tool_result", "finished", "error", "warning", "pause"
	Content   string                 // Thought text, tool results, errors
	ToolName  string                 // Name of the tool being called
	ToolArgs  map[string]interface{} // Arguments passed to the tool
	Iteration int                    // Current loop iteration
	Done      bool                   // True if build task is complete
}

// BuildMode runs the autonomous ReAct agent execution loop with tool access.
type BuildMode struct {
	mu                 sync.RWMutex
	client             *api.Client
	modelManager       *modelmanager.Manager
	contextManager     *agent.ContextManager
	budgetManager      *agent.BudgetManager
	cfg                *config.Config
	registry           *tools.Registry
	retriever          *rag.Retriever
	classifier         *safety.Classifier
	history            []api.Message
	session            *BuildSession
	memoryManager      *memory.MemoryManager
	consecutiveErrors  int
	confirmCallback    func(msg safety.ConfirmMsg) bool
	updateCallback     func(msg AgentStepMsg)
	askUserCallback    func(question string) (string, error)
}

// NewBuildMode creates a new BuildMode execution handler.
func NewBuildMode(
	client *api.Client,
	mm *modelmanager.Manager,
	cm *agent.ContextManager,
	bm *agent.BudgetManager,
	cfg *config.Config,
	reg *tools.Registry,
	retriever *rag.Retriever,
	memManager *memory.MemoryManager,
	confirmCB func(msg safety.ConfirmMsg) bool,
	updateCB func(msg AgentStepMsg),
) *BuildMode {
	return &BuildMode{
		client:          client,
		modelManager:    mm,
		contextManager:  cm,
		budgetManager:   bm,
		cfg:             cfg,
		registry:        reg,
		retriever:       retriever,
		classifier:      safety.NewClassifier(cfg.Tools.Shell.Blocklist),
		history:         make([]api.Message, 0),
		memoryManager:   memManager,
		confirmCallback: confirmCB,
		updateCallback:  updateCB,
	}
}

// SetConfirmCallback sets the confirmation query callback.
func (bm *BuildMode) SetConfirmCallback(cb func(msg safety.ConfirmMsg) bool) {
	bm.confirmCallback = cb
}

// SetUpdateCallback sets the update progress callback.
func (bm *BuildMode) SetUpdateCallback(cb func(msg AgentStepMsg)) {
	bm.updateCallback = cb
}

// SetAskUserCallback sets the ask user input query callback.
func (bm *BuildMode) SetAskUserCallback(cb func(question string) (string, error)) {
	bm.askUserCallback = cb
}

// Reset clears session logs and history.
func (bm *BuildMode) Reset() {
	bm.mu.Lock()
	bm.history = make([]api.Message, 0)
	bm.mu.Unlock()
	bm.session = nil
	bm.consecutiveErrors = 0
}

// History returns a copy of the conversation history.
func (bm *BuildMode) History() []api.Message {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	hist := make([]api.Message, len(bm.history))
	copy(hist, bm.history)
	return hist
}

// SetHistory replaces the current conversation history.
func (bm *BuildMode) SetHistory(hist []api.Message) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.history = hist
}

// Session returns the active build session.
func (bm *BuildMode) Session() *BuildSession {
	return bm.session
}

func (bm *BuildMode) summarizeHistory(ctx context.Context, modelID string) error {
	bm.updateCallback(AgentStepMsg{
		StepType: "thought",
		Content:  "Context window is 90% full. Compressing history...",
	})

	bm.mu.Lock()
	messages := make([]api.Message, len(bm.history))
	copy(messages, bm.history)
	bm.mu.Unlock()

	messages = append(messages, api.Message{
		Role:    "user",
		Content: "Summarize the key progress, files modified, shell commands run, and current state from the conversation history above. Keep it concise, under 300 words.",
	})

	req := api.ChatRequest{
		Model:       modelID,
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   500,
	}

	resp, err := bm.client.ChatCompletion(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to generate context summary: %w", err)
	}

	if len(resp.Choices) == 0 {
		return fmt.Errorf("no summary returned")
	}

	summary := resp.Choices[0].Message.Content

	bm.mu.Lock()
	defer bm.mu.Unlock()

	var trailing []api.Message
	if len(bm.history) > 2 {
		trailing = bm.history[len(bm.history)-2:]
	} else {
		trailing = bm.history
	}

	newHistory := []api.Message{
		{
			Role:    "system",
			Content: fmt.Sprintf("Summary of previous conversation history:\n%s", summary),
		},
	}
	newHistory = append(newHistory, trailing...)

	bm.history = newHistory
	return nil
}

// SetSession updates the active session.
func (bm *BuildMode) SetSession(s *BuildSession) {
	bm.session = s
}

// LoadPlan reads and parses a saved plan file.
func (bm *BuildMode) LoadPlan(path string) (*plan.Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read plan file: %w", err)
	}

	p, err := plan.ParsePlanJSON(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse plan: %w", err)
	}

	return p, nil
}

// ExecuteTask runs the ReAct loop in a non-blocking background goroutine.
func (bm *BuildMode) ExecuteTask(ctx context.Context, modelID, task, projectContext, memoryFacts, gitStatus string, temp float64, maxToks int) {
	if bm.session == nil {
		bm.session = NewBuildSession(bm.registry.ScopeRoot(), bm.cfg.Agent.MaxIterations)
	}

	go func() {
		// Retrieve RAG chunks matching the task if enabled
		var ragChunks string
		if bm.retriever != nil && bm.cfg.RAG.Enabled {
			retrieved, err := bm.retriever.Retrieve(ctx, task, bm.cfg.RAG.TopK, bm.cfg.RAG.MaxTokens)
			if err == nil && len(retrieved) > 0 {
				var ragPieces []string
				for _, chunk := range retrieved {
					piece := fmt.Sprintf("[%s:%d-%d]\n%s", chunk.FilePath, chunk.StartLine, chunk.EndLine, chunk.Content)
					ragPieces = append(ragPieces, piece)
				}
				ragChunks = strings.Join(ragPieces, "\n\n")
			}
		}

		// First step: add the user task if history is empty
		bm.mu.Lock()
		histEmpty := len(bm.history) == 0
		if histEmpty {
			if bm.session.PlanRef != nil {
				bm.session.SetCurrentStep(0)
				step := bm.session.PlanRef.Steps[0]
				bm.history = append(bm.history, api.Message{
					Role:    "user",
					Content: fmt.Sprintf("Please execute Step 1 of our plan:\nDescription: %s\nTarget: %s", step.Description, step.Target),
				})
			} else {
				bm.history = append(bm.history, api.Message{
					Role:    "user",
					Content: task,
				})
			}
		}
		bm.mu.Unlock()

		for {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				bm.updateCallback(AgentStepMsg{
					StepType: "finished",
					Content:  "Task cancelled by user.",
					Done:     true,
				})
				return
			default:
			}

			// Check iteration limit
			iter := bm.session.GetIteration()
			if iter >= bm.session.MaxIterations {
				// Pause and ask user if they want to continue
				if bm.confirmCallback != nil {
					bm.updateCallback(AgentStepMsg{
						StepType:  "pause",
						Content:   fmt.Sprintf("Reached max iterations limit (%d). Do you want to continue?", bm.session.MaxIterations),
						Iteration: iter,
					})

					ch := make(chan bool, 1)
					confirmMsg := safety.ConfirmMsg{
						ToolName:     "continue_agent",
						Level:        tools.Dangerous,
						Description:  fmt.Sprintf("Agent reached maximum iterations ceiling (%d). Allow 10 more iterations?", bm.session.MaxIterations),
						ResponseChan: ch,
					}
					approved := bm.confirmCallback(confirmMsg)
					if approved {
						bm.session.MaxIterations += 10
					} else {
						bm.updateCallback(AgentStepMsg{
							StepType: "finished",
							Content:  "Task aborted: Max iteration ceiling reached.",
							Done:     true,
						})
						return
					}
				} else {
					bm.updateCallback(AgentStepMsg{
						StepType: "finished",
						Content:  "Stopped: Max iteration limit reached.",
						Done:     true,
					})
					return
				}
			}

			// Add warnings at iteration thresholds
			if iter == 10 {
				bm.updateCallback(AgentStepMsg{
					StepType:  "warning",
					Content:   "Approaching maximum iteration limit (10/15). Consider clarifying task parameters.",
					Iteration: iter,
				})
			}

			// Generate system prompt dynamically
			toolSchemas, _ := bm.registry.SchemaJSON()
			allocation := bm.budgetManager.Allocate(projectContext, memoryFacts, ragChunks)
			systemPrompt := agent.RenderBuildPrompt(bm.session.ScopeRoot, gitStatus, runtime.GOOS, bm.cfg.Tools.Shell.AllowedShells[0], toolSchemas, allocation.ProjectContext, allocation.MemoryFacts, allocation.RAGChunks)

			// Check context limits
			metrics := bm.modelManager.Metrics().Get()
			limit := metrics.ContextLimit
			if limit == 0 {
				limit = 32768 // Safe default
			}

			bm.mu.Lock()
			ctxResult := bm.contextManager.ManageContext(
				bm.history,
				systemPrompt,
				limit,
				bm.cfg.Agent.ContextWarnPct,
				bm.cfg.Agent.ContextTrimPct,
				bm.cfg.Agent.ContextSummarizePct,
			)

			if ctxResult.Action == agent.ContextHardStop {
				bm.mu.Unlock()
				bm.updateCallback(AgentStepMsg{
					StepType: "error",
					Content:  fmt.Sprintf("Context limit reached (hard-stop): %s", ctxResult.Log),
					Done:     true,
				})
				return
			}

			if ctxResult.Action == agent.ContextNeedsSummarize {
				bm.mu.Unlock()
				err := bm.summarizeHistory(ctx, modelID)
				bm.mu.Lock()
				if err != nil {
					bm.updateCallback(AgentStepMsg{
						StepType: "thought",
						Content:  fmt.Sprintf("Summarization failed: %v. Trimming instead.", err),
					})
					trimRes := bm.contextManager.ManageContext(
						bm.history,
						systemPrompt,
						limit,
						bm.cfg.Agent.ContextWarnPct,
						bm.cfg.Agent.ContextTrimPct,
						0, // Disable summarization check to force fallback to trim/ok
					)
					if trimRes.Action == agent.ContextTrimmed {
						bm.history = trimRes.Messages
					}
				}
			} else if ctxResult.Action == agent.ContextTrimmed {
				bm.history = ctxResult.Messages
			}

			// Prepare request
			reqMessages := []api.Message{
				{
					Role:    "system",
					Content: systemPrompt,
				},
			}
			reqMessages = append(reqMessages, bm.history...)
			bm.mu.Unlock()

			resolvedTemp := temp
			if resolvedTemp == 0 {
				resolvedTemp = bm.cfg.ModeInference.Build.Temperature
			}
			if resolvedTemp == 0 {
				resolvedTemp = bm.cfg.Inference.Temperature
			}
			if resolvedTemp == 0 {
				resolvedTemp = 0.5
			}

			resolvedMaxToks := maxToks
			if resolvedMaxToks == 0 {
				resolvedMaxToks = bm.cfg.ModeInference.Build.MaxTokens
			}
			if resolvedMaxToks == 0 {
				resolvedMaxToks = bm.cfg.Inference.MaxTokens
			}
			if resolvedMaxToks == 0 {
				resolvedMaxToks = 8192
			}

			req := api.ChatRequest{
				Model:       modelID,
				Messages:    reqMessages,
				Temperature: resolvedTemp,
				MaxTokens:   resolvedMaxToks,
				TopP:        0.95,
			}

			// Execute step
			bm.updateCallback(AgentStepMsg{StepType: "thought", Content: "Thinking...", Iteration: iter})

			stream, err := bm.client.ChatCompletionStream(ctx, req)
			if err != nil {
				bm.updateCallback(AgentStepMsg{
					StepType: "error",
					Content:  fmt.Sprintf("Failed to contact local model: %v", err),
					Done:     true,
				})
				return
			}

			var responseSB strings.Builder
			var nativeCalls []api.ToolCall

			// Stream thinking blocks to UI
			for chunk := range stream {
				if chunk.Error != nil {
					bm.updateCallback(AgentStepMsg{
						StepType: "error",
						Content:  fmt.Sprintf("Streaming error: %v", chunk.Error),
						Done:     true,
					})
					return
				}
				if chunk.Content != "" {
					responseSB.WriteString(chunk.Content)
					bm.updateCallback(AgentStepMsg{
						StepType:  "thought",
						Content:   chunk.Content,
						Iteration: iter,
					})
				}
				if chunk.Done {
					// Update local context window metrics
					bm.mu.Lock()
					totalLen := bm.contextManager.CountMessagesTokens(bm.history) + bm.contextManager.CountTokens(systemPrompt)
					bm.mu.Unlock()
					bm.modelManager.Metrics().UpdateTelemetry(modelID, limit, totalLen, metrics.RAMUsedGB)
				}
			}

			fullResponse := responseSB.String()

			// Parse tool calls
			tcs, parseErr := agent.ParseToolCall(fullResponse, nativeCalls)
			if parseErr != nil {
				bm.consecutiveErrors++
				if bm.consecutiveErrors >= 3 {
					bm.updateCallback(AgentStepMsg{
						StepType: "error",
						Content:  "Parser failed consecutively 3 times. Local model is producing invalid outputs. Adjust system instructions.",
						Done:     true,
					})
					return
				}

				// Feed parser failure feedback back as assistant turn + user observation
				bm.mu.Lock()
				bm.history = append(bm.history, api.Message{
					Role:    "assistant",
					Content: fullResponse,
				})

				feedback := fmt.Sprintf("Your last response could not be parsed as a tool call. Format calls correctly inside <tool_call>...</tool_call>.\nRaw output was:\n%s", fullResponse)
				bm.history = append(bm.history, api.Message{
					Role:    "user",
					Content: feedback,
				})
				bm.mu.Unlock()

				bm.session.IncrementIteration()
				continue
			}

			bm.consecutiveErrors = 0

			// Handle end of task (if no tool calls are requested, agent is done)
			if len(tcs) == 0 {
				bm.mu.Lock()
				bm.history = append(bm.history, api.Message{
					Role:    "assistant",
					Content: fullResponse,
				})
				bm.mu.Unlock()

				if bm.session.PlanRef != nil {
					currentIdx := bm.session.GetCurrentStep()
					if currentIdx < len(bm.session.PlanRef.Steps)-1 {
						nextIdx := currentIdx + 1
						nextStep := bm.session.PlanRef.Steps[nextIdx]

						if nextStep.RequiresConfirm && bm.confirmCallback != nil {
							bm.updateCallback(AgentStepMsg{
								StepType:  "pause",
								Content:   fmt.Sprintf("Step %d completed. Requesting confirmation to proceed to Step %d...", currentIdx+1, nextIdx+1),
								Iteration: iter,
							})

							ch := make(chan bool, 1)
							confirmMsg := safety.ConfirmMsg{
								ToolName:     "next_plan_step",
								Description:  fmt.Sprintf("Proceed to next step: %s?", nextStep.Description),
								ResponseChan: ch,
							}
							approved := bm.confirmCallback(confirmMsg)
							if !approved {
								bm.updateCallback(AgentStepMsg{
									StepType: "finished",
									Content:  "Plan execution paused by user.",
									Done:     true,
								})
								return
							}
						}

						bm.session.SetCurrentStep(nextIdx)
						bm.mu.Lock()
						bm.history = append(bm.history, api.Message{
							Role:    "user",
							Content: fmt.Sprintf("Step %d completed. Now proceed to Step %d:\nDescription: %s\nTarget: %s", currentIdx+1, nextIdx+1, nextStep.Description, nextStep.Target),
						})
						bm.mu.Unlock()
						bm.session.IncrementIteration()
						continue
					}
				}

				if bm.memoryManager != nil {
					bm.mu.RLock()
					histCopy := make([]api.Message, len(bm.history))
					copy(histCopy, bm.history)
					bm.mu.RUnlock()
					_ = bm.memoryManager.ExtractAndStore(ctx, modelID, histCopy)
				}
				bm.updateCallback(AgentStepMsg{
					StepType: "finished",
					Content:  fullResponse,
					Done:     true,
				})
				return
			}

			// Execute the first tool call returned
			tc := tcs[0]
			bm.mu.Lock()
			bm.history = append(bm.history, api.Message{
				Role:      "assistant",
				Content:   fullResponse,
				ToolCalls: nativeCalls, // Attach if native
			})
			bm.mu.Unlock()

			// Retrieve tool instance
			toolInstance, exists := bm.registry.Get(tc.Name)
			if !exists {
				obs := fmt.Sprintf("Tool not found: %s", tc.Name)
				bm.mu.Lock()
				bm.history = append(bm.history, api.Message{Role: "user", Content: obs})
				bm.mu.Unlock()
				bm.session.IncrementIteration()
				continue
			}

			// Validate parameters
			if err := bm.registry.ValidateArgs(toolInstance, tc.Args); err != nil {
				obs := fmt.Sprintf("Parameter validation error: %v", err)
				bm.mu.Lock()
				bm.history = append(bm.history, api.Message{Role: "user", Content: obs})
				bm.mu.Unlock()
				bm.session.IncrementIteration()
				continue
			}

			// Apply Safety Check / Classification
			level := bm.classifier.Classify(toolInstance, tc.Args)
			targetDesc := fmt.Sprintf("Execute %s with parameters %v", tc.Name, tc.Args)
			switch tc.Name {
			case "write_file":
				targetDesc = fmt.Sprintf("Write content to %s", tc.Args["path"])
			case "run_command":
				targetDesc = fmt.Sprintf("Run shell command: %s", tc.Args["cmd"])
			}

			// Handle User Confirmation
			requireConfirm := (level == tools.Dangerous && bm.cfg.Safety.RequireConfirmDangerous) ||
				(level == tools.Warn && bm.cfg.Safety.RequireConfirmWarn) ||
				(tc.Name == "write_file" && bm.cfg.Safety.ShowDiffBeforeWrite)

			var diffStr string
			if tc.Name == "write_file" && bm.cfg.Safety.ShowDiffBeforeWrite {
				pathVal, _ := tc.Args["path"].(string)
				resolvedPath, pathErr := tools.PathInScope(pathVal, bm.session.ScopeRoot)
				if pathErr == nil {
					var oldData []byte
					if _, err := os.Stat(resolvedPath); err == nil {
						oldData, _ = os.ReadFile(resolvedPath)
					}
					newData, _ := tc.Args["content"].(string)
					diff := difflib.UnifiedDiff{
						A:        difflib.SplitLines(string(oldData)),
						B:        difflib.SplitLines(newData),
						FromFile: "original",
						ToFile:   "proposed",
						Context:  3,
					}
					diffStr, _ = difflib.GetUnifiedDiffString(diff)
				}
			}

			if requireConfirm && bm.confirmCallback != nil {
				bm.updateCallback(AgentStepMsg{
					StepType:  "pause",
					Content:   fmt.Sprintf("Requesting confirmation for %s...", tc.Name),
					Iteration: iter,
				})

				ch := make(chan bool, 1)
				confirmMsg := safety.ConfirmMsg{
					ToolName:     tc.Name,
					Args:         tc.Args,
					Level:        level,
					Description:  targetDesc,
					Diff:         diffStr,
					ResponseChan: ch,
				}

				approved := bm.confirmCallback(confirmMsg)
				if !approved {
					bm.mu.Lock()
					bm.history = append(bm.history, api.Message{
						Role:    "user",
						Content: "Observation: Execution rejected by user.",
					})
					bm.mu.Unlock()
					bm.session.IncrementIteration()
					bm.updateCallback(AgentStepMsg{
						StepType:  "tool_result",
						ToolName:  tc.Name,
						Content:   "Execution cancelled by user.",
						Iteration: iter,
					})
					continue
				}
			}

			// Take Undo Snapshot if tool is undoable
			var snapshotErr error
			if toolInstance.Undoable() {
				// Size check first if it's write_file
				var sizeOk = true
				if tc.Name == "write_file" {
					pathVal, _ := tc.Args["path"].(string)
					resolvedPath, pathErr := tools.PathInScope(pathVal, bm.session.ScopeRoot)
					if pathErr == nil {
						if statErr := safety.FileSizeGuard(resolvedPath, bm.cfg.Safety.MaxFileWriteBytes); statErr != nil {
							sizeOk = false
						}
					}
				}

				if sizeOk {
					snapshot, snapErr := toolInstance.Snapshot(ctx, tc.Args)
					if snapErr == nil {
						bm.session.UndoStack.Push(snapshot)
					} else {
						snapshotErr = snapErr
					}
				}
			}

			// Execute tool
			bm.updateCallback(AgentStepMsg{
				StepType:  "tool_call",
				ToolName:  tc.Name,
				ToolArgs:  tc.Args,
				Iteration: iter,
			})

			startTime := time.Now()
			result, err := toolInstance.Execute(ctx, tc.Args)
			duration := time.Since(startTime)

			if err != nil {
				result = tools.ToolResult{
					IsError: true,
					Content: fmt.Sprintf("execution runtime error: %v", err),
				}
			}

			// Record updates in session logs
			bm.session.AddToolCall(tc.Name, tc.Args, result.Content, duration, toolInstance.Undoable())

			if tc.Name == "run_command" {
				var exitCode = 0
				if result.IsError {
					exitCode = -1
					if codeVal, ok := result.Metadata["exit_code"].(int); ok {
						exitCode = codeVal
					}
				}
				bm.session.AddCommandRun(tc.Args["cmd"].(string), exitCode, duration, result.Content, "")
			}

			if tc.Name == "write_file" || tc.Name == "delete_file" || tc.Name == "move_file" {
				pathVal, _ := tc.Args["path"].(string)
				if pathVal == "" {
					pathVal, _ = tc.Args["dst"].(string)
				}
				bm.session.AddFileModified(pathVal)
			}

			// Send update to UI
			var resultString = result.Content
			if snapshotErr != nil {
				resultString = fmt.Sprintf("%s\n(Note: rollback snapshot failed: %v)", resultString, snapshotErr)
			}

			bm.updateCallback(AgentStepMsg{
				StepType:  "tool_result",
				ToolName:  tc.Name,
				Content:   resultString,
				Iteration: iter,
			})

			// Add result observation to conversation history
			bm.mu.Lock()
			bm.history = append(bm.history, api.Message{
				Role:    "user",
				Content: fmt.Sprintf("Observation: %s", resultString),
			})
			bm.mu.Unlock()

			bm.session.IncrementIteration()
		}
	}()
}
