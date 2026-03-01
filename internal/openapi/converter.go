package openapi

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/barun-bash/human/internal/parser"
)

// ToHuman converts an OpenAPI spec to Human language source code.
func ToHuman(spec *Spec, appName string) (string, error) {
	if appName == "" {
		appName = toPascalCase(spec.Info.Title)
	}
	if appName == "" {
		appName = "MyApp"
	}

	var sections []string

	// App declaration
	sections = append(sections, fmt.Sprintf("app %s is a web application", appName))

	// Data models from component schemas
	schemaNames := sortedKeys(spec.Components.Schemas)
	for _, name := range schemaNames {
		schema := spec.Components.Schemas[name]
		if schema.Type == "object" || len(schema.Properties) > 0 {
			block := schemaToData(name, schema)
			if block != "" {
				sections = append(sections, block)
			}
		}
	}

	// API endpoints from paths
	pathKeys := sortedKeys(spec.Paths)
	for _, path := range pathKeys {
		item := spec.Paths[path]
		blocks := pathToAPIs(path, item, spec)
		sections = append(sections, blocks...)
	}

	// Authentication from security schemes
	if len(spec.Components.SecuritySchemes) > 0 {
		authBlock := securityToAuth(spec.Components.SecuritySchemes)
		if authBlock != "" {
			sections = append(sections, authBlock)
		}
	}

	// Build block
	sections = append(sections, "build with:\n  backend using Node with Express\n  database using PostgreSQL")

	code := strings.Join(sections, "\n\n") + "\n"

	// Validate via parser
	if _, err := parser.Parse(code); err != nil {
		return code, fmt.Errorf("generated code has syntax issues (usable but may need edits): %w", err)
	}

	return code, nil
}

// schemaToData converts an OpenAPI schema to a Human data: block.
func schemaToData(name string, schema Schema) string {
	if len(schema.Properties) == 0 {
		return ""
	}

	requiredSet := map[string]bool{}
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("data %s:", toPascalCase(name)))

	// Sort properties for deterministic output
	propNames := sortedKeys(schema.Properties)
	for _, prop := range propNames {
		propSchema := schema.Properties[prop]
		humanType := schemaTypeToHuman(prop, propSchema)
		fieldName := toHumanFieldName(prop)

		if len(propSchema.Enum) > 0 {
			enumVals := enumToStrings(propSchema.Enum)
			lines = append(lines, fmt.Sprintf("  has a %s which is either %s", fieldName, joinEnum(enumVals)))
		} else if propSchema.Type == "array" && propSchema.Items != nil {
			target := refName(propSchema.Items.Ref)
			if target == "" && propSchema.Items.Type != "" {
				target = propSchema.Items.Type
			}
			if target != "" {
				lines = append(lines, fmt.Sprintf("  has many %s", toPascalCase(target)))
			}
		} else if requiredSet[prop] {
			lines = append(lines, fmt.Sprintf("  has a %s which is required %s", fieldName, humanType))
		} else {
			lines = append(lines, fmt.Sprintf("  has a %s which is %s", fieldName, humanType))
		}
	}

	return strings.Join(lines, "\n")
}

// pathToAPIs converts an OpenAPI path + operations to Human api: blocks.
func pathToAPIs(path string, item PathItem, spec *Spec) []string {
	var blocks []string

	ops := []struct {
		method string
		op     *Operation
	}{
		{"GET", item.Get},
		{"POST", item.Post},
		{"PUT", item.Put},
		{"PATCH", item.Patch},
		{"DELETE", item.Delete},
	}

	for _, entry := range ops {
		if entry.op == nil {
			continue
		}
		block := operationToAPI(entry.method, path, entry.op, item.Parameters, spec)
		if block != "" {
			blocks = append(blocks, block)
		}
	}

	return blocks
}

func operationToAPI(method, path string, op *Operation, pathParams []Parameter, spec *Spec) string {
	// Determine name
	name := op.OperationID
	if name == "" {
		name = methodPathToName(method, path)
	}
	name = toPascalCase(name)

	var lines []string
	lines = append(lines, fmt.Sprintf("api %s:", name))

	// Check if authentication is required
	hasAuth := len(op.Security) > 0 || len(spec.Security) > 0
	if hasAuth {
		lines = append(lines, "  requires authentication")
	}

	// Collect all parameters (path-level + operation-level)
	allParams := append(pathParams, op.Parameters...)

	// Describe accepted parameters
	var paramNames []string
	for _, p := range allParams {
		if p.Name != "" && p.In != "header" {
			paramNames = append(paramNames, toHumanFieldName(p.Name))
		}
	}

	// Request body fields
	if op.RequestBody != nil {
		for _, media := range op.RequestBody.Content {
			for prop := range media.Schema.Properties {
				paramNames = append(paramNames, toHumanFieldName(prop))
			}
			// If it's a $ref to a schema, mention the model
			if media.Schema.Ref != "" {
				modelName := refName(media.Schema.Ref)
				paramNames = append(paramNames, toHumanFieldName(modelName))
			}
		}
	}

	if len(paramNames) > 0 {
		lines = append(lines, fmt.Sprintf("  accepts %s", strings.Join(paramNames, " and ")))
	}

	// Describe the action based on method
	model := inferModelFromPath(path)
	switch method {
	case "GET":
		if strings.Contains(path, "{") {
			lines = append(lines, fmt.Sprintf("  fetch the %s by id", model))
		} else {
			lines = append(lines, fmt.Sprintf("  fetch all %s", pluralize(model)))
		}
	case "POST":
		lines = append(lines, fmt.Sprintf("  create the %s", model))
	case "PUT", "PATCH":
		lines = append(lines, fmt.Sprintf("  update the %s", model))
	case "DELETE":
		lines = append(lines, fmt.Sprintf("  delete the %s", model))
	}

	// Describe response
	if resp, ok := op.Responses["200"]; ok && resp.Description != "" {
		lines = append(lines, fmt.Sprintf("  respond with the %s", model))
	} else if _, ok := op.Responses["201"]; ok {
		lines = append(lines, fmt.Sprintf("  respond with the created %s", model))
	}

	return strings.Join(lines, "\n")
}

// securityToAuth converts OpenAPI security schemes to a Human authentication: block.
func securityToAuth(schemes map[string]SecurityScheme) string {
	var lines []string
	lines = append(lines, "authentication:")

	for _, scheme := range schemes {
		switch scheme.Type {
		case "http":
			if strings.ToLower(scheme.Scheme) == "bearer" {
				if strings.ToLower(scheme.BearerFormat) == "jwt" {
					lines = append(lines, "  method JWT tokens")
				} else {
					lines = append(lines, "  method token based")
				}
			} else if strings.ToLower(scheme.Scheme) == "basic" {
				lines = append(lines, "  method email and password")
			}
		case "apiKey":
			lines = append(lines, "  method API keys")
		case "oauth2":
			lines = append(lines, "  method OAuth")
		default:
			lines = append(lines, "  method token based")
		}
		break // only use the first scheme
	}

	if len(lines) == 1 {
		return ""
	}
	return strings.Join(lines, "\n")
}

// ── Helpers ──

func schemaTypeToHuman(fieldName string, schema Schema) string {
	if schema.Format != "" {
		switch schema.Format {
		case "email":
			return "email"
		case "date":
			return "date"
		case "date-time":
			return "datetime"
		case "password":
			return "encrypted text"
		case "uri", "url":
			return "url"
		case "int32", "int64":
			return "number"
		case "float", "double":
			return "number"
		}
	}

	switch schema.Type {
	case "string":
		// Try to infer from field name
		lower := strings.ToLower(fieldName)
		if strings.Contains(lower, "email") {
			return "email"
		}
		if strings.Contains(lower, "password") {
			return "encrypted text"
		}
		if strings.Contains(lower, "date") || strings.Contains(lower, "time") {
			return "datetime"
		}
		if strings.Contains(lower, "url") || strings.Contains(lower, "link") {
			return "url"
		}
		return "text"
	case "integer", "number":
		return "number"
	case "boolean":
		return "text" // Human IR uses text for boolean-like fields
	case "array":
		return "text" // handled separately in schemaToData
	default:
		return "text"
	}
}

func enumToStrings(vals []interface{}) []string {
	var result []string
	for _, v := range vals {
		result = append(result, fmt.Sprintf("%v", v))
	}
	return result
}

func joinEnum(vals []string) string {
	if len(vals) == 0 {
		return ""
	}
	quoted := make([]string, len(vals))
	for i, v := range vals {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	if len(quoted) == 1 {
		return quoted[0]
	}
	return strings.Join(quoted[:len(quoted)-1], ", ") + " or " + quoted[len(quoted)-1]
}

func toHumanFieldName(name string) string {
	// Convert camelCase/snake_case to "space separated"
	var result []rune
	for i, r := range name {
		if r == '_' || r == '-' {
			result = append(result, ' ')
			continue
		}
		if unicode.IsUpper(r) && i > 0 && !unicode.IsUpper(rune(name[i-1])) {
			result = append(result, ' ')
		}
		result = append(result, unicode.ToLower(r))
	}
	return strings.TrimSpace(string(result))
}

func inferModelFromPath(path string) string {
	// /tasks/{id} → "task", /api/v1/users → "user"
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		p := parts[i]
		if p == "" || strings.HasPrefix(p, "{") {
			continue
		}
		// Skip version segments
		if strings.HasPrefix(p, "v") && len(p) <= 3 {
			continue
		}
		if p == "api" {
			continue
		}
		return singularize(p)
	}
	return "resource"
}

func singularize(s string) string {
	s = strings.ToLower(s)
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "sses") {
		// addresses → address
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "xes") || strings.HasSuffix(s, "zes") {
		return s[:len(s)-2]
	}
	// Don't strip trailing s from words ending in "us", "ss", "is"
	if strings.HasSuffix(s, "us") || strings.HasSuffix(s, "ss") || strings.HasSuffix(s, "is") {
		return s
	}
	if strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}

func pluralize(s string) string {
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") || strings.HasSuffix(s, "z") {
		return s + "es"
	}
	return s + "s"
}

func methodPathToName(method, path string) string {
	model := inferModelFromPath(path)
	switch method {
	case "GET":
		if strings.Contains(path, "{") {
			return "Get" + toPascalCase(model)
		}
		return "List" + toPascalCase(pluralize(model))
	case "POST":
		return "Create" + toPascalCase(model)
	case "PUT", "PATCH":
		return "Update" + toPascalCase(model)
	case "DELETE":
		return "Delete" + toPascalCase(model)
	}
	return toPascalCase(model)
}

func toPascalCase(s string) string {
	if s == "" {
		return ""
	}
	// Split on spaces, hyphens, underscores
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_' || r == '/'
	})
	var result []string
	for _, w := range words {
		if w == "" {
			continue
		}
		result = append(result, strings.ToUpper(w[:1])+w[1:])
	}
	return strings.Join(result, "")
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
