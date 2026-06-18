package plan

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/yonatanzilberman/lmhub/internal/agent"
	"github.com/yonatanzilberman/lmhub/internal/api"
	"github.com/yonatanzilberman/lmhub/internal/config"
	"github.com/yonatanzilberman/lmhub/internal/modelmanager"
	"github.com/yonatanzilberman/lmhub/internal/rag"
)

// PlanMode coordinates plan generation, parsing, and correction retries.
type PlanMode struct {
	client         *api.Client
	modelManager   *modelmanager.Manager
	contextManager *agent.ContextManager
	budgetManager  *agent.BudgetManager
	cfg            *config.Config
	retriever      *rag.Retriever
	parseFailures  int
}

// NewPlanMode creates a new PlanMode instance.
func NewPlanMode(
	client *api.Client,
	modelManager *modelmanager.Manager,
	contextManager *agent.ContextManager,
	budgetManager *agent.BudgetManager,
	cfg *config.Config,
	retriever *rag.Retriever,
) *PlanMode {
	return &PlanMode{
		client:         client,
		modelManager:   modelManager,
		contextManager: contextManager,
		budgetManager:  budgetManager,
		cfg:            cfg,
		retriever:      retriever,
	}
}

// GetParseFailures returns the number of consecutive parse failures.
func (pm *PlanMode) GetParseFailures() int {
	return pm.parseFailures
}

// ResetParseFailures resets the parse failure counter.
func (pm *PlanMode) ResetParseFailures() {
	pm.parseFailures = 0
}

// GeneratePlan generates a structured Plan based on the task description.
// It retrieves the model settings, constructs the prompts, requests a chat completion,
// and parses the response. If parsing fails, it retries once with a correction prompt.
// It returns the parsed Plan, the raw model output (useful for debugging/UI), and any error.
func (pm *PlanMode) GeneratePlan(ctx context.Context, modelID, task string, projectRoot string, ragChunks string, memoryFacts string) (*Plan, string, error) {
	// If RAG is enabled, retrieve relevant chunks for the task automatically if not pre-provided
	if ragChunks == "" && pm.retriever != nil && pm.cfg.RAG.Enabled {
		retrieved, err := pm.retriever.Retrieve(ctx, task, pm.cfg.RAG.TopK, pm.cfg.RAG.MaxTokens)
		if err == nil && len(retrieved) > 0 {
			var ragPieces []string
			for _, chunk := range retrieved {
				piece := fmt.Sprintf("[%s:%d-%d]\n%s", chunk.FilePath, chunk.StartLine, chunk.EndLine, chunk.Content)
				ragPieces = append(ragPieces, piece)
			}
			ragChunks = strings.Join(ragPieces, "\n\n")
		}
	}

	// Load project context from file
	projectCtx, err := agent.LoadProjectContext(projectRoot, pm.contextManager, pm.cfg.ContextBudget.ProjectContextMaxTokens)
	if err != nil {
		return nil, "", fmt.Errorf("plan: failed to load project context: %w", err)
	}

	// Allocate budget
	allocation := pm.budgetManager.Allocate(projectCtx, memoryFacts, ragChunks)

	// Get system characteristics
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	osName := runtime.GOOS
	shell := "zsh" // default to zsh, or standard shell

	systemPrompt := agent.RenderPlanPrompt(cwd, osName, shell, allocation.ProjectContext, allocation.MemoryFacts, allocation.RAGChunks, modelID)

	req := api.ChatRequest{
		Model: modelID,
		Messages: []api.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: task},
		},
		Temperature: pm.cfg.ModeInference.Plan.Temperature,
		MaxTokens:   pm.cfg.ModeInference.Plan.MaxTokens,
		Stream:      false,
	}

	resp, err := pm.client.ChatCompletion(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("plan: api request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, "", fmt.Errorf("plan: no choices returned from api")
	}

	rawResponse := resp.Choices[0].Message.Content
	plan, parseErr := ParsePlanJSON(rawResponse)
	if parseErr == nil {
		pm.parseFailures = 0
		return plan, rawResponse, nil
	}

	// Parsing failed; increment parse failure counter and try once to correct
	pm.parseFailures++

	correctionPrompt := fmt.Sprintf(
		"Your previous response was not valid JSON or did not match the required schema. Error: %v. Please return ONLY a valid JSON object matching the schema. Do not include markdown formatting or explanation. Previous response:\n%s",
		parseErr,
		rawResponse,
	)

	retryReq := api.ChatRequest{
		Model: modelID,
		Messages: []api.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: task},
			{Role: "assistant", Content: rawResponse},
			{Role: "user", Content: correctionPrompt},
		},
		Temperature: pm.cfg.ModeInference.Plan.Temperature,
		MaxTokens:   pm.cfg.ModeInference.Plan.MaxTokens,
		Stream:      false,
	}

	retryResp, err := pm.client.ChatCompletion(ctx, retryReq)
	if err != nil {
		return nil, rawResponse, fmt.Errorf("plan: retry api request failed: %w (original parse error: %v)", err, parseErr)
	}

	if len(retryResp.Choices) == 0 {
		return nil, rawResponse, fmt.Errorf("plan: no choices returned from retry api (original parse error: %v)", parseErr)
	}

	retryRawResponse := retryResp.Choices[0].Message.Content
	retryPlan, retryParseErr := ParsePlanJSON(retryRawResponse)
	if retryParseErr != nil {
		pm.parseFailures++
		return nil, retryRawResponse, fmt.Errorf("plan: retry parsing failed: %w (original parse error: %v)", retryParseErr, parseErr)
	}

	pm.parseFailures = 0
	return retryPlan, retryRawResponse, nil
}
