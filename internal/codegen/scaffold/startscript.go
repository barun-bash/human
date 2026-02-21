package scaffold

import (
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// generateStartScript produces an executable start.sh that installs
// dependencies, sets up the database, and starts the dev servers.
// The script adapts to the configured stack (Node/Python/Go, Postgres, etc.).
func generateStartScript(app *ir.Application) string {
	frontend := ""
	backend := ""
	database := ""
	if app.Config != nil {
		frontend = strings.ToLower(app.Config.Frontend)
		backend = strings.ToLower(app.Config.Backend)
		database = strings.ToLower(app.Config.Database)
	}

	hasJS := strings.Contains(backend, "node") ||
		strings.Contains(frontend, "react") ||
		strings.Contains(frontend, "vue") ||
		strings.Contains(frontend, "angular") ||
		strings.Contains(frontend, "svelte")
	hasNode := strings.Contains(backend, "node")
	hasPython := strings.Contains(backend, "python")
	hasGo := matchesGoBackend(backend)
	hasPostgres := strings.Contains(database, "postgres")

	var b strings.Builder

	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -e\n\n")

	// Install dependencies based on stack
	if hasJS {
		b.WriteString("npm install\n\n")
	}
	if hasPython {
		b.WriteString("# Install Python dependencies\n")
		b.WriteString("if [ -f python/requirements.txt ]; then\n")
		b.WriteString("  pip install -r python/requirements.txt\n")
		b.WriteString("fi\n\n")
	}
	if hasGo {
		b.WriteString("# Build Go backend\n")
		b.WriteString("if [ -d go ]; then\n")
		b.WriteString("  cd go && go build -o ../bin/server ./cmd/server && cd ..\n")
		b.WriteString("fi\n\n")
	}

	// Create .env from example if missing
	b.WriteString("if [ ! -f .env ]; then\n")
	b.WriteString("  cp .env.example .env\n")
	b.WriteString("  echo \"Created .env — edit with your values\"\n")
	b.WriteString("fi\n\n")

	// Check PostgreSQL is reachable (only if using Postgres)
	if hasPostgres {
		b.WriteString("# Check PostgreSQL is reachable\n")
		b.WriteString("source .env 2>/dev/null || true\n")
		b.WriteString("if command -v pg_isready &>/dev/null; then\n")
		b.WriteString("  if ! pg_isready -q 2>/dev/null; then\n")
		b.WriteString("    echo \"Error: PostgreSQL is not running.\"\n")
		b.WriteString("    echo \"Start it with: docker compose up db -d   (or start your local PostgreSQL)\"\n")
		b.WriteString("    exit 1\n")
		b.WriteString("  fi\n")
		b.WriteString("else\n")
		b.WriteString("  echo \"Note: pg_isready not found — skipping database check.\"\n")
		b.WriteString("  echo \"Make sure PostgreSQL is running before continuing.\"\n")
		b.WriteString("fi\n\n")
	}

	// Prisma setup (only for Node backend)
	if hasNode {
		b.WriteString("(cd node && npx prisma generate && npx prisma db push)\n")
	}

	// Start dev servers
	if hasJS {
		b.WriteString("npm run dev\n")
	} else if hasPython {
		b.WriteString("cd python && uvicorn app.main:app --reload --port 8000\n")
	} else if hasGo {
		b.WriteString("./bin/server\n")
	}

	return b.String()
}

// matchesGoBackend checks if the backend config indicates Go without
// false-matching strings like "django" or "mongodb".
func matchesGoBackend(backend string) bool {
	lower := strings.ToLower(backend)
	if lower == "go" || strings.HasPrefix(lower, "go ") {
		return true
	}
	for _, kw := range []string{"gin", "fiber", "golang"} {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
