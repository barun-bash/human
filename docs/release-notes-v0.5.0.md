# v0.5.0 — LLM Integration Release

Human now speaks to AI — MCP server enables Claude to write and build .human files.

### What's New
- **MCP Server:** We have added an MCP server that exposes the Human compiler to LLMs like Claude via 6 new tools (`human_build`, `human_validate`, `human_ir`, etc.), unlocking direct AI generation and compilation of `.human` files.
- **Figma Intelligence:** A new component classifier with 4-tier heuristics has been added. It can infer data models from UI patterns and acts as a code generator directly from Figma node trees.
- **Form Bindings:** Svelte and Angular generators have been upgraded with native form bindings (`bind:value` for Svelte, `ReactiveFormsModule` for Angular) to dramatically improve interaction fidelity in the generated apps.
- **Docker Hardening:** Node.js backend Dockerfiles now auto-generate a `start.sh` script to run database migrations (`prisma migrate deploy`) prior to starting the server, enabling seamless zero-touch `docker-compose up`.
- **4 New Examples:** fitness, events, inventory, and figma-demo examples.
- **Language Specification & Prompts:** Comprehensive language spec and an optimized ~3600 token LLM prompt.

### The Demo
This release unlocks the **Figma → Claude → Human → Running App** workflow. You can now use Claude with our MCP server to read a Figma file, infer the data models and component structure, generate a valid `.human` intent file, validate it against our new Language Specification, build it using the Human compiler, and stand up the running app natively using the hardened Docker compose configuration.

### By the Numbers
- **14 Generators:** Emitting robust architectures across frontends, backends, infrastructure, and monitoring.
- **12 Examples:** From e-commerce to task flow, spanning multiple frameworks.
- **33 Packages:** The Human internal compiler is structured cleanly, all fully tested.
- **7 Design Systems:** Extensive support for styling, from Tailwind to Material.
- **5 Integrations:** Built-in hooks for Stripe, SendGrid, Slack, and more.

### What's Next
We are aiming to expand our `docker-compose` end-to-end validation across the remaining frameworks, introduce a dynamic plugin system to let developers extend the compiler, and begin foundational work on Human Cloud.

### Getting Started
**Install the Human CLI via Go:**
```bash
go install github.com/barun-bash/human/cmd/human@v0.5.0
```
**Explore the code:** Check out our new examples and documentation in the `/examples` and `/docs` directories!
