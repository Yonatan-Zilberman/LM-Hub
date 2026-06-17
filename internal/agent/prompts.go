package agent

import (
	"strings"
)

const AskSystemPromptTemplate = `You are an expert assistant specializing in software development and IT operations.
Be concise, accurate, and technical. Use code blocks for any code or commands.
Current working directory: {cwd}
OS: {os} | Shell: {shell}

{project_context_block}
{memory_facts_block}`

const PlanSystemPromptTemplate = `You are an expert technical architect and planner.
Your task is to analyze the user's request and create a detailed, executable, structured plan to implement it.
You MUST output your plan as a single, valid JSON object matching the JSON schema below.
Do not output any normal conversational text, explanation, or markdown formatting around the JSON, unless explicitly instructed otherwise.

JSON Schema:
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "title": { "type": "string", "description": "Short, clear title for the plan" },
    "summary": { "type": "string", "description": "High-level summary of what the plan achieves" },
    "confidence": { "type": "number", "minimum": 0.0, "maximum": 1.0, "description": "Confidence score from 0.0 to 1.0" },
    "estimated_steps": { "type": "integer", "description": "Total number of steps" },
    "risks": { "type": "array", "items": { "type": "string" }, "description": "Potential risks or side-effects" },
    "files_affected": { "type": "array", "items": { "type": "string" }, "description": "Files that will be created, modified, or deleted" },
    "rollback_strategy": { "type": "string", "description": "How to revert these changes if something goes wrong" },
    "steps": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "id": { "type": "integer", "description": "1-based step ID" },
          "description": { "type": "string", "description": "Detailed description of what to do in this step" },
          "type": { "type": "string", "enum": ["file_edit", "shell", "git", "docker", "info"], "description": "Type of action" },
          "target": { "type": "string", "description": "Target file path, command, or parameter" },
          "reversible": { "type": "boolean", "description": "Whether this step can be automatically undone" },
          "requires_confirm": { "type": "boolean", "description": "Whether this step requires explicit user confirmation" }
        },
        "required": ["id", "description", "type", "target", "reversible", "requires_confirm"]
      }
    }
  },
  "required": ["title", "summary", "confidence", "estimated_steps", "steps", "files_affected", "rollback_strategy"]
}

Current working directory: {cwd}
OS: {os} | Shell: {shell}

{project_context_block}
{memory_facts_block}
{gemma_hint}`

const BuildSystemPromptTemplate = `You are an expert software engineer and systems administrator executing tasks autonomously.
Work in small, focused steps. Prefer reading before writing. One tool call per response.
You have access to the following tools:
{tool_schemas}

Rules:
- Reason inside <thought> tags before every action
- Use exactly one tool per response
- Always read a file before writing it
- Never delete without explicit confirmation from the user
- If uncertain about scope, use ask_user before proceeding
- Prefer targeted edits over full file rewrites

Current working directory: {cwd} (you may not write outside this directory)
Git status: {git_status}
OS: {os} | Shell: {shell}

{project_context_block}
{memory_facts_block}`

// RenderAskPrompt generates the system prompt for Ask mode.
func RenderAskPrompt(cwd, osName, shell, projectContext, memoryFacts string) string {
	prompt := AskSystemPromptTemplate
	prompt = strings.ReplaceAll(prompt, "{cwd}", cwd)
	prompt = strings.ReplaceAll(prompt, "{os}", osName)
	prompt = strings.ReplaceAll(prompt, "{shell}", shell)

	var pcBlock string
	if projectContext != "" {
		pcBlock = "== Project Context ==\n" + projectContext + "\n"
	}
	prompt = strings.ReplaceAll(prompt, "{project_context_block}", pcBlock)

	var memBlock string
	if memoryFacts != "" {
		memBlock = "== What I know about this project ==\n" + memoryFacts + "\n"
	}
	prompt = strings.ReplaceAll(prompt, "{memory_facts_block}", memBlock)

	return strings.TrimSpace(prompt)
}

// RenderPlanPrompt generates the system prompt for Plan mode.
func RenderPlanPrompt(cwd, osName, shell, projectContext, memoryFacts, modelName string) string {
	prompt := PlanSystemPromptTemplate
	prompt = strings.ReplaceAll(prompt, "{cwd}", cwd)
	prompt = strings.ReplaceAll(prompt, "{os}", osName)
	prompt = strings.ReplaceAll(prompt, "{shell}", shell)

	var pcBlock string
	if projectContext != "" {
		pcBlock = "== Project Context ==\n" + projectContext + "\n"
	}
	prompt = strings.ReplaceAll(prompt, "{project_context_block}", pcBlock)

	var memBlock string
	if memoryFacts != "" {
		memBlock = "== What I know about this project ==\n" + memoryFacts + "\n"
	}
	prompt = strings.ReplaceAll(prompt, "{memory_facts_block}", memBlock)

	var gemmaHint string
	if strings.Contains(strings.ToLower(modelName), "gemma") {
		gemmaHint = "IMPORTANT: Your response must be raw JSON only. No markdown, no explanation, no code fences."
	}
	prompt = strings.ReplaceAll(prompt, "{gemma_hint}", gemmaHint)

	return strings.TrimSpace(prompt)
}

// RenderBuildPrompt generates the system prompt for Build mode.
func RenderBuildPrompt(cwd, gitStatus, osName, shell, toolSchemas, projectContext, memoryFacts string) string {
	prompt := BuildSystemPromptTemplate
	prompt = strings.ReplaceAll(prompt, "{cwd}", cwd)
	prompt = strings.ReplaceAll(prompt, "{git_status}", gitStatus)
	prompt = strings.ReplaceAll(prompt, "{os}", osName)
	prompt = strings.ReplaceAll(prompt, "{shell}", shell)
	prompt = strings.ReplaceAll(prompt, "{tool_schemas}", toolSchemas)

	var pcBlock string
	if projectContext != "" {
		pcBlock = "== Project Context ==\n" + projectContext + "\n"
	}
	prompt = strings.ReplaceAll(prompt, "{project_context_block}", pcBlock)

	var memBlock string
	if memoryFacts != "" {
		memBlock = "== What I know about this project ==\n" + memoryFacts + "\n"
	}
	prompt = strings.ReplaceAll(prompt, "{memory_facts_block}", memBlock)

	return strings.TrimSpace(prompt)
}
