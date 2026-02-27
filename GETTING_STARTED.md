# Getting Started with Human

Human compiles structured English into production-ready full-stack applications.

## Installation

```bash
# From source
git clone https://github.com/barun-bash/human.git
cd human
make build
make install   # installs to /usr/local/bin
```

Verify installation:

```bash
human --version
human doctor     # check your environment
```

## Your First Project

### 1. Create a project

```bash
human init my-app
```

This creates `my-app.human` with a starter template.

### 2. Write your app

Edit `my-app.human`:

```
app MyApp is a web application

data Task:
  has a title which is text
  has a completed flag which is boolean
  has a created datetime

page TaskList:
  show a list of tasks sorted by created date newest first
  each task shows its title and completed status
  clicking a task toggles its completed status
  there is a text input to add a new task
  there is a form to create a Task
  if no tasks match, show "No tasks yet"
  while loading, show a spinner

build with:
  frontend using React with TypeScript
  backend using Node with Express
  database using PostgreSQL
  deploy to Docker
```

### 3. Check for errors

```bash
human check my-app.human
```

### 4. Build

```bash
human build my-app.human
```

This generates:
- React frontend with TypeScript
- Node.js/Express backend
- PostgreSQL migrations and seeds
- Docker configuration
- Tests, security audit, and QA trail

### 5. Run

```bash
human run
```

## Key Commands

| Command | Description |
|---------|-------------|
| `human check <file>` | Validate syntax and semantics |
| `human build <file>` | Compile to full-stack code |
| `human run` | Start the dev server |
| `human test` | Run generated tests |
| `human explain <topic>` | Learn Human syntax |
| `human syntax` | Full syntax reference |
| `human fix <file>` | Find and auto-fix common issues |
| `human doctor` | Check environment health |

## Interactive REPL

Run `human` with no arguments to enter the interactive shell:

```
$ human
HUMAN_> /open my-app.human
HUMAN_> /build
HUMAN_> /explain data
HUMAN_> /doctor
HUMAN_> /help
```

## Learning the Language

- `human explain data` — Learn about data models
- `human explain apis` — Learn about API endpoints
- `human explain pages` — Learn about frontend pages
- `human explain security` — Learn about authentication
- `human syntax --search "button"` — Search for specific patterns

See [SYNTAX_QUICK_REFERENCE.md](SYNTAX_QUICK_REFERENCE.md) for a one-page cheat sheet.

## AI-Assisted Features (Optional)

Set up an LLM provider for AI-powered features:

```bash
human connect   # or set ANTHROPIC_API_KEY env var
```

Then use:
- `human ask "describe a blog with users and posts"` — Generate .human code
- `human suggest my-app.human` — Get improvement suggestions
- `human edit my-app.human` — Interactive AI-assisted editing

## Next Steps

- Browse `examples/` for complete project examples
- Run `human explain` to see all syntax topics
- Read [CLI_REFERENCE.md](CLI_REFERENCE.md) for the full command reference
