# LinkedIn Launch Post

---

I built a programming language where you describe what you want in structured English and the compiler generates production-ready full-stack applications.

It's called Human. The compiler is written in Go, open source (MIT), and fully deterministic.

**The problem it solves:** AI can generate code faster than ever, but the output is only reviewable by engineers. When an LLM writes React components, a product manager or designer has no way to verify correctness.

**The approach:** Human acts as an auditable intermediate representation. A .human file reads like a spec — pages, data models, APIs, security policies, deployment config — all in structured English. Non-technical stakeholders can read and review it. The compiler then generates real code across 14 targets: React, Vue, Angular, Svelte, Node, Python, Go, PostgreSQL, Docker, Terraform, CI/CD, and monitoring.

**What exists today:**
- Complete compiler pipeline (lexer, parser, semantic analyzer, IR, code generators)
- 7 design systems (Material, Shadcn, Ant, Chakra, Bootstrap, Tailwind, Untitled UI)
- MCP server for Claude integration
- Figma component classifier for design-to-code
- 12 example apps, 400+ tests across 33 packages

I built this in 4 days as a solo developer using AI-assisted multi-agent development — 4 Claude instances working in parallel on different compiler subsystems. The agents also wrote .human files to test the compiler. The workflow was the proof of concept.

Looking for feedback, real-world use cases, and contributors. If you're interested in a world where describing software is the same as building it, I'd love to hear from you.

GitHub: https://github.com/barun-bash/human
Website: https://barun-bash.github.io/human/

---
