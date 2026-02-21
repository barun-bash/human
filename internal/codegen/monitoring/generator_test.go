package monitoring

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
			Backend: "Node with Express",
		},
		APIs: []*ir.Endpoint{
			{Name: "CreateTask"},
			{Name: "GetTasks"},
		},
		Monitoring: []*ir.MonitoringRule{
			{Kind: "track", Metric: "page views"},
			{Kind: "alert", Condition: "error rate is above 5%", Channel: "Slack"},
			{Kind: "log", Metric: "all API requests", Service: "CloudWatch", Duration: "90 days"},
		},
	}
}

// ── Generate tests ──

func TestGenerateNodeBackend(t *testing.T) {
	app := testApp()
	tmpDir := t.TempDir()

	g := Generator{}
	if err := g.Generate(app, tmpDir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expectedFiles := []string{
		"prometheus/prometheus.yml",
		"prometheus/alerts.yml",
		"grafana/provisioning/datasources/prometheus.yml",
		"grafana/provisioning/dashboards/dashboards.yml",
		"grafana/dashboards/app.json",
		"docker-compose.monitoring.yml",
		"instrumentation/metrics.ts",
		"instrumentation/middleware.ts",
	}

	for _, name := range expectedFiles {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Expected %s to exist: %v", name, err)
		}
	}
}

func TestGeneratePythonBackend(t *testing.T) {
	app := testApp()
	app.Config.Backend = "Python with FastAPI"
	tmpDir := t.TempDir()

	g := Generator{}
	if err := g.Generate(app, tmpDir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Python instrumentation files
	for _, name := range []string{"instrumentation/metrics.py", "instrumentation/middleware.py"} {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Expected %s to exist: %v", name, err)
		}
	}

	// Should NOT have Node files
	if _, err := os.Stat(filepath.Join(tmpDir, "instrumentation", "metrics.ts")); err == nil {
		t.Error("Python backend should not generate .ts instrumentation files")
	}
}

func TestGenerateGoBackend(t *testing.T) {
	app := testApp()
	app.Config.Backend = "Go with Gin"
	tmpDir := t.TempDir()

	g := Generator{}
	if err := g.Generate(app, tmpDir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	for _, name := range []string{"instrumentation/metrics.go", "instrumentation/middleware.go"} {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Expected %s to exist: %v", name, err)
		}
	}
}

// ── Prometheus config tests ──

func TestPrometheusConfigContainsScrapeTargets(t *testing.T) {
	app := testApp()
	content := generatePrometheusConfig(app)

	if !strings.Contains(content, "scrape_configs:") {
		t.Error("Prometheus config should contain scrape_configs")
	}
	if !strings.Contains(content, "testapp-backend:3000") {
		t.Error("Prometheus config should scrape the backend")
	}
	if !strings.Contains(content, "metrics_path: /metrics") {
		t.Error("Prometheus config should set metrics path")
	}
}

func TestPrometheusConfigMicroservices(t *testing.T) {
	app := testApp()
	app.Architecture = &ir.Architecture{
		Style: "microservices",
		Services: []*ir.ServiceDef{
			{Name: "UserService", Port: 3001},
			{Name: "TaskService", Port: 3002},
		},
	}
	content := generatePrometheusConfig(app)

	if !strings.Contains(content, "userservice:3001") {
		t.Error("Should scrape UserService on port 3001")
	}
	if !strings.Contains(content, "taskservice:3002") {
		t.Error("Should scrape TaskService on port 3002")
	}
}

func TestPrometheusAlertmanagerIncluded(t *testing.T) {
	app := testApp() // has alert rules
	content := generatePrometheusConfig(app)

	if !strings.Contains(content, "alertmanagers:") {
		t.Error("Prometheus config should include alertmanager when alerts exist")
	}
}

func TestPrometheusNoAlertmanager(t *testing.T) {
	app := testApp()
	app.Monitoring = []*ir.MonitoringRule{
		{Kind: "track", Metric: "page views"},
	}
	content := generatePrometheusConfig(app)

	if strings.Contains(content, "alertmanagers:") {
		t.Error("Prometheus config should not include alertmanager when no alerts exist")
	}
}

// ── Alert rules tests ──

func TestAlertRulesIncludeDefaults(t *testing.T) {
	app := testApp()
	content := generateAlertRules(app)

	if !strings.Contains(content, "HighErrorRate") {
		t.Error("Alert rules should include HighErrorRate default")
	}
	if !strings.Contains(content, "HighLatency") {
		t.Error("Alert rules should include HighLatency default")
	}
	if !strings.Contains(content, "ServiceDown") {
		t.Error("Alert rules should include ServiceDown default")
	}
}

func TestAlertRulesIncludeCustom(t *testing.T) {
	app := testApp()
	content := generateAlertRules(app)

	if !strings.Contains(content, "error rate is above 5%") {
		t.Error("Alert rules should include custom alert conditions")
	}
}

// ── Grafana tests ──

func TestGrafanaDatasource(t *testing.T) {
	content := generateGrafanaDatasource()

	if !strings.Contains(content, "type: prometheus") {
		t.Error("Grafana datasource should be prometheus")
	}
	if !strings.Contains(content, "http://prometheus:9090") {
		t.Error("Grafana datasource should point to prometheus:9090")
	}
}

func TestGrafanaDashboardContainsPanels(t *testing.T) {
	app := testApp()
	content := generateGrafanaDashboard(app)

	if !strings.Contains(content, "Request Rate") {
		t.Error("Dashboard should include Request Rate panel")
	}
	if !strings.Contains(content, "Error Rate") {
		t.Error("Dashboard should include Error Rate panel")
	}
	if !strings.Contains(content, "Request Latency") {
		t.Error("Dashboard should include Request Latency panel")
	}
	if !strings.Contains(content, "page views") {
		t.Error("Dashboard should include custom tracked metric")
	}
}

// ── Docker Compose tests ──

func TestMonitoringComposeContainsServices(t *testing.T) {
	app := testApp()
	content := generateMonitoringCompose(app)

	if !strings.Contains(content, "prometheus:") {
		t.Error("Monitoring compose should include prometheus service")
	}
	if !strings.Contains(content, "grafana:") {
		t.Error("Monitoring compose should include grafana service")
	}
}

func TestMonitoringComposeAlertmanager(t *testing.T) {
	app := testApp() // has alerts
	content := generateMonitoringCompose(app)

	if !strings.Contains(content, "alertmanager:") {
		t.Error("Monitoring compose should include alertmanager when alerts exist")
	}
}

func TestMonitoringComposeNoAlertmanager(t *testing.T) {
	app := testApp()
	app.Monitoring = []*ir.MonitoringRule{
		{Kind: "track", Metric: "page views"},
	}
	content := generateMonitoringCompose(app)

	if strings.Contains(content, "alertmanager:") {
		t.Error("Monitoring compose should not include alertmanager when no alerts")
	}
}

// ── Node instrumentation tests ──

func TestNodeMetricsContainsCounters(t *testing.T) {
	app := testApp()
	content := generateNodeMetrics(app)

	if !strings.Contains(content, "http_requests_total") {
		t.Error("Node metrics should define http_requests_total")
	}
	if !strings.Contains(content, "http_request_duration_seconds") {
		t.Error("Node metrics should define http_request_duration_seconds")
	}
	if !strings.Contains(content, "page_views") {
		t.Error("Node metrics should define custom tracked metric")
	}
}

func TestNodeMiddleware(t *testing.T) {
	app := testApp()
	content := generateNodeMiddleware(app)

	if !strings.Contains(content, "metricsMiddleware") {
		t.Error("Node middleware should export metricsMiddleware")
	}
	if !strings.Contains(content, "metricsEndpoint") {
		t.Error("Node middleware should export metricsEndpoint")
	}
}

// ── Python instrumentation tests ──

func TestPythonMetrics(t *testing.T) {
	app := testApp()
	content := generatePythonMetrics(app)

	if !strings.Contains(content, "http_requests_total") {
		t.Error("Python metrics should define http_requests_total")
	}
	if !strings.Contains(content, "prometheus_client") {
		t.Error("Python metrics should import prometheus_client")
	}
	if !strings.Contains(content, "page_views") {
		t.Error("Python metrics should define custom tracked metric")
	}
}

// ── Go instrumentation tests ──

func TestGoMetrics(t *testing.T) {
	app := testApp()
	content := generateGoMetrics(app)

	if !strings.Contains(content, "HTTPRequestsTotal") {
		t.Error("Go metrics should define HTTPRequestsTotal")
	}
	if !strings.Contains(content, "promauto") {
		t.Error("Go metrics should use promauto")
	}
}

func TestGoMiddleware(t *testing.T) {
	app := testApp()
	content := generateGoMiddleware(app)

	if !strings.Contains(content, "MetricsMiddleware") {
		t.Error("Go middleware should export MetricsMiddleware")
	}
	if !strings.Contains(content, "MetricsHandler") {
		t.Error("Go middleware should export MetricsHandler")
	}
}

// ── Backend port tests ──

func TestBackendPortNode(t *testing.T) {
	app := testApp()
	port := backendPort(app)
	if port != "3000" {
		t.Errorf("Node backend port = %q, want \"3000\"", port)
	}
}

func TestBackendPortPython(t *testing.T) {
	app := testApp()
	app.Config.Backend = "Python with FastAPI"
	port := backendPort(app)
	if port != "8000" {
		t.Errorf("Python backend port = %q, want \"8000\"", port)
	}
}

func TestBackendPortGo(t *testing.T) {
	app := testApp()
	app.Config.Backend = "Go with Gin"
	port := backendPort(app)
	if port != "8080" {
		t.Errorf("Go backend port = %q, want \"8080\"", port)
	}
}

func TestPrometheusConfigPythonPort(t *testing.T) {
	app := testApp()
	app.Config.Backend = "Python with FastAPI"
	content := generatePrometheusConfig(app)

	if !strings.Contains(content, "testapp-backend:8000") {
		t.Error("Python backend should use port 8000 in Prometheus scrape target")
	}
}

// ── conditionToPromQL tests ──

func TestConditionToPromQLExceeds(t *testing.T) {
	// "exceeds" should be recognized like "above"
	result := conditionToPromQL("error rate exceeds 5 percent")
	if strings.Contains(result, "TODO") {
		t.Errorf("conditionToPromQL should handle 'exceeds': got %q", result)
	}
	if !strings.Contains(result, "0.05") {
		t.Errorf("conditionToPromQL should extract 5%% as 0.05: got %q", result)
	}
}

func TestConditionToPromQLResponseTimeThreshold(t *testing.T) {
	result := conditionToPromQL("response time exceeds 2 seconds")
	if !strings.Contains(result, "> 2") {
		t.Errorf("conditionToPromQL should extract threshold 2: got %q", result)
	}
}

func TestConditionToPromQLAbove(t *testing.T) {
	result := conditionToPromQL("error rate is above 10%")
	if !strings.Contains(result, "0.10") {
		t.Errorf("conditionToPromQL should extract 10%% as 0.10: got %q", result)
	}
}

// ── Monitoring compose network test ──

func TestMonitoringComposeNetworkNotExternal(t *testing.T) {
	app := testApp()
	content := generateMonitoringCompose(app)

	if strings.Contains(content, "external: true") {
		t.Error("Monitoring compose network should not be external by default")
	}
}

// ── Python middleware test ──

func TestPythonMiddlewareAsyncEndpoint(t *testing.T) {
	app := testApp()
	app.Config.Backend = "Python with FastAPI"
	content := generatePythonMiddleware(app)

	if !strings.Contains(content, "async def metrics_endpoint") {
		t.Error("Python metrics_endpoint should be async")
	}
}

// ── Tracking metric mapping tests ──

func TestTrackingToPromQLResponseTime(t *testing.T) {
	expr := trackingToPromQL("response times for all api endpoints", "myapp")
	if !strings.Contains(expr, "http_request_duration_seconds_bucket") {
		t.Errorf("response time tracking should use duration histogram: got %q", expr)
	}
}

func TestTrackingToPromQLErrorRate(t *testing.T) {
	expr := trackingToPromQL("error rates per endpoint", "myapp")
	if !strings.Contains(expr, "http_requests_total") {
		t.Errorf("error rate tracking should use http_requests_total: got %q", expr)
	}
	if !strings.Contains(expr, "status=~\"5..\"") {
		t.Errorf("error rate tracking should filter 5xx status: got %q", expr)
	}
}

func TestTrackingToPromQLActiveUsers(t *testing.T) {
	expr := trackingToPromQL("active users daily and monthly", "myapp")
	if !strings.Contains(expr, "app_active_users") {
		t.Errorf("active users should use app_active_users metric: got %q", expr)
	}
}

func TestIsStandardMetric(t *testing.T) {
	if !isStandardMetric("response times for all api endpoints") {
		t.Error("response time should be standard")
	}
	if !isStandardMetric("error rates per endpoint") {
		t.Error("error rate should be standard")
	}
	if isStandardMetric("page views") {
		t.Error("page views should not be standard")
	}
	if isStandardMetric("active users daily and monthly") {
		t.Error("active users should not be standard")
	}
}

func TestNodeMetricsSkipsStandardTracking(t *testing.T) {
	app := &ir.Application{
		Name:   "TestApp",
		Config: &ir.BuildConfig{Backend: "Node with Express"},
		Monitoring: []*ir.MonitoringRule{
			{Kind: "track", Metric: "response times for all api endpoints"},
			{Kind: "track", Metric: "error rates per endpoint"},
			{Kind: "track", Metric: "active users daily and monthly"},
		},
	}
	content := generateNodeMetrics(app)

	// Standard metrics should NOT create separate custom counters
	if strings.Contains(content, "response_times_for_all_api_endpoints") {
		t.Error("should not create custom metric for response times (already covered by http_request_duration_seconds)")
	}
	if strings.Contains(content, "error_rates_per_endpoint") {
		t.Error("should not create custom metric for error rates (already covered by http_requests_total)")
	}

	// Business metric should be created as Gauge
	if !strings.Contains(content, "app_active_users") {
		t.Error("should create custom metric for active users")
	}
	if !strings.Contains(content, "new Gauge") {
		t.Error("custom business metrics should be Gauge, not Counter")
	}
}

func TestGrafanaDashboardUsesRealPromQL(t *testing.T) {
	app := &ir.Application{
		Name:   "TestApp",
		Config: &ir.BuildConfig{Backend: "Node with Express"},
		Monitoring: []*ir.MonitoringRule{
			{Kind: "track", Metric: "response times for all api endpoints"},
			{Kind: "track", Metric: "error rates per endpoint"},
		},
	}
	content := generateGrafanaDashboard(app)

	// Dashboard should use real PromQL, not fake metric names
	if strings.Contains(content, "response_times_for_all_api_endpoints{") {
		t.Error("dashboard should not use raw description as metric name")
	}
	if !strings.Contains(content, "http_request_duration_seconds_bucket") {
		t.Error("response time panel should use duration histogram PromQL")
	}
	if !strings.Contains(content, "http_requests_total") {
		t.Error("error rate panel should use http_requests_total PromQL")
	}
}

// ── Helper tests ──

func TestSanitizeAlertName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"error rate is above 5%", "ErrorRateIsAbove5"},
		{"cpu usage high", "CpuUsageHigh"},
		{"", "CustomAlert"},
	}
	for _, tt := range tests {
		got := sanitizeAlertName(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeAlertName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
