package templates

// BuiltinTemplates returns the default set of 20 prompt templates.
func BuiltinTemplates() []Template {
	return []Template{
		// 1. Debugging & Errors
		{
			Name:        "Debug Go error / panic",
			Description: "Analyze a Go error message, trace log, or panic to find the cause and fix it.",
			Tags:        []string{"go", "debugging", "error", "panic"},
			Mode:        "ask",
			Prompt: `I am getting the following error or panic in my Go application:

{cursor}

Please:
1. Explain what this error/panic means.
2. Identify the most likely root causes.
3. Show me a corrected code snippet or system fix.
4. Describe how to prevent this class of errors in the future.`,
		},
		{
			Name:        "Explain shell command exit code",
			Description: "Explain why a command failed with a specific exit code and how to solve it.",
			Tags:        []string{"shell", "exit-code", "linux", "mac"},
			Mode:        "ask",
			Prompt: `My shell command exited with an unexpected code.
Command run: {cursor}

Please explain:
1. What the exit status code represents.
2. Why this command would exit with this status under normal and abnormal conditions.
3. How to modify the command or system state to run successfully.`,
		},
		{
			Name:        "Parse and fix Docker build error",
			Description: "Diagnose a failure during 'docker build' and provide steps to resolve it.",
			Tags:        []string{"docker", "build", "container"},
			Mode:        "ask",
			Prompt: `My Docker build failed with the following log output:

{cursor}

Please diagnose the issue:
1. Identify the line or instruction in the Dockerfile that caused the failure.
2. Explain the root cause of the error.
3. Provide a corrected version of the Dockerfile or the commands to fix it.`,
		},
		{
			Name:        "Interpret git conflict markers",
			Description: "Analyze conflict markers in a file and show how to resolve the merge conflict.",
			Tags:        []string{"git", "merge", "conflict"},
			Mode:        "ask",
			Prompt: `I have a git merge conflict in my file:

{cursor}

Please help me resolve it:
1. Explain what both versions (HEAD and incoming branch) are attempting to do.
2. Show the resolved file content combining or selecting the changes logically.
3. Provide the git commands to mark the conflict resolved and complete the merge.`,
		},

		// 2. Code Generation
		{
			Name:        "Write a Go HTTP handler (REST)",
			Description: "Generate a clean Go handler function for a REST API endpoint.",
			Tags:        []string{"go", "http", "rest", "handler"},
			Mode:        "build",
			Prompt: `Write a Go HTTP handler function for the following endpoint:
{cursor}

Requirements:
- Follow standard Go HTTP idioms (e.g. net/http or a common router like chi if configured).
- Parse request body/params safely and validate inputs.
- Handle errors gracefully and return appropriate JSON payloads and HTTP status codes.
- Use context.Context for database or downstream service calls.`,
		},
		{
			Name:        "Generate a Dockerfile",
			Description: "Create a production-ready, multi-stage Dockerfile for an application.",
			Tags:        []string{"docker", "dockerfile", "container"},
			Mode:        "build",
			Prompt: `Create a production-ready multi-stage Dockerfile for:
{cursor}

Follow container best practices:
- Use small, secure base images (e.g. alpine or distroless).
- Run as a non-root user.
- Utilize builder stages to keep the final image minimal.
- Correctly cache dependencies.`,
		},
		{
			Name:        "Write a systemd unit file",
			Description: "Create a systemd service definition file for a background process.",
			Tags:        []string{"systemd", "linux", "service"},
			Mode:        "build",
			Prompt: `Create a systemd unit file (.service) for:
{cursor}

Ensure:
- Proper dependencies (e.g. network.target, database services).
- Sandboxing and security configurations.
- Automatic restart policies on failure.
- Logging output redirected to journald.`,
		},
		{
			Name:        "Write a docker-compose.yml",
			Description: "Generate a docker-compose environment with multiple services.",
			Tags:        []string{"docker", "docker-compose", "yaml"},
			Mode:        "build",
			Prompt: `Write a docker-compose.yml file to set up the following services:
{cursor}

Include:
- Service dependencies, health checks, and restart policies.
- Network isolation and persistent volumes.
- Environment variables or secrets configuration.`,
		},

		// 3. Code Review
		{
			Name:        "Review this function for correctness",
			Description: "Analyze a function for logical bugs, edge cases, and optimization possibilities.",
			Tags:        []string{"review", "bug", "correctness"},
			Mode:        "ask",
			Prompt: `Please review the following function for correctness, performance, and style:

{cursor}

Specifically check for:
1. Off-by-one errors, nil pointer dereferences, or panic risks.
2. Edge cases (empty inputs, large inputs, boundaries).
3. Performance bottlenecks or unnecessary allocations.
4. Suggestions for idiomatic improvements.`,
		},
		{
			Name:        "Check this SQL query for N+1 problems",
			Description: "Audit a SQL query or database interaction pattern for performance issues.",
			Tags:        []string{"sql", "database", "performance", "n+1"},
			Mode:        "ask",
			Prompt: `Review the following database operations or SQL queries for N+1 query problems and performance bottlenecks:

{cursor}

Please identify:
1. Any N+1 query patterns.
2. Missing indexes or redundant joins.
3. A refactored approach (e.g., eager loading, subqueries, or bulk joins) to improve speed.`,
		},
		{
			Name:        "Audit this shell script for safety issues",
			Description: "Scan a bash/zsh script for common scripting errors and safety issues.",
			Tags:        []string{"shell", "bash", "safety", "audit"},
			Mode:        "ask",
			Prompt: `Audit the following shell script for safety, portability, and common pitfalls (e.g. unquoted variables, shell injection risks):

{cursor}

Please point out:
1. Critical bugs or safety vulnerabilities.
2. Best practice improvements (e.g., set -euo pipefail).
3. A refactored, safer version of the script.`,
		},
		{
			Name:        "Review this Dockerfile for best practices",
			Description: "Audit an existing Dockerfile for size, security, and cache optimization.",
			Tags:        []string{"docker", "review", "security", "best-practices"},
			Mode:        "ask",
			Prompt: `Review this Dockerfile for security vulnerabilities, image size optimization, and layer cache efficiency:

{cursor}

Suggest concrete improvements for:
1. Base image choice.
2. Layer ordering.
3. Build secret handling.
4. Least privilege execution.`,
		},

		// 4. IT Operations
		{
			Name:        "Diagnose this nginx error log",
			Description: "Analyze nginx access or error logs to determine why a request failed.",
			Tags:        []string{"nginx", "log", "debug", "ops"},
			Mode:        "ask",
			Prompt: `Diagnose the issue in the following nginx log lines:

{cursor}

Explain:
1. What the HTTP status/error code indicates.
2. What configuration in nginx (e.g. proxy settings, buffers, permissions) might be causing it.
3. How to update the nginx configuration or backend service to resolve the issue.`,
		},
		{
			Name:        "Write a cron job",
			Description: "Generate a cron schedule and command for a periodic task.",
			Tags:        []string{"cron", "schedule", "linux"},
			Mode:        "ask",
			Prompt: `Write a crontab entry for:
{cursor}

Provide:
- The standard cron expression.
- Best practices for logging output (e.g., redirecting to syslog, handling exit codes).
- Protection against concurrent runs if the task takes longer than the interval.`,
		},
		{
			Name:        "Explain this kernel log message",
			Description: "Analyze dmesg / journalctl kernel logs (e.g. OOM killer, disk issues).",
			Tags:        []string{"kernel", "log", "oom", "dmesg"},
			Mode:        "ask",
			Prompt: `Explain the following kernel/system log message:

{cursor}

Please explain:
1. What hardware or subsystem generated the message.
2. What happened (e.g., OOM killer invoked, disk write failure, PCI driver crash).
3. What diagnostics or corrective steps should be taken.`,
		},
		{
			Name:        "Generate an SSH config",
			Description: "Create a custom ~/.ssh/config snippet for a specific network setup.",
			Tags:        []string{"ssh", "config", "network"},
			Mode:        "ask",
			Prompt: `Create an ~/.ssh/config entry for the following scenario:
{cursor}

Include:
- Use of jump hosts (ProxyJump) if applicable.
- Custom ports, identity files, and user definitions.
- Keepalive and multiplexing options for stable connections.`,
		},

		// 5. Refactoring
		{
			Name:        "Refactor this function to be testable",
			Description: "Re-architect a function to make it easy to write unit tests for.",
			Tags:        []string{"refactor", "test", "design"},
			Mode:        "plan",
			Prompt: `Refactor the following code to make it highly testable and clean:

{cursor}

Focus on:
1. Decoupling dependencies (e.g. introducing interfaces or dependency injection).
2. Removing global state or hardcoded side-effects.
3. Show a sample unit test demonstrating how to test the refactored code.`,
		},
		{
			Name:        "Extract this logic into a reusable package",
			Description: "Plan how to extract inline logic into a standalone package/module.",
			Tags:        []string{"refactor", "modular", "package"},
			Mode:        "plan",
			Prompt: `Review this block of code and plan how to extract it into a separate, clean, reusable package:

{cursor}

Provide:
1. The new package's API design (exported functions, types).
2. The code implementation for the new package.
3. How to update the original codebase to import and use the new package.`,
		},
		{
			Name:        "Add proper error handling to this function",
			Description: "Replace lazy error handling (e.g. panics/log.Fatal) with proper Go error wrapping.",
			Tags:        []string{"go", "refactor", "error-handling"},
			Mode:        "build",
			Prompt: `Refactor the following function to use proper idiomatic Go error handling:

{cursor}

Guidelines:
- Do not use panics, log.Fatal, or return arbitrary strings.
- Define custom error types or wrap errors with context using fmt.Errorf("operation: %w", err).
- Propagate errors up to the caller to decide logging/handling.`,
		},
		{
			Name:        "Convert callback to use context.Context",
			Description: "Refactor a long-running, asynchronous, or network function to support cancellation via context.",
			Tags:        []string{"refactor", "context", "async", "cancel"},
			Mode:        "build",
			Prompt: `Convert the following code to accept and correctly respect context.Context for cancellation and timeouts:

{cursor}

Ensure:
- Goroutines exit cleanly when ctx.Done() is triggered.
- Propagate context to all nested API/database/network calls.
- Handle context timeout/cancellation errors gracefully.`,
		},
	}
}
