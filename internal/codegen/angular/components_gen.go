package angular

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

func generatePage(page *ir.Page, app *ir.Application) string {
	var b strings.Builder

	needsRouter := false
	needsState := false

	for _, a := range page.Content {
		switch a.Type {
		case "navigate":
			needsRouter = true
		case "interact":
			if strings.Contains(strings.ToLower(a.Text), "navigate") {
				needsRouter = true
			}
		case "query", "input", "loop":
			needsState = true
		}
	}

	b.WriteString("import { Component, OnInit, signal, inject } from '@angular/core';\n")
	b.WriteString("import { CommonModule } from '@angular/common';\n")
	b.WriteString("import { RouterModule, Router } from '@angular/router';\n")
	b.WriteString("import { ApiService } from '../../services/api.service';\n")

	compName := toPascalCase(page.Name) + "Component"
	selector := "app-" + toKebabCase(page.Name)

	b.WriteString(fmt.Sprintf("\n@Component({\n"))
	b.WriteString(fmt.Sprintf("  selector: '%s',\n", selector))
	b.WriteString("  standalone: true,\n")
	b.WriteString("  imports: [CommonModule, RouterModule],\n")
	b.WriteString("  template: `\n")

	b.WriteString(fmt.Sprintf("    <div class=\"%s-page\">\n", toKebabCase(page.Name)))
	for _, a := range page.Content {
		writeTemplateAction(&b, a, "      ")
	}
	b.WriteString("    </div>\n  `\n})\n")

	b.WriteString(fmt.Sprintf("export class %s implements OnInit {\n", compName))
	
	if needsRouter {
		b.WriteString("  private router = inject(Router);\n")
	}
	if needsState {
		b.WriteString("  private api = inject(ApiService);\n")
		b.WriteString("  loading = signal(true);\n")
		b.WriteString("  data = signal<any[]>([]);\n\n")
		b.WriteString("  ngOnInit() {\n")
		b.WriteString("    // TODO: fetch data via this.api\n")
		b.WriteString("    this.loading.set(false);\n")
		b.WriteString("  }\n")
	} else {
		b.WriteString("  ngOnInit() {}\n")
	}

	if needsRouter {
		b.WriteString("\n  navigate(path: string) {\n    this.router.navigate([path]);\n  }\n")
	}

	b.WriteString("}\n")
	return b.String()
}

func generateComponent(comp *ir.Component, app *ir.Application) string {
	var b strings.Builder

	b.WriteString("import { Component, Input, Output, EventEmitter } from '@angular/core';\n")
	b.WriteString("import { CommonModule } from '@angular/common';\n")

	hasDataModelImport := false
	for _, prop := range comp.Props {
		if prop.Type != "" && isDataModel(prop.Type, app) {
			hasDataModelImport = true
			break
		}
	}

	if hasDataModelImport {
		models := []string{}
		for _, prop := range comp.Props {
			if prop.Type != "" && isDataModel(prop.Type, app) {
				models = append(models, prop.Type)
			}
		}
		fmt.Fprintf(&b, "import type { %s } from '../../models/types';\n", strings.Join(models, ", "))
	}

	compName := toPascalCase(comp.Name) + "Component"
	selector := "app-" + toKebabCase(comp.Name)

	b.WriteString(fmt.Sprintf("\n@Component({\n"))
	b.WriteString(fmt.Sprintf("  selector: '%s',\n", selector))
	b.WriteString("  standalone: true,\n")
	b.WriteString("  imports: [CommonModule],\n")
	b.WriteString("  template: `\n")

	hasClick := hasClickHandler(comp)

	if hasClick {
		b.WriteString(fmt.Sprintf("    <div class=\"%s\" (click)=\"onClick.emit()\">\n", toKebabCase(comp.Name)))
	} else {
		b.WriteString(fmt.Sprintf("    <div class=\"%s\">\n", toKebabCase(comp.Name)))
	}

	for _, a := range comp.Content {
		writeTemplateAction(&b, a, "      ")
	}
	b.WriteString("    </div>\n  `\n})\n")

	b.WriteString(fmt.Sprintf("export class %s {\n", compName))

	for _, prop := range comp.Props {
		propType := "unknown"
		if prop.Type != "" {
			if isDataModel(prop.Type, app) {
				propType = prop.Type
			} else {
				propType = tsType(prop.Type)
			}
		}
		fmt.Fprintf(&b, "  @Input() %s!: %s;\n", prop.Name, propType)
	}

	if hasClick {
		b.WriteString("  @Output() onClick = new EventEmitter<void>();\n")
	}

	b.WriteString("}\n")
	return b.String()
}

func writeTemplateAction(b *strings.Builder, a *ir.Action, indent string) {
	lower := strings.ToLower(a.Text)

	fmt.Fprintf(b, "%s<!-- %s -->\n", indent, a.Text)

	switch a.Type {
	case "display":
		className := slugify(a.Text)
		fmt.Fprintf(b, "%s<div class=\"%s\">%s</div>\n", indent, className, a.Text)

	case "input":
		if strings.Contains(lower, "search") {
			fmt.Fprintf(b, "%s<input type=\"text\" placeholder=\"Search...\" (input)=\"/* TODO */\" />\n", indent)
		} else if strings.Contains(lower, "dropdown") || strings.Contains(lower, "select") {
			fmt.Fprintf(b, "%s<select (change)=\"/* TODO */\">\n", indent)
			fmt.Fprintf(b, "%s  <option>Select...</option>\n", indent)
			fmt.Fprintf(b, "%s</select>\n", indent)
		} else {
			fmt.Fprintf(b, "%s<input type=\"text\" placeholder=\"%s\" />\n", indent, a.Text)
		}

	case "interact":
		if strings.Contains(lower, "clicking") || strings.Contains(lower, "click") {
			label := extractButtonLabel(a.Text)
			if strings.Contains(lower, "navigate") {
				target := extractNavTarget(a.Text)
				fmt.Fprintf(b, "%s<button (click)=\"navigate('/%s')\">%s</button>\n", indent, toKebabCase(target), label)
			} else {
				fmt.Fprintf(b, "%s<button (click)=\"/* TODO */\">%s</button>\n", indent, label)
			}
		} else {
			fmt.Fprintf(b, "%s<div class=\"interactive\">%s</div>\n", indent, a.Text)
		}

	case "navigate":
		target := extractNavTarget(a.Text)
		fmt.Fprintf(b, "%s<button (click)=\"navigate('/%s')\">Go to %s</button>\n", indent, toKebabCase(target), target)

	case "condition":
		if strings.Contains(lower, "while loading") || strings.Contains(lower, "is loading") {
			fmt.Fprintf(b, "%s@if (loading()) {\n%s  <div class=\"loading\">Loading...</div>\n%s}\n", indent, indent, indent)
		} else if strings.Contains(lower, "if no ") || strings.Contains(lower, "if there are no") {
			fmt.Fprintf(b, "%s@if (data().length === 0) {\n%s  <div class=\"empty-state\">%s</div>\n%s}\n", indent, indent, a.Text, indent)
		} else {
			fmt.Fprintf(b, "%s<!-- Condition: %s -->\n", indent, a.Text)
		}

	case "loop":
		fmt.Fprintf(b, "%s@for (item of data(); track $index) {\n", indent)
		fmt.Fprintf(b, "%s  <!-- %s -->\n", indent, a.Text)
		fmt.Fprintf(b, "%s}\n", indent)

	case "query":
		// Queries handled in TS code

	default:
		fmt.Fprintf(b, "%s<div>%s</div>\n", indent, a.Text)
	}
}

func slugify(s string) string {
	var result []rune
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			result = append(result, r)
		} else if r == ' ' || r == '-' || r == '_' {
			if len(result) > 0 && result[len(result)-1] != '-' {
				result = append(result, '-')
			}
		}
	}
	if len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}
	return string(result)
}

func extractButtonLabel(text string) string {
	lower := strings.ToLower(text)
	if idx := strings.Index(lower, "clicking the "); idx != -1 {
		after := text[idx+len("clicking the "):]
		if end := strings.Index(strings.ToLower(after), " button"); end != -1 {
			return after[:end]
		}
		if end := strings.Index(strings.ToLower(after), " navigates"); end != -1 {
			return after[:end]
		}
	}
	return "Click"
}

func extractNavTarget(text string) string {
	lower := strings.ToLower(text)
	for _, marker := range []string{"navigates to ", "navigate to ", "go to "} {
		if idx := strings.Index(lower, marker); idx != -1 {
			target := strings.TrimSpace(text[idx+len(marker):])
			if space := strings.IndexByte(target, ' '); space != -1 {
				target = target[:space]
			}
			return target
		}
	}
	return "home"
}

func isDataModel(typeName string, app *ir.Application) bool {
	for _, m := range app.Data {
		if m.Name == typeName {
			return true
		}
	}
	return false
}

func hasClickHandler(comp *ir.Component) bool {
	for _, a := range comp.Content {
		lower := strings.ToLower(a.Text)
		if strings.Contains(lower, "on_click") || strings.Contains(lower, "onclick") || strings.Contains(lower, "click") {
			return true
		}
	}
	return false
}
