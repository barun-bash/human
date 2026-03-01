// Package openapi parses OpenAPI/Swagger JSON specifications and converts
// them to Human language source files.
package openapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Spec represents a parsed OpenAPI 3.x specification.
type Spec struct {
	OpenAPI    string                `json:"openapi"`
	Swagger    string                `json:"swagger"` // Swagger 2.0 detection
	Info       SpecInfo              `json:"info"`
	Paths      map[string]PathItem   `json:"paths"`
	Components Components            `json:"components"`
	Security   []SecurityRequirement `json:"security"` // global security
	// Swagger 2.0 fields
	Definitions map[string]Schema            `json:"definitions"`
	SecurityDefs map[string]SecurityScheme    `json:"securityDefinitions"`
}

type SpecInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type Components struct {
	Schemas         map[string]Schema         `json:"schemas"`
	SecuritySchemes map[string]SecurityScheme  `json:"securitySchemes"`
	Parameters      map[string]Parameter       `json:"parameters"`
	RequestBodies   map[string]RequestBody     `json:"requestBodies"`
	Responses       map[string]Response        `json:"responses"`
}

type PathItem struct {
	Get     *Operation `json:"get"`
	Post    *Operation `json:"post"`
	Put     *Operation `json:"put"`
	Patch   *Operation `json:"patch"`
	Delete  *Operation `json:"delete"`
	Parameters []Parameter `json:"parameters"` // shared path params
}

type Operation struct {
	OperationID string                `json:"operationId"`
	Summary     string                `json:"summary"`
	Description string                `json:"description"`
	Tags        []string              `json:"tags"`
	Parameters  []Parameter           `json:"parameters"`
	RequestBody *RequestBody          `json:"requestBody"`
	Responses   map[string]Response   `json:"responses"`
	Security    []SecurityRequirement `json:"security"`
}

type Parameter struct {
	Ref      string `json:"$ref"`
	Name     string `json:"name"`
	In       string `json:"in"` // "path", "query", "header", "cookie"
	Required bool   `json:"required"`
	Schema   Schema `json:"schema"`
}

type RequestBody struct {
	Ref      string             `json:"$ref"`
	Required bool               `json:"required"`
	Content  map[string]MediaType `json:"content"`
}

type MediaType struct {
	Schema Schema `json:"schema"`
}

type Response struct {
	Ref         string             `json:"$ref"`
	Description string             `json:"description"`
	Content     map[string]MediaType `json:"content"`
}

type Schema struct {
	Ref        string            `json:"$ref"`
	Type       string            `json:"type"`
	Format     string            `json:"format"`
	Properties map[string]Schema `json:"properties"`
	Items      *Schema           `json:"items"`
	Enum       []interface{}     `json:"enum"` // can be string or number
	Required   []string          `json:"required"`
	AllOf      []Schema          `json:"allOf"`
}

type SecurityScheme struct {
	Type         string `json:"type"`   // "http", "apiKey", "oauth2", "openIdConnect"
	Scheme       string `json:"scheme"` // "bearer", "basic"
	BearerFormat string `json:"bearerFormat"`
	In           string `json:"in"`     // "header", "query" (for apiKey)
	Name         string `json:"name"`   // header/query param name (for apiKey)
}

type SecurityRequirement map[string][]string

// Parse reads an OpenAPI specification from a file path or URL.
// Only JSON format is supported. For YAML specs, convert first with: yq -o json spec.yaml > spec.json
func Parse(source string) (*Spec, error) {
	var data []byte
	var err error

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		data, err = fetchURL(source)
	} else {
		data, err = os.ReadFile(source)
	}
	if err != nil {
		return nil, fmt.Errorf("reading spec: %w", err)
	}

	var spec Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing spec (only JSON is supported — for YAML, convert with: yq -o json spec.yaml > spec.json): %w", err)
	}

	// Detect Swagger 2.0 and migrate
	if spec.Swagger != "" && strings.HasPrefix(spec.Swagger, "2") {
		migrateSwagger2(&spec)
	}

	// Validate it's OpenAPI
	if spec.OpenAPI == "" && spec.Swagger == "" {
		return nil, fmt.Errorf("not a valid OpenAPI specification (missing 'openapi' or 'swagger' field)")
	}

	// Resolve $ref pointers
	resolveRefs(&spec)

	return &spec, nil
}

// migrateSwagger2 converts Swagger 2.0 fields to OpenAPI 3.x equivalents.
func migrateSwagger2(spec *Spec) {
	if spec.OpenAPI == "" {
		spec.OpenAPI = "3.0.0"
	}

	// Move definitions → components.schemas
	if len(spec.Definitions) > 0 && spec.Components.Schemas == nil {
		spec.Components.Schemas = spec.Definitions
	}

	// Move securityDefinitions → components.securitySchemes
	if len(spec.SecurityDefs) > 0 && spec.Components.SecuritySchemes == nil {
		spec.Components.SecuritySchemes = spec.SecurityDefs
	}
}

// resolveRefs resolves $ref pointers in schemas, parameters, and request bodies.
func resolveRefs(spec *Spec) {
	// Build lookup map for schemas
	schemas := spec.Components.Schemas

	// Resolve schema refs recursively
	for name, schema := range schemas {
		resolved := resolveSchemaRef(schema, schemas)
		schemas[name] = resolved
	}

	// Resolve refs in paths
	for path, item := range spec.Paths {
		resolveOperationRefs(item.Get, spec)
		resolveOperationRefs(item.Post, spec)
		resolveOperationRefs(item.Put, spec)
		resolveOperationRefs(item.Patch, spec)
		resolveOperationRefs(item.Delete, spec)
		spec.Paths[path] = item
	}
}

func resolveOperationRefs(op *Operation, spec *Spec) {
	if op == nil {
		return
	}

	// Resolve parameter refs
	for i, param := range op.Parameters {
		if param.Ref != "" {
			if resolved, ok := lookupParameter(param.Ref, spec); ok {
				op.Parameters[i] = resolved
			}
		}
	}

	// Resolve request body ref
	if op.RequestBody != nil && op.RequestBody.Ref != "" {
		if resolved, ok := lookupRequestBody(op.RequestBody.Ref, spec); ok {
			op.RequestBody = &resolved
		}
	}

	// Resolve schema refs in request body content
	if op.RequestBody != nil {
		for mt, media := range op.RequestBody.Content {
			media.Schema = resolveSchemaRef(media.Schema, spec.Components.Schemas)
			op.RequestBody.Content[mt] = media
		}
	}
}

func resolveSchemaRef(schema Schema, schemas map[string]Schema) Schema {
	if schema.Ref != "" {
		name := refName(schema.Ref)
		if resolved, ok := schemas[name]; ok {
			return resolved
		}
	}

	// Resolve nested properties
	for prop, s := range schema.Properties {
		schema.Properties[prop] = resolveSchemaRef(s, schemas)
	}

	// Resolve items
	if schema.Items != nil && schema.Items.Ref != "" {
		resolved := resolveSchemaRef(*schema.Items, schemas)
		schema.Items = &resolved
	}

	// Resolve allOf
	if len(schema.AllOf) > 0 {
		merged := Schema{Properties: map[string]Schema{}}
		for _, sub := range schema.AllOf {
			sub = resolveSchemaRef(sub, schemas)
			for k, v := range sub.Properties {
				merged.Properties[k] = v
			}
			merged.Required = append(merged.Required, sub.Required...)
			if sub.Type != "" {
				merged.Type = sub.Type
			}
		}
		if merged.Type == "" {
			merged.Type = "object"
		}
		return merged
	}

	return schema
}

func lookupParameter(ref string, spec *Spec) (Parameter, bool) {
	name := refName(ref)
	if p, ok := spec.Components.Parameters[name]; ok {
		return p, true
	}
	return Parameter{}, false
}

func lookupRequestBody(ref string, spec *Spec) (RequestBody, bool) {
	name := refName(ref)
	if rb, ok := spec.Components.RequestBodies[name]; ok {
		return rb, true
	}
	return RequestBody{}, false
}

// refName extracts the type name from a $ref pointer.
// "#/components/schemas/Task" → "Task"
// "#/definitions/Task" → "Task" (Swagger 2.0)
func refName(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ref
}

func fetchURL(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: HTTP %d", url, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
