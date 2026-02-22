# Human Compiler — Demo Guide

This guide walks you through the Human compiler end-to-end: writing `.human` files, validating, building, and using the MCP server with Claude Desktop.

## Prerequisites

- **Go 1.21+** installed
- **Git** for cloning the repository
- Clone and build:

```bash
git clone https://github.com/barun-bash/human.git
cd human
make build        # builds ./human binary
make install      # optional: installs to /usr/local/bin
```

Verify the install:

```bash
human --version
```

---

## Quick Demo: Write → Check → Build → Inspect

### 1. Write a `.human` file

Create `hello/app.human`:

```
app HelloWorld is a web application

── theme ──

theme:
  primary color is #4F46E5
  secondary color is #10B981
  font is Inter for body and headings
  border radius is rounded
  design system is tailwind

── frontend ──

page Home:
  show a hero section with the app name
  show a list of messages sorted by date descending
  there is a form to create a new message
  clicking "Send" creates the message

── backend ──

data Message:
  has a content which is text
  has a created datetime

api CreateMessage:
  accepts content
  check that content is not empty
  create a Message with the given fields
  respond with the created message

api ListMessages:
  fetch all Message
  sort by created descending
  respond with messages

── database ──

database:
  use PostgreSQL

── build ──

build with:
  frontend using React with TypeScript
  backend using Node with Express
  database using PostgreSQL
  deploy to Docker
```

### 2. Validate

```bash
human check hello/app.human
```

Expected output:

```
✓ hello/app.human is valid — 1 data model, 1 page, 2 APIs
```

### 3. Build

```bash
human build hello/app.human
```

This generates production-ready code in `.human/output/`:

```
.human/output/
├── react/          # Frontend (React + TypeScript)
├── node/           # Backend (Node + Express)
├── postgres/       # Database migrations
├── docker-compose.yml
├── start.sh
├── package.json
└── ...
```

### 4. Inspect generated code

Browse the output:

```bash
ls .human/output/react/src/
ls .human/output/node/src/
cat .human/output/docker-compose.yml
```

---

## MCP Server Setup (Claude Desktop)

The Human compiler includes an MCP server that lets Claude interact with the compiler directly.

### Build the MCP server

```bash
go build -o human-mcp ./cmd/human-mcp/main.go
```

### Configure Claude Desktop

Add to your Claude Desktop MCP config (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "human": {
      "command": "/path/to/human-mcp"
    }
  }
}
```

Replace `/path/to/human-mcp` with the actual path to the built binary.

### Restart Claude Desktop

After updating the config, restart Claude Desktop. The Human tools will appear in the tools menu.

---

## MCP Tools

| Tool | Description |
|------|-------------|
| `human_validate` | Validate `.human` source without generating code. Returns structured diagnostics. |
| `human_build` | Compile `.human` source through the full pipeline. Returns a file manifest and key files. |
| `human_ir` | Parse `.human` source and return the Intent IR as YAML. |
| `human_examples` | List available examples, or retrieve a specific example's source code. |
| `human_spec` | Return the complete Human language specification. |
| `human_read_file` | Read a file from the last build output. |

### Example prompts for Claude

- "Validate this .human code for me" → uses `human_validate`
- "Build me a todo app in Human" → uses `human_examples` + `human_build`
- "Show me the language spec for data models" → uses `human_spec`
- "What files were generated?" → uses `human_read_file`

---

## Design System Showcase

The Human compiler supports 7 design systems across 4 frontend frameworks:

| Design System | React | Vue | Angular | Svelte |
|--------------|-------|-----|---------|--------|
| **Material UI** | @mui/material | Vuetify | @angular/material | Tailwind fallback |
| **Shadcn/ui** | Radix + Tailwind | Radix-Vue + Tailwind | — | Bits UI + Tailwind |
| **Ant Design** | antd | ant-design-vue | ng-zorro-antd | — |
| **Chakra UI** | @chakra-ui/react | — | — | — |
| **Bootstrap** | react-bootstrap | bootstrap-vue-next | ngx-bootstrap | sveltestrap |
| **Tailwind CSS** | tailwindcss | tailwindcss | tailwindcss | tailwindcss |
| **Untitled UI** | tailwindcss | tailwindcss | tailwindcss | tailwindcss |

To use a design system, add it to your theme:

```
theme:
  primary color is #4F46E5
  design system is chakra
```

---

## Cross-Framework Targeting

The same `.human` source can target different frameworks by changing the `build with` section:

```
# React + Node
build with:
  frontend using React with TypeScript
  backend using Node with Express

# Vue + Python
build with:
  frontend using Vue with TypeScript
  backend using Python with FastAPI

# Angular + Go
build with:
  frontend using Angular with TypeScript
  backend using Go with Gin

# Svelte + Node
build with:
  frontend using Svelte with TypeScript
  backend using Node with Express
```

The compiler's Intent IR is framework-agnostic — the same intermediate representation generates correct code for any target.

---

## Example Gallery

| Example | Description | Frontend | Backend | Design System |
|---------|-------------|----------|---------|---------------|
| **taskflow** | Task management app | React | Node | — |
| **blog** | Blog platform with comments | Vue | Python | — |
| **ecommerce** | Online store with orders | Angular | Go | — |
| **saas** | Team project management | Svelte | Node | Shadcn |
| **recipes** | Recipe sharing community | React | Node | Tailwind |
| **projects** | Project hub with boards | React | Node | Shadcn |
| **api-only** | Payment gateway API | — | Node | — |
| **fitness** | Fitness tracker with goals | Vue | Python | Material |
| **events** | Event booking with tickets | Angular | Node | Ant Design |
| **inventory** | Stock management system | React | Go | Chakra |
| **figma-demo** | SaaS analytics dashboard | React | Python | Untitled UI |

Build any example:

```bash
human build examples/fitness/app.human
human build examples/events/app.human
human build examples/inventory/app.human
human build examples/figma-demo/app.human
```

---

## Troubleshooting

### `human check` reports validation errors

- **E104 "references model X which does not exist"**: Your API references a model name that doesn't match any `data` declaration. Model names are case-sensitive — use `User`, not `user` or `users`.
- **W304 "Unknown border radius"**: Valid values are `sharp`, `smooth`, `rounded`, or `pill`.

### Build produces no frontend output

Make sure your `build with:` section includes a `frontend using` line:

```
build with:
  frontend using React with TypeScript
```

### MCP server doesn't connect

1. Verify the binary path in your Claude Desktop config is correct and absolute.
2. Check that the binary has execute permissions: `chmod +x human-mcp`
3. Test manually: `echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./human-mcp`

### Design system deps not appearing

Ensure the design system supports your chosen framework (see the compatibility table above). Unsupported combos fall back to Tailwind CSS.

### Build output location

All build output goes to `.human/output/` relative to your current working directory. Use `human build --inspect` to view the IR without generating files.
