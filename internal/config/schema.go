// Package config defines the configuration structure and schema for LM Hub.
package config

// Config represents the complete application configuration.
type Config struct {
	LMStudio      LMStudioConfig      `mapstructure:"lmstudio" yaml:"lmstudio"`
	ModeModels    ModeModelsConfig    `mapstructure:"mode_models" yaml:"mode_models"`
	Inference     InferenceConfig     `mapstructure:"inference" yaml:"inference"`
	ModeInference ModeInferenceConfig `mapstructure:"mode_inference" yaml:"mode_inference"`
	Agent         AgentConfig         `mapstructure:"agent" yaml:"agent"`
	ContextBudget ContextBudgetConfig `mapstructure:"context_budget" yaml:"context_budget"`
	RAG           RAGConfig           `mapstructure:"rag" yaml:"rag"`
	Memory        MemoryConfig        `mapstructure:"memory" yaml:"memory"`
	ProjectCtx    ProjectContextConfig `mapstructure:"project_context" yaml:"project_context"`
	Templates     TemplatesConfig     `mapstructure:"templates" yaml:"templates"`
	Tools         ToolsConfig         `mapstructure:"tools" yaml:"tools"`
	Safety        SafetyConfig        `mapstructure:"safety" yaml:"safety"`
	UI            UIConfig            `mapstructure:"ui" yaml:"ui"`
	Sessions      SessionsConfig      `mapstructure:"sessions" yaml:"sessions"`
	Log           LogConfig           `mapstructure:"log" yaml:"log"`
}

// LMStudioConfig contains connection settings for LM Studio.
type LMStudioConfig struct {
	BaseURL               string `mapstructure:"base_url" yaml:"base_url"`
	TimeoutSeconds        int    `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
	Stream                bool   `mapstructure:"stream" yaml:"stream"`
	MetricsPollIntervalMs int    `mapstructure:"metrics_poll_interval_ms" yaml:"metrics_poll_interval_ms"`
	EmbeddingModel        string `mapstructure:"embedding_model" yaml:"embedding_model"`
}

// ModeModelsConfig pins specific models for each operational mode.
type ModeModelsConfig struct {
	Ask   string `mapstructure:"ask" yaml:"ask"`
	Plan  string `mapstructure:"plan" yaml:"plan"`
	Build string `mapstructure:"build" yaml:"build"`
}

// InferenceConfig specifies default hyperparameters for model execution.
// Note: max_tokens / context window size is NOT configured here — it is set in LM Studio
// when loading a model. LMH reads context_length from the loaded model instance.
type InferenceConfig struct {
	Temperature   float64 `mapstructure:"temperature" yaml:"temperature"`
	TopP          float64 `mapstructure:"top_p" yaml:"top_p"`
	RepeatPenalty float64 `mapstructure:"repeat_penalty" yaml:"repeat_penalty"`
}

// ModeInferenceConfig overrides inference params for specific modes.
type ModeInferenceConfig struct {
	Ask   InferenceConfig `mapstructure:"ask" yaml:"ask"`
	Plan  InferenceConfig `mapstructure:"plan" yaml:"plan"`
	Build InferenceConfig `mapstructure:"build" yaml:"build"`
}

// AgentConfig regulates iteration and token limits for the autonomous loops.
type AgentConfig struct {
	MaxIterations      int `mapstructure:"max_iterations" yaml:"max_iterations"`
	ContextWarnPct     int `mapstructure:"context_warn_pct" yaml:"context_warn_pct"`
	ContextTrimPct     int `mapstructure:"context_trim_pct" yaml:"context_trim_pct"`
	ContextSummarizePct int `mapstructure:"context_summarize_pct" yaml:"context_summarize_pct"`
}

// ContextBudgetConfig enforces context window allocation ceilings.
type ContextBudgetConfig struct {
	ProjectContextMaxTokens int `mapstructure:"project_context_max_tokens" yaml:"project_context_max_tokens"`
	MemoryMaxTokens         int `mapstructure:"memory_max_tokens" yaml:"memory_max_tokens"`
	RAGMaxTokens            int `mapstructure:"rag_max_tokens" yaml:"rag_max_tokens"`
	TotalMaxTokens          int `mapstructure:"total_max_tokens" yaml:"total_max_tokens"`
}

// RAGConfig contains codebase indexing settings.
type RAGConfig struct {
	Enabled         bool     `mapstructure:"enabled" yaml:"enabled"`
	TopK            int      `mapstructure:"top_k" yaml:"top_k"`
	MaxTokens       int      `mapstructure:"max_tokens" yaml:"max_tokens"`
	MinScore        float64  `mapstructure:"min_score" yaml:"min_score"`
	ReindexOnStart  bool     `mapstructure:"reindex_on_start" yaml:"reindex_on_start"`
	ExcludePatterns []string `mapstructure:"exclude_patterns" yaml:"exclude_patterns"`
}

// MemoryConfig configures agent episodic/semantic memory settings.
type MemoryConfig struct {
	Enabled              bool    `mapstructure:"enabled" yaml:"enabled"`
	AutoExtract          bool    `mapstructure:"auto_extract" yaml:"auto_extract"`
	AutoExtractThreshold float64 `mapstructure:"auto_extract_threshold" yaml:"auto_extract_threshold"`
	MaxFactsPerProject   int     `mapstructure:"max_facts_per_project" yaml:"max_facts_per_project"`
	MaxFactsGlobal       int     `mapstructure:"max_facts_global" yaml:"max_facts_global"`
}

// ProjectContextConfig configures project context file loading.
type ProjectContextConfig struct {
	Enabled   bool   `mapstructure:"enabled" yaml:"enabled"`
	MaxTokens int    `mapstructure:"max_tokens" yaml:"max_tokens"`
	FileName  string `mapstructure:"file_name" yaml:"file_name"`
}

// TemplatesConfig configures prompt templates directories.
type TemplatesConfig struct {
	BuiltinEnabled bool   `mapstructure:"builtin_enabled" yaml:"builtin_enabled"`
	UserDir        string `mapstructure:"user_dir" yaml:"user_dir"`
}

// ToolsConfig dictates settings for executing external commands/processes.
type ToolsConfig struct {
	Shell  ShellToolConfig  `mapstructure:"shell" yaml:"shell"`
	Web    WebToolConfig    `mapstructure:"web" yaml:"web"`
	Docker DockerToolConfig `mapstructure:"docker" yaml:"docker"`
}

// ShellToolConfig configures safe/unsafe shell operations.
type ShellToolConfig struct {
	TimeoutSeconds int      `mapstructure:"timeout_seconds" yaml:"timeout_seconds"`
	AllowedShells  []string `mapstructure:"allowed_shells" yaml:"allowed_shells"`
	Blocklist      []string `mapstructure:"blocklist" yaml:"blocklist"`
}

// WebToolConfig configures internet connectivity integrations.
type WebToolConfig struct {
	SearchProvider       string `mapstructure:"search_provider" yaml:"search_provider"`
	SerperAPIKey         string `mapstructure:"serper_api_key" yaml:"serper_api_key"`
	FetchTimeoutSeconds  int    `mapstructure:"fetch_timeout_seconds" yaml:"fetch_timeout_seconds"`
	CacheTTLMinutes      int    `mapstructure:"cache_ttl_minutes" yaml:"cache_ttl_minutes"`
}

// DockerToolConfig contains docker integration settings.
type DockerToolConfig struct {
	Socket string `mapstructure:"socket" yaml:"socket"`
}

// SafetyConfig outlines human confirmation rules.
type SafetyConfig struct {
	RequireConfirmDangerous bool  `mapstructure:"require_confirm_dangerous" yaml:"require_confirm_dangerous"`
	RequireConfirmWarn      bool  `mapstructure:"require_confirm_warn" yaml:"require_confirm_warn"`
	ShowDiffBeforeWrite     bool  `mapstructure:"show_diff_before_write" yaml:"show_diff_before_write"`
	MaxFileWriteBytes       int64 `mapstructure:"max_file_write_bytes" yaml:"max_file_write_bytes"`
}

// UIConfig dictates color schemes and elements of the Bubbletea view.
type UIConfig struct {
	Theme                    string `mapstructure:"theme" yaml:"theme"`
	MarkdownStyle            string `mapstructure:"markdown_style" yaml:"markdown_style"`
	ShowTokenCount           bool   `mapstructure:"show_token_count" yaml:"show_token_count"`
	ShowThinkingTags         bool   `mapstructure:"show_thinking_tags" yaml:"show_thinking_tags"`
	ShowContextBar           bool   `mapstructure:"show_context_bar" yaml:"show_context_bar"`
	ShowMetricsInStatusbar   bool   `mapstructure:"show_metrics_in_statusbar" yaml:"show_metrics_in_statusbar"`
}

// SessionsConfig specifies parameters for local conversation logging.
type SessionsConfig struct {
	SaveDir     string `mapstructure:"save_dir" yaml:"save_dir"`
	AutoSave    bool   `mapstructure:"auto_save" yaml:"auto_save"`
	MaxHistory  int    `mapstructure:"max_history" yaml:"max_history"`
}

// LogConfig defines logging verbosity and files.
type LogConfig struct {
	Level string `mapstructure:"level" yaml:"level"`
	File  string `mapstructure:"file" yaml:"file"`
}

// DefaultConfig returns a fully-populated Config struct with default settings.
func DefaultConfig() Config {
	return Config{
		LMStudio: LMStudioConfig{
			BaseURL:               "http://localhost:1234",
			TimeoutSeconds:        120,
			Stream:                true,
			MetricsPollIntervalMs: 2000,
			EmbeddingModel:        "text-embedding-nomic-embed-text-v1.5",
		},
		ModeModels: ModeModelsConfig{
			// All empty: LMH will use whatever model is currently loaded in LM Studio.
			// Set these to specific model keys only if you want automatic model switching per mode.
			Ask:   "",
			Plan:  "",
			Build: "",
		},
		Inference: InferenceConfig{
			Temperature:   0.7,
			TopP:          0.95,
			RepeatPenalty: 1.1,
		},
		ModeInference: ModeInferenceConfig{
			Ask: InferenceConfig{
				Temperature:   0.7,
				TopP:          0.95,
				RepeatPenalty: 1.1,
			},
			Plan: InferenceConfig{
				Temperature:   0.3,
				TopP:          0.95,
				RepeatPenalty: 1.1,
			},
			Build: InferenceConfig{
				Temperature:   0.5,
				TopP:          0.95,
				RepeatPenalty: 1.1,
			},
		},
		Agent: AgentConfig{
			MaxIterations:      15,
			ContextWarnPct:     70,
			ContextTrimPct:     85,
			ContextSummarizePct: 90,
		},
		ContextBudget: ContextBudgetConfig{
			ProjectContextMaxTokens: 800,
			MemoryMaxTokens:         800,
			RAGMaxTokens:            1200,
			TotalMaxTokens:          2800,
		},
		RAG: RAGConfig{
			Enabled:        true,
			TopK:           3,
			MaxTokens:      1200,
			MinScore:       0.72,
			ReindexOnStart: false,
			ExcludePatterns: []string{
				"*.lock",
				"*.sum",
				"testdata/**",
				"vendor/**",
				"node_modules/**",
				"*.pb.go",
			},
		},
		Memory: MemoryConfig{
			Enabled:              true,
			AutoExtract:          true,
			AutoExtractThreshold: 0.8,
			MaxFactsPerProject:   50,
			MaxFactsGlobal:       20,
		},
		ProjectCtx: ProjectContextConfig{
			Enabled:   true,
			MaxTokens: 800,
			FileName:  "context.md",
		},
		Templates: TemplatesConfig{
			BuiltinEnabled: true,
			UserDir:        "~/.config/lmhub/templates/",
		},
		Tools: ToolsConfig{
			Shell: ShellToolConfig{
				TimeoutSeconds: 30,
				AllowedShells:  []string{"zsh", "bash"},
				Blocklist: []string{
					"rm -rf /",
					"mkfs",
					"dd if=",
					":(){:|:&};:",
				},
			},
			Web: WebToolConfig{
				SearchProvider:      "duckduckgo",
				SerperAPIKey:        "",
				FetchTimeoutSeconds: 10,
				CacheTTLMinutes:     60,
			},
			Docker: DockerToolConfig{
				Socket: "/var/run/docker.sock",
			},
		},
		Safety: SafetyConfig{
			RequireConfirmDangerous: true,
			RequireConfirmWarn:      false,
			ShowDiffBeforeWrite:     true,
			MaxFileWriteBytes:       10485760, // 10MB
		},
		UI: UIConfig{
			Theme:                  "dark",
			MarkdownStyle:          "dracula",
			ShowTokenCount:         true,
			ShowThinkingTags:       false,
			ShowContextBar:         true,
			ShowMetricsInStatusbar: true,
		},
		Sessions: SessionsConfig{
			SaveDir:    "~/.local/share/lmhub/sessions",
			AutoSave:   true,
			MaxHistory: 50,
		},
		Log: LogConfig{
			Level: "warn",
			File:  "~/.local/share/lmhub/logs/lmhub.log",
		},
	}
}
