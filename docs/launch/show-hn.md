# Show HN Post

## Title

Show HN: Human – A programming language that compiles structured English to full-stack apps

## Post Body

Human is an open-source programming language where you write in structured English and the compiler generates full-stack applications. The compiler is written in Go, deterministic, and produces standalone code you fully own.

Here's what a complete app looks like:

```
app TaskFlow is a web application

data Task:
  belongs to a User
  has a title which is text
  has a status which is either "pending" or "done"
  has a due date

page Dashboard:
  show a list of all tasks sorted by due date
  each task shows its title, status, and due date
  clicking a task toggles its status
  if no tasks match, show "No tasks found"

api CreateTask:
  requires authentication
  accepts title and due date
  check that title is not empty
  create the task for the current user
  respond with the created task

build with:
  frontend using React with TypeScript
  backend using Node with Express
  database using PostgreSQL
```

From that, the compiler generates a React frontend, Node/Express backend, PostgreSQL migrations, Docker config, CI/CD pipelines, and tests. Same source can target Vue, Angular, Svelte, Python/FastAPI, or Go/Gin by changing the `build with` block.

**Why this exists:** When LLMs generate code, the output is hard for non-technical stakeholders to review. Human serves as an auditable intermediate representation. A product manager can read a .human file and understand what the app does. An LLM can write .human files and the compiler enforces correctness deterministically.

**What works today:**
- Full compiler pipeline: lexer, parser, semantic analyzer, Intent IR, 14 code generators
- 7 built-in design systems (Material, Shadcn, Ant, Chakra, Bootstrap, Tailwind, Untitled UI)
- MCP server so Claude can write and compile .human files directly
- Figma component classifier for design-to-code
- 400+ tests across 33 packages

**What's still in progress:** Runtime quality engine (generated tests, security audit), production deployment hardening, and more real-world testing.

Built solo in 4 days using AI-assisted multi-agent development.

GitHub: https://github.com/barun-bash/human
Website: https://barun-bash.github.io/human/
Getting started: https://barun-bash.github.io/human/getting-started.html
MIT License.

Looking for feedback on the language design, real-world use cases to test against, and contributors interested in adding new code generation targets.
