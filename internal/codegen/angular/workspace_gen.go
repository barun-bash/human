package angular

import (
	"fmt"
	"sort"
	"strings"

	"github.com/barun-bash/human/internal/codegen/themes"
	"github.com/barun-bash/human/internal/ir"
)

func generateAngularJson(app *ir.Application) string {
	return `{
  "$schema": "./node_modules/@angular/cli/lib/config/schema.json",
  "version": 1,
  "newProjectRoot": "projects",
  "projects": {
    "app": {
      "projectType": "application",
      "schematics": {},
      "root": "",
      "sourceRoot": "src",
      "prefix": "app",
      "architect": {
        "build": {
          "builder": "@angular-devkit/build-angular:application",
          "options": {
            "outputPath": "dist/app",
            "index": "src/index.html",
            "browser": "src/main.ts",
            "polyfills": ["zone.js"],
            "tsConfig": "tsconfig.json",
            "assets": ["src/favicon.ico", "src/assets"],
            "styles": ["src/styles.css"],
            "scripts": []
          }
        },
        "serve": {
          "builder": "@angular-devkit/build-angular:dev-server",
          "configurations": {
            "production": {
              "buildTarget": "app:build:production"
            }
          },
          "defaultConfiguration": "development"
        }
      }
    }
  }
}`
}

func generateTsConfig(app *ir.Application) string {
	return `{
  "compileOnSave": false,
  "compilerOptions": {
    "outDir": "./dist/out-tsc",
    "strict": true,
    "noImplicitOverride": true,
    "noPropertyAccessFromIndexSignature": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true,
    "skipLibCheck": true,
    "esModuleInterop": true,
    "sourceMap": true,
    "declaration": false,
    "experimentalDecorators": true,
    "moduleResolution": "node",
    "importHelpers": true,
    "target": "ES2022",
    "module": "ES2022",
    "lib": ["ES2022", "dom"]
  }
}`
}

func generateIndexHtml(app *ir.Application) string {
	title := app.Name
	if title == "" {
		title = "Human App"
	}
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>%s</title>
  <base href="/">
  <meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
  <app-root></app-root>
</body>
</html>
`, title)
}

func generateMainTs(app *ir.Application) string {
	return `import { bootstrapApplication } from '@angular/platform-browser';
import { appConfig } from './app/app.config';
import { AppComponent } from './app/app.component';

bootstrapApplication(AppComponent, appConfig)
  .catch((err) => console.error(err));
`
}

func generateAppConfig(app *ir.Application) string {
	return `import { ApplicationConfig } from '@angular/core';
import { provideRouter } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideRouter(routes),
    provideHttpClient()
  ]
};
`
}

func generatePackageJson(app *ir.Application) string {
	name := toKebabCase(app.Name)
	if name == "" {
		name = "app"
	}

	deps := map[string]string{
		"@angular/animations":              "^17.0.0",
		"@angular/common":                  "^17.0.0",
		"@angular/compiler":                "^17.0.0",
		"@angular/core":                    "^17.0.0",
		"@angular/forms":                   "^17.0.0",
		"@angular/platform-browser":        "^17.0.0",
		"@angular/platform-browser-dynamic": "^17.0.0",
		"@angular/router":                  "^17.0.0",
		"rxjs":                             "~7.8.0",
		"tslib":                            "^2.3.0",
		"zone.js":                          "~0.14.2",
	}
	devDeps := map[string]string{
		"@angular-devkit/build-angular": "^17.0.0",
		"@angular/cli":                 "^17.0.0",
		"@angular/compiler-cli":        "^17.0.0",
		"@types/node":                  "^18.18.0",
		"typescript":                   "~5.2.2",
	}

	// Inject design system dependencies
	if app.Theme != nil && app.Theme.DesignSystem != "" {
		dsDeps, dsDevDeps := themes.Dependencies(app.Theme.DesignSystem, "angular")
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
	b.WriteString("  \"scripts\": {\n")
	b.WriteString("    \"ng\": \"ng\",\n")
	b.WriteString("    \"start\": \"ng serve\",\n")
	b.WriteString("    \"build\": \"ng build\",\n")
	b.WriteString("    \"watch\": \"ng build --watch --configuration development\",\n")
	b.WriteString("    \"test\": \"ng test\"\n")
	b.WriteString("  },\n")
	b.WriteString("  \"private\": true,\n")

	writeSortedDeps(&b, "dependencies", deps)
	b.WriteString(",\n")
	writeSortedDeps(&b, "devDependencies", devDeps)
	b.WriteString("\n}\n")

	return b.String()
}

func generateRoutes(app *ir.Application) string {
	var b strings.Builder
	b.WriteString("import { Routes } from '@angular/router';\n\n")
	b.WriteString("export const routes: Routes = [\n")

	for _, page := range app.Pages {
		routePath := ""
		if strings.ToLower(page.Name) != "home" {
			routePath = toKebabCase(page.Name)
		}
		fileName := toKebabCase(page.Name)
		compName := toPascalCase(page.Name) + "Component"
		b.WriteString(fmt.Sprintf("  { path: '%s', loadComponent: () => import('./pages/%s/%s.component').then(m => m.%s) },\n", routePath, fileName, fileName, compName))
	}
	b.WriteString("  { path: '**', loadComponent: () => import('./pages/not-found/not-found.component').then(m => m.NotFoundComponent) },\n")

	b.WriteString("];\n")
	return b.String()
}

func generateAppComponent(app *ir.Application) string {
	return `import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: '<router-outlet></router-outlet>'
})
export class AppComponent {}
`
}

func generateNotFoundComponent() string {
	return `import { Component } from '@angular/core';

@Component({
  selector: 'app-not-found',
  standalone: true,
  template: '<div style="text-align:center;padding:4rem"><h1>404</h1><p>Page not found</p></div>'
})
export class NotFoundComponent {}
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
