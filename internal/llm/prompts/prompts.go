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

BUILD TARGET:
  build with:
    frontend using <framework>
    backend using <language>
    database using <database>
    deploy to <platform>

RULES:
- Indentation-based scoping (like Python)
- Keywords are case-insensitive
- Strings in double quotes
- Comments start with #
- Section headers use ── name ── format

OUTPUT FORMAT: When generating .human code, output ONLY valid .human code wrapped in a ` + "```human" + ` code fence. Do not include explanations outside the code fence unless specifically asked.`

// AskPrompt builds a message sequence for the "ask" command.
// The user provides a freeform English description, and the LLM generates .human code.
func AskPrompt(query string) []Message {
	return []Message{
		{Role: RoleSystem, Content: SystemPrompt},
		{Role: RoleUser, Content: query},
	}
}

// SuggestPrompt builds a message sequence for the "suggest" command.
// The LLM analyzes existing .human source and returns improvement suggestions.
func SuggestPrompt(source string) []Message {
	return []Message{
		{Role: RoleSystem, Content: SystemPrompt},
		{Role: RoleUser, Content: "Analyze the following .human source code and suggest improvements. " +
			"Categorize suggestions as: [performance], [security], [usability], [structure], or [feature]. " +
			"Format each suggestion as a single line starting with the category tag.\n\n" +
			"```human\n" + source + "\n```"},
	}
}

// EditPrompt builds a message sequence for the "edit" command.
// Supports conversational editing by including message history.
func EditPrompt(source, instruction string, history []Message) []Message {
	msgs := []Message{
		{Role: RoleSystem, Content: SystemPrompt + "\n\nYou are editing an existing .human file. " +
			"Apply the user's requested changes and return the complete updated file. " +
			"Preserve all existing code that isn't being changed."},
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
