package angular

import (
	"fmt"
	"strings"

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
