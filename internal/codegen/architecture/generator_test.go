package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
)

func testMicroservicesApp() *ir.Application {
	return &ir.Application{
		Name:     "TestApp",
		Platform: "web",
		Config: &ir.BuildConfig{
			Backend: "Node with Express",
		},
		Data: []*ir.DataModel{
			{Name: "User"},
			{Name: "Task"},
			{Name: "Order"},
		},
		APIs: []*ir.Endpoint{
			{Name: "CreateUser", Steps: []*ir.Action{{Type: "create", Text: "create a User"}}},
			{Name: "GetTasks", Steps: []*ir.Action{{Type: "query", Text: "fetch Task list"}}},
		},
		Architecture: &ir.Architecture{
			Style: "microservices",
			Services: []*ir.ServiceDef{
				{
					Name:           "UserService",
					Handles:        "user management",
					Port:           3001,
					Models:         []string{"User"},
					HasOwnDatabase: true,
					TalksTo:        []string{"TaskService"},
				},
				{
					Name:           "TaskService",
					Handles:        "task management",
					Port:           3002,
					Models:         []string{"Task"},
					HasOwnDatabase: true,
				},
			},
			Gateway: &ir.GatewayDef{
				Routes: map[string]string{
					"/api/users": "UserService",
					"/api/tasks": "TaskService",
				},
			},
			Broker: "RabbitMQ",
		},
	}
}

func testServerlessApp() *ir.Application {
	return &ir.Application{
		Name:     "TestApp",
		Platform: "api",
		Config: &ir.BuildConfig{
			Backend: "Node with Express",
		},
		APIs: []*ir.Endpoint{
			{Name: "CreateUser", Steps: []*ir.Action{{Type: "create", Text: "create a User"}}},
			{Name: "GetTasks", Steps: []*ir.Action{{Type: "query", Text: "fetch Task list"}}},
			{Name: "DeleteTask", Steps: []*ir.Action{{Type: "delete", Text: "delete the Task"}}},
		},
		Architecture: &ir.Architecture{
			Style: "serverless",
		},
	}
}

// ── Monolith ──

func TestMonolithGeneratesNothing(t *testing.T) {
	app := &ir.Application{Name: "TestApp"}
	tmpDir := t.TempDir()

	g := Generator{}
	if err := g.Generate(app, tmpDir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// No files should be created for monolith (nil architecture)
	entries, _ := os.ReadDir(tmpDir)
	if len(entries) != 0 {
		t.Errorf("Monolith should not generate files, got %d entries", len(entries))
	}
}

func TestExplicitMonolithGeneratesNothing(t *testing.T) {
	app := &ir.Application{
		Name:         "TestApp",
		Architecture: &ir.Architecture{Style: "monolith"},
	}
	tmpDir := t.TempDir()

	g := Generator{}
	if err := g.Generate(app, tmpDir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	entries, _ := os.ReadDir(tmpDir)
	if len(entries) != 0 {
		t.Errorf("Explicit monolith should not generate files, got %d entries", len(entries))
	}
}

// ── Microservices ──

func TestMicroservicesGeneratesFiles(t *testing.T) {
	app := testMicroservicesApp()
	tmpDir := t.TempDir()

	g := Generator{}
	if err := g.Generate(app, tmpDir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expectedFiles := []string{
		"docker-compose.services.yml",
		"services/userservice/Dockerfile",
		"services/userservice/README.md",
		"services/taskservice/Dockerfile",
		"services/taskservice/README.md",
		"gateway/nginx.conf",
		"gateway/Dockerfile",
	}

	for _, name := range expectedFiles {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Expected %s to exist: %v", name, err)
		}
	}
}

func TestServicesComposeContainsServices(t *testing.T) {
	app := testMicroservicesApp()
	content := generateServicesCompose(app)

	if !strings.Contains(content, "userservice:") {
		t.Error("Compose should define userservice")
	}
	if !strings.Contains(content, "taskservice:") {
		t.Error("Compose should define taskservice")
	}
	if !strings.Contains(content, "3001:3001") {
		t.Error("UserService should expose port 3001")
	}
	if !strings.Contains(content, "3002:3002") {
		t.Error("TaskService should expose port 3002")
	}
}

func TestServicesComposeContainsDatabases(t *testing.T) {
	app := testMicroservicesApp()
	content := generateServicesCompose(app)

	if !strings.Contains(content, "userservice-db:") {
		t.Error("Compose should define userservice-db")
	}
	if !strings.Contains(content, "taskservice-db:") {
		t.Error("Compose should define taskservice-db")
	}
	if !strings.Contains(content, "postgres:16-alpine") {
		t.Error("Service databases should use postgres:16-alpine")
	}
}

func TestServicesComposeContainsGateway(t *testing.T) {
	app := testMicroservicesApp()
	content := generateServicesCompose(app)

	if !strings.Contains(content, "gateway:") {
		t.Error("Compose should define gateway service")
	}
	if !strings.Contains(content, "80:80") {
		t.Error("Gateway should expose port 80")
	}
}

func TestServicesComposeContainsRabbitMQ(t *testing.T) {
	app := testMicroservicesApp()
	content := generateServicesCompose(app)

	if !strings.Contains(content, "rabbitmq:") {
		t.Error("Compose should include RabbitMQ broker")
	}
	if !strings.Contains(content, "5672:5672") {
		t.Error("RabbitMQ should expose port 5672")
	}
}

func TestServicesComposeKafka(t *testing.T) {
	app := testMicroservicesApp()
	app.Architecture.Broker = "Kafka"
	content := generateServicesCompose(app)

	if !strings.Contains(content, "kafka:") {
		t.Error("Compose should include Kafka")
	}
	if !strings.Contains(content, "zookeeper:") {
		t.Error("Compose should include Zookeeper for Kafka")
	}
}

func TestNginxGatewayContainsRoutes(t *testing.T) {
	app := testMicroservicesApp()
	content := generateNginxGateway(app)

	if !strings.Contains(content, "upstream userservice") {
		t.Error("Nginx should define userservice upstream")
	}
	if !strings.Contains(content, "upstream taskservice") {
		t.Error("Nginx should define taskservice upstream")
	}
	if !strings.Contains(content, "location /api/users") {
		t.Error("Nginx should route /api/users")
	}
	if !strings.Contains(content, "proxy_pass http://userservice") {
		t.Error("Nginx should proxy to userservice")
	}
	if !strings.Contains(content, "/health") {
		t.Error("Nginx should include health endpoint")
	}
}

func TestNginxGatewayAutoRoutes(t *testing.T) {
	app := testMicroservicesApp()
	app.Architecture.Gateway.Routes = nil // clear explicit routes
	content := generateNginxGateway(app)

	// Should auto-generate routes from service names
	if !strings.Contains(content, "location /api/userservice/") {
		t.Error("Nginx should auto-generate route for userservice")
	}
}

func TestServiceDockerfileNode(t *testing.T) {
	app := testMicroservicesApp()
	svc := app.Architecture.Services[0]
	content := generateServiceDockerfile(app, svc)

	if !strings.Contains(content, "FROM node:20-alpine") {
		t.Error("Node service Dockerfile should use node:20-alpine")
	}
	if !strings.Contains(content, "EXPOSE 3001") {
		t.Error("Should expose port 3001")
	}
}

func TestServiceDockerfilePython(t *testing.T) {
	app := testMicroservicesApp()
	app.Config.Backend = "Python with FastAPI"
	svc := app.Architecture.Services[0]
	content := generateServiceDockerfile(app, svc)

	if !strings.Contains(content, "FROM python:3.12-slim") {
		t.Error("Python service should use python:3.12-slim")
	}
}

func TestServiceDockerfileGo(t *testing.T) {
	app := testMicroservicesApp()
	app.Config.Backend = "Go with Gin"
	svc := app.Architecture.Services[0]
	content := generateServiceDockerfile(app, svc)

	if !strings.Contains(content, "FROM golang:1.21-alpine") {
		t.Error("Go service should use golang:1.21-alpine builder")
	}
	if !strings.Contains(content, "CGO_ENABLED=0") {
		t.Error("Go service should use static build")
	}
}

func TestServiceReadme(t *testing.T) {
	app := testMicroservicesApp()
	svc := app.Architecture.Services[0]
	content := generateServiceReadme(app, svc)

	if !strings.Contains(content, "# UserService") {
		t.Error("README should contain service name")
	}
	if !strings.Contains(content, "user management") {
		t.Error("README should contain handles description")
	}
	if !strings.Contains(content, "3001") {
		t.Error("README should contain port")
	}
	if !strings.Contains(content, "User") {
		t.Error("README should list owned models")
	}
	if !strings.Contains(content, "TaskService") {
		t.Error("README should list communication targets")
	}
}

// ── Serverless ──

func TestServerlessGeneratesFiles(t *testing.T) {
	app := testServerlessApp()
	tmpDir := t.TempDir()

	g := Generator{}
	if err := g.Generate(app, tmpDir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expectedFiles := []string{
		"template.yaml",
		"functions/createuser/index.ts",
		"functions/gettasks/index.ts",
		"functions/deletetask/index.ts",
	}

	for _, name := range expectedFiles {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Expected %s to exist: %v", name, err)
		}
	}
}

func TestSAMTemplateContainsFunctions(t *testing.T) {
	app := testServerlessApp()
	content := generateSAMTemplate(app)

	if !strings.Contains(content, "AWS::Serverless-2016-10-31") {
		t.Error("SAM template should use serverless transform")
	}
	if !strings.Contains(content, "CreateUserFunction:") {
		t.Error("SAM template should define CreateUser function")
	}
	if !strings.Contains(content, "GetTasksFunction:") {
		t.Error("SAM template should define GetTasks function")
	}
	if !strings.Contains(content, "DeleteTaskFunction:") {
		t.Error("SAM template should define DeleteTask function")
	}
	if !strings.Contains(content, "Method: POST") {
		t.Error("CreateUser should use POST method")
	}
	if !strings.Contains(content, "Method: GET") {
		t.Error("GetTasks should use GET method")
	}
	if !strings.Contains(content, "Method: DELETE") {
		t.Error("DeleteTask should use DELETE method")
	}
}

func TestSAMTemplatePython(t *testing.T) {
	app := testServerlessApp()
	app.Config.Backend = "Python"
	content := generateSAMTemplate(app)

	if !strings.Contains(content, "python3.12") {
		t.Error("Python SAM template should use python3.12 runtime")
	}
}

func TestLambdaHandler(t *testing.T) {
	app := testServerlessApp()
	api := app.APIs[0]
	content := generateLambdaHandler(app, api)

	if !strings.Contains(content, "APIGatewayProxyEvent") {
		t.Error("Lambda handler should use APIGatewayProxyEvent")
	}
	if !strings.Contains(content, "handler") {
		t.Error("Lambda handler should export handler function")
	}
}

// ── HTTP method inference ──

func TestInferHTTPMethod(t *testing.T) {
	tests := []struct {
		name     string
		steps    []*ir.Action
		expected string
	}{
		{"CreateUser", []*ir.Action{{Type: "create", Text: "create a User"}}, "POST"},
		{"UpdateTask", []*ir.Action{{Type: "update", Text: "update the Task"}}, "PUT"},
		{"DeleteTask", []*ir.Action{{Type: "delete", Text: "delete the Task"}}, "DELETE"},
		{"GetTasks", []*ir.Action{{Type: "query", Text: "fetch Task list"}}, "GET"},
		{"Login", nil, "POST"},     // inferred from name
		{"GetProfile", nil, "GET"}, // inferred from name
	}

	for _, tt := range tests {
		api := &ir.Endpoint{Name: tt.name, Steps: tt.steps}
		got := inferHTTPMethod(api)
		if got != tt.expected {
			t.Errorf("inferHTTPMethod(%q) = %q, want %q", tt.name, got, tt.expected)
		}
	}
}
