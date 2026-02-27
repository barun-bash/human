package prompts

// Role identifies the sender of a message.
// Mirrors llm.Role but defined here to avoid an import cycle
// (llm imports prompts, so prompts cannot import llm).
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is a single conversation message.
// The connector converts between prompts.Message and llm.Message.
type Message struct {
	Role    Role
	Content string
}

// SystemPrompt is a condensed reference of the Human language spec,
// provided to the LLM so it can generate valid .human code.
const SystemPrompt = `You are an expert in the Human programming language. Human is a structured-English language that compiles to production-ready full-stack applications.

FILE EXTENSION: .human

TOP-LEVEL DECLARATIONS:
  app <Name> is a <platform> application     # platform: web, mobile, desktop, api
  # ── <section> ──                           # section headers organize code

DATA MODELS:
  data <Name>:
    <field> is <type>                          # text, number, decimal, boolean, date, datetime, email, url, file, image, json
    <field> is <type>, required                # required field
    <field> is <type>, optional                # nullable field
    <field> is <type>, unique                  # unique constraint
    <field> is <type>, encrypted               # encrypted at rest
    <field> is either "a" or "b"               # enum
    <field> defaults to <value>                # default value
    belongs to <Model>                         # foreign key (many-to-one)
    has many <Model>                           # one-to-many
    has many <Model> through <JoinModel>       # many-to-many

PAGES:
  page <Name>:
    show <element>                             # display content
    show a list of <data>                      # render collection
    show each <item>'s <field>                 # specify fields
    clicking <element> navigates to <page>     # navigation
    clicking <element> does <action>           # interaction
    typing in <element> does <action>          # input handler
    there is a <input_type> for <purpose>      # form inputs
    if <condition>, show <thing>               # conditional display
    while loading, show a spinner              # loading state

COMPONENTS:
  component <Name>:
    accepts <prop> as <type>
    <content_statements>

APIs:
  api <Name>:
    requires authentication                    # auth guard
    accepts <fields>                           # input parameters
    check that <validation>                    # validation rules
    create/update/delete <data>                # CRUD operations
    fetch <data> from <source>                 # queries
    respond with <data>                        # response

AUTHENTICATION:
  authentication:
    method JWT tokens that expire in <duration>
    method <Provider> OAuth
    passwords are hashed with bcrypt
    rate limit all endpoints to <n> requests per <period>
    sanitize all text inputs against XSS
    enable CORS only for <domain>

POLICIES:
  policy <Name>:
    can <permission>
    cannot <restriction>

WORKFLOWS:
  workflow: when <event>:
    <action_sequence>

ERROR HANDLING:
  if <error_condition>:
    retry <n> times with <delay>
    respond with <message>
    alert <channel>

DATABASE:
  database:
    use <type>                                 # PostgreSQL, MySQL, SQLite
    index <Model> by <fields>
    backup <schedule>

INTEGRATIONS:
  integrate with <Service>:
    api key from environment variable <VAR>
    use for <purpose>

BUILD TARGET (REQUIRED — always include this at the end of the file):
  build with:
    frontend using <framework>                   # React with TypeScript, Vue with TypeScript, Angular with TypeScript, Svelte with TypeScript
    backend using <language>                      # Node with Express, Python with FastAPI, Go with Gin
    database using <database>                     # PostgreSQL, MySQL
    deploy to <platform>                          # Docker, AWS, GCP

RULES:
- Indentation-based scoping (like Python)
- Keywords are case-insensitive
- Strings in double quotes
- Comments start with #
- Section headers use ── name ── format
- ALL declarations (page, data, api, etc.) must be at the TOP LEVEL (zero indentation), not nested under app
- The build with: block is REQUIRED — without it, no frontend/backend/database code is generated

OUTPUT FORMAT: When generating .human code, output ONLY valid .human code wrapped in a ` + "```human" + ` code fence. Do not include explanations outside the code fence unless specifically asked. ALWAYS include a build with: block at the end.`

// buildSystemPrompt returns the system prompt, optionally appending project
// instructions from HUMAN.md when they are provided.
func buildSystemPrompt(base, instructions string) string {
	if instructions == "" {
		return base
	}
	return base + "\n\n── PROJECT INSTRUCTIONS (from HUMAN.md) ──\n" + instructions
}

// AskPrompt builds a message sequence for the "ask" command.
// The user provides a freeform English description, and the LLM generates .human code.
// instructions is optional project context from HUMAN.md (pass "" to omit).
func AskPrompt(query, instructions string) []Message {
	return []Message{
		{Role: RoleSystem, Content: buildSystemPrompt(SystemPrompt, instructions)},
		{Role: RoleUser, Content: query},
	}
}

// SuggestPrompt builds a message sequence for the "suggest" command.
// The LLM analyzes existing .human source and returns improvement suggestions.
// instructions is optional project context from HUMAN.md (pass "" to omit).
func SuggestPrompt(source, instructions string) []Message {
	return []Message{
		{Role: RoleSystem, Content: buildSystemPrompt(SystemPrompt, instructions)},
		{Role: RoleUser, Content: "Analyze the following .human source code and suggest improvements. " +
			"Categorize suggestions as: [performance], [security], [usability], [structure], or [feature]. " +
			"Format each suggestion as a single line starting with the category tag.\n\n" +
			"```human\n" + source + "\n```"},
	}
}

// EditPrompt builds a message sequence for the "edit" command.
// Supports conversational editing by including message history.
// instructions is optional project context from HUMAN.md (pass "" to omit).
func EditPrompt(source, instruction string, history []Message, instructions string) []Message {
	base := SystemPrompt + "\n\nYou are editing an existing .human file. " +
		"Apply the user's requested changes and return the complete updated file. " +
		"Preserve all existing code that isn't being changed."

	msgs := []Message{
		{Role: RoleSystem, Content: buildSystemPrompt(base, instructions)},
	}

	// Include conversation history (capped).
	maxHistory := 10
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}
	msgs = append(msgs, history...)

	msgs = append(msgs, Message{
		Role: RoleUser,
		Content: "Here is the current .human file:\n\n```human\n" + source + "\n```\n\n" +
			"Apply this change: " + instruction,
	})

	return msgs
}

// HowPrompt builds a message sequence for the "how" command.
// The LLM answers questions about Human language usage without generating code.
func HowPrompt(question, instructions string) []Message {
	base := SystemPrompt + `

You are a helpful guide for the Human programming language. The user is asking a question about how to do something in Human. Provide a clear, concise answer with:
1. A brief explanation
2. One or two code examples (using ` + "```human" + ` fences)
3. Related tips or caveats if relevant

Keep answers focused and practical. Do not generate a complete .human file unless asked — just show the relevant snippet.`

	return []Message{
		{Role: RoleSystem, Content: buildSystemPrompt(base, instructions)},
		{Role: RoleUser, Content: question},
	}
}

// RewritePrompt builds a message sequence for the "rewrite" command.
// The LLM regenerates the entire .human file based on an approach description.
func RewritePrompt(source, approach, instructions string) []Message {
	base := SystemPrompt + `

You are rewriting an existing .human file with a different approach. Study the original file carefully — understand its purpose, data models, pages, APIs, and build target. Then regenerate the entire file incorporating the requested changes. The output must be a complete, valid .human file that replaces the original.`

	return []Message{
		{Role: RoleSystem, Content: buildSystemPrompt(base, instructions)},
		{Role: RoleUser, Content: "Here is the current .human file:\n\n```human\n" + source + "\n```\n\n" +
			"Rewrite this file with the following approach: " + approach},
	}
}

// AddPrompt builds a message sequence for the "add" command.
// The LLM adds a new section (page, data model, API, etc.) to an existing file.
func AddPrompt(source, description, instructions string) []Message {
	base := SystemPrompt + `

You are adding a new section to an existing .human file. Study the existing file to understand naming conventions, design patterns, and the build target. Add the requested section in the appropriate location within the file. Return the complete updated file with the new section integrated naturally.`

	return []Message{
		{Role: RoleSystem, Content: buildSystemPrompt(base, instructions)},
		{Role: RoleUser, Content: "Here is the current .human file:\n\n```human\n" + source + "\n```\n\n" +
			"Add the following to this file: " + description},
	}
}
