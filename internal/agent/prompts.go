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
