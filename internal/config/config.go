// Package config defines the configuration structure and schema for LM Hub.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Load reads and parses the configuration file.
// If the configuration file does not exist, it writes the default config.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Default values
	defaultCfg := DefaultConfig()

	// Set defaults in viper
	v.SetDefault("lmstudio.base_url", defaultCfg.LMStudio.BaseURL)
	v.SetDefault("lmstudio.timeout_seconds", defaultCfg.LMStudio.TimeoutSeconds)
	v.SetDefault("lmstudio.stream", defaultCfg.LMStudio.Stream)
	v.SetDefault("lmstudio.metrics_poll_interval_ms", defaultCfg.LMStudio.MetricsPollIntervalMs)
	v.SetDefault("lmstudio.embedding_model", defaultCfg.LMStudio.EmbeddingModel)

	v.SetDefault("mode_models.ask", defaultCfg.ModeModels.Ask)
	v.SetDefault("mode_models.plan", defaultCfg.ModeModels.Plan)
	v.SetDefault("mode_models.build", defaultCfg.ModeModels.Build)

	v.SetDefault("inference.temperature", defaultCfg.Inference.Temperature)
	v.SetDefault("inference.top_p", defaultCfg.Inference.TopP)
	v.SetDefault("inference.repeat_penalty", defaultCfg.Inference.RepeatPenalty)

	v.SetDefault("mode_inference.ask.temperature", defaultCfg.ModeInference.Ask.Temperature)
	v.SetDefault("mode_inference.plan.temperature", defaultCfg.ModeInference.Plan.Temperature)
	v.SetDefault("mode_inference.build.temperature", defaultCfg.ModeInference.Build.Temperature)

	v.SetDefault("agent.max_iterations", defaultCfg.Agent.MaxIterations)
	v.SetDefault("agent.context_warn_pct", defaultCfg.Agent.ContextWarnPct)
	v.SetDefault("agent.context_trim_pct", defaultCfg.Agent.ContextTrimPct)
	v.SetDefault("agent.context_summarize_pct", defaultCfg.Agent.ContextSummarizePct)

	v.SetDefault("context_budget.project_context_max_tokens", defaultCfg.ContextBudget.ProjectContextMaxTokens)
	v.SetDefault("context_budget.memory_max_tokens", defaultCfg.ContextBudget.MemoryMaxTokens)
	v.SetDefault("context_budget.rag_max_tokens", defaultCfg.ContextBudget.RAGMaxTokens)
	v.SetDefault("context_budget.total_max_tokens", defaultCfg.ContextBudget.TotalMaxTokens)

	v.SetDefault("rag.enabled", defaultCfg.RAG.Enabled)
	v.SetDefault("rag.top_k", defaultCfg.RAG.TopK)
	v.SetDefault("rag.max_tokens", defaultCfg.RAG.MaxTokens)
	v.SetDefault("rag.min_score", defaultCfg.RAG.MinScore)
	v.SetDefault("rag.reindex_on_start", defaultCfg.RAG.ReindexOnStart)
	v.SetDefault("rag.exclude_patterns", defaultCfg.RAG.ExcludePatterns)

	v.SetDefault("memory.enabled", defaultCfg.Memory.Enabled)
	v.SetDefault("memory.auto_extract", defaultCfg.Memory.AutoExtract)
	v.SetDefault("memory.auto_extract_threshold", defaultCfg.Memory.AutoExtractThreshold)
	v.SetDefault("memory.max_facts_per_project", defaultCfg.Memory.MaxFactsPerProject)
	v.SetDefault("memory.max_facts_global", defaultCfg.Memory.MaxFactsGlobal)

	v.SetDefault("project_context.enabled", defaultCfg.ProjectCtx.Enabled)
	v.SetDefault("project_context.max_tokens", defaultCfg.ProjectCtx.MaxTokens)
	v.SetDefault("project_context.file_name", defaultCfg.ProjectCtx.FileName)

	v.SetDefault("templates.builtin_enabled", defaultCfg.Templates.BuiltinEnabled)
	v.SetDefault("templates.user_dir", defaultCfg.Templates.UserDir)

	v.SetDefault("tools.shell.timeout_seconds", defaultCfg.Tools.Shell.TimeoutSeconds)
	v.SetDefault("tools.shell.allowed_shells", defaultCfg.Tools.Shell.AllowedShells)
	v.SetDefault("tools.shell.blocklist", defaultCfg.Tools.Shell.Blocklist)
	v.SetDefault("tools.web.search_provider", defaultCfg.Tools.Web.SearchProvider)
	v.SetDefault("tools.web.serper_api_key", defaultCfg.Tools.Web.SerperAPIKey)
	v.SetDefault("tools.web.fetch_timeout_seconds", defaultCfg.Tools.Web.FetchTimeoutSeconds)
	v.SetDefault("tools.web.cache_ttl_minutes", defaultCfg.Tools.Web.CacheTTLMinutes)
	v.SetDefault("tools.docker.socket", defaultCfg.Tools.Docker.Socket)

	v.SetDefault("safety.require_confirm_dangerous", defaultCfg.Safety.RequireConfirmDangerous)
	v.SetDefault("safety.require_confirm_warn", defaultCfg.Safety.RequireConfirmWarn)
	v.SetDefault("safety.show_diff_before_write", defaultCfg.Safety.ShowDiffBeforeWrite)
	v.SetDefault("safety.max_file_write_bytes", defaultCfg.Safety.MaxFileWriteBytes)

	v.SetDefault("ui.theme", defaultCfg.UI.Theme)
	v.SetDefault("ui.markdown_style", defaultCfg.UI.MarkdownStyle)
	v.SetDefault("ui.show_token_count", defaultCfg.UI.ShowTokenCount)
	v.SetDefault("ui.show_thinking_tags", defaultCfg.UI.ShowThinkingTags)
	v.SetDefault("ui.show_context_bar", defaultCfg.UI.ShowContextBar)
	v.SetDefault("ui.show_metrics_in_statusbar", defaultCfg.UI.ShowMetricsInStatusbar)

	v.SetDefault("sessions.save_dir", defaultCfg.Sessions.SaveDir)
	v.SetDefault("sessions.auto_save", defaultCfg.Sessions.AutoSave)
	v.SetDefault("sessions.max_history", defaultCfg.Sessions.MaxHistory)

	v.SetDefault("log.level", defaultCfg.Log.Level)
	v.SetDefault("log.file", defaultCfg.Log.File)

	// Set config file path
	if configPath != "" {
		resolvedPath, err := ExpandTilde(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve config path tilde: %w", err)
		}
		v.SetConfigFile(resolvedPath)
	} else {
		// Default config path: ~/.config/lmhub/config.yaml
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		dir := filepath.Join(home, ".config", "lmhub")
		v.AddConfigPath(dir)
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	// Environment overrides (e.g. LMHUB_LMSTUDIO_BASE_URL)
	v.SetEnvPrefix("LMHUB")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file if it exists, or create default
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok || os.IsNotExist(err) {
			// Write default configuration if not found
			cfgPath := v.ConfigFileUsed()
			if cfgPath == "" {
				home, _ := os.UserHomeDir()
				cfgPath = filepath.Join(home, ".config", "lmhub", "config.yaml")
			}
			err := writeDefaultConfig(cfgPath)
			if err != nil {
				return nil, fmt.Errorf("failed to write default config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Resolve tilde in all configured paths
	if err := cfg.resolvePaths(); err != nil {
		return nil, fmt.Errorf("failed to resolve paths in config: %w", err)
	}

	return &cfg, nil
}

// ExpandTilde replaces leading ~ in path with user home directory.
func ExpandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	if path == "~" {
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func (c *Config) resolvePaths() error {
	var err error
	c.Templates.UserDir, err = ExpandTilde(c.Templates.UserDir)
	if err != nil {
		return err
	}
	c.Sessions.SaveDir, err = ExpandTilde(c.Sessions.SaveDir)
	if err != nil {
		return err
	}
	c.Log.File, err = ExpandTilde(c.Log.File)
	if err != nil {
		return err
	}
	return nil
}

func writeDefaultConfig(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	// Write simple default YAML string to file
	defaultYAML := `# ~/.config/lmhub/config.yaml

lmstudio:
  base_url: "http://localhost:1234"
  timeout_seconds: 120
  stream: true
  metrics_poll_interval_ms: 2000
  embedding_model: "text-embedding-nomic-embed-text-v1.5"

mode_models:
  # Leave empty to use whatever model is currently loaded in LM Studio.
  # Set to a specific model key (e.g. "qwen/qwen3-8b") to enable automatic
  # model switching when entering that mode.
  ask:   ""
  plan:  ""
  build: ""

inference:
  temperature: 0.7
  top_p: 0.95
  repeat_penalty: 1.1

mode_inference:
  ask:
    temperature: 0.7
  plan:
    temperature: 0.3
  build:
    temperature: 0.5

agent:
  max_iterations: 15
  context_warn_pct: 70
  context_trim_pct: 85
  context_summarize_pct: 90

context_budget:
  project_context_max_tokens: 800
  memory_max_tokens: 800
  rag_max_tokens: 1200
  total_max_tokens: 2800

rag:
  enabled: true
  top_k: 3
  max_tokens: 1200
  min_score: 0.72
  reindex_on_start: false
  exclude_patterns:
    - "*.lock"
    - "*.sum"
    - "testdata/**"
    - "vendor/**"
    - "node_modules/**"
    - "*.pb.go"

memory:
  enabled: true
  auto_extract: true
  auto_extract_threshold: 0.8
  max_facts_per_project: 50
  max_facts_global: 20

project_context:
  enabled: true
  max_tokens: 800
  file_name: "context.md"

templates:
  builtin_enabled: true
  user_dir: "~/.config/lmhub/templates/"

tools:
  shell:
    timeout_seconds: 30
    allowed_shells: ["zsh", "bash"]
    blocklist:
      - "rm -rf /"
      - "mkfs"
      - "dd if="
      - ":(){:|:&};:"
  web:
    search_provider: "duckduckgo"
    serper_api_key: ""
    fetch_timeout_seconds: 10
    cache_ttl_minutes: 60
  docker:
    socket: "/var/run/docker.sock"

safety:
  require_confirm_dangerous: true
  require_confirm_warn: false
  show_diff_before_write: true
  max_file_write_bytes: 10485760

ui:
  theme: "dark"
  markdown_style: "dracula"
  show_token_count: true
  show_thinking_tags: false
  show_context_bar: true
  show_metrics_in_statusbar: true

sessions:
  save_dir: "~/.local/share/lmhub/sessions"
  auto_save: true
  max_history: 50

log:
  level: "warn"
  file: "~/.local/share/lmhub/logs/lmhub.log"
`
	return os.WriteFile(path, []byte(defaultYAML), 0644)
}
