package cicd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// Generator produces GitHub Actions workflows and repository templates from Intent IR.
type Generator struct{}

// Generate writes CI/CD workflows and GitHub templates to outputDir.
func (g Generator) Generate(app *ir.Application, outputDir string) error {
	files := map[string]string{
		filepath.Join(outputDir, ".github", "workflows", "ci.yml"):              generateCIWorkflow(app),
		filepath.Join(outputDir, ".github", "workflows", "deploy.yml"):          generateDeployWorkflow(app),
		filepath.Join(outputDir, ".github", "workflows", "security.yml"):        generateSecurityWorkflow(app),
		filepath.Join(outputDir, ".github", "PULL_REQUEST_TEMPLATE.md"):         generatePRTemplate(app),
		filepath.Join(outputDir, ".github", "ISSUE_TEMPLATE", "bug_report.md"):  generateBugReport(app),
		filepath.Join(outputDir, ".github", "ISSUE_TEMPLATE", "feature_request.md"): generateFeatureRequest(app),
	}

	for path, content := range files {
		if err := writeFile(path, content); err != nil {
			return err
		}
	}

	return nil
}

func writeFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// ── Stack Detection ──

func isNodeBackend(app *ir.Application) bool {
	if app.Config == nil || app.Config.Backend == "" {
		return true // default to Node
	}
	return strings.Contains(strings.ToLower(app.Config.Backend), "node")
}

func isPythonBackend(app *ir.Application) bool {
	if app.Config == nil {
		return false
	}
	return strings.Contains(strings.ToLower(app.Config.Backend), "python")
}

func isGoBackend(app *ir.Application) bool {
	if app.Config == nil {
		return false
	}
	return strings.Contains(strings.ToLower(app.Config.Backend), "go")
}

func isPostgres(app *ir.Application) bool {
	if app.Config != nil && strings.Contains(strings.ToLower(app.Config.Database), "postgres") {
		return true
	}
	if app.Database != nil && strings.Contains(strings.ToLower(app.Database.Engine), "postgres") {
		return true
	}
	return false
}

func isMySQL(app *ir.Application) bool {
	if app.Config != nil && strings.Contains(strings.ToLower(app.Config.Database), "mysql") {
		return true
	}
	if app.Database != nil && strings.Contains(strings.ToLower(app.Database.Engine), "mysql") {
		return true
	}
	return false
}

func appNameLower(app *ir.Application) string {
	if app.Name != "" {
		return strings.ToLower(strings.ReplaceAll(app.Name, " ", "-"))
	}
	return "app"
}

func deployTarget(app *ir.Application) string {
	if app.Config == nil || app.Config.Deploy == "" {
		return "docker"
	}
	return strings.ToLower(app.Config.Deploy)
}

// ── CI Workflow ──

func generateCIWorkflow(app *ir.Application) string {
	var b strings.Builder

	name := appNameLower(app)
	b.WriteString(fmt.Sprintf("name: %s-ci\n\n", name))
	b.WriteString("on:\n")
	b.WriteString("  push:\n")
	b.WriteString("    branches: [main]\n")
	b.WriteString("  pull_request:\n")
	b.WriteString("    branches: [main]\n\n")
	b.WriteString("jobs:\n")
	b.WriteString("  ci:\n")
	b.WriteString("    runs-on: ubuntu-latest\n")

	// Service containers
	if isPostgres(app) {
		b.WriteString("    services:\n")
		b.WriteString("      postgres:\n")
		b.WriteString("        image: postgres:16\n")
		b.WriteString("        env:\n")
		b.WriteString("          POSTGRES_USER: postgres\n")
		b.WriteString("          POSTGRES_PASSWORD: postgres\n")
		b.WriteString(fmt.Sprintf("          POSTGRES_DB: %s_test\n", strings.ReplaceAll(appNameLower(app), "-", "_")))
		b.WriteString("        ports:\n")
		b.WriteString("          - 5432:5432\n")
		b.WriteString("        options: >-\n")
		b.WriteString("          --health-cmd pg_isready\n")
		b.WriteString("          --health-interval 10s\n")
		b.WriteString("          --health-timeout 5s\n")
		b.WriteString("          --health-retries 5\n")
	} else if isMySQL(app) {
		b.WriteString("    services:\n")
		b.WriteString("      mysql:\n")
		b.WriteString("        image: mysql:8\n")
		b.WriteString("        env:\n")
		b.WriteString("          MYSQL_ROOT_PASSWORD: root\n")
		b.WriteString(fmt.Sprintf("          MYSQL_DATABASE: %s_test\n", strings.ReplaceAll(appNameLower(app), "-", "_")))
		b.WriteString("        ports:\n")
		b.WriteString("          - 3306:3306\n")
		b.WriteString("        options: >-\n")
		b.WriteString("          --health-cmd \"mysqladmin ping\"\n")
		b.WriteString("          --health-interval 10s\n")
		b.WriteString("          --health-timeout 5s\n")
		b.WriteString("          --health-retries 5\n")
	}

	b.WriteString("    steps:\n")
	b.WriteString("      - uses: actions/checkout@v4\n")

	// Backend-specific steps
	if isPythonBackend(app) {
		b.WriteString("      - name: Set up Python\n")
		b.WriteString("        uses: actions/setup-python@v5\n")
		b.WriteString("        with:\n")
		b.WriteString("          python-version: '3.12'\n")
		b.WriteString("          cache: pip\n")
		b.WriteString("      - name: Install dependencies\n")
		b.WriteString("        run: pip install -r requirements.txt\n")
		b.WriteString("      - name: Lint\n")
		b.WriteString("        run: flake8\n")
		b.WriteString("      - name: Test\n")
		b.WriteString("        run: pytest\n")
	} else if isGoBackend(app) {
		b.WriteString("      - name: Set up Go\n")
		b.WriteString("        uses: actions/setup-go@v5\n")
		b.WriteString("        with:\n")
		b.WriteString("          go-version: '1.21'\n")
		b.WriteString("      - name: Vet\n")
		b.WriteString("        run: go vet ./...\n")
		b.WriteString("      - name: Test\n")
		b.WriteString("        run: go test ./...\n")
		b.WriteString("      - name: Build\n")
		b.WriteString("        run: go build ./...\n")
	} else {
		// Default: Node
		b.WriteString("      - name: Set up Node\n")
		b.WriteString("        uses: actions/setup-node@v4\n")
		b.WriteString("        with:\n")
		b.WriteString("          node-version: 20\n")
		b.WriteString("          cache: npm\n")
		b.WriteString("      - name: Install dependencies\n")
		b.WriteString("        run: npm ci\n")
		b.WriteString("      - name: Lint\n")
		b.WriteString("        run: npm run lint\n")
		b.WriteString("      - name: Test\n")
		b.WriteString("        run: npm test\n")
		b.WriteString("      - name: Build\n")
		b.WriteString("        run: npm run build\n")
	}

	return b.String()
}

// ── Deploy Workflow ──

func generateDeployWorkflow(app *ir.Application) string {
	var b strings.Builder

	name := appNameLower(app)
	b.WriteString(fmt.Sprintf("name: %s-deploy\n\n", name))
	b.WriteString("on:\n")
	b.WriteString("  push:\n")
	b.WriteString("    branches: [main]\n\n")
	b.WriteString("jobs:\n")
	b.WriteString("  deploy:\n")
	b.WriteString("    runs-on: ubuntu-latest\n")
	b.WriteString("    steps:\n")
	b.WriteString("      - uses: actions/checkout@v4\n")

	target := deployTarget(app)

	switch target {
	case "vercel":
		b.WriteString("      - name: Install Vercel CLI\n")
		b.WriteString("        run: npm install -g vercel\n")
		b.WriteString("      - name: Deploy to Vercel\n")
		b.WriteString("        run: vercel --prod --token ${{ secrets.VERCEL_TOKEN }}\n")
		b.WriteString("        env:\n")
		b.WriteString("          VERCEL_TOKEN: ${{ secrets.VERCEL_TOKEN }}\n")

	case "aws":
		b.WriteString("      - name: Configure AWS credentials\n")
		b.WriteString("        uses: aws-actions/configure-aws-credentials@v4\n")
		b.WriteString("        with:\n")
		b.WriteString("          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}\n")
		b.WriteString("          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}\n")
		b.WriteString("          aws-region: us-east-1\n")
		b.WriteString("      - name: Login to Amazon ECR\n")
		b.WriteString("        uses: aws-actions/amazon-ecr-login@v2\n")
		b.WriteString("      - name: Build and push Docker image\n")
		b.WriteString(fmt.Sprintf("        run: |\n          docker build -t %s .\n          docker tag %s:latest ${{ secrets.AWS_ACCOUNT_ID }}.dkr.ecr.us-east-1.amazonaws.com/%s:latest\n          docker push ${{ secrets.AWS_ACCOUNT_ID }}.dkr.ecr.us-east-1.amazonaws.com/%s:latest\n", name, name, name, name))
		b.WriteString("      - name: Deploy to ECS\n")
		b.WriteString(fmt.Sprintf("        run: aws ecs update-service --cluster %s-cluster --service %s-service --force-new-deployment\n", name, name))

	case "gcp":
		b.WriteString("      - name: Authenticate to Google Cloud\n")
		b.WriteString("        uses: google-github-actions/auth@v2\n")
		b.WriteString("        with:\n")
		b.WriteString("          credentials_json: ${{ secrets.GCP_SA_KEY }}\n")
		b.WriteString("      - name: Set up Cloud SDK\n")
		b.WriteString("        uses: google-github-actions/setup-gcloud@v2\n")
		b.WriteString("      - name: Build and push to GCR\n")
		b.WriteString(fmt.Sprintf("        run: |\n          gcloud builds submit --tag gcr.io/${{ secrets.GCP_PROJECT_ID }}/%s\n", name))
		b.WriteString("      - name: Deploy to Cloud Run\n")
		b.WriteString(fmt.Sprintf("        run: |\n          gcloud run deploy %s --image gcr.io/${{ secrets.GCP_PROJECT_ID }}/%s --region us-central1 --platform managed\n", name, name))

	default: // docker
		b.WriteString("      - name: Log in to Docker Hub\n")
		b.WriteString("        uses: docker/login-action@v3\n")
		b.WriteString("        with:\n")
		b.WriteString("          username: ${{ secrets.DOCKER_USERNAME }}\n")
		b.WriteString("          password: ${{ secrets.DOCKER_PASSWORD }}\n")
		b.WriteString("      - name: Build and push Docker image\n")
		b.WriteString("        uses: docker/build-push-action@v5\n")
		b.WriteString("        with:\n")
		b.WriteString("          context: .\n")
		b.WriteString("          push: true\n")
		b.WriteString(fmt.Sprintf("          tags: ${{ secrets.DOCKER_USERNAME }}/%s:latest\n", name))
	}

	return b.String()
}

// ── Security Workflow ──

func generateSecurityWorkflow(app *ir.Application) string {
	var b strings.Builder

	name := appNameLower(app)
	b.WriteString(fmt.Sprintf("name: %s-security\n\n", name))
	b.WriteString("on:\n")
	b.WriteString("  schedule:\n")
	b.WriteString("    - cron: '0 0 * * 0'\n")
	b.WriteString("  pull_request:\n")
	b.WriteString("    branches: [main]\n\n")
	b.WriteString("jobs:\n")
	b.WriteString("  security:\n")
	b.WriteString("    runs-on: ubuntu-latest\n")
	b.WriteString("    steps:\n")
	b.WriteString("      - uses: actions/checkout@v4\n")

	if isPythonBackend(app) {
		b.WriteString("      - name: Set up Python\n")
		b.WriteString("        uses: actions/setup-python@v5\n")
		b.WriteString("        with:\n")
		b.WriteString("          python-version: '3.12'\n")
		b.WriteString("      - name: Install dependencies\n")
		b.WriteString("        run: pip install -r requirements.txt\n")
		b.WriteString("      - name: Security audit\n")
		b.WriteString("        run: pip install pip-audit && pip-audit\n")
	} else if isGoBackend(app) {
		b.WriteString("      - name: Set up Go\n")
		b.WriteString("        uses: actions/setup-go@v5\n")
		b.WriteString("        with:\n")
		b.WriteString("          go-version: '1.21'\n")
		b.WriteString("      - name: Vet\n")
		b.WriteString("        run: go vet ./...\n")
		b.WriteString("      - name: Vulnerability check\n")
		b.WriteString("        run: go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...\n")
	} else {
		// Default: Node
		b.WriteString("      - name: Set up Node\n")
		b.WriteString("        uses: actions/setup-node@v4\n")
		b.WriteString("        with:\n")
		b.WriteString("          node-version: 20\n")
		b.WriteString("      - name: Install dependencies\n")
		b.WriteString("        run: npm ci\n")
		b.WriteString("      - name: Security audit\n")
		b.WriteString("        run: npm audit --audit-level=high\n")
	}

	return b.String()
}

// ── PR Template ──

func generatePRTemplate(app *ir.Application) string {
	var b strings.Builder

	b.WriteString("## Description\n\n")
	b.WriteString("<!-- Briefly describe the changes in this PR -->\n\n")
	b.WriteString("## Type of change\n\n")
	b.WriteString("- [ ] Bug fix\n")
	b.WriteString("- [ ] New feature\n")
	b.WriteString("- [ ] Breaking change\n")
	b.WriteString("- [ ] Documentation update\n")
	b.WriteString("- [ ] Refactoring\n\n")
	b.WriteString("## Checklist\n\n")
	b.WriteString("- [ ] Tests pass\n")
	b.WriteString("- [ ] Security audit passes\n")
	b.WriteString("- [ ] Documentation updated\n")
	b.WriteString("- [ ] No sensitive data exposed\n")

	return b.String()
}

// ── Bug Report Template ──

func generateBugReport(app *ir.Application) string {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString("name: Bug Report\n")
	b.WriteString("description: Report a bug\n")
	b.WriteString("labels: [bug]\n")
	b.WriteString("---\n\n")
	b.WriteString("## Describe the bug\n\n")
	b.WriteString("<!-- A clear and concise description of what the bug is -->\n\n")
	b.WriteString("## To reproduce\n\n")
	b.WriteString("1. Go to '...'\n")
	b.WriteString("2. Click on '...'\n")
	b.WriteString("3. See error\n\n")
	b.WriteString("## Expected behavior\n\n")
	b.WriteString("<!-- What you expected to happen -->\n\n")
	b.WriteString("## Environment\n\n")
	b.WriteString("- OS: \n")
	b.WriteString("- Browser: \n")
	b.WriteString("- Version: \n")

	return b.String()
}

// ── Feature Request Template ──

func generateFeatureRequest(app *ir.Application) string {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString("name: Feature Request\n")
	b.WriteString("description: Suggest a new feature\n")
	b.WriteString("labels: [enhancement]\n")
	b.WriteString("---\n\n")
	b.WriteString("## Problem description\n\n")
	b.WriteString("<!-- Describe the problem you'd like solved -->\n\n")
	b.WriteString("## Proposed solution\n\n")
	b.WriteString("<!-- Describe the solution you'd like -->\n\n")
	b.WriteString("## Alternatives considered\n\n")
	b.WriteString("<!-- Describe alternatives you've considered -->\n")

	return b.String()
}
