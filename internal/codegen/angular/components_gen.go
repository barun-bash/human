package angular

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// pageContext carries shared state for template generation within a page or component.
type pageContext struct {
	app             *ir.Application
	modelName       string            // primary data model (e.g. "Post")
	varName         string            // plural signal name (e.g. "posts")
	itemVar         string            // loop item variable (e.g. "post")
	props           map[string]string // component props: name → type
	hasSuccessState bool
	hasErrorState   bool
	isComponent     bool              // true when generating a component (not a page)
}

func generatePage(page *ir.Page, app *ir.Application) string {
	var b strings.Builder

	modelName, varName, itemVar := detectPageModel(page, app)

	needsRouter := false
	needsDataState := false
	needsEffect := false
	needsAuth := false
	needsFormState := false
	needsSuccess := false
	needsError := false

	for _, a := range page.Content {
		lower := strings.ToLower(a.Text)
		switch a.Type {
		case "navigate":
			needsRouter = true
		case "interact":
			if strings.Contains(lower, "navigate") {
				needsRouter = true
			}
			if strings.Contains(lower, "opens a form") || strings.Contains(lower, "open a form") {
				needsFormState = true
			}
		case "query":
			needsDataState = true
			needsEffect = true
		case "loop":
			needsDataState = true
			if modelName != "" {
				needsEffect = true
			}
		case "input":
			if strings.Contains(lower, "button") && (strings.Contains(lower, "create") || strings.Contains(lower, "new") || strings.Contains(lower, "add")) {
				needsFormState = true
			}
		case "condition":
			if strings.Contains(lower, "logged in") {
				needsAuth = true
			}
			if strings.Contains(lower, "succeed") || strings.Contains(lower, "success") {
				needsSuccess = true
			}
			if strings.Contains(lower, "error") {
				needsError = true
			}
		}
	}

	ctx := &pageContext{
		app:             app,
		modelName:       modelName,
		varName:         varName,
		itemVar:         itemVar,
		hasSuccessState: needsSuccess,
		hasErrorState:   needsError,
	}

	// Imports
	coreImports := []string{"Component", "OnInit", "signal", "inject"}
	b.WriteString(fmt.Sprintf("import { %s } from '@angular/core';\n", strings.Join(coreImports, ", ")))
	b.WriteString("import { CommonModule } from '@angular/common';\n")
	if needsRouter {
		b.WriteString("import { RouterModule, Router } from '@angular/router';\n")
	}
	if needsDataState || needsEffect {
		b.WriteString("import { ApiService } from '../../services/api.service';\n")
	}
	if modelName != "" {
		fmt.Fprintf(&b, "import type { %s } from '../../models/types';\n", modelName)
	}

	compName := toPascalCase(page.Name) + "Component"
	selector := "app-" + toKebabCase(page.Name)

	fmt.Fprintf(&b, "\n@Component({\n")
	fmt.Fprintf(&b, "  selector: '%s',\n", selector)
	b.WriteString("  standalone: true,\n")
	if needsRouter {
		b.WriteString("  imports: [CommonModule, RouterModule],\n")
	} else {
		b.WriteString("  imports: [CommonModule],\n")
	}
	b.WriteString("  template: `\n")

	// Template
	fmt.Fprintf(&b, "    <div class=\"%s-page\">\n", toKebabCase(page.Name))

	loopFields := collectLoopFields(page, ctx)
	loopRendered := false
	for _, a := range page.Content {
		if a.Type == "loop" && loopRendered {
			continue
		}
		if a.Type == "loop" {
			loopRendered = true
			writeLoopNG(&b, a.Text, "      ", ctx, loopFields)
			continue
		}
		writeTemplateAction(&b, a, "      ", ctx)
	}

	if needsFormState {
		fmt.Fprintf(&b, "      @if (showForm()) {\n")
		b.WriteString("        <div class=\"modal-overlay\" (click)=\"showForm.set(false)\">\n")
		b.WriteString("          <div class=\"modal\" (click)=\"$event.stopPropagation()\">\n")
		b.WriteString("            <button class=\"modal-close\" (click)=\"showForm.set(false)\">&times;</button>\n")
		if modelName != "" {
			fmt.Fprintf(&b, "            <h2>New %s</h2>\n", modelName)
		}
		b.WriteString("            <!-- TODO: form fields -->\n")
		b.WriteString("          </div>\n")
		b.WriteString("        </div>\n")
		b.WriteString("      }\n")
	}

	b.WriteString("    </div>\n  `\n})\n")

	// Class
	fmt.Fprintf(&b, "export class %s implements OnInit {\n", compName)

	if needsRouter {
		b.WriteString("  private router = inject(Router);\n")
	}
	if needsDataState || needsEffect {
		b.WriteString("  private api = inject(ApiService);\n")
	}
	b.WriteString("  loading = signal(true);\n")
	if needsDataState {
		if modelName != "" {
			fmt.Fprintf(&b, "  %s = signal<%s[]>([]);\n", varName, modelName)
		} else {
			b.WriteString("  data = signal<any[]>([]);\n")
		}
	}
	if needsAuth {
		b.WriteString("  isLoggedIn = signal(false); // TODO: connect to auth\n")
	}
	if needsFormState {
		b.WriteString("  showForm = signal(false);\n")
	}
	if needsSuccess {
		b.WriteString("  success = signal('');\n")
	}
	if needsError {
		b.WriteString("  error = signal('');\n")
	}

	// ngOnInit
	if needsEffect {
		apiPath := "/api/" + toKebabCase(varName)
		b.WriteString("\n  ngOnInit() {\n")
		fmt.Fprintf(&b, "    fetch('%s')\n", apiPath)
		b.WriteString("      .then(res => res.json())\n")
		if modelName != "" {
			fmt.Fprintf(&b, "      .then(res => { this.%s.set(res.data ?? []); this.loading.set(false); })\n", varName)
		} else {
			b.WriteString("      .then(res => { this.data.set(res.data ?? []); this.loading.set(false); })\n")
		}
		b.WriteString("      .catch(() => this.loading.set(false));\n")
		b.WriteString("  }\n")
	} else {
		b.WriteString("\n  ngOnInit() {}\n")
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

	fmt.Fprintf(&b, "\n@Component({\n")
	fmt.Fprintf(&b, "  selector: '%s',\n", selector)
	b.WriteString("  standalone: true,\n")
	b.WriteString("  imports: [CommonModule],\n")
	b.WriteString("  template: `\n")

	hasClick := hasClickHandler(comp)

	if hasClick {
		fmt.Fprintf(&b, "    <div class=\"%s\" (click)=\"onClick.emit()\">\n", toKebabCase(comp.Name))
	} else {
		fmt.Fprintf(&b, "    <div class=\"%s\">\n", toKebabCase(comp.Name))
	}

	// Build context for template generation
	propsMap := make(map[string]string)
	for _, p := range comp.Props {
		propsMap[p.Name] = p.Type
	}
	ctx := &pageContext{
		app:         app,
		props:       propsMap,
		isComponent: true,
	}

	for _, a := range comp.Content {
		writeTemplateAction(&b, a, "      ", ctx)
	}
	b.WriteString("    </div>\n  `\n})\n")

	fmt.Fprintf(&b, "export class %s {\n", compName)

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

func writeTemplateAction(b *strings.Builder, a *ir.Action, indent string, ctx *pageContext) {
	switch a.Type {
	case "display":
		writeDisplayNG(b, a.Text, indent, ctx)
	case "input":
		writeInputNG(b, a.Text, indent, ctx)
	case "interact":
		writeInteractNG(b, a.Text, indent, ctx)
	case "condition":
		writeConditionNG(b, a.Text, indent, ctx)
	case "loop":
		writeLoopNG(b, a.Text, indent, ctx, nil)
	case "query":
		// handled by ngOnInit
	default:
		fmt.Fprintf(b, "%s<!-- TODO: %s -->\n", indent, a.Text)
	}
}

// ── Display ──

func writeDisplayNG(b *strings.Builder, text string, indent string, ctx *pageContext) {
	cleaned := text
	lowerCleaned := strings.ToLower(cleaned)
	for _, prefix := range []string{"show ", "display "} {
		if strings.HasPrefix(lowerCleaned, prefix) {
			cleaned = cleaned[len(prefix):]
			break
		}
	}
	lower := strings.ToLower(cleaned)

	// Hero section
	if strings.Contains(lower, "hero") {
		appName := ""
		if ctx.app != nil {
			appName = ctx.app.Name
		}
		if appName == "" {
			appName = "Welcome"
		}
		fmt.Fprintf(b, "%s<section class=\"hero\">\n", indent)
		fmt.Fprintf(b, "%s  <h1>%s</h1>\n", indent, appName)
		fmt.Fprintf(b, "%s  <p>%s</p>\n", indent, extractTagline(cleaned))
		fmt.Fprintf(b, "%s</section>\n", indent)
		return
	}

	// Summary card
	if strings.Contains(lower, "summary") && (strings.Contains(lower, "card") || strings.Contains(lower, "with")) {
		metrics := extractMetricLabels(cleaned)
		fmt.Fprintf(b, "%s<div class=\"summary-cards\">\n", indent)
		for _, m := range metrics {
			fmt.Fprintf(b, "%s  <div class=\"stat-card\">\n", indent)
			fmt.Fprintf(b, "%s    <span class=\"stat-value\">0</span>\n", indent)
			fmt.Fprintf(b, "%s    <span class=\"stat-label\">%s</span>\n", indent, capitalize(m))
			fmt.Fprintf(b, "%s  </div>\n", indent)
		}
		fmt.Fprintf(b, "%s</div>\n", indent)
		return
	}

	// Greeting
	if strings.Contains(lower, "greeting") {
		fmt.Fprintf(b, "%s<h2 class=\"greeting\">Welcome back!</h2>\n", indent)
		return
	}

	// Explicit button
	if strings.Contains(lower, "button") {
		label := extractQuotedText(cleaned)
		if label == "" {
			label = extractButtonPurpose(lower)
		}
		if strings.Contains(lower, "create") || strings.Contains(lower, "new") {
			fmt.Fprintf(b, "%s<button class=\"fab\" (click)=\"showForm.set(true)\">+ %s</button>\n", indent, label)
		} else {
			fmt.Fprintf(b, "%s<button class=\"btn\">%s</button>\n", indent, label)
		}
		return
	}

	// Badge
	if strings.Contains(lower, "as a badge") || strings.Contains(lower, "as a colored badge") || strings.Contains(lower, "as a small badge") {
		expr := resolveFieldExpr(cleaned, ctx)
		fmt.Fprintf(b, "%s<span class=\"badge\">{{ %s }}</span>\n", indent, expr)
		return
	}

	// Bold
	if strings.Contains(lower, "in bold") {
		expr := resolveFieldExpr(cleaned, ctx)
		fmt.Fprintf(b, "%s<strong>{{ %s }}</strong>\n", indent, expr)
		return
	}

	// Large heading
	if strings.Contains(lower, "large heading") || strings.Contains(lower, "heading") {
		expr := resolveFieldExpr(cleaned, ctx)
		if expr != "null" {
			fmt.Fprintf(b, "%s<h1>{{ %s }}</h1>\n", indent, expr)
		} else {
			fmt.Fprintf(b, "%s<h1>%s</h1>\n", indent, cleaned)
		}
		return
	}

	// Rich text
	if strings.Contains(lower, "rich text") {
		expr := resolveFieldExpr(cleaned, ctx)
		if expr != "null" {
			fmt.Fprintf(b, "%s<div class=\"rich-text\" [innerHTML]=\"%s\"></div>\n", indent, expr)
		} else {
			fmt.Fprintf(b, "%s<div class=\"rich-text\"><!-- rich text content --></div>\n", indent)
		}
		return
	}

	// Icon
	if strings.Contains(lower, "with an icon") {
		expr := resolveFieldExpr(cleaned, ctx)
		fmt.Fprintf(b, "%s<span class=\"with-icon\">{{ %s }}</span>\n", indent, expr)
		return
	}

	// Link
	if strings.Contains(lower, "link") {
		label := extractQuotedText(text)
		if label == "" {
			label = cleaned
		}
		target := "/"
		if strings.Contains(lower, "home") {
			target = "/"
		}
		fmt.Fprintf(b, "%s<a routerLink=\"%s\" class=\"link\">%s</a>\n", indent, target, label)
		return
	}

	// Tags as badges
	if strings.Contains(lower, "tags") && strings.Contains(lower, "badge") {
		fmt.Fprintf(b, "%s<div class=\"badge-list\">\n", indent)
		fmt.Fprintf(b, "%s  <!-- TODO: render tags as badges -->\n", indent)
		fmt.Fprintf(b, "%s</div>\n", indent)
		return
	}

	// List reference
	if strings.Contains(lower, "list of") || strings.Contains(lower, "list ") {
		fmt.Fprintf(b, "%s<!-- %s — rendered by @for below -->\n", indent, text)
		return
	}

	// Field list: "the user's name, email, and avatar"
	if strings.Contains(lower, "'s ") {
		fields := extractFieldList(cleaned)
		if len(fields) > 0 {
			fmt.Fprintf(b, "%s<div class=\"field-group\">\n", indent)
			for _, f := range fields {
				fmt.Fprintf(b, "%s  <div class=\"field\"><span class=\"field-label\">%s</span></div>\n", indent, capitalize(f))
			}
			fmt.Fprintf(b, "%s</div>\n", indent)
			return
		}
	}

	// Account/date meta
	if strings.HasPrefix(lower, "account ") {
		display := cleaned
		if idx := strings.Index(strings.ToLower(display), " like "); idx != -1 {
			display = strings.TrimSpace(display[:idx])
		}
		fmt.Fprintf(b, "%s<p class=\"meta\">%s</p>\n", indent, display)
		return
	}

	// Relative format
	if strings.Contains(lower, "relative format") {
		expr := resolveFieldExpr(cleaned, ctx)
		fmt.Fprintf(b, "%s<time>{{ %s }}</time>\n", indent, expr)
		return
	}
	if strings.Contains(lower, "in red") {
		expr := resolveFieldExpr(cleaned, ctx)
		fmt.Fprintf(b, "%s<span class=\"text-danger\">{{ %s }}</span>\n", indent, expr)
		return
	}

	// Sidebar
	if strings.Contains(lower, "sidebar") {
		fmt.Fprintf(b, "%s<aside class=\"sidebar\">\n", indent)
		fmt.Fprintf(b, "%s  <!-- %s -->\n", indent, cleaned)
		fmt.Fprintf(b, "%s</aside>\n", indent)
		return
	}

	// Generic "the X"
	if strings.HasPrefix(lower, "the ") {
		expr := resolveFieldExpr(cleaned, ctx)
		if expr != "null" {
			fmt.Fprintf(b, "%s<p>{{ %s }}</p>\n", indent, expr)
			return
		}
	}

	// Fallback
	fmt.Fprintf(b, "%s<!-- TODO: %s -->\n", indent, text)
	fmt.Fprintf(b, "%s<div class=\"%s\"></div>\n", indent, slugify(text))
}

// ── Input ──

func writeInputNG(b *strings.Builder, text string, indent string, ctx *pageContext) {
	lower := strings.ToLower(text)

	if strings.Contains(lower, "search") {
		fmt.Fprintf(b, "%s<input type=\"search\" placeholder=\"Search...\" class=\"search-input\" />\n", indent)
		return
	}
	if strings.Contains(lower, "dropdown") || strings.Contains(lower, "filter by") || strings.Contains(lower, "select") {
		label := "All"
		if strings.Contains(lower, "status") {
			label = "All Statuses"
		} else if strings.Contains(lower, "priority") {
			label = "All Priorities"
		} else if strings.Contains(lower, "category") {
			label = "Select Category"
		}
		fmt.Fprintf(b, "%s<select class=\"filter-select\">\n", indent)
		fmt.Fprintf(b, "%s  <option value=\"\">%s</option>\n", indent, label)
		fmt.Fprintf(b, "%s</select>\n", indent)
		return
	}
	if strings.Contains(lower, "date") && (strings.Contains(lower, "picker") || strings.Contains(lower, "range")) {
		fmt.Fprintf(b, "%s<input type=\"date\" class=\"date-filter\" />\n", indent)
		return
	}
	if strings.Contains(lower, "button") && (strings.Contains(lower, "create") || strings.Contains(lower, "new") || strings.Contains(lower, "add")) {
		label := "New"
		fmt.Fprintf(b, "%s<button class=\"fab\" (click)=\"showForm.set(true)\">+ %s</button>\n", indent, label)
		return
	}
	if strings.Contains(lower, "form") {
		writeFormNG(b, text, indent, ctx)
		return
	}
	if strings.Contains(lower, "toggle") {
		label := "Toggle"
		if strings.Contains(lower, "published") {
			label = "Published"
		}
		fmt.Fprintf(b, "%s<label class=\"toggle\"><input type=\"checkbox\" /> <span>%s</span></label>\n", indent, label)
		return
	}
	if strings.Contains(lower, "file") || strings.Contains(lower, "upload") {
		label := "Upload file"
		if strings.Contains(lower, "avatar") {
			label = "Upload avatar"
		} else if strings.Contains(lower, "cover") || strings.Contains(lower, "image") {
			label = "Upload image"
		}
		fmt.Fprintf(b, "%s<div class=\"file-upload\">\n", indent)
		fmt.Fprintf(b, "%s  <label>%s</label>\n", indent, label)
		fmt.Fprintf(b, "%s  <input type=\"file\" accept=\"image/*\" />\n", indent)
		fmt.Fprintf(b, "%s</div>\n", indent)
		return
	}
	if strings.Contains(lower, "tag") && strings.Contains(lower, "selector") {
		fmt.Fprintf(b, "%s<div class=\"tag-selector\"><!-- TODO: tag selector --></div>\n", indent)
		return
	}
	if strings.Contains(lower, "rich text") || strings.Contains(lower, "editor") {
		fmt.Fprintf(b, "%s<div class=\"rich-text-editor\">\n", indent)
		fmt.Fprintf(b, "%s  <textarea placeholder=\"Write your content...\"></textarea>\n", indent)
		fmt.Fprintf(b, "%s</div>\n", indent)
		return
	}
	if strings.Contains(lower, "text input") || strings.Contains(lower, "input for") {
		fieldName := "field"
		for _, marker := range []string{"input for ", "text input for "} {
			if idx := strings.Index(lower, marker); idx != -1 {
				fieldName = strings.TrimSpace(text[idx+len(marker):])
				break
			}
		}
		fmt.Fprintf(b, "%s<div class=\"form-field\">\n", indent)
		fmt.Fprintf(b, "%s  <label>%s</label>\n", indent, capitalize(fieldName))
		fmt.Fprintf(b, "%s  <input type=\"text\" placeholder=\"%s\" />\n", indent, fieldName)
		fmt.Fprintf(b, "%s</div>\n", indent)
		return
	}
	fmt.Fprintf(b, "%s<input type=\"text\" placeholder=\"%s\" />\n", indent, text)
}

func writeFormNG(b *strings.Builder, text string, indent string, ctx *pageContext) {
	lower := strings.ToLower(text)
	fields := extractFormFields(lower, ctx)

	onSubmit := "/* TODO: submit */"
	if ctx.hasSuccessState && ctx.hasErrorState {
		onSubmit = "error.set(''); success.set('Saved successfully')"
	} else if ctx.hasSuccessState {
		onSubmit = "success.set('Saved successfully')"
	}

	fmt.Fprintf(b, "%s<form class=\"form\" (ngSubmit)=\"%s\">\n", indent, onSubmit)
	for _, f := range fields {
		inputType := "text"
		fl := strings.ToLower(f)
		if strings.Contains(fl, "email") {
			inputType = "email"
		} else if strings.Contains(fl, "password") {
			inputType = "password"
		} else if strings.Contains(fl, "date") {
			inputType = "date"
		} else if strings.Contains(fl, "number") || strings.Contains(fl, "count") {
			inputType = "number"
		}
		fmt.Fprintf(b, "%s  <div class=\"form-field\">\n", indent)
		fmt.Fprintf(b, "%s    <label>%s</label>\n", indent, capitalize(f))
		fmt.Fprintf(b, "%s    <input type=\"%s\" name=\"%s\" placeholder=\"%s\" />\n", indent, inputType, toCamelCase(f), capitalize(f))
		fmt.Fprintf(b, "%s  </div>\n", indent)
	}
	fmt.Fprintf(b, "%s  <button type=\"submit\">Save</button>\n", indent)
	fmt.Fprintf(b, "%s</form>\n", indent)
}

// ── Loop ──

func writeLoopNG(b *strings.Builder, text string, indent string, ctx *pageContext, fields []string) {
	dataVar := ctx.varName
	if dataVar == "" {
		dataVar = "data"
	}
	item := ctx.itemVar
	if item == "" {
		item = "item"
	}

	compRef := extractComponentRef(text)
	if compRef != "" {
		fmt.Fprintf(b, "%s@for (%s of %s(); track %s.id) {\n", indent, item, dataVar, item)
		compSelector := "app-" + toKebabCase(compRef)
		fmt.Fprintf(b, "%s  <%s [%s]=\"%s\" (onClick)=\"/* TODO */\"></%s>\n", indent, compSelector, item, item, compSelector)
		fmt.Fprintf(b, "%s}\n", indent)
		return
	}

	if len(fields) == 0 {
		fields = extractLoopFields(text, ctx)
	}

	modelClass := toKebabCase(ctx.modelName)
	if modelClass == "" {
		modelClass = "item"
	}
	fmt.Fprintf(b, "%s@for (%s of %s(); track %s.id) {\n", indent, item, dataVar, item)
	fmt.Fprintf(b, "%s  <div class=\"%s-item\">\n", indent, modelClass)
	if len(fields) > 0 {
		for _, f := range fields {
			fieldExpr := item + "." + f
			fl := strings.ToLower(f)
			if fl == "status" || fl == "role" || fl == "priority" || fl == "category" {
				fmt.Fprintf(b, "%s    <span class=\"badge\">{{ %s }}</span>\n", indent, fieldExpr)
			} else if fl == "title" || fl == "name" {
				fmt.Fprintf(b, "%s    <h3>{{ %s }}</h3>\n", indent, fieldExpr)
			} else if strings.Contains(fl, "date") || fl == "due" || fl == "created" || strings.Contains(fl, "published") {
				fmt.Fprintf(b, "%s    <time>{{ %s }}</time>\n", indent, fieldExpr)
			} else if strings.Contains(fl, "excerpt") {
				fmt.Fprintf(b, "%s    <p>{{ %s }}</p>\n", indent, fieldExpr)
			} else if strings.Contains(fl, "count") || strings.Contains(fl, "view") {
				fmt.Fprintf(b, "%s    <span class=\"count\">{{ %s }}</span>\n", indent, fieldExpr)
			} else {
				fmt.Fprintf(b, "%s    <span>{{ %s }}</span>\n", indent, fieldExpr)
			}
		}
	} else {
		fmt.Fprintf(b, "%s    <span>{{ %s | json }}</span>\n", indent, item)
	}
	fmt.Fprintf(b, "%s  </div>\n", indent)
	fmt.Fprintf(b, "%s}\n", indent)
}

// ── Condition ──

func writeConditionNG(b *strings.Builder, text string, indent string, ctx *pageContext) {
	// Components don't have page-level state (loading, data, isLoggedIn) — emit as comment
	if ctx.isComponent {
		fmt.Fprintf(b, "%s<!-- %s -->\n", indent, text)
		return
	}

	lower := strings.ToLower(text)
	dataVar := ctx.varName
	if dataVar == "" {
		dataVar = "data"
	}

	// Loading
	if strings.Contains(lower, "while loading") || strings.Contains(lower, "is loading") {
		if strings.Contains(lower, "skeleton") {
			fmt.Fprintf(b, "%s@if (loading()) {\n", indent)
			fmt.Fprintf(b, "%s  <div class=\"skeleton-screen\">\n", indent)
			fmt.Fprintf(b, "%s    @for (i of [1,2,3]; track i) {\n", indent)
			fmt.Fprintf(b, "%s      <div class=\"skeleton-item\"></div>\n", indent)
			fmt.Fprintf(b, "%s    }\n", indent)
			fmt.Fprintf(b, "%s  </div>\n", indent)
			fmt.Fprintf(b, "%s}\n", indent)
		} else {
			fmt.Fprintf(b, "%s@if (loading()) {\n", indent)
			fmt.Fprintf(b, "%s  <div class=\"loading-spinner\">\n", indent)
			fmt.Fprintf(b, "%s    <div class=\"spinner\"></div>\n", indent)
			fmt.Fprintf(b, "%s  </div>\n", indent)
			fmt.Fprintf(b, "%s}\n", indent)
		}
		return
	}

	// Empty state
	if strings.Contains(lower, "if no ") || strings.Contains(lower, "if there are no") {
		message := extractQuotedText(text)
		if message == "" {
			message = extractConditionMessage(text)
		}
		if message == "" {
			message = "No items found."
		}
		fmt.Fprintf(b, "%s@if (!loading() && %s().length === 0) {\n", indent, dataVar)
		fmt.Fprintf(b, "%s  <div class=\"empty-state\">%s</div>\n", indent, message)
		fmt.Fprintf(b, "%s}\n", indent)
		return
	}

	// "does not exist" / "not found"
	if strings.Contains(lower, "does not exist") || strings.Contains(lower, "not found") {
		fmt.Fprintf(b, "%s<!-- %s -->\n", indent, text)
		return
	}

	// Auth
	if strings.Contains(lower, "not logged in") || strings.Contains(lower, "is not logged in") {
		content := extractConditionContent(text)
		fmt.Fprintf(b, "%s@if (!isLoggedIn()) {\n", indent)
		fmt.Fprintf(b, "%s  <div class=\"auth-prompt\">\n", indent)
		writeConditionButtonsNG(b, content, indent+"    ", ctx)
		fmt.Fprintf(b, "%s  </div>\n", indent)
		fmt.Fprintf(b, "%s}\n", indent)
		return
	}
	if strings.Contains(lower, "is logged in") || strings.Contains(lower, "logged in") {
		content := extractConditionContent(text)
		fmt.Fprintf(b, "%s@if (isLoggedIn()) {\n", indent)
		fmt.Fprintf(b, "%s  <div>\n", indent)
		writeConditionButtonsNG(b, content, indent+"    ", ctx)
		fmt.Fprintf(b, "%s  </div>\n", indent)
		fmt.Fprintf(b, "%s}\n", indent)
		return
	}

	// Success
	if strings.Contains(lower, "succeed") || strings.Contains(lower, "success") {
		message := extractQuotedText(text)
		if message == "" {
			message = extractConditionMessage(text)
		}
		if message == "" {
			message = "Success!"
		}
		fmt.Fprintf(b, "%s@if (success()) {\n", indent)
		fmt.Fprintf(b, "%s  <div class=\"alert alert-success\">{{ success() || '%s' }}</div>\n", indent, message)
		fmt.Fprintf(b, "%s}\n", indent)
		return
	}

	// Error
	if strings.Contains(lower, "error") {
		fmt.Fprintf(b, "%s@if (error()) {\n", indent)
		fmt.Fprintf(b, "%s  <div class=\"alert alert-error\">{{ error() }}</div>\n", indent)
		fmt.Fprintf(b, "%s}\n", indent)
		return
	}

	// Nested/replies
	if strings.Contains(lower, "has replies") || strings.Contains(lower, "nested") {
		fmt.Fprintf(b, "%s<!-- %s -->\n", indent, text)
		return
	}

	// Overdue
	if strings.Contains(lower, "overdue") {
		expr := resolveFieldExpr(text, ctx)
		fmt.Fprintf(b, "%s<!-- %s -->\n", indent, text)
		if expr != "null" {
			fmt.Fprintf(b, "%s<span class=\"text-danger\">{{ %s }}</span>\n", indent, expr)
		}
		return
	}

	fmt.Fprintf(b, "%s<!-- TODO: %s -->\n", indent, text)
}

func writeConditionButtonsNG(b *strings.Builder, content string, indent string, ctx *pageContext) {
	lower := strings.ToLower(content)

	labels := extractAllQuotedText(content)
	if len(labels) > 0 {
		for _, label := range labels {
			target := toKebabCase(label)
			if strings.Contains(strings.ToLower(label), "dashboard") {
				target = "dashboard"
			}
			fmt.Fprintf(b, "%s<button (click)=\"navigate('/%s')\">%s</button>\n", indent, target, label)
		}
		return
	}

	if strings.Contains(lower, "login") && strings.Contains(lower, "signup") {
		fmt.Fprintf(b, "%s<button (click)=\"navigate('/login')\">Log In</button>\n", indent)
		fmt.Fprintf(b, "%s<button (click)=\"navigate('/sign-up')\">Sign Up</button>\n", indent)
		return
	}

	if strings.Contains(lower, "form") {
		writeFormNG(b, content, indent, ctx)
		return
	}

	if strings.Contains(lower, "button") {
		label := extractButtonPurpose(lower)
		target := ""
		for _, word := range strings.Fields(lower) {
			if word != "go" && word != "to" && word != "button" && word != "the" && word != "a" {
				target = word
				break
			}
		}
		if target == "" {
			target = "home"
		}
		fmt.Fprintf(b, "%s<button (click)=\"navigate('/%s')\">%s</button>\n", indent, toKebabCase(target), label)
		return
	}

	fmt.Fprintf(b, "%s<p>%s</p>\n", indent, content)
}

// ── Interaction ──

func writeInteractNG(b *strings.Builder, text string, indent string, ctx *pageContext) {
	lower := strings.ToLower(text)

	if strings.Contains(lower, "clicking") || strings.Contains(lower, "click") {
		label := extractButtonLabel(text)

		if strings.Contains(lower, "navigate") {
			target := extractNavTarget(text)
			fmt.Fprintf(b, "%s<button (click)=\"navigate('/%s')\">%s</button>\n", indent, toKebabCase(target), label)
			return
		}

		if strings.Contains(lower, "opens a form") || strings.Contains(lower, "open a form") {
			fmt.Fprintf(b, "%s<button (click)=\"showForm.set(true)\">%s</button>\n", indent, capitalize(label))
			return
		}

		if strings.Contains(lower, "triggers") || strings.Contains(lower, "on_click") || strings.Contains(lower, "onclick") {
			fmt.Fprintf(b, "%s<!-- clicking triggers onClick — handled by component wrapper -->\n", indent)
			return
		}

		if strings.Contains(lower, "update") || strings.Contains(lower, "save") {
			quoted := extractQuotedText(text)
			if quoted != "" {
				label = quoted
			} else if label == "Click" {
				label = "Save"
			}
			handler := "/* TODO: save */"
			if ctx.hasSuccessState && ctx.hasErrorState {
				handler = "error.set(''); success.set('Saved successfully')"
			} else if ctx.hasSuccessState {
				handler = "success.set('Saved successfully')"
			}
			fmt.Fprintf(b, "%s<button (click)=\"%s\">%s</button>\n", indent, handler, label)
			return
		}

		if strings.Contains(lower, "saves") || strings.Contains(lower, "submit") {
			quoted := extractQuotedText(text)
			if quoted != "" {
				label = quoted
			}
			handler := "/* TODO: save */"
			if ctx.hasSuccessState && ctx.hasErrorState {
				handler = "error.set(''); success.set('Saved successfully')"
			}
			fmt.Fprintf(b, "%s<button (click)=\"%s\">%s</button>\n", indent, handler, label)
			return
		}

		if strings.Contains(lower, "opens") {
			if label != "Click" && strings.Contains(label, " ") {
				fmt.Fprintf(b, "%s<button>%s</button>\n", indent, label)
			} else {
				fmt.Fprintf(b, "%s<!-- %s — handled by item click handler -->\n", indent, text)
			}
			return
		}

		fmt.Fprintf(b, "%s<button>%s</button>\n", indent, label)
		return
	}

	fmt.Fprintf(b, "%s<!-- TODO: %s -->\n", indent, text)
}

// ── Helpers ──

func detectPageModel(page *ir.Page, app *ir.Application) (modelName, varName, itemVar string) {
	for _, a := range page.Content {
		if a.Type == "query" || a.Type == "loop" {
			for _, m := range app.Data {
				lowerText := strings.ToLower(a.Text)
				lowerModel := strings.ToLower(m.Name)
				if strings.Contains(lowerText, lowerModel+"s") || strings.Contains(lowerText, lowerModel) {
					return m.Name, strings.ToLower(m.Name) + "s", strings.ToLower(m.Name)
				}
			}
		}
	}
	return "", "data", "item"
}

func findModel(app *ir.Application, name string) *ir.DataModel {
	for _, m := range app.Data {
		if strings.EqualFold(m.Name, name) {
			return m
		}
	}
	return nil
}

func collectLoopFields(page *ir.Page, ctx *pageContext) []string {
	seen := map[string]bool{}
	var fields []string
	for _, a := range page.Content {
		if a.Type == "loop" {
			for _, f := range extractLoopFields(a.Text, ctx) {
				if !seen[f] {
					seen[f] = true
					fields = append(fields, f)
				}
			}
		}
	}
	return fields
}

func extractLoopFields(text string, ctx *pageContext) []string {
	lower := strings.ToLower(text)
	for _, marker := range []string{"shows its ", "shows the ", "shows "} {
		if idx := strings.Index(lower, marker); idx != -1 {
			rest := text[idx+len(marker):]
			return parseFieldNames(rest, ctx)
		}
	}
	return nil
}

func parseFieldNames(text string, ctx *pageContext) []string {
	for _, mod := range []string{" as a colored badge", " as a badge", " in bold", " with an icon"} {
		text = strings.Replace(strings.ToLower(text), mod, "", -1)
	}
	text = strings.ReplaceAll(text, " and ", ", ")
	parts := strings.Split(text, ",")
	var fields []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		resolved := resolveFieldName(p, ctx)
		if resolved != "" {
			fields = append(fields, resolved)
		}
	}
	return fields
}

func resolveFieldName(name string, ctx *pageContext) string {
	name = strings.TrimSpace(strings.ToLower(name))
	// Reject strings that clearly aren't field names
	if strings.Contains(name, ",") {
		return ""
	}
	model := findModel(ctx.app, ctx.modelName)
	if model == nil {
		if len(strings.Fields(name)) <= 2 {
			return toCamelCase(name)
		}
		return ""
	}
	for _, f := range model.Fields {
		if strings.ToLower(f.Name) == name {
			return f.Name
		}
	}
	for _, f := range model.Fields {
		if strings.ToLower(f.Name+" "+f.Type) == name {
			return f.Name
		}
	}
	for _, f := range model.Fields {
		if strings.Contains(name, strings.ToLower(f.Name)) {
			return f.Name
		}
	}
	// Model exists but no field matched — don't guess
	return ""
}

func resolveFieldExpr(text string, ctx *pageContext) string {
	lower := strings.ToLower(text)
	stripped := lower
	if idx := strings.Index(stripped, " like "); idx != -1 {
		stripped = stripped[:idx]
	}
	for _, mod := range []string{
		"in bold", "as a colored badge", "as a badge", "as a small badge", "with an icon",
		"in relative format", "in red", "the ", "show ", "in large heading",
		"as a heading", "as rich text", "truncated to ",
	} {
		stripped = strings.Replace(stripped, mod, " ", -1)
	}
	if idx := strings.Index(stripped, "truncated"); idx != -1 {
		stripped = stripped[:idx]
	}
	stripped = strings.TrimSpace(stripped)

	// Reject text containing commas (clearly not a single field expression)
	if strings.Contains(stripped, ",") {
		return "null"
	}

	// Component prop resolution
	for propName, propType := range ctx.props {
		propLower := strings.ToLower(propName)
		if strings.Contains(stripped, propLower+" ") {
			fieldPart := strings.TrimSpace(strings.Replace(stripped, propLower+" ", "", 1))
			fieldPart = strings.TrimSpace(strings.TrimPrefix(fieldPart, "its "))
			model := findModel(ctx.app, propType)
			if model != nil {
				for _, f := range model.Fields {
					if strings.Contains(fieldPart, strings.ToLower(f.Name)) {
						return propName + "." + f.Name
					}
					if strings.ToLower(f.Name+" "+f.Type) == fieldPart {
						return propName + "." + f.Name
					}
				}
			}
			if len(strings.Fields(fieldPart)) <= 2 {
				return propName + "." + toCamelCase(fieldPart)
			}
			return "null"
		}
		if model := findModel(ctx.app, propType); model != nil {
			for _, f := range model.Fields {
				if strings.Contains(stripped, strings.ToLower(f.Name)) {
					return propName + "." + f.Name
				}
			}
		}
	}

	if strings.Contains(lower, "'s ") {
		return "null"
	}

	// Note: itemVar (loop variable) is NOT used here because it only exists
	// inside @for blocks. Display actions outside loops must use other sources.

	return "null"
}

func extractQuotedText(text string) string {
	if idx := strings.Index(text, "\""); idx != -1 {
		rest := text[idx+1:]
		if end := strings.Index(rest, "\""); end != -1 {
			return rest[:end]
		}
	}
	return ""
}

func extractAllQuotedText(text string) []string {
	var results []string
	remaining := text
	for {
		idx := strings.Index(remaining, "\"")
		if idx == -1 {
			break
		}
		rest := remaining[idx+1:]
		end := strings.Index(rest, "\"")
		if end == -1 {
			break
		}
		results = append(results, rest[:end])
		remaining = rest[end+1:]
	}
	return results
}

func extractButtonPurpose(lower string) string {
	lower = strings.TrimPrefix(lower, "a ")
	lower = strings.TrimPrefix(lower, "an ")
	lower = strings.TrimSuffix(lower, " button")
	lower = strings.TrimSuffix(lower, " buttons")
	words := strings.Fields(lower)
	for i := range words {
		words[i] = capitalize(words[i])
	}
	if len(words) == 0 {
		return "Click"
	}
	return strings.Join(words, " ")
}

func extractMetricLabels(text string) []string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, "with ")
	if idx == -1 {
		return []string{"Total"}
	}
	rest := text[idx+5:]
	rest = strings.ReplaceAll(rest, " and ", ", ")
	parts := strings.Split(rest, ",")
	var labels []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			labels = append(labels, p)
		}
	}
	if len(labels) == 0 {
		return []string{"Total"}
	}
	return labels
}

func extractFieldList(text string) []string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, "'s ")
	if idx == -1 {
		return nil
	}
	rest := text[idx+3:]
	rest = strings.ReplaceAll(rest, " and ", ", ")
	parts := strings.Split(rest, ",")
	var fields []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			fields = append(fields, p)
		}
	}
	return fields
}

func extractFormFields(lower string, ctx *pageContext) []string {
	for _, marker := range []string{"form to update ", "form to create ", "form to edit "} {
		if idx := strings.Index(lower, marker); idx != -1 {
			rest := lower[idx+len(marker):]
			if strings.HasPrefix(rest, "a ") || strings.HasPrefix(rest, "an ") || strings.HasPrefix(rest, "or edit a ") {
				rest = strings.TrimPrefix(rest, "a ")
				rest = strings.TrimPrefix(rest, "an ")
				rest = strings.TrimPrefix(rest, "or edit a ")
				modelName := strings.TrimSpace(rest)
				model := findModel(ctx.app, modelName)
				if model != nil {
					var fields []string
					for _, f := range model.Fields {
						fl := strings.ToLower(f.Name)
						if fl == "created" || fl == "updated" || fl == "createdat" || fl == "updatedat" {
							continue
						}
						if f.Encrypted {
							continue
						}
						fields = append(fields, f.Name)
					}
					return fields
				}
			}
			rest = strings.ReplaceAll(rest, " and ", ", ")
			parts := strings.Split(rest, ",")
			var fields []string
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					fields = append(fields, p)
				}
			}
			return fields
		}
	}
	return []string{"field"}
}

func extractComponentRef(text string) string {
	lower := strings.ToLower(text)
	for _, marker := range []string{" as a ", " as "} {
		if idx := strings.Index(lower, marker); idx != -1 {
			rest := strings.TrimSpace(text[idx+len(marker):])
			if space := strings.IndexByte(rest, ' '); space != -1 {
				rest = rest[:space]
			}
			if len(rest) > 0 && rest[0] >= 'A' && rest[0] <= 'Z' {
				return rest
			}
		}
	}
	return ""
}

func extractConditionMessage(text string) string {
	lower := strings.ToLower(text)
	for _, marker := range []string{", show ", " show "} {
		if idx := strings.Index(lower, marker); idx != -1 {
			msg := strings.TrimSpace(text[idx+len(marker):])
			msgLower := strings.ToLower(msg)
			if strings.HasPrefix(msgLower, "a ") || strings.HasPrefix(msgLower, "an ") || strings.HasPrefix(msgLower, "the ") {
				return ""
			}
			return msg
		}
	}
	return ""
}

func extractConditionContent(text string) string {
	lower := strings.ToLower(text)
	for _, marker := range []string{", show ", " show "} {
		if idx := strings.Index(lower, marker); idx != -1 {
			return strings.TrimSpace(text[idx+len(marker):])
		}
	}
	return text
}

func extractTagline(text string) string {
	lower := strings.ToLower(text)
	for _, marker := range []string{"with the ", "and "} {
		if idx := strings.Index(lower, marker); idx != -1 {
			return strings.TrimSpace(text[idx+len(marker):])
		}
	}
	return "Get things done, beautifully."
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
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
	if q := extractQuotedText(text); q != "" {
		return q
	}
	lower := strings.ToLower(text)
	if idx := strings.Index(lower, "clicking the "); idx != -1 {
		after := text[idx+len("clicking the "):]
		if end := strings.Index(strings.ToLower(after), " button"); end != -1 {
			return after[:end]
		}
		if end := strings.Index(strings.ToLower(after), " navigates"); end != -1 {
			return after[:end]
		}
		if end := strings.Index(strings.ToLower(after), " opens"); end != -1 {
			return after[:end]
		}
		if end := strings.Index(strings.ToLower(after), " updates"); end != -1 {
			return after[:end]
		}
	}
	if idx := strings.Index(lower, "clicking a "); idx != -1 {
		after := text[idx+len("clicking a "):]
		if space := strings.IndexByte(after, ' '); space != -1 {
			return after[:space]
		}
		return after
	}
	if idx := strings.Index(lower, "clicking "); idx != -1 {
		after := text[idx+len("clicking "):]
		words := strings.Fields(after)
		var labelParts []string
		for _, w := range words {
			wl := strings.ToLower(w)
			if wl == "updates" || wl == "opens" || wl == "navigates" || wl == "triggers" || wl == "saves" {
				break
			}
			labelParts = append(labelParts, w)
		}
		if len(labelParts) > 0 {
			return strings.Join(labelParts, " ")
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
