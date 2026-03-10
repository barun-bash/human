#!/usr/bin/env bash
set -e

npm install

if [ ! -f .env ]; then
  cp .env.example .env
  echo "Created .env — edit with your values"
fi

set -a
source .env 2>/dev/null || true
set +a

if [ -z "${DATABASE_URL:-}" ]; then
  echo "Error: DATABASE_URL is not set."
  echo "Copy .env.example to .env and set DATABASE_URL before running."
  exit 1
fi

# Check PostgreSQL is reachable
if command -v pg_isready &>/dev/null; then
  if ! pg_isready -q 2>/dev/null; then
    echo "Error: PostgreSQL is not running."
    echo "Start it with: docker compose up db -d   (or start your local PostgreSQL)"
    exit 1
  fi
else
  echo "Note: pg_isready not found — skipping database check."
  echo "Make sure PostgreSQL is running before continuing."
fi

(cd node && npx prisma generate && npx prisma db push)
npm run dev
