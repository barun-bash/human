package repl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/barun-bash/human/internal/cli"
	"github.com/barun-bash/human/internal/config"
	"github.com/barun-bash/human/internal/llm"
)

// cmdConnect handles the /connect command for LLM provider setup.
func cmdConnect(r *REPL, args []string) {
	sub := "status"
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}

	switch sub {
	case "status":
		connectStatus(r)
	case "anthropic":
		connectAPIKey(r, "anthropic", "Anthropic")
	case "openai":
		connectAPIKey(r, "openai", "OpenAI")
	case "groq":
		connectAPIKey(r, "groq", "Groq")
	case "openrouter":
		connectAPIKey(r, "openrouter", "OpenRouter")
	case "ollama":
		connectOllama(r)
	case "gemini":
		connectAPIKey(r, "gemini", "Gemini")
	case "custom":
		connectCustom(r)
	default:
		fmt.Fprintf(r.errOut, "Unknown provider: %s\n", sub)
		fmt.Fprintf(r.errOut, "Supported: %s\n", strings.Join(llm.SupportedProviders, ", "))
	}
}

// connectStatus shows the current LLM provider configuration.
func connectStatus(r *REPL) {
	gc, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not load config: %v", err)))
		return
	}

	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, cli.Heading("LLM Provider"))
	fmt.Fprintln(r.out, strings.Repeat("\u2500", 30))

	if gc.LLM == nil {
		fmt.Fprintln(r.out, "  Not configured.")
		fmt.Fprintln(r.out)
		fmt.Fprintln(r.out, cli.Muted("  Run /connect <provider> to set up."))
		fmt.Fprintln(r.out, cli.Muted("  Providers: anthropic, openai, ollama"))
		fmt.Fprintln(r.out)
		return
	}

	fmt.Fprintf(r.out, "  Provider:  %s\n", gc.LLM.Provider)
	if gc.LLM.Model != "" {
		fmt.Fprintf(r.out, "  Model:     %s\n", gc.LLM.Model)
	}
	if gc.LLM.APIKey != "" {
		fmt.Fprintf(r.out, "  API Key:   %s\n", maskAPIKey(gc.LLM.APIKey))
	}
	if gc.LLM.BaseURL != "" {
		fmt.Fprintf(r.out, "  Base URL:  %s\n", gc.LLM.BaseURL)
	}
	fmt.Fprintln(r.out)
}

// connectAPIKey handles setup for API-key-based providers (anthropic, openai).
func connectAPIKey(r *REPL, provider, displayName string) {
	fmt.Fprintf(r.out, "Enter your %s API key: ", displayName)

	key, _ := r.scanLine()
	if key == "" {
		fmt.Fprintln(r.errOut, cli.Error("No API key provided. Aborting."))
		return
	}

	// Test the key with a minimal API call.
	fmt.Fprintln(r.out, cli.Muted("  Verifying..."))

	if err := validateProvider(provider, key, ""); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Verification failed: %v", err)))
		return
	}

	// Determine default model.
	defaults := config.DefaultLLMConfig(provider)
	model := defaults.Model

	// Prompt for model selection.
	models := knownModels(provider)
	if len(models) > 0 {
		fmt.Fprintf(r.out, "Model (%s) [%s]: ", strings.Join(models, ", "), model)
		choice, _ := r.scanLine()
		if choice != "" {
			model = choice
		}
	}

	// Load existing config to preserve MCP settings.
	gc, _ := config.LoadGlobalConfig()
	gc.LLM = &config.GlobalLLMConfig{
		Provider: provider,
		Model:    model,
		APIKey:   key,
	}

	if err := config.SaveGlobalConfig(gc); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not save config: %v", err)))
		return
	}

	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Connected to %s (%s, key: %s)", displayName, model, maskAPIKey(key))))
}

// connectOllama handles setup for the Ollama local provider.
func connectOllama(r *REPL) {
	fmt.Fprintln(r.out, cli.Muted("  Ollama uses local models — no API key needed."))
	fmt.Fprintf(r.out, "Base URL (default http://localhost:11434): ")

	baseURL, _ := r.scanLine()
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// Test the connection.
	fmt.Fprintln(r.out, cli.Muted("  Verifying..."))

	if err := validateProvider("ollama", "", baseURL); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not connect to Ollama: %v", err)))
		fmt.Fprintln(r.errOut, cli.Muted("  Make sure Ollama is running: ollama serve"))
		return
	}

	defaults := config.DefaultLLMConfig("ollama")
	model := defaults.Model

	// Try to list installed models from the Ollama API.
	installedModels := fetchOllamaModels(baseURL)
	if len(installedModels) > 0 {
		fmt.Fprintln(r.out, cli.Muted("  Installed models:"))
		for _, m := range installedModels {
			fmt.Fprintf(r.out, "    %s\n", m)
		}
		fmt.Fprintf(r.out, "Model [%s]: ", model)
		choice, _ := r.scanLine()
		if choice != "" {
			model = choice
		}
	}

	// Load existing config to preserve MCP settings.
	gc, _ := config.LoadGlobalConfig()
	gc.LLM = &config.GlobalLLMConfig{
		Provider: "ollama",
		Model:    model,
		BaseURL:  baseURL,
	}

	if err := config.SaveGlobalConfig(gc); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not save config: %v", err)))
		return
	}

	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Connected to Ollama at %s (%s)", baseURL, model)))
}

// cmdDisconnect handles the /disconnect command — removes the saved LLM provider.
func cmdDisconnect(r *REPL, args []string) {
	gc, err := config.LoadGlobalConfig()
	if err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not load config: %v", err)))
		return
	}

	if gc.LLM == nil {
		fmt.Fprintln(r.out, cli.Muted("No LLM provider is configured."))
		return
	}

	provider := gc.LLM.Provider
	fmt.Fprintf(r.out, "Disconnect from %s? (y/n): ", provider)
	answer, ok := r.scanLine()
	if !ok || !isYes(answer) {
		fmt.Fprintln(r.out, cli.Info("Cancelled."))
		return
	}

	gc.LLM = nil
	if err := config.SaveGlobalConfig(gc); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not save config: %v", err)))
		return
	}

	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Disconnected from %s.", provider)))
}

// maskAPIKey returns a masked version of an API key, showing only the last 4 characters.
func maskAPIKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return "..." + key[len(key)-4:]
}

// connectCustom handles setup for a custom OpenAI-compatible provider.
func connectCustom(r *REPL) {
	fmt.Fprintln(r.out, cli.Muted("  Custom provider uses an OpenAI-compatible API endpoint."))
	fmt.Fprintf(r.out, "Base URL (e.g. https://api.example.com/v1/chat/completions): ")

	baseURL, _ := r.scanLine()
	if baseURL == "" {
		fmt.Fprintln(r.errOut, cli.Error("No base URL provided. Aborting."))
		return
	}

	fmt.Fprintf(r.out, "API key (leave empty if not required): ")
	apiKey, _ := r.scanLine()

	fmt.Fprintf(r.out, "Model name: ")
	model, _ := r.scanLine()
	if model == "" {
		model = "default"
	}

	// Load existing config to preserve MCP settings.
	gc, _ := config.LoadGlobalConfig()
	gc.LLM = &config.GlobalLLMConfig{
		Provider: "custom",
		Model:    model,
		APIKey:   apiKey,
		BaseURL:  baseURL,
	}

	if err := config.SaveGlobalConfig(gc); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not save config: %v", err)))
		return
	}

	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Connected to custom provider at %s (%s)", baseURL, model)))
}

// knownModels returns popular model names for a provider.
func knownModels(provider string) []string {
	switch provider {
	case "anthropic":
		return []string{"claude-sonnet-4-20250514", "claude-opus-4-20250514", "claude-haiku-4-20250514"}
	case "openai":
		return []string{"gpt-4o", "gpt-4o-mini", "o1", "o1-mini"}
	case "groq":
		return []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768"}
	case "openrouter":
		return []string{"anthropic/claude-sonnet-4-20250514", "openai/gpt-4o", "google/gemini-2.0-flash"}
	default:
		return nil
	}
}

// fetchOllamaModels queries the Ollama /api/tags endpoint for installed models.
// Returns nil on any error (connection refused, timeout, etc.).
func fetchOllamaModels(baseURL string) []string {
	url := strings.TrimRight(baseURL, "/") + "/api/tags"
	client := &http.Client{Timeout: 3 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}

	var names []string
	for _, m := range result.Models {
		names = append(names, m.Name)
	}
	return names
}

// validateProvider creates a provider and makes a minimal test call.
func validateProvider(provider, apiKey, baseURL string) error {
	cfg := config.DefaultLLMConfig(provider)
	cfg.APIKey = apiKey
	cfg.MaxTokens = 1
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}

	p, err := llm.NewProvider(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err = p.Complete(ctx, &llm.Request{
		Messages:  []llm.Message{{Role: llm.RoleUser, Content: "Hi"}},
		Model:     cfg.Model,
		MaxTokens: 1,
	})
	return err
}
