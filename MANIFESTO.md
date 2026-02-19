# The Human Manifesto

## Software is broken. Not the code — the process.

We have mass-produced more software in the last decade than in the previous fifty years combined. And yet, building software remains one of the most wasteful human activities on the planet.

A feature that takes two sentences to describe takes two weeks to ship. A page that a designer mocks up in an afternoon takes a team of engineers a sprint to build. A product that a founder can sketch on a napkin takes six months and six figures before a single user touches it.

This is not because software is inherently complex. It's because we've been speaking the wrong language.

---

## The gap is not technical. It's linguistic.

Every piece of software begins as an idea expressed in English.

*"I want a dashboard that shows my team's tasks, sorted by deadline."*

*"When a user signs up, send them a welcome email."*

*"Only admins can delete published content."*

These statements are precise. They are unambiguous. They contain everything a machine needs to produce working software.

And yet, between this English and the running application, we force every idea through layers of translation — from English to user stories, to designs, to tickets, to code, to tests, to deployments. Each translation introduces delay. Each handoff introduces error. Each layer employs people whose primary job is to translate, not to create.

We don't have a technology problem. We have a translation problem.

---

## What if English was the code?

Not pseudocode. Not prompts. Not "low-code" drag-and-drop with hidden limitations. Not AI hallucinations wrapped in a chat interface.

Actual, compilable English. Structured enough to be deterministic. Natural enough to be readable by anyone. Powerful enough to produce full-stack, production-grade, tested, secure, deployable applications.

That's Human.

```
app TaskFlow is a web application

data Task:
  belongs to a User
  has a title which is text
  has a status which is either "pending" or "done"
  has a due date

page Dashboard:
  show a list of all tasks sorted by due date
  clicking a task toggles its status
  if no tasks match, show "No tasks found"

api CreateTask:
  requires authentication
  accepts title and due date
  check that title is not empty
  create the task for the current user
  respond with the created task
```

Read it once. You understand the entire application. That's the point.

---

## Our beliefs

### 1. If you can describe it, you can build it.

The ability to build software should not be gated by knowing that a React `useEffect` hook needs a dependency array, or that your Express middleware has to call `next()`, or that PostgreSQL requires `ON DELETE CASCADE` on a foreign key.

These are implementation details. They have nothing to do with what the software does.

Human separates **intent** from **implementation**. You describe what you want. The compiler handles the how.

### 2. Determinism over magic.

Every LLM-powered tool on the market right now has the same fundamental problem: you cannot trust the output. Run the same prompt twice, get different code. Subtle bugs. Missing edge cases. Hallucinated APIs.

Human is a compiler, not a chatbot. The same `.human` file always produces the same output. No randomness. No temperature settings. No "try again and hope for the better." If it compiles, it works.

AI is available as an *optional* enhancement — for interpreting freeform English, suggesting patterns, or importing designs. But the core compiler runs without any AI dependency. You can build on an airplane.

### 3. Quality is not optional.

In the traditional workflow, testing is a chore that gets cut when deadlines loom. Security audits happen quarterly, if at all. Code quality is enforced by linters that developers disable when they're annoyed.

In Human, quality is part of the compiler. You cannot build without tests. You cannot deploy without a security audit. You cannot ship code that doesn't meet the quality bar.

This isn't about being strict for the sake of it. It's about recognizing that every shortcut in quality becomes a tax on every future change. Human eliminates the shortcut entirely.

### 4. The developer chooses the target, not the language.

Your business requirements don't change because your frontend team prefers Vue over React. Your data model doesn't change because you're deploying to AWS instead of GCP.

Human is target-agnostic. The same `.human` source compiles to React, Angular, Vue, Svelte, or HTMX. The same backend logic compiles to Node, Python, Go, or Rust. Switch frameworks without rewriting your application. That's not a feature — it's the architecture.

### 5. Generated code belongs to you.

Human is not a platform. There is no vendor lock-in. There is no runtime dependency. There is no monthly subscription required to keep your app running.

`human eject` gives you a clean, readable, well-structured codebase in the framework of your choice. Walk away with your code any time. We believe the best way to earn trust is to never require it.

### 6. Design is a first-class input.

In most workflows, a designer creates a Figma mockup, then an engineer recreates it from scratch in code. The mockup becomes a suggestion, not a source of truth.

In Human, Figma files, images, and screenshots are compiler inputs alongside English. The design is not an inspiration — it's part of the specification.

---

## Who is Human for?

**Founders** who can describe their product but can't build it. Write a `.human` file. Get a working application. Iterate from there.

**Developers** who are tired of boilerplate. Stop writing the same CRUD endpoints, the same auth flows, the same CI/CD pipelines. Describe what's unique about your application and let the compiler handle the infrastructure.

**Teams** where the gap between "what we want" and "what we shipped" keeps growing. Human makes the specification the source code. When the spec changes, the application changes. No translation layer.

**Agencies** that build similar applications for different clients. Write the pattern once in Human, swap the data models and branding, compile to the client's preferred stack.

**Anyone** who has ever looked at a piece of software and thought: *"Why did this take so long to build?"*

---

## What Human is not

**Human is not AI-generated code.** It is compiled code. There is a critical difference. AI-generated code is probabilistic — it might work, it might not, and you won't know until you test it. Compiled code is deterministic — it either compiles or it doesn't, and if it compiles, the output is guaranteed to match the specification.

**Human is not no-code.** No-code tools trade power for simplicity. You get a working prototype fast, but the moment you need something custom, you hit a wall. Human has no walls. The language is expressive enough to describe any application, and when you eject, you have full source code to extend however you like.

**Human is not a framework.** Frameworks are opinions about how to organize code. Human is a language that compiles to frameworks. It sits above them, not beside them.

---

## The road ahead

Human is in early development. The language specification is stable. The compiler is being built from scratch in Go. The path is long and the scope is ambitious.

We are building in public because we believe this idea is bigger than any one team. If this resonates with you — if you've felt the frustration of translating ideas into code, if you've watched good software take too long to build, if you believe there's a better way — we want to hear from you.

The gap between human intent and working software has existed since the first line of code was written. We think it's time to close it.

---

*Human: the programming language where you describe what you want, and get what you described.*
