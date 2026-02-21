package storybook

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/barun-bash/human/internal/ir"
)

type Generator struct{}

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

	files := map[string]string{
		filepath.Join(outputDir, ".storybook", "main.ts"):            generateMainTs(app),
		filepath.Join(outputDir, ".storybook", "preview.ts"):         generatePreviewTs(app),
		filepath.Join(outputDir, "src", "mocks", "data.ts"):          generateMockData(app),
		filepath.Join(outputDir, "src", "stories", "Introduction.mdx"): generateIntroduction(app, inventory),
		filepath.Join(outputDir, "storybook-dependencies.json"):      generateDependencies(app),
	}

	for _, comp := range inventory.Components {
		path := filepath.Join(outputDir, "src", "stories", "components", comp.Name+".stories.tsx")
		files[path] = generateComponentStory(comp, app)
	}

	for _, page := range inventory.Pages {
		path := filepath.Join(outputDir, "src", "stories", "pages", page.Name+".stories.tsx")
		files[path] = generatePageStory(page, app)
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

func generateMainTs(app *ir.Application) string {
	return `import type { StorybookConfig } from '@storybook/react-vite';

const config: StorybookConfig = {
  stories: ['../src/**/*.mdx', '../src/**/*.stories.@(js|jsx|mjs|ts|tsx)'],
  addons: [
    '@storybook/addon-links',
    '@storybook/addon-essentials',
    '@storybook/addon-interactions',
  ],
  framework: {
    name: '@storybook/react-vite',
    options: {},
  },
  docs: {
    autodocs: 'tag',
  },
};
export default config;
`
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

func generateDependencies(app *ir.Application) string {
	return `{
  "devDependencies": {
    "@storybook/addon-essentials": "^8.0.0",
    "@storybook/addon-interactions": "^8.0.0",
    "@storybook/addon-links": "^8.0.0",
    "@storybook/blocks": "^8.0.0",
    "@storybook/react": "^8.0.0",
    "@storybook/react-vite": "^8.0.0",
    "@storybook/test": "^8.0.0",
    "storybook": "^8.0.0"
  },
  "scripts": {
    "storybook": "storybook dev -p 6006",
    "build-storybook": "storybook build"
  }
}
`
}

func generateIntroduction(app *ir.Application, inv *ComponentInventory) string {
	return `# UI Storyboard for ` + app.Name + `

Welcome to the auto-generated Storybook. Here you can find all components and pages extracted from your Human declarations.
`
}
