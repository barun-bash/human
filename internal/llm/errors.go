package llm

import "fmt"

// LLMError is a structured error from the LLM connector.
// Error messages are plain English with actionable suggestions,
// matching the project's error style.
type LLMError struct {
	Code    string // short identifier, e.g. "no_api_key"
	Message string // user-facing message
}

func (e *LLMError) Error() string {
	return e.Message
}

// ErrNoAPIKey returns an error when no API key is configured for a provider.
func ErrNoAPIKey(provider string) error {
	var envVar string
	switch provider {
	case "anthropic":
		envVar = "ANTHROPIC_API_KEY"
	case "openai":
		envVar = "OPENAI_API_KEY"
	default:
		envVar = "the appropriate API key"
	}
	return &LLMError{
		Code:    "no_api_key",
		Message: fmt.Sprintf("No API key found for %s. Set the %s environment variable.", provider, envVar),
	}
}

// ErrAuthFailed returns an error when the API rejects the provided key.
func ErrAuthFailed(provider string) error {
	return &LLMError{
		Code:    "auth_failed",
		Message: fmt.Sprintf("Authentication failed for %s. Check that your API key is valid and has not expired.", provider),
	}
}

// ErrRateLimit returns an error when the API rate limit is exceeded.
func ErrRateLimit(provider string) error {
	return &LLMError{
		Code:    "rate_limit",
		Message: fmt.Sprintf("Rate limit exceeded for %s. Wait a moment and try again.", provider),
	}
}

// ErrNetworkFailure returns an error when the provider can't be reached.
func ErrNetworkFailure(provider string, detail string) error {
	return &LLMError{
		Code:    "network_failure",
		Message: fmt.Sprintf("Could not connect to %s: %s", provider, detail),
	}
}

// ErrOllamaNotRunning returns an error when Ollama is not reachable.
func ErrOllamaNotRunning() error {
	return &LLMError{
		Code:    "ollama_not_running",
		Message: "Could not connect to Ollama. Make sure Ollama is running (ollama serve) and accessible at its configured URL.",
	}
}

// ErrProviderError returns an error for an unexpected provider response.
func ErrProviderError(provider string, statusCode int, body string) error {
	return &LLMError{
		Code:    "provider_error",
		Message: fmt.Sprintf("%s returned an error (HTTP %d): %s", provider, statusCode, body),
	}
}

// ErrNoProvider returns an error when no LLM provider is configured.
func ErrNoProvider() error {
	return &LLMError{
		Code:    "no_provider",
		Message: "No LLM provider configured. Run a command like 'human ask' to set one up, or set the ANTHROPIC_API_KEY or OPENAI_API_KEY environment variable.",
	}
}
