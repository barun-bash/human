package svelte

import (
	"fmt"

	"github.com/barun-bash/human/internal/ir"
)

func generatePackageJson(app *ir.Application) string {
	name := toKebabCase(app.Name)
	if name == "" {
		name = "app"
	}
	return fmt.Sprintf(`{
  "name": "%s",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "vite dev",
    "build": "vite build",
    "preview": "vite preview",
    "start": "vite dev",
    "check": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json",
    "check:watch": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json --watch"
  },
  "devDependencies": {
    "@sveltejs/adapter-auto": "^3.0.0",
    "@sveltejs/kit": "^2.0.0",
    "@sveltejs/vite-plugin-svelte": "^3.0.0",
    "svelte": "^5.0.0",
    "svelte-check": "^3.6.0",
    "tslib": "^2.4.1",
    "typescript": "^5.0.0",
    "vite": "^5.0.3"
  },
  "type": "module"
}
`, name)
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
}
`
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
