package parser

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/lexer"
)

// Parse lexes and parses a .human source string into an AST.
// Returns the program and any parse errors. On partial failure the
// program may still contain successfully parsed declarations.
func Parse(source string) (*Program, error) {
	lex := lexer.New(source)
	tokens, err := lex.Tokenize()
	if err != nil {
		return nil, fmt.Errorf("lexer error: %w", err)
	}
	return ParseTokens(tokens)
}

// ParseTokens parses a pre-built token stream into an AST.
func ParseTokens(tokens []lexer.Token) (*Program, error) {
	p := &parser{tokens: tokens}
	prog := p.parse()
	if len(p.errors) > 0 {
		return prog, fmt.Errorf("parse errors:\n  %s", strings.Join(p.errors, "\n  "))
	}
	return prog, nil
}

// parser holds the state for a single parse run.
type parser struct {
	tokens []lexer.Token
	pos    int
	errors []string
}

// ── Public parse entry point ──

func (p *parser) parse() *Program {
	prog := &Program{}
	p.skipNoise()

	for !p.isAtEnd() {
		line := p.peek().Line

		switch p.peek().Type {
		case lexer.TOKEN_SECTION_HEADER:
			prog.Sections = append(prog.Sections, p.peek().Literal)
			p.advance()

		case lexer.TOKEN_APP:
			if decl := p.parseAppDeclaration(); decl != nil {
				prog.App = decl
			}

		case lexer.TOKEN_DATA:
			if decl := p.parseDataDeclaration(); decl != nil {
				prog.Data = append(prog.Data, decl)
			}

		case lexer.TOKEN_PAGE:
			if decl := p.parsePageDeclaration(); decl != nil {
				prog.Pages = append(prog.Pages, decl)
			}

		case lexer.TOKEN_COMPONENT:
			if decl := p.parseComponentDeclaration(); decl != nil {
				prog.Components = append(prog.Components, decl)
			}

		case lexer.TOKEN_API:
			if decl := p.parseAPIDeclaration(); decl != nil {
				prog.APIs = append(prog.APIs, decl)
			}

		case lexer.TOKEN_POLICY:
			if decl := p.parsePolicyDeclaration(); decl != nil {
				prog.Policies = append(prog.Policies, decl)
			}

		case lexer.TOKEN_WHEN:
			if decl := p.parseWorkflowDeclaration(); decl != nil {
				prog.Workflows = append(prog.Workflows, decl)
			}

		case lexer.TOKEN_THEME:
			if decl := p.parseThemeDeclaration(); decl != nil {
				prog.Theme = decl
			}

		case lexer.TOKEN_AUTHENTICATION:
			if decl := p.parseAuthenticationDeclaration(); decl != nil {
				prog.Authentication = decl
			}

		case lexer.TOKEN_DATABASE:
			if decl := p.parseDatabaseDeclaration(); decl != nil {
				prog.Database = decl
			}

		case lexer.TOKEN_INTEGRATE:
			if decl := p.parseIntegrationDeclaration(); decl != nil {
				prog.Integrations = append(prog.Integrations, decl)
			}

		case lexer.TOKEN_ENVIRONMENT:
			if decl := p.parseEnvironmentDeclaration(); decl != nil {
				prog.Environments = append(prog.Environments, decl)
			}

		case lexer.TOKEN_BUILD:
			if decl := p.parseBuildDeclaration(); decl != nil {
				prog.Build = decl
			}

		case lexer.TOKEN_ARCHITECTURE:
			if decl := p.parseArchitectureDeclaration(); decl != nil {
				prog.Architecture = decl
			}

		case lexer.TOKEN_IF:
			if decl := p.parseErrorHandler(); decl != nil {
				prog.ErrorHandlers = append(prog.ErrorHandlers, decl)
			}

		case lexer.TOKEN_BRANCHES:
			// branches: block — parse as generic statement block
			p.advance()
			stmts := p.parseIndentedBody()
			for _, s := range stmts {
				prog.Statements = append(prog.Statements, s)
			}

		default:
			// Top-level statement (source control, repository, track, alert, etc.)
			stmt := p.parseTopLevelStatement()
			if stmt != nil {
				prog.Statements = append(prog.Statements, stmt)
			}
		}

		// Safety: ensure we always advance to avoid infinite loops
		if p.peek().Line == line && !p.isAtEnd() {
			if p.peek().Type == lexer.TOKEN_NEWLINE || p.peek().Type == lexer.TOKEN_DEDENT {
				p.advance()
			} else if p.peek().Line == line {
				// Still stuck on the same line — skip token
				p.advance()
			}
		}
		p.skipNoise()
	}

	return prog
}

// ── Declaration parsers ──

// parseAppDeclaration parses: app <Name> is a <platform> application
func (p *parser) parseAppDeclaration() *AppDeclaration {
	line := p.peek().Line
	p.advance() // consume APP

	name := p.advanceLiteral() // name

	// Consume "is a <platform> application"
	p.match(lexer.TOKEN_IS)
	p.matchAny(lexer.TOKEN_A, lexer.TOKEN_AN)

	platform := p.advanceLiteral() // "web", "mobile", etc.
	p.skipRestOfLine()             // "application" and anything else

	return &AppDeclaration{Name: name, Platform: platform, Line: line}
}

// parseDataDeclaration parses a data model with fields and relationships.
func (p *parser) parseDataDeclaration() *DataDeclaration {
	line := p.peek().Line
	p.advance() // consume DATA

	name := p.advanceLiteral()
	decl := &DataDeclaration{Name: name, Line: line}

	if !p.match(lexer.TOKEN_COLON) {
		p.addError(fmt.Sprintf("line %d: expected ':' after data %s", line, name))
		p.synchronize()
		return decl
	}
	p.skipNewlines()

	if !p.match(lexer.TOKEN_INDENT) {
		return decl
	}

	for !p.check(lexer.TOKEN_DEDENT) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(lexer.TOKEN_DEDENT) || p.isAtEnd() {
			break
		}

		startPos := p.pos
		switch p.peek().Type {
		case lexer.TOKEN_HAS:
			p.parseDataHas(decl)
		case lexer.TOKEN_BELONGS:
			p.parseDataBelongs(decl)
		default:
			p.skipRestOfLine()
		}
		if p.pos == startPos {
			p.advance()
		}
		p.skipNewlines()
	}

	p.match(lexer.TOKEN_DEDENT)
	return decl
}

// parseDataHas parses "has a/an ... " or "has many ..." within a data block.
func (p *parser) parseDataHas(decl *DataDeclaration) {
	line := p.peek().Line
	p.advance() // consume HAS

	// has many → relationship
	if p.check(lexer.TOKEN_MANY) {
		p.advance() // consume MANY
		target := p.advanceLiteral()
		through := ""
		if p.match(lexer.TOKEN_THROUGH) {
			through = p.advanceLiteral()
		}
		p.skipRestOfLine()
		decl.Relationships = append(decl.Relationships, &Relationship{
			Kind: "has_many", Target: target, Through: through, Line: line,
		})
		return
	}

	// has a/an [optional] <name> [which is [modifiers] <type>]
	p.matchAny(lexer.TOKEN_A, lexer.TOKEN_AN)

	field := &Field{Line: line}

	// Check for "optional" modifier before the field name
	if p.check(lexer.TOKEN_OPTIONAL) {
		field.Modifiers = append(field.Modifiers, "optional")
		p.advance()
	}

	// Field name
	field.Name = p.advanceLiteral()

	// Check for "which is" (full form) vs type keyword (shorthand)
	if p.match(lexer.TOKEN_WHICH) {
		p.match(lexer.TOKEN_IS) // consume IS

		// Check for modifiers: unique, encrypted
		for p.check(lexer.TOKEN_UNIQUE) || p.check(lexer.TOKEN_ENCRYPTED) {
			field.Modifiers = append(field.Modifiers, strings.ToLower(p.advance().Literal))
		}

		// Check for "either" (enum)
		if p.check(lexer.TOKEN_EITHER) {
			p.advance()
			field.EnumValues = p.parseEnumValues()
		} else if p.isTypeKeyword() {
			field.Type = strings.ToLower(p.advance().Literal)
		} else {
			// Unknown type — take whatever word is there
			field.Type = p.advanceLiteral()
		}
	} else if p.match(lexer.TOKEN_WHICH) {
		// shouldn't get here, but safety
		p.skipRestOfLine()
	} else if p.isTypeKeyword() {
		// Shorthand: has a created datetime
		field.Type = strings.ToLower(p.advance().Literal)
	} else if p.check(lexer.TOKEN_DEFAULTS) {
		// has a <name> which defaults to <value>
		p.advance() // defaults
		p.match(lexer.TOKEN_TO)
		field.Default = p.collectRestOfLine()
	}

	p.skipRestOfLine()
	decl.Fields = append(decl.Fields, field)
}

// parseDataBelongs parses "belongs to a <Data>" within a data block.
func (p *parser) parseDataBelongs(decl *DataDeclaration) {
	line := p.peek().Line
	p.advance() // consume BELONGS
	p.match(lexer.TOKEN_TO)
	p.matchAny(lexer.TOKEN_A, lexer.TOKEN_AN)

	target := p.advanceLiteral()
	p.skipRestOfLine()

	decl.Relationships = append(decl.Relationships, &Relationship{
		Kind: "belongs_to", Target: target, Line: line,
	})
}

// parseEnumValues parses: "value1" or "value2" or "value3"
func (p *parser) parseEnumValues() []string {
	var values []string
	if p.check(lexer.TOKEN_STRING_LIT) {
		values = append(values, p.advance().Literal)
	}
	for p.match(lexer.TOKEN_OR) {
		if p.check(lexer.TOKEN_STRING_LIT) {
			values = append(values, p.advance().Literal)
		}
	}
	return values
}

// parsePageDeclaration parses a page with display/interaction statements.
func (p *parser) parsePageDeclaration() *PageDeclaration {
	line := p.peek().Line
	p.advance() // consume PAGE

	name := p.advanceLiteral()
	decl := &PageDeclaration{Name: name, Line: line}
	decl.Statements = p.parseIndentedBody()
	return decl
}

// parseComponentDeclaration parses a reusable component.
func (p *parser) parseComponentDeclaration() *ComponentDeclaration {
	line := p.peek().Line
	p.advance() // consume COMPONENT

	name := p.advanceLiteral()
	decl := &ComponentDeclaration{Name: name, Line: line}

	if !p.match(lexer.TOKEN_COLON) {
		p.addError(fmt.Sprintf("line %d: expected ':' after component %s", line, name))
		p.synchronize()
		return decl
	}
	p.skipNewlines()
	if !p.match(lexer.TOKEN_INDENT) {
		return decl
	}

	for !p.check(lexer.TOKEN_DEDENT) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(lexer.TOKEN_DEDENT) || p.isAtEnd() {
			break
		}
		startPos := p.pos
		if p.check(lexer.TOKEN_ACCEPTS) {
			p.advance()
			decl.Accepts = p.parseParamList()
		} else {
			stmt := p.parseBodyStatement()
			if stmt != nil {
				decl.Statements = append(decl.Statements, stmt)
			}
		}
		if p.pos == startPos {
			p.advance()
		}
		p.skipNewlines()
	}
	p.match(lexer.TOKEN_DEDENT)
	return decl
}

// parseAPIDeclaration parses an API endpoint.
func (p *parser) parseAPIDeclaration() *APIDeclaration {
	line := p.peek().Line
	p.advance() // consume API

	name := p.advanceLiteral()
	decl := &APIDeclaration{Name: name, Line: line}

	if !p.match(lexer.TOKEN_COLON) {
		p.addError(fmt.Sprintf("line %d: expected ':' after api %s", line, name))
		p.synchronize()
		return decl
	}
	p.skipNewlines()
	if !p.match(lexer.TOKEN_INDENT) {
		return decl
	}

	for !p.check(lexer.TOKEN_DEDENT) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(lexer.TOKEN_DEDENT) || p.isAtEnd() {
			break
		}

		startPos := p.pos
		switch p.peek().Type {
		case lexer.TOKEN_REQUIRES:
			p.advance() // consume REQUIRES
			if p.check(lexer.TOKEN_AUTHENTICATION) {
				p.advance()
				decl.Auth = true
				p.skipRestOfLine()
			} else {
				text := "requires " + p.collectRestOfLine()
				decl.Statements = append(decl.Statements, &Statement{
					Kind: "requires", Text: text, Line: p.peek().Line,
				})
			}
		case lexer.TOKEN_ACCEPTS:
			p.advance() // consume ACCEPTS
			decl.Accepts = p.parseParamList()
		default:
			stmt := p.parseBodyStatement()
			if stmt != nil {
				decl.Statements = append(decl.Statements, stmt)
			}
		}
		if p.pos == startPos {
			p.advance()
		}
		p.skipNewlines()
	}

	p.match(lexer.TOKEN_DEDENT)
	return decl
}

// parsePolicyDeclaration parses a policy with can/cannot rules.
func (p *parser) parsePolicyDeclaration() *PolicyDeclaration {
	line := p.peek().Line
	p.advance() // consume POLICY

	name := p.advanceLiteral()
	decl := &PolicyDeclaration{Name: name, Line: line}

	if !p.match(lexer.TOKEN_COLON) {
		p.addError(fmt.Sprintf("line %d: expected ':' after policy %s", line, name))
		p.synchronize()
		return decl
	}
	p.skipNewlines()
	if !p.match(lexer.TOKEN_INDENT) {
		return decl
	}

	for !p.check(lexer.TOKEN_DEDENT) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(lexer.TOKEN_DEDENT) || p.isAtEnd() {
			break
		}

		startPos := p.pos
		ruleLine := p.peek().Line
		switch p.peek().Type {
		case lexer.TOKEN_CAN:
			p.advance()
			text := p.collectRestOfLine()
			decl.Rules = append(decl.Rules, &PolicyRule{
				Allowed: true, Text: text, Line: ruleLine,
			})
		case lexer.TOKEN_CANNOT:
			p.advance()
			text := p.collectRestOfLine()
			decl.Rules = append(decl.Rules, &PolicyRule{
				Allowed: false, Text: text, Line: ruleLine,
			})
		default:
			p.skipRestOfLine()
		}
		if p.pos == startPos {
			p.advance()
		}
		p.skipNewlines()
	}

	p.match(lexer.TOKEN_DEDENT)
	return decl
}

// parseWorkflowDeclaration parses: when <event>: <body>
// Also handles CI/CD pipelines (when code is pushed/merged).
func (p *parser) parseWorkflowDeclaration() *WorkflowDeclaration {
	line := p.peek().Line
	p.advance() // consume WHEN

	// Collect the event description up to the colon
	event := p.collectUntilColon()
	decl := &WorkflowDeclaration{Event: event, Line: line}
	decl.Statements = p.parseIndentedBody()
	return decl
}

// parseThemeDeclaration parses theme properties.
func (p *parser) parseThemeDeclaration() *ThemeDeclaration {
	line := p.peek().Line
	p.advance() // consume THEME

	decl := &ThemeDeclaration{Line: line}
	decl.Properties = p.parseIndentedBody()
	return decl
}

// parseAuthenticationDeclaration parses security/auth configuration.
func (p *parser) parseAuthenticationDeclaration() *AuthenticationDeclaration {
	line := p.peek().Line
	p.advance() // consume AUTHENTICATION

	decl := &AuthenticationDeclaration{Line: line}
	decl.Statements = p.parseIndentedBody()
	return decl
}

// parseDatabaseDeclaration parses database configuration.
func (p *parser) parseDatabaseDeclaration() *DatabaseDeclaration {
	line := p.peek().Line
	p.advance() // consume DATABASE

	decl := &DatabaseDeclaration{Line: line}
	decl.Statements = p.parseIndentedBody()
	return decl
}

// parseIntegrationDeclaration parses: integrate with <Service>: <body>
func (p *parser) parseIntegrationDeclaration() *IntegrationDeclaration {
	line := p.peek().Line
	p.advance() // consume INTEGRATE
	p.match(lexer.TOKEN_WITH)

	// Service name may be multiple words (e.g., "AWS S3")
	service := p.collectUntilColon()
	decl := &IntegrationDeclaration{Service: service, Line: line}
	decl.Statements = p.parseIndentedBody()
	return decl
}

// parseEnvironmentDeclaration parses: environment <name>: <body>
func (p *parser) parseEnvironmentDeclaration() *EnvironmentDeclaration {
	line := p.peek().Line
	p.advance() // consume ENVIRONMENT

	name := p.advanceLiteral()
	decl := &EnvironmentDeclaration{Name: name, Line: line}
	decl.Statements = p.parseIndentedBody()
	return decl
}

// parseBuildDeclaration parses build target configuration.
func (p *parser) parseBuildDeclaration() *BuildDeclaration {
	line := p.peek().Line
	p.advance() // consume BUILD

	// "build with:" — consume "with" if present
	p.match(lexer.TOKEN_WITH)

	decl := &BuildDeclaration{Line: line}
	decl.Statements = p.parseIndentedBody()
	return decl
}

// parseErrorHandler parses: if <condition>: <body>
func (p *parser) parseErrorHandler() *ErrorHandlerDeclaration {
	line := p.peek().Line
	p.advance() // consume IF

	condition := p.collectUntilColon()
	decl := &ErrorHandlerDeclaration{Condition: condition, Line: line}
	decl.Statements = p.parseIndentedBody()
	return decl
}

// parseArchitectureDeclaration parses: architecture: <style> [body]
// The style is extracted from the text after the colon on the same line.
// An optional indented body may follow with service/gateway definitions.
func (p *parser) parseArchitectureDeclaration() *ArchitectureDeclaration {
	line := p.peek().Line
	p.advance() // consume ARCHITECTURE

	// Consume colon
	p.match(lexer.TOKEN_COLON)

	// Collect the style from the rest of the line
	style := strings.TrimSpace(p.collectRestOfLine())

	decl := &ArchitectureDeclaration{Style: style, Line: line}

	// Check for an optional indented body (microservices service defs, etc.)
	p.skipNewlines()
	if p.check(lexer.TOKEN_INDENT) {
		p.advance() // consume INDENT
		for !p.isAtEnd() && !p.check(lexer.TOKEN_DEDENT) {
			if p.check(lexer.TOKEN_NEWLINE) {
				p.advance()
				continue
			}
			stmt := p.parseBodyStatement()
			if stmt != nil {
				decl.Statements = append(decl.Statements, stmt)
			}
		}
		p.match(lexer.TOKEN_DEDENT)
	}

	return decl
}

// parseTopLevelStatement parses a single-line top-level statement.
// Handles: source control, repository, track, alert, log, keep, etc.
func (p *parser) parseTopLevelStatement() *Statement {
	line := p.peek().Line

	// Special case: "repository:" with a value
	if p.check(lexer.TOKEN_REPOSITORY) {
		p.advance()
		if p.match(lexer.TOKEN_COLON) {
			val := p.collectRestOfLine()
			return &Statement{Kind: "repository", Text: "repository: " + val, Line: line}
		}
		text := "repository " + p.collectRestOfLine()
		return &Statement{Kind: "repository", Text: text, Line: line}
	}

	return p.parseBodyStatement()
}

// ── Body/statement parsing ──

// parseIndentedBody parses a colon-delimited indented block of statements.
// Expects the cursor at the COLON token. Handles nested INDENT/DEDENT pairs
// (e.g., continuation lines indented further within the block).
func (p *parser) parseIndentedBody() []*Statement {
	if !p.match(lexer.TOKEN_COLON) {
		// No colon — not a block
		p.skipRestOfLine()
		return nil
	}
	p.skipNewlines()

	if !p.match(lexer.TOKEN_INDENT) {
		return nil
	}

	var stmts []*Statement
	depth := 0

	for !p.isAtEnd() {
		// Skip newlines and comments (but NOT dedents — we track those explicitly)
		for p.check(lexer.TOKEN_NEWLINE) || p.check(lexer.TOKEN_COMMENT) {
			p.advance()
		}

		if p.isAtEnd() {
			break
		}

		// Handle nested INDENT: track depth so we don't exit early
		if p.check(lexer.TOKEN_INDENT) {
			depth++
			p.advance()
			continue
		}

		// Handle DEDENT: if nested, decrement depth; otherwise exit body
		if p.check(lexer.TOKEN_DEDENT) {
			if depth > 0 {
				depth--
				p.advance()
				continue
			}
			break // closing DEDENT for this body
		}

		startPos := p.pos
		stmt := p.parseBodyStatement()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
		if p.pos == startPos {
			p.advance()
		}
	}

	p.match(lexer.TOKEN_DEDENT)
	return stmts
}

// parseBodyStatement parses a single statement within an indented block.
func (p *parser) parseBodyStatement() *Statement {
	if p.isAtEnd() || p.check(lexer.TOKEN_DEDENT) || p.check(lexer.TOKEN_EOF) {
		return nil
	}
	line := p.peek().Line
	kind := strings.ToLower(p.peek().Literal)
	text := p.collectRestOfLine()
	if text == "" {
		return nil
	}
	return &Statement{Kind: kind, Text: text, Line: line}
}

// parseParamList parses a comma/and-separated list of parameter names.
// Used for "accepts" clauses.
func (p *parser) parseParamList() []string {
	var params []string

	param := p.collectParamName()
	if param != "" {
		params = append(params, param)
	}

	for p.check(lexer.TOKEN_COMMA) || p.check(lexer.TOKEN_AND) {
		p.advance() // consume comma or "and"
		// Handle ", and" pattern
		if p.check(lexer.TOKEN_AND) {
			p.advance()
		}
		param = p.collectParamName()
		if param != "" {
			params = append(params, param)
		}
	}

	return params
}

// collectParamName collects a parameter name which may be multiple words.
func (p *parser) collectParamName() string {
	var parts []string
	for !p.isAtEnd() &&
		!p.check(lexer.TOKEN_COMMA) &&
		!p.check(lexer.TOKEN_AND) &&
		!p.check(lexer.TOKEN_NEWLINE) &&
		!p.check(lexer.TOKEN_DEDENT) &&
		!p.check(lexer.TOKEN_EOF) {
		parts = append(parts, p.advance().Literal)
	}
	return strings.Join(parts, " ")
}

// ── Token collection helpers ──

// collectRestOfLine collects all tokens until end-of-line, joining their literals.
func (p *parser) collectRestOfLine() string {
	var parts []string
	for !p.isAtEnd() &&
		!p.check(lexer.TOKEN_NEWLINE) &&
		!p.check(lexer.TOKEN_DEDENT) &&
		!p.check(lexer.TOKEN_EOF) {
		tok := p.advance()
		// Attach possessive and comma directly to previous word
		if tok.Type == lexer.TOKEN_POSSESSIVE && len(parts) > 0 {
			parts[len(parts)-1] += tok.Literal
		} else if tok.Type == lexer.TOKEN_COMMA && len(parts) > 0 {
			parts[len(parts)-1] += tok.Literal
		} else if tok.Type == lexer.TOKEN_COLON && len(parts) > 0 {
			parts[len(parts)-1] += tok.Literal
		} else {
			parts = append(parts, tok.Literal)
		}
	}
	return strings.Join(parts, " ")
}

// collectUntilColon collects token literals until a COLON is found.
// The colon is NOT consumed — callers use parseIndentedBody which expects it.
func (p *parser) collectUntilColon() string {
	var parts []string
	for !p.isAtEnd() &&
		!p.check(lexer.TOKEN_COLON) &&
		!p.check(lexer.TOKEN_NEWLINE) &&
		!p.check(lexer.TOKEN_EOF) {
		tok := p.advance()
		if tok.Type == lexer.TOKEN_POSSESSIVE && len(parts) > 0 {
			parts[len(parts)-1] += tok.Literal
		} else if tok.Type == lexer.TOKEN_COMMA && len(parts) > 0 {
			parts[len(parts)-1] += tok.Literal
		} else {
			parts = append(parts, tok.Literal)
		}
	}
	return strings.Join(parts, " ")
}

// ── Token movement ──

func (p *parser) peek() lexer.Token {
	if p.pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.TOKEN_EOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() lexer.Token {
	tok := p.peek()
	if tok.Type != lexer.TOKEN_EOF {
		p.pos++
	}
	return tok
}

// advanceLiteral advances and returns the literal of the current token.
// Works for identifiers and keywords alike.
func (p *parser) advanceLiteral() string {
	return p.advance().Literal
}

func (p *parser) check(t lexer.TokenType) bool {
	return p.peek().Type == t
}

func (p *parser) match(t lexer.TokenType) bool {
	if p.check(t) {
		p.advance()
		return true
	}
	return false
}

func (p *parser) matchAny(types ...lexer.TokenType) bool {
	for _, t := range types {
		if p.check(t) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *parser) isAtEnd() bool {
	return p.pos >= len(p.tokens) || p.peek().Type == lexer.TOKEN_EOF
}

// isTypeKeyword returns true if the current token is a type keyword.
func (p *parser) isTypeKeyword() bool {
	switch p.peek().Type {
	case lexer.TOKEN_TEXT, lexer.TOKEN_NUMBER, lexer.TOKEN_DECIMAL,
		lexer.TOKEN_BOOLEAN, lexer.TOKEN_DATE, lexer.TOKEN_DATETIME,
		lexer.TOKEN_EMAIL, lexer.TOKEN_URL, lexer.TOKEN_FILE,
		lexer.TOKEN_IMAGE, lexer.TOKEN_JSON:
		return true
	}
	return false
}

// ── Skip helpers ──

// skipNoise skips newlines, comments, and dedents at the top level.
func (p *parser) skipNoise() {
	for !p.isAtEnd() {
		switch p.peek().Type {
		case lexer.TOKEN_NEWLINE, lexer.TOKEN_COMMENT, lexer.TOKEN_DEDENT:
			p.advance()
		default:
			return
		}
	}
}

// skipNewlines skips newline and comment tokens.
func (p *parser) skipNewlines() {
	for p.check(lexer.TOKEN_NEWLINE) || p.check(lexer.TOKEN_COMMENT) {
		p.advance()
	}
}

// skipRestOfLine consumes tokens until end-of-line without collecting them.
func (p *parser) skipRestOfLine() {
	for !p.isAtEnd() &&
		!p.check(lexer.TOKEN_NEWLINE) &&
		!p.check(lexer.TOKEN_DEDENT) &&
		!p.check(lexer.TOKEN_EOF) {
		p.advance()
	}
}

// ── Error handling ──

func (p *parser) addError(msg string) {
	p.errors = append(p.errors, msg)
}

// synchronize skips tokens until the next top-level declaration start.
func (p *parser) synchronize() {
	for !p.isAtEnd() {
		// If we hit a newline, check next token
		if p.peek().Type == lexer.TOKEN_NEWLINE {
			p.advance()
		}
		switch p.peek().Type {
		case lexer.TOKEN_APP, lexer.TOKEN_DATA, lexer.TOKEN_PAGE,
			lexer.TOKEN_COMPONENT, lexer.TOKEN_API, lexer.TOKEN_POLICY,
			lexer.TOKEN_WHEN, lexer.TOKEN_THEME, lexer.TOKEN_AUTHENTICATION,
			lexer.TOKEN_DATABASE, lexer.TOKEN_INTEGRATE, lexer.TOKEN_ENVIRONMENT,
			lexer.TOKEN_BUILD, lexer.TOKEN_IF, lexer.TOKEN_SOURCE,
			lexer.TOKEN_REPOSITORY, lexer.TOKEN_BRANCHES,
			lexer.TOKEN_SECTION_HEADER, lexer.TOKEN_EOF:
			return
		default:
			p.advance()
		}
	}
}
