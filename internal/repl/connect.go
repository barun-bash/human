package repl

import (
	"bufio"
	"context"
	"fmt"
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
	case "ollama":
		connectOllama(r)
	default:
		fmt.Fprintf(r.errOut, "Unknown provider: %s\n", sub)
		fmt.Fprintln(r.errOut, "Supported: anthropic, openai, ollama")
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

	key := readLine(r.in)
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

	gc := &config.GlobalConfig{
		LLM: &config.GlobalLLMConfig{
			Provider: provider,
			Model:    defaults.Model,
			APIKey:   key,
		},
	}

	if err := config.SaveGlobalConfig(gc); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not save config: %v", err)))
		return
	}

	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Connected to %s (key: %s)", displayName, maskAPIKey(key))))
}

// connectOllama handles setup for the Ollama local provider.
func connectOllama(r *REPL) {
	fmt.Fprintln(r.out, cli.Muted("  Ollama uses local models — no API key needed."))
	fmt.Fprintf(r.out, "Base URL (default http://localhost:11434): ")

	baseURL := readLine(r.in)
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

	gc := &config.GlobalConfig{
		LLM: &config.GlobalLLMConfig{
			Provider: "ollama",
			Model:    defaults.Model,
			BaseURL:  baseURL,
		},
	}

	if err := config.SaveGlobalConfig(gc); err != nil {
		fmt.Fprintln(r.errOut, cli.Error(fmt.Sprintf("Could not save config: %v", err)))
		return
	}

	fmt.Fprintln(r.out, cli.Success(fmt.Sprintf("Connected to Ollama at %s", baseURL)))
}

// maskAPIKey returns a masked version of an API key, showing only the last 4 characters.
func maskAPIKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return "..." + key[len(key)-4:]
}

// readLine reads a single line from the reader, trimming whitespace.
func readLine(r interface{}) string {
	scanner := bufio.NewScanner(r.(interface{ Read([]byte) (int, error) }))
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
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
