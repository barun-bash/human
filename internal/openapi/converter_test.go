package openapi

import (
	"strings"
	"testing"
)

func TestSchemaToData(t *testing.T) {
	schema := Schema{
		Type: "object",
		Required: []string{"title"},
		Properties: map[string]Schema{
			"title":  {Type: "string"},
			"status": {Type: "string", Enum: []interface{}{"pending", "done"}},
			"count":  {Type: "integer"},
		},
	}

	block := schemaToData("Task", schema)
	if !strings.Contains(block, "data Task:") {
		t.Error("expected 'data Task:' header")
	}
	if !strings.Contains(block, "title") {
		t.Error("expected title field")
	}
	if !strings.Contains(block, "either") {
		t.Error("expected enum to produce 'either' syntax")
	}
	if !strings.Contains(block, "number") {
		t.Error("expected integer to map to number")
	}
}

func TestSchemaToDataWithArray(t *testing.T) {
	schema := Schema{
		Type: "object",
		Properties: map[string]Schema{
			"name": {Type: "string"},
			"tasks": {
				Type: "array",
				Items: &Schema{Ref: "#/components/schemas/Task"},
			},
		},
	}

	block := schemaToData("User", schema)
	if !strings.Contains(block, "has many Task") {
		t.Errorf("expected 'has many Task', got:\n%s", block)
	}
}

func TestSchemaToDataEmpty(t *testing.T) {
	schema := Schema{Type: "object"}
	block := schemaToData("Empty", schema)
	if block != "" {
		t.Errorf("expected empty block for schema with no properties, got %q", block)
	}
}

func TestSchemaTypeToHuman(t *testing.T) {
	tests := []struct {
		field  string
		schema Schema
		want   string
	}{
		{"name", Schema{Type: "string"}, "text"},
		{"email", Schema{Type: "string"}, "email"},
		{"password", Schema{Type: "string"}, "encrypted text"},
		{"count", Schema{Type: "integer"}, "number"},
		{"price", Schema{Type: "number"}, "number"},
		{"active", Schema{Type: "boolean"}, "text"},
		{"createdAt", Schema{Type: "string", Format: "date-time"}, "datetime"},
		{"birthday", Schema{Type: "string", Format: "date"}, "date"},
		{"email", Schema{Type: "string", Format: "email"}, "email"},
		{"website", Schema{Type: "string", Format: "uri"}, "url"},
		{"secret", Schema{Type: "string", Format: "password"}, "encrypted text"},
	}

	for _, tt := range tests {
		got := schemaTypeToHuman(tt.field, tt.schema)
		if got != tt.want {
			t.Errorf("schemaTypeToHuman(%q, %+v) = %q, want %q", tt.field, tt.schema, got, tt.want)
		}
	}
}

func TestOperationToAPI(t *testing.T) {
	spec := &Spec{OpenAPI: "3.0.0"}
	op := &Operation{
		OperationID: "createTask",
		Security:    []SecurityRequirement{{"bearer": {}}},
		Responses:   map[string]Response{"201": {Description: "Created"}},
	}

	block := operationToAPI("POST", "/tasks", op, nil, spec)
	if !strings.Contains(block, "api CreateTask:") {
		t.Errorf("expected 'api CreateTask:', got:\n%s", block)
	}
	if !strings.Contains(block, "requires authentication") {
		t.Error("expected 'requires authentication'")
	}
	if !strings.Contains(block, "create the task") {
		t.Error("expected 'create the task'")
	}
}

func TestOperationToAPIWithQueryParams(t *testing.T) {
	spec := &Spec{OpenAPI: "3.0.0"}
	op := &Operation{
		OperationID: "listTasks",
		Parameters: []Parameter{
			{Name: "status", In: "query", Schema: Schema{Type: "string"}},
			{Name: "limit", In: "query", Schema: Schema{Type: "integer"}},
		},
		Responses: map[string]Response{"200": {Description: "OK"}},
	}

	block := operationToAPI("GET", "/tasks", op, nil, spec)
	if !strings.Contains(block, "accepts") {
		t.Error("expected 'accepts' for query params")
	}
	if !strings.Contains(block, "fetch all tasks") {
		t.Error("expected 'fetch all tasks'")
	}
}

func TestOperationWithGlobalSecurity(t *testing.T) {
	spec := &Spec{
		OpenAPI:  "3.0.0",
		Security: []SecurityRequirement{{"bearer": {}}},
	}
	op := &Operation{
		OperationID: "getProfile",
		Responses:   map[string]Response{"200": {Description: "OK"}},
	}

	block := operationToAPI("GET", "/profile/{id}", op, nil, spec)
	if !strings.Contains(block, "requires authentication") {
		t.Error("expected global security to trigger 'requires authentication'")
	}
}

func TestSecurityToAuth(t *testing.T) {
	tests := []struct {
		name   string
		scheme SecurityScheme
		expect string
	}{
		{"JWT", SecurityScheme{Type: "http", Scheme: "bearer", BearerFormat: "JWT"}, "JWT tokens"},
		{"bearer", SecurityScheme{Type: "http", Scheme: "bearer"}, "token based"},
		{"basic", SecurityScheme{Type: "http", Scheme: "basic"}, "email and password"},
		{"apiKey", SecurityScheme{Type: "apiKey"}, "API keys"},
		{"oauth2", SecurityScheme{Type: "oauth2"}, "OAuth"},
	}

	for _, tt := range tests {
		schemes := map[string]SecurityScheme{tt.name: tt.scheme}
		block := securityToAuth(schemes)
		if !strings.Contains(block, tt.expect) {
			t.Errorf("securityToAuth(%q) expected %q, got:\n%s", tt.name, tt.expect, block)
		}
	}
}

func TestToHumanFieldName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"firstName", "first name"},
		{"due_date", "due date"},
		{"email", "email"},
		{"created-at", "created at"},
		{"ID", "id"},
	}

	for _, tt := range tests {
		got := toHumanFieldName(tt.input)
		if got != tt.want {
			t.Errorf("toHumanFieldName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestInferModelFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/tasks", "task"},
		{"/tasks/{id}", "task"},
		{"/api/v1/users", "user"},
		{"/api/v1/users/{id}/posts", "post"},
		{"/items/{itemId}/reviews", "review"},
	}

	for _, tt := range tests {
		got := inferModelFromPath(tt.path)
		if got != tt.want {
			t.Errorf("inferModelFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestSingularize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"tasks", "task"},
		{"users", "user"},
		{"categories", "category"},
		{"addresses", "address"},
		{"status", "status"},
	}

	for _, tt := range tests {
		got := singularize(tt.input)
		if got != tt.want {
			t.Errorf("singularize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestJoinEnum(t *testing.T) {
	got := joinEnum([]string{"pending", "active", "done"})
	if got != `"pending", "active" or "done"` {
		t.Errorf("joinEnum = %q", got)
	}

	got2 := joinEnum([]string{"yes"})
	if got2 != `"yes"` {
		t.Errorf("joinEnum single = %q", got2)
	}
}

func TestMethodPathToName(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   string
	}{
		{"GET", "/tasks", "ListTasks"},
		{"GET", "/tasks/{id}", "GetTask"},
		{"POST", "/tasks", "CreateTask"},
		{"PUT", "/tasks/{id}", "UpdateTask"},
		{"DELETE", "/tasks/{id}", "DeleteTask"},
	}

	for _, tt := range tests {
		got := methodPathToName(tt.method, tt.path)
		if got != tt.want {
			t.Errorf("methodPathToName(%q, %q) = %q, want %q", tt.method, tt.path, got, tt.want)
		}
	}
}

func TestToHumanFullSpec(t *testing.T) {
	spec := &Spec{
		OpenAPI: "3.0.0",
		Info:    SpecInfo{Title: "Task Manager", Version: "1.0"},
		Paths: map[string]PathItem{
			"/tasks": {
				Get: &Operation{
					OperationID: "listTasks",
					Responses:   map[string]Response{"200": {Description: "OK"}},
				},
				Post: &Operation{
					OperationID: "createTask",
					Security:    []SecurityRequirement{{"bearer": {}}},
					Responses:   map[string]Response{"201": {Description: "Created"}},
				},
			},
		},
		Components: Components{
			Schemas: map[string]Schema{
				"Task": {
					Type:     "object",
					Required: []string{"title"},
					Properties: map[string]Schema{
						"title":  {Type: "string"},
						"status": {Type: "string", Enum: []interface{}{"pending", "done"}},
					},
				},
			},
			SecuritySchemes: map[string]SecurityScheme{
				"bearer": {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
			},
		},
	}

	code, err := ToHuman(spec, "")
	// err may be a syntax warning — code should still be produced
	if code == "" {
		t.Fatalf("ToHuman produced empty output, err: %v", err)
	}

	if !strings.Contains(code, "app TaskManager") {
		t.Error("expected app name from spec title")
	}
	if !strings.Contains(code, "data Task:") {
		t.Error("expected data block")
	}
	if !strings.Contains(code, "api") {
		t.Error("expected api blocks")
	}
	if !strings.Contains(code, "authentication:") {
		t.Error("expected authentication block")
	}
	if !strings.Contains(code, "build with:") {
		t.Error("expected build block")
	}
}

func TestToHumanCustomAppName(t *testing.T) {
	spec := &Spec{
		OpenAPI: "3.0.0",
		Info:    SpecInfo{Title: "Some API", Version: "1.0"},
		Components: Components{
			Schemas: map[string]Schema{},
		},
	}

	code, _ := ToHuman(spec, "CustomApp")
	if !strings.Contains(code, "app CustomApp") {
		t.Errorf("expected custom app name, got:\n%s", code)
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"task manager", "TaskManager"},
		{"my-api", "MyApi"},
		{"user_profile", "UserProfile"},
		{"", ""},
	}

	for _, tt := range tests {
		got := toPascalCase(tt.input)
		if got != tt.want {
			t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
