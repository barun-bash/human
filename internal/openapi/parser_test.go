package openapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSimpleSpec(t *testing.T) {
	spec := `{
		"openapi": "3.0.0",
		"info": {"title": "Test API", "version": "1.0.0"},
		"paths": {
			"/tasks": {
				"get": {
					"operationId": "listTasks",
					"summary": "List all tasks",
					"responses": {"200": {"description": "OK"}}
				},
				"post": {
					"operationId": "createTask",
					"summary": "Create a task",
					"requestBody": {
						"content": {
							"application/json": {
								"schema": {"$ref": "#/components/schemas/CreateTaskInput"}
							}
						}
					},
					"responses": {"201": {"description": "Created"}}
				}
			}
		},
		"components": {
			"schemas": {
				"Task": {
					"type": "object",
					"required": ["title"],
					"properties": {
						"id": {"type": "integer"},
						"title": {"type": "string"},
						"status": {"type": "string", "enum": ["pending", "done"]},
						"dueDate": {"type": "string", "format": "date"}
					}
				},
				"CreateTaskInput": {
					"type": "object",
					"properties": {
						"title": {"type": "string"},
						"status": {"type": "string"}
					}
				}
			}
		}
	}`

	file := writeTemp(t, spec)
	parsed, err := Parse(file)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Info.Title != "Test API" {
		t.Errorf("title = %q, want %q", parsed.Info.Title, "Test API")
	}
	if len(parsed.Paths) != 1 {
		t.Errorf("paths = %d, want 1", len(parsed.Paths))
	}
	if len(parsed.Components.Schemas) != 2 {
		t.Errorf("schemas = %d, want 2", len(parsed.Components.Schemas))
	}
}

func TestParseWithRef(t *testing.T) {
	spec := `{
		"openapi": "3.0.0",
		"info": {"title": "Ref Test", "version": "1.0"},
		"paths": {
			"/items": {
				"get": {
					"operationId": "listItems",
					"parameters": [
						{"$ref": "#/components/parameters/LimitParam"}
					],
					"responses": {"200": {"description": "OK"}}
				}
			}
		},
		"components": {
			"schemas": {
				"Item": {
					"type": "object",
					"properties": {
						"name": {"type": "string"}
					}
				}
			},
			"parameters": {
				"LimitParam": {
					"name": "limit",
					"in": "query",
					"required": false,
					"schema": {"type": "integer"}
				}
			}
		}
	}`

	file := writeTemp(t, spec)
	parsed, err := Parse(file)
	if err != nil {
		t.Fatal(err)
	}

	op := parsed.Paths["/items"].Get
	if len(op.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(op.Parameters))
	}
	if op.Parameters[0].Name != "limit" {
		t.Errorf("param name = %q, want %q", op.Parameters[0].Name, "limit")
	}
}

func TestParseSwagger2(t *testing.T) {
	spec := `{
		"swagger": "2.0",
		"info": {"title": "Swagger Test", "version": "1.0"},
		"paths": {},
		"definitions": {
			"Pet": {
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				}
			}
		},
		"securityDefinitions": {
			"api_key": {
				"type": "apiKey",
				"name": "api_key",
				"in": "header"
			}
		}
	}`

	file := writeTemp(t, spec)
	parsed, err := Parse(file)
	if err != nil {
		t.Fatal(err)
	}

	// Should have migrated definitions to components.schemas
	if _, ok := parsed.Components.Schemas["Pet"]; !ok {
		t.Error("Swagger 2.0 definitions should be migrated to components.schemas")
	}
	if _, ok := parsed.Components.SecuritySchemes["api_key"]; !ok {
		t.Error("Swagger 2.0 securityDefinitions should be migrated to components.securitySchemes")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	file := writeTemp(t, "not json at all")
	_, err := Parse(file)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "JSON") {
		t.Errorf("error should mention JSON: %v", err)
	}
}

func TestParseNotOpenAPI(t *testing.T) {
	file := writeTemp(t, `{"foo": "bar"}`)
	_, err := Parse(file)
	if err == nil {
		t.Fatal("expected error for non-OpenAPI JSON")
	}
}

func TestParseMissingFile(t *testing.T) {
	_, err := Parse("/nonexistent/path/spec.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRefName(t *testing.T) {
	tests := []struct {
		ref  string
		want string
	}{
		{"#/components/schemas/Task", "Task"},
		{"#/definitions/Pet", "Pet"},
		{"#/components/parameters/LimitParam", "LimitParam"},
		{"Task", "Task"},
	}
	for _, tt := range tests {
		got := refName(tt.ref)
		if got != tt.want {
			t.Errorf("refName(%q) = %q, want %q", tt.ref, got, tt.want)
		}
	}
}

func TestParseAllOf(t *testing.T) {
	spec := `{
		"openapi": "3.0.0",
		"info": {"title": "AllOf Test", "version": "1.0"},
		"paths": {},
		"components": {
			"schemas": {
				"Base": {
					"type": "object",
					"properties": {
						"id": {"type": "integer"}
					}
				},
				"Extended": {
					"allOf": [
						{"$ref": "#/components/schemas/Base"},
						{
							"type": "object",
							"properties": {
								"name": {"type": "string"}
							}
						}
					]
				}
			}
		}
	}`

	file := writeTemp(t, spec)
	parsed, err := Parse(file)
	if err != nil {
		t.Fatal(err)
	}

	extended := parsed.Components.Schemas["Extended"]
	if len(extended.Properties) != 2 {
		t.Errorf("expected 2 properties in merged allOf, got %d", len(extended.Properties))
	}
	if _, ok := extended.Properties["id"]; !ok {
		t.Error("expected 'id' from base schema")
	}
	if _, ok := extended.Properties["name"]; !ok {
		t.Error("expected 'name' from extension")
	}
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// Verify spec can round-trip through JSON marshaling
func TestSpecMarshalRoundTrip(t *testing.T) {
	spec := &Spec{
		OpenAPI: "3.0.0",
		Info:    SpecInfo{Title: "Test", Version: "1.0"},
		Components: Components{
			Schemas: map[string]Schema{
				"Item": {Type: "object", Properties: map[string]Schema{
					"name": {Type: "string"},
				}},
			},
		},
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}

	var parsed Spec
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed.Info.Title != "Test" {
		t.Errorf("round trip failed: title = %q", parsed.Info.Title)
	}
}
