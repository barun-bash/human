package storybook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// Generator produces Storybook configuration and story files for a frontend project.
type Generator struct{}

// GetFramework returns the frontend framework name from the IR config.
func GetFramework(app *ir.Application) string {
	if app.Config == nil || app.Config.Frontend == "" {
		return "react"
	}
	lower := strings.ToLower(app.Config.Frontend)
	if strings.Contains(lower, "vue") {
		return "vue"
	}
	if strings.Contains(lower, "svelte") {
		return "svelte"
	}
	if strings.Contains(lower, "angular") {
		return "angular"
	}
	return "react"
}

func getStoryExtension(fw string) string {
	if fw == "react" {
		return ".stories.tsx"
	}
	return ".stories.ts"
}

// Generate writes Storybook config, story files, and mock data into outputDir.
// outputDir should be the frontend directory (e.g. .human/output/react).
func (g Generator) Generate(app *ir.Application, outputDir string) error {
	dirs := []string{
		filepath.Join(outputDir, ".storybook"),
		filepath.Join(outputDir, "src", "stories", "components"),
		filepath.Join(outputDir, "src", "stories", "pages"),
		filepath.Join(outputDir, "src", "mocks"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	inventory := BuildInventory(app)
	fw := GetFramework(app)
	ext := getStoryExtension(fw)

	files := map[string]string{
		filepath.Join(outputDir, ".storybook", "main.ts"):    generateMainTs(fw),
		filepath.Join(outputDir, ".storybook", "preview.ts"): generatePreviewTs(fw),
		filepath.Join(outputDir, "src", "mocks", "data.ts"):  generateMockData(app, fw),
	}

	for _, comp := range inventory.Components {
		path := filepath.Join(outputDir, "src", "stories", "components", comp.Name+ext)
		files[path] = generateComponentStory(comp, app, fw)
	}

	for _, page := range inventory.Pages {
		path := filepath.Join(outputDir, "src", "stories", "pages", page.Name+ext)
		files[path] = generatePageStory(page, app, fw)
	}

	for path, content := range files {
		if err := writeFile(path, content); err != nil {
			return err
		}
	}

	return nil
}

func writeFile(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func generateMainTs(fw string) string {
	frameworkPkg := "@storybook/react-vite"
	if fw == "vue" {
		frameworkPkg = "@storybook/vue3-vite"
	} else if fw == "svelte" {
		frameworkPkg = "@storybook/sveltekit"
	} else if fw == "angular" {
		frameworkPkg = "@storybook/angular"
	}

	return fmt.Sprintf(`import type { StorybookConfig } from '%s';

const config: StorybookConfig = {
  stories: ['../src/**/*.stories.@(js|jsx|mjs|ts|tsx)'],
  addons: [
    '@storybook/addon-essentials',
    '@storybook/addon-interactions',
  ],
  framework: {
    name: '%s',
    options: {},
  },
};
export default config;
`, frameworkPkg, frameworkPkg)
}

func generatePreviewTs(fw string) string {
	previewType := "@storybook/react"
	if fw == "vue" {
		previewType = "@storybook/vue3"
	} else if fw == "svelte" {
		previewType = "@storybook/svelte"
	} else if fw == "angular" {
		previewType = "@storybook/angular"
	}

	return fmt.Sprintf(`import type { Preview } from '%s';

const preview: Preview = {
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
  },
};

export default preview;
`, previewType)
}

// DevDependencies returns the Storybook devDependencies map for a given framework.
// This is used by the scaffold generator to merge into the frontend package.json.
func DevDependencies(fw string) map[string]string {
	deps := map[string]string{
		"@storybook/addon-essentials":    "^8.6.0",
		"@storybook/addon-interactions":  "^8.6.0",
		"@storybook/blocks":             "^8.6.0",
		"@storybook/test":               "^8.6.0",
		"storybook":                     "^8.6.0",
	}

	switch fw {
	case "vue":
		deps["@storybook/vue3"] = "^8.6.0"
		deps["@storybook/vue3-vite"] = "^8.6.0"
	case "svelte":
		deps["@storybook/svelte"] = "^8.6.0"
		deps["@storybook/sveltekit"] = "^8.6.0"
	case "angular":
		deps["@storybook/angular"] = "^8.6.0"
	default: // react
		deps["@storybook/react"] = "^8.6.0"
		deps["@storybook/react-vite"] = "^8.6.0"
	}

	return deps
}

// Scripts returns the Storybook npm scripts to merge into the frontend package.json.
func Scripts() map[string]string {
	return map[string]string{
		"storybook":       "storybook dev -p 6006",
		"build-storybook": "storybook build",
	}
}
