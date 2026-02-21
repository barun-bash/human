package svelte

import (
	"fmt"
	"sort"
	"strings"

	"github.com/barun-bash/human/internal/codegen/themes"
	"github.com/barun-bash/human/internal/ir"
)

func generatePackageJson(app *ir.Application) string {
	name := toKebabCase(app.Name)
	if name == "" {
		name = "app"
	}

	devDeps := map[string]string{
		"@sveltejs/adapter-auto":       "^3.0.0",
		"@sveltejs/kit":                "^2.0.0",
		"@sveltejs/vite-plugin-svelte": "^3.0.0",
		"svelte":                       "^5.0.0",
		"svelte-check":                 "^3.6.0",
		"tslib":                        "^2.4.1",
		"typescript":                   "^5.0.0",
		"vite":                         "^5.0.3",
	}

	// Inject design system dependencies
	deps := map[string]string{}
	if app.Theme != nil && app.Theme.DesignSystem != "" {
		dsDeps, dsDevDeps := themes.Dependencies(app.Theme.DesignSystem, "svelte")
		for k, v := range dsDeps {
			deps[k] = v
		}
		for k, v := range dsDevDeps {
			devDeps[k] = v
		}
	}

	var b strings.Builder
	b.WriteString("{\n")
	fmt.Fprintf(&b, "  \"name\": \"%s\",\n", name)
	b.WriteString("  \"version\": \"0.1.0\",\n")
	b.WriteString("  \"private\": true,\n")
	b.WriteString("  \"scripts\": {\n")
	b.WriteString("    \"dev\": \"vite dev\",\n")
	b.WriteString("    \"build\": \"vite build\",\n")
	b.WriteString("    \"preview\": \"vite preview\",\n")
	b.WriteString("    \"start\": \"vite dev\",\n")
	b.WriteString("    \"check\": \"svelte-kit sync && svelte-check --tsconfig ./tsconfig.json\",\n")
	b.WriteString("    \"check:watch\": \"svelte-kit sync && svelte-check --tsconfig ./tsconfig.json --watch\"\n")
	b.WriteString("  },\n")

	if len(deps) > 0 {
		writeSortedDeps(&b, "dependencies", deps)
		b.WriteString(",\n")
	}

	writeSortedDeps(&b, "devDependencies", devDeps)
	b.WriteString(",\n")
	b.WriteString("  \"type\": \"module\"\n")
	b.WriteString("}\n")

	return b.String()
}

func generateSvelteConfig() string {
	return `import adapter from '@sveltejs/adapter-auto';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter()
	}
};

export default config;
`
}

func generateViteConfig() string {
	return `import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()]
});
`
}

func generateTsConfig() string {
	return `{
  "extends": "./.svelte-kit/tsconfig.json",
  "compilerOptions": {
    "allowJs": true,
    "checkJs": true,
    "esModuleInterop": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    "skipLibCheck": true,
    "sourceMap": true,
    "strict": true,
    "moduleResolution": "bundler"
  }
}`
}

func generateAppHtml(app *ir.Application) string {
	title := app.Name
	if title == "" {
		title = "Human App"
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="utf-8" />
		<link rel="icon" href="%%sveltekit.assets%%/favicon.png" />
		<meta name="viewport" content="width=device-width, initial-scale=1" />
		<title>%s</title>
		%%sveltekit.head%%
	</head>
	<body data-sveltekit-preload-data="hover">
		<div style="display: contents">%%sveltekit.body%%</div>
	</body>
</html>
`, title)
}

func generateAppDts() string {
	return `// See https://kit.svelte.dev/docs/types#app
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
}

export {};
`
}

// writeSortedDeps writes a JSON object with sorted keys.
func writeSortedDeps(b *strings.Builder, label string, m map[string]string) {
	b.WriteString(fmt.Sprintf("  \"%s\": {\n", label))
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		fmt.Fprintf(b, "    \"%s\": \"%s\"", k, m[k])
		if i < len(keys)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("  }")
}
