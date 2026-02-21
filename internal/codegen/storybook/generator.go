package storybook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

type Generator struct{}

func getFramework(app *ir.Application) string {
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
	fw := getFramework(app)
	ext := getStoryExtension(fw)

	files := map[string]string{
		filepath.Join(outputDir, ".storybook", "main.ts"):            generateMainTs(app, fw),
		filepath.Join(outputDir, ".storybook", "preview.ts"):         generatePreviewTs(app),
		filepath.Join(outputDir, "src", "mocks", "data.ts"):          generateMockData(app),
		filepath.Join(outputDir, "src", "stories", "Introduction.mdx"): generateIntroduction(app, inventory),
		filepath.Join(outputDir, "storybook-dependencies.json"):      generateDependencies(app, fw),
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

func generateMainTs(app *ir.Application, fw string) string {
	addon := "@storybook/react-vite"
	if fw == "vue" {
		addon = "@storybook/vue3-vite"
	} else if fw == "svelte" {
		addon = "@storybook/sveltekit"
	} else if fw == "angular" {
		addon = "@storybook/angular"
	}

	return fmt.Sprintf(`import type { StorybookConfig } from '%s';

const config: StorybookConfig = {
  stories: ['../src/**/*.mdx', '../src/**/*.stories.@(js|jsx|mjs|ts|tsx|svelte)'],
  addons: [
    '@storybook/addon-links',
    '@storybook/addon-essentials',
    '@storybook/addon-interactions',
  ],
  framework: {
    name: '%s',
    options: {},
  },
  docs: {
    autodocs: 'tag',
  },
};
export default config;
`, addon, addon)
}

func generatePreviewTs(app *ir.Application) string {
	return `import type { Preview } from '@storybook/react';

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
`
}

func generateDependencies(app *ir.Application, fw string) string {
	sbDep := `"@storybook/react": "^8.0.0",
    "@storybook/react-vite": "^8.0.0"`
	
	if fw == "vue" {
		sbDep = `"@storybook/vue3": "^8.0.0",
    "@storybook/vue3-vite": "^8.0.0"`
	} else if fw == "svelte" {
		sbDep = `"@storybook/svelte": "^8.0.0",
    "@storybook/sveltekit": "^8.0.0"`
	} else if fw == "angular" {
		sbDep = `"@storybook/angular": "^8.0.0"`
	}

	return fmt.Sprintf(`{
  "devDependencies": {
    "@storybook/addon-essentials": "^8.0.0",
    "@storybook/addon-interactions": "^8.0.0",
    "@storybook/addon-links": "^8.0.0",
    "@storybook/blocks": "^8.0.0",
    %s,
    "@storybook/test": "^8.0.0",
    "storybook": "^8.0.0"
  },
  "scripts": {
    "storybook": "storybook dev -p 6006",
    "build-storybook": "storybook build"
  }
}
`, sbDep)
}

func generateIntroduction(app *ir.Application, inv *ComponentInventory) string {
	return `# UI Storyboard for ` + app.Name + `

Welcome to the auto-generated Storybook. Here you can find all components and pages extracted from your Human declarations.
`
}
