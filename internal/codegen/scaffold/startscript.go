package scaffold

import (
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// generateStartScript produces an executable start.sh that installs
// dependencies, sets up the database, and starts the dev servers.
func generateStartScript(app *ir.Application) string {
	_ = app // reserved for future app-specific logic
	var b strings.Builder

	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -e\n\n")

	// Install dependencies
	b.WriteString("npm install\n\n")

	// Create .env from example if missing
	b.WriteString("if [ ! -f .env ]; then\n")
	b.WriteString("  cp .env.example .env\n")
	b.WriteString("  echo \"Created .env — edit with your values\"\n")
	b.WriteString("fi\n\n")

	// Check PostgreSQL is reachable
	b.WriteString("# Check PostgreSQL is reachable before running Prisma\n")
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

	// Prisma setup and start
	b.WriteString("cd node && npx prisma generate && npx prisma db push && cd ..\n")
	b.WriteString("npm run dev\n")

	return b.String()
}
