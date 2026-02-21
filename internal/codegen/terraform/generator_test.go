package terraform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
)

func testApp() *ir.Application {
	return &ir.Application{
		Name:     "TestApp",
		Platform: "web",
		Config: &ir.BuildConfig{
			Frontend: "React with TypeScript",
			Backend:  "Node with Express",
			Database: "PostgreSQL",
			Deploy:   "AWS",
		},
		Data: []*ir.DataModel{
			{Name: "User", Fields: []*ir.DataField{
				{Name: "name", Type: "text"},
				{Name: "email", Type: "email"},
			}},
			{Name: "Task", Fields: []*ir.DataField{
				{Name: "title", Type: "text"},
			}, Relations: []*ir.Relation{
				{Kind: "belongs_to", Target: "User"},
			}},
		},
		Pages: []*ir.Page{
			{Name: "Home"},
			{Name: "Dashboard"},
		},
		APIs: []*ir.Endpoint{
			{Name: "CreateTask", Auth: true},
		},
		Database: &ir.DatabaseConfig{
			Engine: "PostgreSQL",
		},
		Environments: []*ir.Environment{
			{Name: "staging", Config: map[string]string{"region": "us-west-2"}},
			{Name: "production", Config: map[string]string{"region": "us-east-1"}},
		},
	}
}

// ── Generate tests ──

func TestGenerateAWS(t *testing.T) {
	app := testApp()
	tmpDir := t.TempDir()

	g := Generator{}
	if err := g.Generate(app, tmpDir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify core files exist
	for _, name := range []string{"main.tf", "variables.tf", "outputs.tf", "terraform.tfvars.example"} {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Expected %s to exist: %v", name, err)
		}
	}

	// AWS-specific files
	for _, name := range []string{"aws_ecs.tf", "aws_rds.tf", "aws_networking.tf", "aws_cdn.tf"} {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Expected %s to exist: %v", name, err)
		}
	}

	// Per-environment tfvars
	for _, env := range []string{"staging.tfvars", "production.tfvars"} {
		path := filepath.Join(tmpDir, "envs", env)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Expected envs/%s to exist: %v", env, err)
		}
	}
}

func TestGenerateGCP(t *testing.T) {
	app := testApp()
	app.Config.Deploy = "GCP"
	tmpDir := t.TempDir()

	g := Generator{}
	if err := g.Generate(app, tmpDir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	for _, name := range []string{"main.tf", "gcp_cloudrun.tf", "gcp_cloudsql.tf", "gcp_cdn.tf"} {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Expected %s to exist: %v", name, err)
		}
	}
}

func TestGenerateDockerProd(t *testing.T) {
	app := testApp()
	app.Config.Deploy = "Docker"
	tmpDir := t.TempDir()

	g := Generator{}
	if err := g.Generate(app, tmpDir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	path := filepath.Join(tmpDir, "docker_prod.tf")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("Expected docker_prod.tf to exist: %v", err)
	}

	// Should NOT have AWS files
	if _, err := os.Stat(filepath.Join(tmpDir, "aws_ecs.tf")); err == nil {
		t.Error("Docker deploy should not generate aws_ecs.tf")
	}
}

func TestGenerateNoCDNWithoutFrontend(t *testing.T) {
	app := testApp()
	app.Config.Frontend = ""
	app.Pages = nil
	tmpDir := t.TempDir()

	g := Generator{}
	if err := g.Generate(app, tmpDir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// No CDN file when no frontend
	if _, err := os.Stat(filepath.Join(tmpDir, "aws_cdn.tf")); err == nil {
		t.Error("Should not generate aws_cdn.tf without frontend")
	}
}

// ── Content tests ──

func TestMainTFContainsAWSProvider(t *testing.T) {
	app := testApp()
	content := generateMainTF(app, "aws")

	if !strings.Contains(content, "hashicorp/aws") {
		t.Error("AWS main.tf should contain hashicorp/aws provider")
	}
	if !strings.Contains(content, "backend \"s3\"") {
		t.Error("AWS main.tf should use S3 backend for state")
	}
	if !strings.Contains(content, "provider \"aws\"") {
		t.Error("AWS main.tf should configure aws provider")
	}
}

func TestMainTFContainsGCPProvider(t *testing.T) {
	app := testApp()
	content := generateMainTF(app, "gcp")

	if !strings.Contains(content, "hashicorp/google") {
		t.Error("GCP main.tf should contain hashicorp/google provider")
	}
	if !strings.Contains(content, "backend \"gcs\"") {
		t.Error("GCP main.tf should use GCS backend for state")
	}
}

func TestMainTFContainsDockerProvider(t *testing.T) {
	app := testApp()
	content := generateMainTF(app, "docker")

	if !strings.Contains(content, "kreuzwerker/docker") {
		t.Error("Docker main.tf should contain kreuzwerker/docker provider")
	}
}

func TestVariablesTFIncludesDBVars(t *testing.T) {
	app := testApp()
	content := generateVariablesTF(app, "aws")

	if !strings.Contains(content, "db_instance_class") {
		t.Error("AWS variables.tf should include db_instance_class when database is configured")
	}
	if !strings.Contains(content, "db_password") {
		t.Error("AWS variables.tf should include db_password when database is configured")
	}
}

func TestVariablesTFNoDBVarsWithoutDB(t *testing.T) {
	app := testApp()
	app.Config.Database = ""
	app.Database = nil
	content := generateVariablesTF(app, "aws")

	if strings.Contains(content, "db_instance_class") {
		t.Error("Should not include db variables when no database is configured")
	}
}

func TestECSContainsECRAndCluster(t *testing.T) {
	app := testApp()
	content := generateAWSECS(app)

	if !strings.Contains(content, "aws_ecr_repository") {
		t.Error("AWS ECS should define an ECR repository")
	}
	if !strings.Contains(content, "aws_ecs_cluster") {
		t.Error("AWS ECS should define a cluster")
	}
	if !strings.Contains(content, "aws_ecs_service") {
		t.Error("AWS ECS should define a service")
	}
	if !strings.Contains(content, "FARGATE") {
		t.Error("AWS ECS should use Fargate")
	}
}

func TestRDSPostgres(t *testing.T) {
	app := testApp()
	content := generateAWSRDS(app)

	if !strings.Contains(content, "engine         = \"postgres\"") {
		t.Error("RDS should use postgres engine")
	}
	if !strings.Contains(content, "engine_version = \"16\"") {
		t.Error("RDS should use postgres 16")
	}
}

func TestRDSMySQL(t *testing.T) {
	app := testApp()
	app.Config.Database = "MySQL"
	app.Database.Engine = "MySQL"
	content := generateAWSRDS(app)

	if !strings.Contains(content, "engine         = \"mysql\"") {
		t.Error("RDS should use mysql engine")
	}
}

func TestRDSSkippedWithoutDB(t *testing.T) {
	app := testApp()
	app.Config.Database = ""
	app.Database = nil
	content := generateAWSRDS(app)

	if strings.Contains(content, "aws_db_instance") {
		t.Error("RDS should be skipped when no database is configured")
	}
}

func TestCloudRunContainsService(t *testing.T) {
	app := testApp()
	content := generateGCPCloudRun(app)

	if !strings.Contains(content, "google_cloud_run_v2_service") {
		t.Error("GCP should define a Cloud Run service")
	}
	if !strings.Contains(content, "google_artifact_registry_repository") {
		t.Error("GCP should define an Artifact Registry repository")
	}
}

func TestCloudSQLPostgres(t *testing.T) {
	app := testApp()
	content := generateGCPCloudSQL(app)

	if !strings.Contains(content, "POSTGRES_16") {
		t.Error("Cloud SQL should use POSTGRES_16")
	}
}

func TestDockerProdContainsContainers(t *testing.T) {
	app := testApp()
	content := generateDockerProd(app)

	if !strings.Contains(content, "docker_container") {
		t.Error("Docker prod should define containers")
	}
	if !strings.Contains(content, "docker_network") {
		t.Error("Docker prod should define a network")
	}
	if !strings.Contains(content, "postgres:16-alpine") {
		t.Error("Docker prod should include postgres container when DB is configured")
	}
}

func TestEnvTFVarsProduction(t *testing.T) {
	app := testApp()
	env := &ir.Environment{
		Name:   "production",
		Config: map[string]string{"region": "us-east-1"},
	}
	content := generateEnvTFVars(app, env, "aws")

	if !strings.Contains(content, "environment = \"production\"") {
		t.Error("Production tfvars should set environment")
	}
	if !strings.Contains(content, "desired_count") {
		t.Error("Production tfvars should set desired_count")
	}
}

func TestECSEnvVarNameGoBackend(t *testing.T) {
	app := testApp()
	app.Config.Backend = "Go with Gin"
	content := generateAWSECS(app)

	if strings.Contains(content, "NODE_ENV") {
		t.Error("Go backend should not use NODE_ENV")
	}
	if !strings.Contains(content, "APP_ENV") {
		t.Error("Go backend should use APP_ENV")
	}
}

func TestECSDatabaseURLMySQL(t *testing.T) {
	app := testApp()
	app.Config.Database = "MySQL"
	app.Database.Engine = "MySQL"
	content := generateAWSECS(app)

	if strings.Contains(content, "postgresql://") {
		t.Error("MySQL backend should not use postgresql:// scheme")
	}
	if !strings.Contains(content, "mysql://") {
		t.Error("MySQL backend should use mysql:// scheme")
	}
}

func TestCloudRunDatabaseURLFormat(t *testing.T) {
	app := testApp()
	content := generateGCPCloudRun(app)

	if strings.Contains(content, "@//cloudsql/") {
		t.Error("DATABASE_URL should not use @//cloudsql/ format")
	}
	if !strings.Contains(content, "?host=/cloudsql/") {
		t.Error("PostgreSQL DATABASE_URL should use ?host=/cloudsql/ format")
	}
}

func TestCloudRunDatabaseURLMySQL(t *testing.T) {
	app := testApp()
	app.Config.Database = "MySQL"
	app.Database.Engine = "MySQL"
	content := generateGCPCloudRun(app)

	if !strings.Contains(content, "@unix(/cloudsql/") {
		t.Error("MySQL DATABASE_URL should use @unix(/cloudsql/) format")
	}
}

func TestDockerProdBuildContext(t *testing.T) {
	app := testApp()
	content := generateDockerProd(app)

	if !strings.Contains(content, "context = \"../node\"") {
		t.Error("Node backend should use ../node build context")
	}

	app.Config.Backend = "Go with Gin"
	content = generateDockerProd(app)

	if !strings.Contains(content, "context = \"../go\"") {
		t.Error("Go backend should use ../go build context")
	}
}

func TestDockerProdVolumeReference(t *testing.T) {
	app := testApp()
	content := generateDockerProd(app)

	if !strings.Contains(content, "volume_name    = docker_volume.db_data.name") {
		t.Error("Docker container should reference docker_volume.db_data.name")
	}
}

// ── Helper tests ──

func TestBackendLang(t *testing.T) {
	tests := []struct {
		backend  string
		expected string
	}{
		{"Node with Express", "node"},
		{"Go with Gin", "go"},
		{"Python with Flask", "python"},
		{"Python with Django", "python"},
		{"", "node"},
	}
	for _, tt := range tests {
		app := &ir.Application{Config: &ir.BuildConfig{Backend: tt.backend}}
		got := backendLang(app)
		if got != tt.expected {
			t.Errorf("backendLang(%q) = %q, want %q", tt.backend, got, tt.expected)
		}
	}
}

func TestEnvVarName(t *testing.T) {
	app := &ir.Application{Config: &ir.BuildConfig{Backend: "Node with Express"}}
	if got := envVarName(app); got != "NODE_ENV" {
		t.Errorf("envVarName(Node) = %q, want NODE_ENV", got)
	}

	app.Config.Backend = "Go with Gin"
	if got := envVarName(app); got != "APP_ENV" {
		t.Errorf("envVarName(Go) = %q, want APP_ENV", got)
	}
}

func TestDeployTarget(t *testing.T) {
	tests := []struct {
		deploy   string
		expected string
	}{
		{"AWS", "aws"},
		{"aws", "aws"},
		{"GCP", "gcp"},
		{"Google Cloud", "gcp"},
		{"Docker", "docker"},
		{"", "docker"},
	}
	for _, tt := range tests {
		app := &ir.Application{Config: &ir.BuildConfig{Deploy: tt.deploy}}
		got := deployTarget(app)
		if got != tt.expected {
			t.Errorf("deployTarget(%q) = %q, want %q", tt.deploy, got, tt.expected)
		}
	}
}

func TestAppNameLower(t *testing.T) {
	app := &ir.Application{Name: "My App"}
	if got := appNameLower(app); got != "my-app" {
		t.Errorf("appNameLower() = %q, want %q", got, "my-app")
	}
}

func TestHasFrontend(t *testing.T) {
	app := &ir.Application{Config: &ir.BuildConfig{Frontend: "React"}}
	if !hasFrontend(app) {
		t.Error("hasFrontend should return true when frontend is configured")
	}

	app2 := &ir.Application{Pages: []*ir.Page{{Name: "Home"}}}
	if !hasFrontend(app2) {
		t.Error("hasFrontend should return true when pages exist")
	}

	app3 := &ir.Application{}
	if hasFrontend(app3) {
		t.Error("hasFrontend should return false when no frontend and no pages")
	}
}
