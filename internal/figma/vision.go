package figma

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/llm"
	"github.com/barun-bash/human/internal/parser"
)

const visionPrompt = `You are an expert in the Human programming language.

Analyze this UI screenshot and generate a .human file that recreates the design.

Rules:
1. Identify all visible UI components (navigation, forms, lists, cards, buttons, modals)
2. Infer the data models from the content shown
3. Describe every visible interaction (clickable elements, forms, navigation)
4. Extract the color scheme and typography for the theme block
5. Include empty states and loading states where logical
6. Use only valid Human language syntax

Output format:
- Start with: app <AppName> is a web application
- Include theme: block with colors extracted from the design
- Include data: blocks for each inferred model
- Include page: blocks for each screen
- Include api: blocks for CRUD operations
- Include authentication: if login/signup is visible
- Include build with: block

Output the .human file only, no explanation or markdown.`

// AnalyzeImage uses LLM vision to interpret a UI screenshot and produce a .human file.
func AnalyzeImage(imagePath string, cfg *GenerateConfig, provider llm.Provider) (string, error) {
	if cfg == nil {
		name := strings.TrimSuffix(filepath.Base(imagePath), filepath.Ext(imagePath))
		cfg = &GenerateConfig{
			AppName:  toPascalCase(name),
			Platform: "web",
			Frontend: "React",
			Backend:  "Node",
			Database: "PostgreSQL",
		}
	}

	// Check provider supports vision
	if !SupportsVision(provider) {
		return "", fmt.Errorf("LLM provider %q does not support vision/image analysis. Use Anthropic or OpenAI", provider.Name())
	}

	// Read and encode image
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("reading image %s: %w", imagePath, err)
	}

	// Validate image size (max 10MB)
	if len(data) > 10*1024*1024 {
		return "", fmt.Errorf("image too large (%d bytes, max 10MB). Resize the image and try again", len(data))
	}

	mimeType := detectMIMEType(imagePath)
	imageB64 := base64.StdEncoding.EncodeToString(data)

	prompt := fmt.Sprintf("%s\n\nApplication name: %s\nFrontend: %s\nBackend: %s\nDatabase: %s",
		visionPrompt, cfg.AppName, cfg.Frontend, cfg.Backend, cfg.Database)

	// Build request with image
	req := &llm.Request{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: prompt},
		},
		MaxTokens:   8192,
		Temperature: 0.0,
		Images: []llm.ImageInput{
			{
				Data:     imageB64,
				MIMEType: mimeType,
			},
		},
	}

	resp, err := provider.Complete(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("LLM vision analysis failed: %w", err)
	}

	// Extract and validate the .human code from response
	code := extractHumanCode(resp.Content)

	// Validate via parser
	if _, err := parser.Parse(code); err != nil {
		// Return the code with a warning — partial output is still useful
		return code, fmt.Errorf("generated code has syntax issues (usable but may need edits): %w", err)
	}

	return code, nil
}

// AnalyzeMultipleImages processes several screenshots and merges them into one .human file.
func AnalyzeMultipleImages(imagePaths []string, cfg *GenerateConfig, provider llm.Provider) (string, error) {
	if len(imagePaths) == 0 {
		return "", fmt.Errorf("no image paths provided")
	}

	if len(imagePaths) == 1 {
		return AnalyzeImage(imagePaths[0], cfg, provider)
	}

	// Analyze each image individually
	var results []string
	for _, path := range imagePaths {
		code, err := AnalyzeImage(path, cfg, provider)
		if err != nil {
			// Log warning but continue
			fmt.Printf("  warning: %s: %v\n", filepath.Base(path), err)
			if code != "" {
				results = append(results, code)
			}
			continue
		}
		results = append(results, code)
	}

	if len(results) == 0 {
		return "", fmt.Errorf("all image analyses failed")
	}

	if len(results) == 1 {
		return results[0], nil
	}

	// Merge multiple results via LLM
	return mergeViaLLM(results, cfg, provider)
}

// mergeViaLLM uses the LLM to merge multiple .human files into one.
func mergeViaLLM(results []string, cfg *GenerateConfig, provider llm.Provider) (string, error) {
	var sb strings.Builder
	for i, r := range results {
		fmt.Fprintf(&sb, "--- Screen %d ---\n%s\n\n", i+1, r)
	}

	mergePrompt := fmt.Sprintf(`You are an expert in the Human programming language.

Below are multiple .human file fragments, each generated from a different screen of the same application.
Merge them into a single cohesive .human file.

Rules:
1. Combine data models — deduplicate fields, keep the most complete version
2. Combine all pages into one app
3. Unify the theme (use the most common colors/fonts)
4. Keep all API endpoints but deduplicate
5. Use app name: %s
6. Output only the merged .human file, no explanation

Fragments:
%s`, cfg.AppName, sb.String())

	req := &llm.Request{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: mergePrompt},
		},
		MaxTokens:   8192,
		Temperature: 0.0,
	}

	resp, err := provider.Complete(context.Background(), req)
	if err != nil {
		// Fall back to first result
		return results[0], nil
	}

	return extractHumanCode(resp.Content), nil
}

// SupportsVision checks if a provider supports image/vision inputs.
func SupportsVision(provider llm.Provider) bool {
	name := strings.ToLower(provider.Name())
	return name == "anthropic" || name == "openai" || name == "gemini"
}

// IsImageFile checks if a file path points to a supported image format.
func IsImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".webp"
}

func detectMIMEType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "image/png"
	}
}

// extractHumanCode strips markdown fences and extracts .human code from LLM output.
func extractHumanCode(content string) string {
	// Strip markdown code fences
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		// Remove first and last fence lines
		if len(lines) >= 2 {
			lines = lines[1:]
			if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
				lines = lines[:len(lines)-1]
			}
			content = strings.Join(lines, "\n")
		}
	}
	return strings.TrimSpace(content)
}
