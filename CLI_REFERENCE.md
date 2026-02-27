# CLI Reference

## Core Commands

### `human check <file>`
Validate a `.human` file for syntax and semantic errors.

```bash
human check app.human
```

### `human build <file>`
Compile a `.human` file into full-stack code.

```bash
human build app.human              # Full build
human build --inspect app.human    # Print IR as YAML
human build --watch app.human      # Rebuild on file changes
```

### `human init [name]`
Create a new Human project with a starter template.

```bash
human init my-app
```

### `human run`
Start the development server from the last build output.

```bash
human run
```

### `human test`
Run generated tests from the build output.

```bash
human test
```

### `human audit`
Display the security and quality report from the last build.

```bash
human audit
```

### `human eject [path]`
Export generated code as a standalone project. Removes Human compiler dependency.

```bash
human eject           # exports to ./output/
human eject my-app    # exports to ./my-app/
```

### `human deploy [file]`
Deploy the application to the configured target.

```bash
human deploy app.human              # Deploy
human deploy --dry-run app.human    # Preview without deploying
human deploy --env staging app.human  # Deploy to a specific environment
```

### `human storybook`
Launch the Storybook dev server from build output.

```bash
human storybook
```

## Reference Commands

### `human explain [topic]`
Learn about Human language syntax by topic.

```bash
human explain          # List all topics
human explain data     # Data models reference
human explain apis     # API endpoints reference
human explain pages    # Pages and navigation
human explain security # Authentication
human explain color    # Color-related patterns
```

### `human syntax [section]`
Full syntax reference, with optional section filter or search.

```bash
human syntax                    # Full reference (uses pager)
human syntax data               # Data section only
human syntax --search "button"  # Search for patterns
```

### `human fix [--dry-run] <file>`
Analyze a `.human` file and suggest auto-fixes for common issues.

```bash
human fix app.human              # Analyze and offer fixes
human fix --dry-run app.human    # Show issues without fixing
```

Detected issues (W601-W605):
- **W601**: Page fetches data but has no loading state
- **W602**: Page shows list but has no empty state
- **W603**: Form without error display
- **W604**: API modifies data without authentication
- **W605**: Queried data with no database index

### `human doctor`
Check environment health: tools, configuration, and project validity.

```bash
human doctor
```

Checks:
- Compiler version, Go runtime
- Docker, Node.js, Python, Terraform availability
- Project config and LLM setup
- `.human` file validity

## AI-Assisted Commands

These require an LLM provider (set via `human connect` or environment variables).

### `human ask "<description>"`
Generate `.human` code from a natural language description.

```bash
human ask "describe a blog with users, posts, and comments"
```

### `human suggest <file>`
Get improvement suggestions for an existing `.human` file.

```bash
human suggest app.human
```

### `human edit <file>`
Interactive AI-assisted editing session.

```bash
human edit app.human
```

### `human convert "<description>"`
Convert a description to `.human` code (design import planned).

```bash
human convert "a todo app with categories"
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--no-color` | Disable colored output |
| `--version`, `-v` | Print compiler version |
| `--help`, `-h` | Show help |

## REPL Commands

Run `human` with no arguments to enter the interactive shell. All CLI commands are available as `/command`:

| REPL | CLI Equivalent | Description |
|------|---------------|-------------|
| `/open <file>` | — | Load a .human file |
| `/build` | `human build` | Compile loaded project |
| `/check` | `human check` | Validate loaded project |
| `/explain [topic]` | `human explain` | Syntax reference |
| `/syntax [section]` | `human syntax` | Full syntax reference |
| `/fix` | `human fix` | Analyze and fix issues |
| `/doctor` | `human doctor` | Environment health check |
| `/ask <desc>` | `human ask` | Generate .human code |
| `/edit [instr]` | `human edit` | AI-assisted editing |
| `/suggest` | `human suggest` | Improvement suggestions |
| `/deploy` | `human deploy` | Deploy application |
| `/run` | `human run` | Start dev server |
| `/test` | `human test` | Run tests |
| `/connect` | — | Set up LLM provider |
| `/theme [name]` | — | Change color theme |
| `/config` | — | View/change settings |
| `/help` | — | Show all commands |
