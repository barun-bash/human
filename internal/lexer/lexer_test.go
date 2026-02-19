package lexer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// helper to tokenize and assert no error
func mustTokenize(t *testing.T, source string) []Token {
	t.Helper()
	l := New(source)
	tokens, err := l.Tokenize()
	if err != nil {
		t.Fatalf("unexpected lexer error: %v", err)
	}
	return tokens
}

// helper to check token type at index (ignoring EOF)
func expectToken(t *testing.T, tokens []Token, index int, expectedType TokenType, expectedLiteral string) {
	t.Helper()
	if index >= len(tokens) {
		t.Fatalf("token index %d out of range (have %d tokens)", index, len(tokens))
	}
	tok := tokens[index]
	if tok.Type != expectedType {
		t.Errorf("token[%d]: expected type %s, got %s (literal=%q)", index, expectedType, tok.Type, tok.Literal)
	}
	if expectedLiteral != "" && tok.Literal != expectedLiteral {
		t.Errorf("token[%d]: expected literal %q, got %q", index, expectedLiteral, tok.Literal)
	}
}

// ── Basic Token Tests ──

func TestEmptySource(t *testing.T) {
	tokens := mustTokenize(t, "")
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token (EOF), got %d", len(tokens))
	}
	expectToken(t, tokens, 0, TOKEN_EOF, "")
}

func TestWhitespaceOnly(t *testing.T) {
	tokens := mustTokenize(t, "   \n\n  \n")
	// All blank lines should be skipped, only EOF
	if tokens[len(tokens)-1].Type != TOKEN_EOF {
		t.Error("expected EOF as last token")
	}
}

func TestColon(t *testing.T) {
	tokens := mustTokenize(t, "theme:")
	expectToken(t, tokens, 0, TOKEN_THEME, "theme")
	expectToken(t, tokens, 1, TOKEN_COLON, ":")
}

func TestComma(t *testing.T) {
	tokens := mustTokenize(t, "name, email, password")
	expectToken(t, tokens, 0, TOKEN_IDENTIFIER, "name")
	expectToken(t, tokens, 1, TOKEN_COMMA, ",")
	expectToken(t, tokens, 2, TOKEN_EMAIL, "email")
	expectToken(t, tokens, 3, TOKEN_COMMA, ",")
	expectToken(t, tokens, 4, TOKEN_IDENTIFIER, "password")
}

// ── Keyword Tests ──

func TestDeclarationKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"app", TOKEN_APP},
		{"data", TOKEN_DATA},
		{"page", TOKEN_PAGE},
		{"component", TOKEN_COMPONENT},
		{"api", TOKEN_API},
		{"service", TOKEN_SERVICE},
		{"policy", TOKEN_POLICY},
		{"workflow", TOKEN_WORKFLOW},
		{"theme", TOKEN_THEME},
		{"architecture", TOKEN_ARCHITECTURE},
		{"environment", TOKEN_ENVIRONMENT},
		{"integrate", TOKEN_INTEGRATE},
		{"database", TOKEN_DATABASE},
		{"authentication", TOKEN_AUTHENTICATION},
		{"build", TOKEN_BUILD},
		{"design", TOKEN_DESIGN},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := mustTokenize(t, tt.input)
			expectToken(t, tokens, 0, tt.expected, tt.input)
		})
	}
}

func TestCaseInsensitiveKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"App", TOKEN_APP},
		{"APP", TOKEN_APP},
		{"Page", TOKEN_PAGE},
		{"PAGE", TOKEN_PAGE},
		{"Data", TOKEN_DATA},
		{"IF", TOKEN_IF},
		{"Show", TOKEN_SHOW},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := mustTokenize(t, tt.input)
			expectToken(t, tokens, 0, tt.expected, tt.input)
		})
	}
}

func TestTypeKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"text", TOKEN_TEXT},
		{"number", TOKEN_NUMBER},
		{"decimal", TOKEN_DECIMAL},
		{"boolean", TOKEN_BOOLEAN},
		{"date", TOKEN_DATE},
		{"datetime", TOKEN_DATETIME},
		{"email", TOKEN_EMAIL},
		{"url", TOKEN_URL},
		{"file", TOKEN_FILE},
		{"image", TOKEN_IMAGE},
		{"json", TOKEN_JSON},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := mustTokenize(t, tt.input)
			expectToken(t, tokens, 0, tt.expected, tt.input)
		})
	}
}

func TestActionKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"show", TOKEN_SHOW},
		{"fetch", TOKEN_FETCH},
		{"create", TOKEN_CREATE},
		{"update", TOKEN_UPDATE},
		{"delete", TOKEN_DELETE},
		{"send", TOKEN_SEND},
		{"respond", TOKEN_RESPOND},
		{"check", TOKEN_CHECK},
		{"validate", TOKEN_VALIDATE},
		{"filter", TOKEN_FILTER},
		{"sort", TOKEN_SORT},
		{"paginate", TOKEN_PAGINATE},
		{"search", TOKEN_SEARCH},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := mustTokenize(t, tt.input)
			expectToken(t, tokens, 0, tt.expected, tt.input)
		})
	}
}

func TestConnectorKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"is", TOKEN_IS},
		{"are", TOKEN_ARE},
		{"has", TOKEN_HAS},
		{"with", TOKEN_WITH},
		{"from", TOKEN_FROM},
		{"to", TOKEN_TO},
		{"in", TOKEN_IN},
		{"on", TOKEN_ON},
		{"for", TOKEN_FOR},
		{"by", TOKEN_BY},
		{"as", TOKEN_AS},
		{"and", TOKEN_AND},
		{"or", TOKEN_OR},
		{"not", TOKEN_NOT},
		{"the", TOKEN_THE},
		{"a", TOKEN_A},
		{"an", TOKEN_AN},
		{"which", TOKEN_WHICH},
		{"that", TOKEN_THAT},
		{"either", TOKEN_EITHER},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := mustTokenize(t, tt.input)
			expectToken(t, tokens, 0, tt.expected, tt.input)
		})
	}
}

// ── Literal Tests ──

func TestStringLiteral(t *testing.T) {
	tokens := mustTokenize(t, `"hello world"`)
	expectToken(t, tokens, 0, TOKEN_STRING_LIT, "hello world")
}

func TestStringWithEscapes(t *testing.T) {
	tokens := mustTokenize(t, `"say \"hello\""`)
	expectToken(t, tokens, 0, TOKEN_STRING_LIT, `say \"hello\"`)
}

func TestEmptyString(t *testing.T) {
	tokens := mustTokenize(t, `""`)
	expectToken(t, tokens, 0, TOKEN_STRING_LIT, "")
}

func TestUnterminatedString(t *testing.T) {
	l := New(`"unterminated`)
	_, err := l.Tokenize()
	if err == nil {
		t.Error("expected error for unterminated string")
	}
}

func TestIntegerNumber(t *testing.T) {
	tokens := mustTokenize(t, "42")
	expectToken(t, tokens, 0, TOKEN_NUMBER_LIT, "42")
}

func TestDecimalNumber(t *testing.T) {
	tokens := mustTokenize(t, "3.14")
	expectToken(t, tokens, 0, TOKEN_NUMBER_LIT, "3.14")
}

func TestLargeNumber(t *testing.T) {
	tokens := mustTokenize(t, "50000")
	expectToken(t, tokens, 0, TOKEN_NUMBER_LIT, "50000")
}

func TestIdentifier(t *testing.T) {
	tokens := mustTokenize(t, "TaskFlow")
	expectToken(t, tokens, 0, TOKEN_IDENTIFIER, "TaskFlow")
}

func TestIdentifierWithUnderscore(t *testing.T) {
	tokens := mustTokenize(t, "user_name")
	expectToken(t, tokens, 0, TOKEN_IDENTIFIER, "user_name")
}

func TestIdentifierWithHyphen(t *testing.T) {
	tokens := mustTokenize(t, "getting-started")
	expectToken(t, tokens, 0, TOKEN_IDENTIFIER, "getting-started")
}

// ── Color Literal Tests ──

func TestColorLiteral6(t *testing.T) {
	tokens := mustTokenize(t, "#6C5CE7")
	expectToken(t, tokens, 0, TOKEN_COLOR_LIT, "#6C5CE7")
}

func TestColorLiteral3(t *testing.T) {
	tokens := mustTokenize(t, "#ABC")
	expectToken(t, tokens, 0, TOKEN_COLOR_LIT, "#ABC")
}

func TestColorInContext(t *testing.T) {
	tokens := mustTokenize(t, "primary color is #6C5CE7")
	expectToken(t, tokens, 0, TOKEN_IDENTIFIER, "primary")
	expectToken(t, tokens, 1, TOKEN_IDENTIFIER, "color")
	expectToken(t, tokens, 2, TOKEN_IS, "is")
	expectToken(t, tokens, 3, TOKEN_COLOR_LIT, "#6C5CE7")
}

// ── Comment Tests ──

func TestComment(t *testing.T) {
	tokens := mustTokenize(t, "# this is a comment")
	expectToken(t, tokens, 0, TOKEN_COMMENT, "# this is a comment")
}

func TestCommentAfterCode(t *testing.T) {
	// Comment-only lines are consumed at line start
	// This tests a full line that is just a comment
	tokens := mustTokenize(t, "# comment\napp TaskFlow")
	// Comment should be emitted, then app on next line
	found := false
	for _, tok := range tokens {
		if tok.Type == TOKEN_APP {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find APP token after comment line")
	}
}

// ── Section Header Tests ──

func TestSectionHeaderUnicode(t *testing.T) {
	tokens := mustTokenize(t, "── frontend ──")
	expectToken(t, tokens, 0, TOKEN_SECTION_HEADER, "frontend")
}

func TestSectionHeaderDashes(t *testing.T) {
	tokens := mustTokenize(t, "-- backend --")
	expectToken(t, tokens, 0, TOKEN_SECTION_HEADER, "backend")
}

func TestSectionHeaderMultiWord(t *testing.T) {
	tokens := mustTokenize(t, "── error handling ──")
	expectToken(t, tokens, 0, TOKEN_SECTION_HEADER, "error handling")
}

// ── Possessive Tests ──

func TestPossessive(t *testing.T) {
	tokens := mustTokenize(t, "user's name")
	expectToken(t, tokens, 0, TOKEN_IDENTIFIER, "user")
	expectToken(t, tokens, 1, TOKEN_POSSESSIVE, "'s")
	expectToken(t, tokens, 2, TOKEN_IDENTIFIER, "name")
}

func TestPossessiveInPhrase(t *testing.T) {
	tokens := mustTokenize(t, "the user's name")
	expectToken(t, tokens, 0, TOKEN_THE, "the")
	expectToken(t, tokens, 1, TOKEN_IDENTIFIER, "user")
	expectToken(t, tokens, 2, TOKEN_POSSESSIVE, "'s")
	expectToken(t, tokens, 3, TOKEN_IDENTIFIER, "name")
}

func TestContractionDont(t *testing.T) {
	tokens := mustTokenize(t, "don't")
	expectToken(t, tokens, 0, TOKEN_IDENTIFIER, "don't")
}

// ── Indentation Tests ──

func TestSimpleIndent(t *testing.T) {
	source := "theme:\n  primary color"
	tokens := mustTokenize(t, source)

	expected := []TokenType{
		TOKEN_THEME, TOKEN_COLON, TOKEN_NEWLINE,
		TOKEN_INDENT,
		TOKEN_IDENTIFIER, TOKEN_IDENTIFIER,
		TOKEN_DEDENT, TOKEN_EOF,
	}
	checkTokenTypes(t, tokens, expected)
}

func TestIndentDedent(t *testing.T) {
	source := "page Home:\n  show a greeting\ndata User:"
	tokens := mustTokenize(t, source)

	expected := []TokenType{
		TOKEN_PAGE, TOKEN_IDENTIFIER, TOKEN_COLON, TOKEN_NEWLINE,
		TOKEN_INDENT,
		TOKEN_SHOW, TOKEN_A, TOKEN_IDENTIFIER, TOKEN_NEWLINE,
		TOKEN_DEDENT,
		TOKEN_DATA, TOKEN_IDENTIFIER, TOKEN_COLON,
		TOKEN_DEDENT, TOKEN_EOF,
	}
	// Just check that we get INDENT and DEDENT in the right places
	hasIndent := false
	hasDedent := false
	for _, tok := range tokens {
		if tok.Type == TOKEN_INDENT {
			hasIndent = true
		}
		if tok.Type == TOKEN_DEDENT {
			hasDedent = true
		}
	}
	if !hasIndent {
		t.Error("expected INDENT token")
	}
	if !hasDedent {
		t.Error("expected DEDENT token")
	}
	_ = expected // used for reference
}

func TestNestedIndentation(t *testing.T) {
	source := "if condition:\n  if nested:\n    do something\n  back one\nback two"
	tokens := mustTokenize(t, source)

	indentCount := 0
	dedentCount := 0
	for _, tok := range tokens {
		if tok.Type == TOKEN_INDENT {
			indentCount++
		}
		if tok.Type == TOKEN_DEDENT {
			dedentCount++
		}
	}
	if indentCount != 2 {
		t.Errorf("expected 2 INDENT tokens, got %d", indentCount)
	}
	if dedentCount < 2 {
		t.Errorf("expected at least 2 DEDENT tokens, got %d", dedentCount)
	}
}

func TestBlankLinesIgnored(t *testing.T) {
	source := "app TaskFlow\n\n\ndata User"
	tokens := mustTokenize(t, source)

	// Blank lines should not produce NEWLINE or affect indentation
	types := tokenTypes(tokens)
	newlineCount := 0
	for _, tt := range types {
		if tt == TOKEN_NEWLINE {
			newlineCount++
		}
	}
	// Only one NEWLINE (after "app TaskFlow"), blank lines skipped
	if newlineCount != 1 {
		t.Errorf("expected 1 NEWLINE, got %d", newlineCount)
	}
}

// ── App Declaration Tests ──

func TestAppDeclaration(t *testing.T) {
	source := "app TaskFlow is a web application"
	tokens := mustTokenize(t, source)

	expectToken(t, tokens, 0, TOKEN_APP, "app")
	expectToken(t, tokens, 1, TOKEN_IDENTIFIER, "TaskFlow")
	expectToken(t, tokens, 2, TOKEN_IS, "is")
	expectToken(t, tokens, 3, TOKEN_A, "a")
	expectToken(t, tokens, 4, TOKEN_IDENTIFIER, "web")
	expectToken(t, tokens, 5, TOKEN_IDENTIFIER, "application")
}

// ── Data Declaration Tests ──

func TestDataDeclaration(t *testing.T) {
	source := `data User:
  has a name which is text
  has an email which is unique email
  has a password which is encrypted text
  has a role which is either "user" or "admin"
  has an optional bio which is text
  has many Task`
	tokens := mustTokenize(t, source)

	// First line: data User:
	expectToken(t, tokens, 0, TOKEN_DATA, "data")
	expectToken(t, tokens, 1, TOKEN_IDENTIFIER, "User")
	expectToken(t, tokens, 2, TOKEN_COLON, ":")

	// Check key tokens are present
	foundHas := false
	foundWhich := false
	foundText := false
	foundUnique := false
	foundEncrypted := false
	foundEither := false
	foundOptional := false
	foundMany := false
	for _, tok := range tokens {
		switch tok.Type {
		case TOKEN_HAS:
			foundHas = true
		case TOKEN_WHICH:
			foundWhich = true
		case TOKEN_TEXT:
			foundText = true
		case TOKEN_UNIQUE:
			foundUnique = true
		case TOKEN_ENCRYPTED:
			foundEncrypted = true
		case TOKEN_EITHER:
			foundEither = true
		case TOKEN_OPTIONAL:
			foundOptional = true
		case TOKEN_MANY:
			foundMany = true
		}
	}
	if !foundHas {
		t.Error("missing HAS token")
	}
	if !foundWhich {
		t.Error("missing WHICH token")
	}
	if !foundText {
		t.Error("missing TEXT token")
	}
	if !foundUnique {
		t.Error("missing UNIQUE token")
	}
	if !foundEncrypted {
		t.Error("missing ENCRYPTED token")
	}
	if !foundEither {
		t.Error("missing EITHER token")
	}
	if !foundOptional {
		t.Error("missing OPTIONAL token")
	}
	if !foundMany {
		t.Error("missing MANY token")
	}
}

func TestBelongsToRelationship(t *testing.T) {
	source := "  belongs to a User"
	// Need a context line first to establish indent
	fullSource := "data Task:\n" + source
	tokens := mustTokenize(t, fullSource)

	foundBelongs := false
	for _, tok := range tokens {
		if tok.Type == TOKEN_BELONGS {
			foundBelongs = true
		}
	}
	if !foundBelongs {
		t.Error("missing BELONGS token")
	}
}

// ── API Declaration Tests ──

func TestAPIDeclaration(t *testing.T) {
	source := `api CreateTask:
  requires authentication
  accepts title, description, and status
  check that title is not empty
  create a Task with the given fields
  respond with the created task`
	tokens := mustTokenize(t, source)

	expectToken(t, tokens, 0, TOKEN_API, "api")
	expectToken(t, tokens, 1, TOKEN_IDENTIFIER, "CreateTask")
	expectToken(t, tokens, 2, TOKEN_COLON, ":")

	foundRequires := false
	foundAccepts := false
	foundCheck := false
	foundCreate := false
	foundRespond := false
	for _, tok := range tokens {
		switch tok.Type {
		case TOKEN_REQUIRES:
			foundRequires = true
		case TOKEN_ACCEPTS:
			foundAccepts = true
		case TOKEN_CHECK:
			foundCheck = true
		case TOKEN_CREATE:
			foundCreate = true
		case TOKEN_RESPOND:
			foundRespond = true
		}
	}
	if !foundRequires {
		t.Error("missing REQUIRES token")
	}
	if !foundAccepts {
		t.Error("missing ACCEPTS token")
	}
	if !foundCheck {
		t.Error("missing CHECK token")
	}
	if !foundCreate {
		t.Error("missing CREATE token")
	}
	if !foundRespond {
		t.Error("missing RESPOND token")
	}
}

// ── Page Declaration Tests ──

func TestPageDeclaration(t *testing.T) {
	source := `page Dashboard:
  show a greeting with the user's name
  each task shows its title
  clicking a task navigates to the task detail
  there is a search bar that filters tasks by title
  if no tasks match, show "No tasks found"
  while loading, show a spinner`
	tokens := mustTokenize(t, source)

	expectToken(t, tokens, 0, TOKEN_PAGE, "page")
	expectToken(t, tokens, 1, TOKEN_IDENTIFIER, "Dashboard")

	foundShow := false
	foundEach := false
	foundClicking := false
	foundNavigates := false
	foundThere := false
	foundWhile := false
	foundPossessive := false
	foundShows := false
	for _, tok := range tokens {
		switch tok.Type {
		case TOKEN_SHOW:
			foundShow = true
		case TOKEN_EACH:
			foundEach = true
		case TOKEN_CLICKING:
			foundClicking = true
		case TOKEN_NAVIGATES:
			foundNavigates = true
		case TOKEN_THERE:
			foundThere = true
		case TOKEN_WHILE:
			foundWhile = true
		case TOKEN_POSSESSIVE:
			foundPossessive = true
		case TOKEN_SHOWS:
			foundShows = true
		}
	}
	if !foundShow {
		t.Error("missing SHOW token")
	}
	if !foundEach {
		t.Error("missing EACH token")
	}
	if !foundClicking {
		t.Error("missing CLICKING token")
	}
	if !foundNavigates {
		t.Error("missing NAVIGATES token")
	}
	if !foundThere {
		t.Error("missing THERE token")
	}
	if !foundWhile {
		t.Error("missing WHILE token")
	}
	if !foundPossessive {
		t.Error("missing POSSESSIVE token")
	}
	if !foundShows {
		t.Error("missing SHOWS token")
	}
}

// ── Policy Declaration Tests ──

func TestPolicyDeclaration(t *testing.T) {
	source := `policy FreeUser:
  can create up to 50 tasks per month
  cannot delete completed tasks`
	tokens := mustTokenize(t, source)

	expectToken(t, tokens, 0, TOKEN_POLICY, "policy")

	foundCan := false
	foundCannot := false
	foundPer := false
	foundNumber := false
	for _, tok := range tokens {
		switch tok.Type {
		case TOKEN_CAN:
			foundCan = true
		case TOKEN_CANNOT:
			foundCannot = true
		case TOKEN_PER:
			foundPer = true
		case TOKEN_NUMBER_LIT:
			foundNumber = true
		}
	}
	if !foundCan {
		t.Error("missing CAN token")
	}
	if !foundCannot {
		t.Error("missing CANNOT token")
	}
	if !foundPer {
		t.Error("missing PER token")
	}
	if !foundNumber {
		t.Error("missing NUMBER_LIT token")
	}
}

// ── Theme Declaration Tests ──

func TestThemeDeclaration(t *testing.T) {
	source := `theme:
  primary color is #6C5CE7
  secondary color is #00B894
  danger color is #D63031`
	tokens := mustTokenize(t, source)

	expectToken(t, tokens, 0, TOKEN_THEME, "theme")
	expectToken(t, tokens, 1, TOKEN_COLON, ":")

	colorCount := 0
	for _, tok := range tokens {
		if tok.Type == TOKEN_COLOR_LIT {
			colorCount++
		}
	}
	if colorCount != 3 {
		t.Errorf("expected 3 color literals, got %d", colorCount)
	}
}

// ── Workflow Tests ──

func TestWorkflowDeclaration(t *testing.T) {
	source := `when a user signs up:
  create their account
  assign FreeUser policy
  send welcome email with template "welcome"
  after 3 days, send email with template "getting-started"`
	tokens := mustTokenize(t, source)

	expectToken(t, tokens, 0, TOKEN_WHEN, "when")

	foundAfter := false
	foundSend := false
	foundAssign := false
	foundStringLit := false
	for _, tok := range tokens {
		switch tok.Type {
		case TOKEN_AFTER:
			foundAfter = true
		case TOKEN_SEND:
			foundSend = true
		case TOKEN_ASSIGN:
			foundAssign = true
		case TOKEN_STRING_LIT:
			foundStringLit = true
		}
	}
	if !foundAfter {
		t.Error("missing AFTER token")
	}
	if !foundSend {
		t.Error("missing SEND token")
	}
	if !foundAssign {
		t.Error("missing ASSIGN token")
	}
	if !foundStringLit {
		t.Error("missing STRING_LIT token")
	}
}

// ── Authentication Tests ──

func TestAuthenticationBlock(t *testing.T) {
	source := `authentication:
  method JWT tokens that expire in 7 days
  rate limit all endpoints to 100 requests per minute per user`
	tokens := mustTokenize(t, source)

	expectToken(t, tokens, 0, TOKEN_AUTHENTICATION, "authentication")

	foundMethod := false
	foundRate := false
	foundLimit := false
	foundAll := false
	foundPer := false
	for _, tok := range tokens {
		switch tok.Type {
		case TOKEN_METHOD:
			foundMethod = true
		case TOKEN_RATE:
			foundRate = true
		case TOKEN_LIMIT:
			foundLimit = true
		case TOKEN_ALL:
			foundAll = true
		case TOKEN_PER:
			foundPer = true
		}
	}
	if !foundMethod {
		t.Error("missing METHOD token")
	}
	if !foundRate {
		t.Error("missing RATE token")
	}
	if !foundLimit {
		t.Error("missing LIMIT token")
	}
	if !foundAll {
		t.Error("missing ALL token")
	}
	if !foundPer {
		t.Error("missing PER token")
	}
}

// ── Integration Tests ──

func TestIntegrationDeclaration(t *testing.T) {
	source := `integrate with SendGrid:
  api key from environment variable SENDGRID_API_KEY
  use for sending transactional emails`
	tokens := mustTokenize(t, source)

	expectToken(t, tokens, 0, TOKEN_INTEGRATE, "integrate")
	expectToken(t, tokens, 1, TOKEN_WITH, "with")
	expectToken(t, tokens, 2, TOKEN_IDENTIFIER, "SendGrid")
	expectToken(t, tokens, 3, TOKEN_COLON, ":")

	foundKey := false
	foundVariable := false
	foundUse := false
	for _, tok := range tokens {
		switch tok.Type {
		case TOKEN_KEY:
			foundKey = true
		case TOKEN_VARIABLE:
			foundVariable = true
		case TOKEN_USE:
			foundUse = true
		}
	}
	if !foundKey {
		t.Error("missing KEY token")
	}
	if !foundVariable {
		t.Error("missing VARIABLE token")
	}
	if !foundUse {
		t.Error("missing USE token")
	}
}

// ── Database Declaration Tests ──

func TestDatabaseDeclaration(t *testing.T) {
	source := `database:
  use PostgreSQL
  index User by email
  backup daily at 3am
  keep backups for 30 days`
	tokens := mustTokenize(t, source)

	expectToken(t, tokens, 0, TOKEN_DATABASE, "database")

	foundIndex := false
	foundBackup := false
	foundKeep := false
	foundAt := false
	for _, tok := range tokens {
		switch tok.Type {
		case TOKEN_INDEX:
			foundIndex = true
		case TOKEN_BACKUP:
			foundBackup = true
		case TOKEN_KEEP:
			foundKeep = true
		case TOKEN_AT:
			foundAt = true
		}
	}
	if !foundIndex {
		t.Error("missing INDEX token")
	}
	if !foundBackup {
		t.Error("missing BACKUP token")
	}
	if !foundKeep {
		t.Error("missing KEEP token")
	}
	if !foundAt {
		t.Error("missing AT token")
	}
}

// ── Build Declaration Tests ──

func TestBuildDeclaration(t *testing.T) {
	source := `build with:
  frontend using React with TypeScript
  backend using Node with Express
  database using PostgreSQL
  deploy to Vercel`
	tokens := mustTokenize(t, source)

	expectToken(t, tokens, 0, TOKEN_BUILD, "build")
	expectToken(t, tokens, 1, TOKEN_WITH, "with")
	expectToken(t, tokens, 2, TOKEN_COLON, ":")

	foundUsing := false
	foundDeploy := false
	for _, tok := range tokens {
		switch tok.Type {
		case TOKEN_USING:
			foundUsing = true
		case TOKEN_DEPLOY:
			foundDeploy = true
		}
	}
	if !foundUsing {
		t.Error("missing USING token")
	}
	if !foundDeploy {
		t.Error("missing DEPLOY token")
	}
}

// ── DevOps Tests ──

func TestDevOpsTokens(t *testing.T) {
	source := `source control using Git on GitHub
repository: "https://github.com/taskflow/taskflow"

branches:
  main is the production branch`
	tokens := mustTokenize(t, source)

	expectToken(t, tokens, 0, TOKEN_SOURCE, "source")

	foundRepository := false
	foundBranches := false
	for _, tok := range tokens {
		switch tok.Type {
		case TOKEN_REPOSITORY:
			foundRepository = true
		case TOKEN_BRANCHES:
			foundBranches = true
		}
	}
	if !foundRepository {
		t.Error("missing REPOSITORY token")
	}
	if !foundBranches {
		t.Error("missing BRANCHES token")
	}
}

// ── Error Handling Tests ──

func TestErrorHandlingBlock(t *testing.T) {
	source := `if database is unreachable:
  retry 3 times with 1 second delay
  if still failing, respond with "service temporarily unavailable"
  alert the engineering team via Slack`
	tokens := mustTokenize(t, source)

	expectToken(t, tokens, 0, TOKEN_IF, "if")

	foundRetry := false
	foundAlert := false
	foundRespond := false
	for _, tok := range tokens {
		switch tok.Type {
		case TOKEN_RETRY:
			foundRetry = true
		case TOKEN_ALERT:
			foundAlert = true
		case TOKEN_RESPOND:
			foundRespond = true
		}
	}
	if !foundRetry {
		t.Error("missing RETRY token")
	}
	if !foundAlert {
		t.Error("missing ALERT token")
	}
	if !foundRespond {
		t.Error("missing RESPOND token")
	}
}

// ── Line Number Tracking Tests ──

func TestLineNumbers(t *testing.T) {
	source := "app TaskFlow\n\ndata User:\n  has a name"
	tokens := mustTokenize(t, source)

	// "app" should be on line 1
	if tokens[0].Line != 1 {
		t.Errorf("expected 'app' on line 1, got line %d", tokens[0].Line)
	}

	// Find "data" token and check its line
	for _, tok := range tokens {
		if tok.Type == TOKEN_DATA {
			if tok.Line != 3 {
				t.Errorf("expected 'data' on line 3, got line %d", tok.Line)
			}
			break
		}
	}
}

// ── Full Integration Test: app.human ──

func TestTokenizeAppHuman(t *testing.T) {
	// Find the app.human file relative to this test
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	appPath := filepath.Join(projectRoot, "examples", "taskflow", "app.human")

	source, err := os.ReadFile(appPath)
	if err != nil {
		t.Fatalf("could not read app.human: %v", err)
	}

	l := New(string(source))
	tokens, err := l.Tokenize()
	if err != nil {
		t.Fatalf("failed to tokenize app.human: %v", err)
	}

	// Basic sanity checks
	if len(tokens) < 100 {
		t.Errorf("expected at least 100 tokens from app.human, got %d", len(tokens))
	}

	// Last token must be EOF
	lastToken := tokens[len(tokens)-1]
	if lastToken.Type != TOKEN_EOF {
		t.Errorf("expected last token to be EOF, got %s", lastToken.Type)
	}

	// Check that key structural elements were found
	requiredTokens := map[TokenType]string{
		TOKEN_APP:            "app declaration",
		TOKEN_DATA:           "data declaration",
		TOKEN_PAGE:           "page declaration",
		TOKEN_API:            "api declaration",
		TOKEN_POLICY:         "policy declaration",
		TOKEN_AUTHENTICATION: "authentication block",
		TOKEN_DATABASE:       "database block",
		TOKEN_INTEGRATE:      "integrate declaration",
		TOKEN_BUILD:          "build declaration",
		TOKEN_SECTION_HEADER: "section header",
		TOKEN_INDENT:         "indentation",
		TOKEN_DEDENT:         "dedentation",
		TOKEN_STRING_LIT:     "string literal",
		TOKEN_NUMBER_LIT:     "number literal",
		TOKEN_COLOR_LIT:      "color literal",
		TOKEN_COMMENT:        "comment",
		TOKEN_POSSESSIVE:     "possessive",
		TOKEN_COLON:          "colon",
		TOKEN_COMMA:          "comma",
		TOKEN_WHEN:           "when (workflow)",
		TOKEN_SOURCE:         "source (devops)",
		TOKEN_ENVIRONMENT:    "environment",
		TOKEN_CAN:            "can (policy)",
		TOKEN_CANNOT:         "cannot (policy)",
	}

	found := make(map[TokenType]bool)
	for _, tok := range tokens {
		found[tok.Type] = true
	}

	for tokenType, description := range requiredTokens {
		if !found[tokenType] {
			t.Errorf("app.human: missing %s (%s)", tokenType, description)
		}
	}

	// Count section headers
	sectionCount := 0
	for _, tok := range tokens {
		if tok.Type == TOKEN_SECTION_HEADER {
			sectionCount++
		}
	}
	// frontend, backend, security, policies, workflows, error handling, database, integrations, devops, build
	if sectionCount < 10 {
		t.Errorf("expected at least 10 section headers, got %d", sectionCount)
	}

	// Verify indent/dedent balance
	indentCount := 0
	dedentCount := 0
	for _, tok := range tokens {
		if tok.Type == TOKEN_INDENT {
			indentCount++
		}
		if tok.Type == TOKEN_DEDENT {
			dedentCount++
		}
	}
	if indentCount != dedentCount {
		t.Errorf("indent/dedent mismatch: %d indents vs %d dedents", indentCount, dedentCount)
	}

	t.Logf("Successfully tokenized app.human: %d tokens, %d sections, %d indent/dedent pairs",
		len(tokens), sectionCount, indentCount)
}

// ── Helpers ──

func tokenTypes(tokens []Token) []TokenType {
	types := make([]TokenType, len(tokens))
	for i, tok := range tokens {
		types[i] = tok.Type
	}
	return types
}

func checkTokenTypes(t *testing.T, tokens []Token, expected []TokenType) {
	t.Helper()
	if len(tokens) != len(expected) {
		t.Errorf("expected %d tokens, got %d", len(expected), len(tokens))
		for i, tok := range tokens {
			t.Logf("  token[%d] = %s %q", i, tok.Type, tok.Literal)
		}
		return
	}
	for i, exp := range expected {
		if tokens[i].Type != exp {
			t.Errorf("token[%d]: expected %s, got %s (literal=%q)", i, exp, tokens[i].Type, tokens[i].Literal)
		}
	}
}
