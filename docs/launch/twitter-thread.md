# Twitter/X Launch Thread

---

**Tweet 1 (hook)**

I built a programming language where you write in English and the compiler generates full-stack apps.

Not a wrapper around GPT. A real compiler — lexer, parser, IR, 14 code generators. Deterministic. Same input, same output, every time.

Open source. Written in Go.

[VIDEO: demo of `human build` generating a full project from a .human file]

---

**Tweet 2 (the syntax)**

This is the entire source code for a full-stack task manager:

```
app TaskFlow is a web application

data Task:
  has a title which is text
  has a status which is either "pending" or "done"

page Dashboard:
  show a list of tasks sorted by due date
  clicking a task toggles its status

api CreateTask:
  requires authentication
  accepts title
  create the task
  respond with the created task
```

No imports. No semicolons. No framework knowledge.

---

**Tweet 3 (what it produces)**

The compiler generates 85+ files across 14 generators:

- React, Vue, Angular, or Svelte frontend
- Node, Python, or Go backend
- PostgreSQL migrations
- Docker + Compose
- Terraform (AWS/GCP)
- GitHub Actions CI/CD
- Prometheus + Grafana monitoring

Change one line in `build with:` to switch from React+Node to Angular+Go. Same source, different target.

---

**Tweet 4 (the insight)**

The real value isn't saving keystrokes. It's readability.

When an LLM writes React, only developers can review the output. When an LLM writes Human, a product manager can read it and say "that's wrong, users shouldn't be able to delete completed tasks."

Human is an auditable intermediate representation between intent and code.

---

**Tweet 5 (the speed)**

Built the entire compiler in 4 days. Solo developer, 4 Claude agents running in parallel:

- Agent 1: compiler internals (lexer, parser, IR)
- Agent 2: code generators (14 targets)
- Agent 3: infrastructure (Docker, Terraform, CI/CD)
- Agent 4: documentation, examples, tests

400+ tests. 33 packages. 12 example apps. MCP server. Figma classifier.

The agents wrote Human code too. It worked.

---

**Tweet 6 (CTA)**

Human is open source under MIT.

Try it:
```
curl -fsSL https://raw.githubusercontent.com/barun-bash/human/main/install.sh | sh
human build examples/taskflow/app.human
```

GitHub: https://github.com/barun-bash/human

If you think software should be easier to build, give it a star.

---
