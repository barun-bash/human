package lexer

import (
	"fmt"
	"strings"
)

// TokenType represents the type of a lexical token.
type TokenType int

const (
	// Structural tokens
	TOKEN_EOF            TokenType = iota
	TOKEN_NEWLINE                  // end of a logical line
	TOKEN_INDENT                   // increase in indentation level
	TOKEN_DEDENT                   // decrease in indentation level
	TOKEN_COLON                    // :
	TOKEN_COMMA                    // ,
	TOKEN_SECTION_HEADER           // ── name ──
	TOKEN_COMMENT                  // # comment text

	// Literal tokens
	TOKEN_STRING_LIT  // "hello world"
	TOKEN_NUMBER_LIT  // 42, 3.14, 500
	TOKEN_COLOR_LIT   // #6C5CE7, #ABC
	TOKEN_IDENTIFIER  // user_name, Dashboard, etc.
	TOKEN_POSSESSIVE  // 's (as in user's)

	// ── Declaration Keywords ──

	TOKEN_APP            // app
	TOKEN_DATA           // data
	TOKEN_PAGE           // page
	TOKEN_COMPONENT      // component
	TOKEN_API            // api
	TOKEN_SERVICE        // service
	TOKEN_AGENT          // agent
	TOKEN_POLICY         // policy
	TOKEN_WORKFLOW       // workflow
	TOKEN_THEME          // theme
	TOKEN_ARCHITECTURE   // architecture
	TOKEN_ENVIRONMENT    // environment
	TOKEN_INTEGRATE      // integrate
	TOKEN_DATABASE       // database
	TOKEN_AUTHENTICATION // authentication
	TOKEN_BUILD          // build
	TOKEN_DESIGN         // design

	// ── Type Keywords ──

	TOKEN_TEXT     // text
	TOKEN_NUMBER   // number (the type keyword, not a literal)
	TOKEN_DECIMAL  // decimal
	TOKEN_BOOLEAN  // boolean
	TOKEN_DATE     // date
	TOKEN_DATETIME // datetime
	TOKEN_EMAIL    // email
	TOKEN_URL      // url
	TOKEN_FILE     // file
	TOKEN_IMAGE    // image
	TOKEN_JSON     // json

	// ── Action Keywords ──

	TOKEN_SHOW     // show
	TOKEN_FETCH    // fetch
	TOKEN_CREATE   // create
	TOKEN_UPDATE   // update
	TOKEN_DELETE   // delete
	TOKEN_SEND     // send
	TOKEN_RESPOND  // respond
	TOKEN_NAVIGATE // navigate
	TOKEN_CHECK    // check
	TOKEN_VALIDATE // validate
	TOKEN_FILTER   // filter
	TOKEN_SORT     // sort
	TOKEN_PAGINATE // paginate
	TOKEN_SEARCH   // search
	TOKEN_SET      // set
	TOKEN_RETURN   // return
	TOKEN_PUBLISH  // publish
	TOKEN_LISTEN   // listen
	TOKEN_NOTIFY   // notify
	TOKEN_ALERT    // alert
	TOKEN_LOG      // log
	TOKEN_TRACK    // track
	TOKEN_RUN      // run
	TOKEN_DEPLOY   // deploy
	TOKEN_KEEP     // keep
	TOKEN_BACKUP   // backup
	TOKEN_RETRY    // retry
	TOKEN_ROLLBACK // rollback
	TOKEN_INDEX    // index
	TOKEN_ENABLE   // enable
	TOKEN_SUPPORT  // support
	TOKEN_ASSIGN   // assign
	TOKEN_USE      // use

	// ── Condition Keywords ──

	TOKEN_IF     // if
	TOKEN_WHEN   // when
	TOKEN_WHILE  // while
	TOKEN_UNLESS // unless
	TOKEN_UNTIL  // until
	TOKEN_AFTER  // after
	TOKEN_BEFORE // before
	TOKEN_EVERY  // every

	// ── Connector Keywords ──

	TOKEN_IS    // is
	TOKEN_ARE   // are
	TOKEN_HAS   // has
	TOKEN_WITH  // with
	TOKEN_FROM  // from
	TOKEN_TO    // to
	TOKEN_IN    // in
	TOKEN_ON    // on
	TOKEN_FOR   // for
	TOKEN_BY    // by
	TOKEN_AS    // as
	TOKEN_AND   // and
	TOKEN_OR    // or
	TOKEN_NOT   // not
	TOKEN_THE   // the
	TOKEN_A     // a
	TOKEN_AN    // an
	TOKEN_WHICH // which
	TOKEN_THAT  // that
	TOKEN_EITHER // either
	TOKEN_OF    // of
	TOKEN_ITS   // its
	TOKEN_THEIR // their
	TOKEN_USING // using
	TOKEN_PER   // per
	TOKEN_AT    // at

	// ── Modifier Keywords ──

	TOKEN_REQUIRES  // requires
	TOKEN_ACCEPTS   // accepts
	TOKEN_ONLY      // only
	TOKEN_EACH      // each
	TOKEN_ALL       // all
	TOKEN_OPTIONAL  // optional
	TOKEN_UNIQUE    // unique
	TOKEN_ENCRYPTED // encrypted

	// ── Relationship Keywords ──

	TOKEN_BELONGS  // belongs
	TOKEN_MANY     // many
	TOKEN_THROUGH  // through
	TOKEN_DEFAULTS // defaults

	// ── Interaction Subject Keywords ──

	TOKEN_CLICKING  // clicking
	TOKEN_TYPING    // typing
	TOKEN_HOVERING  // hovering
	TOKEN_PRESSING  // pressing
	TOKEN_SCROLLING // scrolling
	TOKEN_DRAGGING  // dragging

	// ── Interaction Verb Keywords ──

	TOKEN_DOES      // does
	TOKEN_NAVIGATES // navigates
	TOKEN_OPENS     // opens
	TOKEN_TRIGGERS  // triggers
	TOKEN_SHOWS     // shows
	TOKEN_LOADS     // loads
	TOKEN_REORDERS  // reorders

	// ── Policy Keywords ──

	TOKEN_CAN    // can
	TOKEN_CANNOT // cannot
	TOKEN_MUST   // must

	// ── Architecture Keywords ──

	TOKEN_MONOLITH      // monolith
	TOKEN_MICROSERVICES // microservices
	TOKEN_SERVERLESS    // serverless
	TOKEN_GATEWAY       // gateway
	TOKEN_BROKER        // broker

	// ── DevOps Keywords ──

	TOKEN_PIPELINE   // pipeline
	TOKEN_MONITOR    // monitor
	TOKEN_RELEASE    // release
	TOKEN_MERGE      // merge
	TOKEN_PUSH       // push
	TOKEN_SOURCE     // source
	TOKEN_REPOSITORY // repository
	TOKEN_BRANCHES   // branches

	// ── Other Keywords ──

	TOKEN_THERE    // there
	TOKEN_NO       // no
	TOKEN_DO       // do
	TOKEN_METHOD   // method
	TOKEN_ENDPOINT // endpoint
	TOKEN_EXCEPT   // except
	TOKEN_RATE     // rate
	TOKEN_LIMIT    // limit
	TOKEN_USES     // uses
	TOKEN_SANITIZE // sanitize
	TOKEN_VARIABLE // variable
	TOKEN_KEY      // key
)

// tokenNames maps token types to their display names.
var tokenNames = map[TokenType]string{
	// Structural
	TOKEN_EOF:            "EOF",
	TOKEN_NEWLINE:        "NEWLINE",
	TOKEN_INDENT:         "INDENT",
	TOKEN_DEDENT:         "DEDENT",
	TOKEN_COLON:          "COLON",
	TOKEN_COMMA:          "COMMA",
	TOKEN_SECTION_HEADER: "SECTION_HEADER",
	TOKEN_COMMENT:        "COMMENT",

	// Literals
	TOKEN_STRING_LIT: "STRING",
	TOKEN_NUMBER_LIT: "NUMBER",
	TOKEN_COLOR_LIT:  "COLOR",
	TOKEN_IDENTIFIER: "IDENTIFIER",
	TOKEN_POSSESSIVE: "POSSESSIVE",

	// Declarations
	TOKEN_APP:            "app",
	TOKEN_DATA:           "data",
	TOKEN_PAGE:           "page",
	TOKEN_COMPONENT:      "component",
	TOKEN_API:            "api",
	TOKEN_SERVICE:        "service",
	TOKEN_AGENT:          "agent",
	TOKEN_POLICY:         "policy",
	TOKEN_WORKFLOW:       "workflow",
	TOKEN_THEME:          "theme",
	TOKEN_ARCHITECTURE:   "architecture",
	TOKEN_ENVIRONMENT:    "environment",
	TOKEN_INTEGRATE:      "integrate",
	TOKEN_DATABASE:       "database",
	TOKEN_AUTHENTICATION: "authentication",
	TOKEN_BUILD:          "build",
	TOKEN_DESIGN:         "design",

	// Types
	TOKEN_TEXT:     "text",
	TOKEN_NUMBER:   "number",
	TOKEN_DECIMAL:  "decimal",
	TOKEN_BOOLEAN:  "boolean",
	TOKEN_DATE:     "date",
	TOKEN_DATETIME: "datetime",
	TOKEN_EMAIL:    "email",
	TOKEN_URL:      "url",
	TOKEN_FILE:     "file",
	TOKEN_IMAGE:    "image",
	TOKEN_JSON:     "json",

	// Actions
	TOKEN_SHOW:     "show",
	TOKEN_FETCH:    "fetch",
	TOKEN_CREATE:   "create",
	TOKEN_UPDATE:   "update",
	TOKEN_DELETE:   "delete",
	TOKEN_SEND:     "send",
	TOKEN_RESPOND:  "respond",
	TOKEN_NAVIGATE: "navigate",
	TOKEN_CHECK:    "check",
	TOKEN_VALIDATE: "validate",
	TOKEN_FILTER:   "filter",
	TOKEN_SORT:     "sort",
	TOKEN_PAGINATE: "paginate",
	TOKEN_SEARCH:   "search",
	TOKEN_SET:      "set",
	TOKEN_RETURN:   "return",
	TOKEN_PUBLISH:  "publish",
	TOKEN_LISTEN:   "listen",
	TOKEN_NOTIFY:   "notify",
	TOKEN_ALERT:    "alert",
	TOKEN_LOG:      "log",
	TOKEN_TRACK:    "track",
	TOKEN_RUN:      "run",
	TOKEN_DEPLOY:   "deploy",
	TOKEN_KEEP:     "keep",
	TOKEN_BACKUP:   "backup",
	TOKEN_RETRY:    "retry",
	TOKEN_ROLLBACK: "rollback",
	TOKEN_INDEX:    "index",
	TOKEN_ENABLE:   "enable",
	TOKEN_SUPPORT:  "support",
	TOKEN_ASSIGN:   "assign",
	TOKEN_USE:      "use",

	// Conditions
	TOKEN_IF:     "if",
	TOKEN_WHEN:   "when",
	TOKEN_WHILE:  "while",
	TOKEN_UNLESS: "unless",
	TOKEN_UNTIL:  "until",
	TOKEN_AFTER:  "after",
	TOKEN_BEFORE: "before",
	TOKEN_EVERY:  "every",

	// Connectors
	TOKEN_IS:    "is",
	TOKEN_ARE:   "are",
	TOKEN_HAS:   "has",
	TOKEN_WITH:  "with",
	TOKEN_FROM:  "from",
	TOKEN_TO:    "to",
	TOKEN_IN:    "in",
	TOKEN_ON:    "on",
	TOKEN_FOR:   "for",
	TOKEN_BY:    "by",
	TOKEN_AS:    "as",
	TOKEN_AND:   "and",
	TOKEN_OR:    "or",
	TOKEN_NOT:   "not",
	TOKEN_THE:   "the",
	TOKEN_A:     "a",
	TOKEN_AN:    "an",
	TOKEN_WHICH: "which",
	TOKEN_THAT:  "that",
	TOKEN_EITHER: "either",
	TOKEN_OF:    "of",
	TOKEN_ITS:   "its",
	TOKEN_THEIR: "their",
	TOKEN_USING: "using",
	TOKEN_PER:   "per",
	TOKEN_AT:    "at",

	// Modifiers
	TOKEN_REQUIRES:  "requires",
	TOKEN_ACCEPTS:   "accepts",
	TOKEN_ONLY:      "only",
	TOKEN_EACH:      "each",
	TOKEN_ALL:       "all",
	TOKEN_OPTIONAL:  "optional",
	TOKEN_UNIQUE:    "unique",
	TOKEN_ENCRYPTED: "encrypted",

	// Relationships
	TOKEN_BELONGS:  "belongs",
	TOKEN_MANY:     "many",
	TOKEN_THROUGH:  "through",
	TOKEN_DEFAULTS: "defaults",

	// Interaction subjects
	TOKEN_CLICKING:  "clicking",
	TOKEN_TYPING:    "typing",
	TOKEN_HOVERING:  "hovering",
	TOKEN_PRESSING:  "pressing",
	TOKEN_SCROLLING: "scrolling",
	TOKEN_DRAGGING:  "dragging",

	// Interaction verbs
	TOKEN_DOES:      "does",
	TOKEN_NAVIGATES: "navigates",
	TOKEN_OPENS:     "opens",
	TOKEN_TRIGGERS:  "triggers",
	TOKEN_SHOWS:     "shows",
	TOKEN_LOADS:     "loads",
	TOKEN_REORDERS:  "reorders",

	// Policy
	TOKEN_CAN:    "can",
	TOKEN_CANNOT: "cannot",
	TOKEN_MUST:   "must",

	// Architecture
	TOKEN_MONOLITH:      "monolith",
	TOKEN_MICROSERVICES: "microservices",
	TOKEN_SERVERLESS:    "serverless",
	TOKEN_GATEWAY:       "gateway",
	TOKEN_BROKER:        "broker",

	// DevOps
	TOKEN_PIPELINE:   "pipeline",
	TOKEN_MONITOR:    "monitor",
	TOKEN_RELEASE:    "release",
	TOKEN_MERGE:      "merge",
	TOKEN_PUSH:       "push",
	TOKEN_SOURCE:     "source",
	TOKEN_REPOSITORY: "repository",
	TOKEN_BRANCHES:   "branches",

	// Other
	TOKEN_THERE:    "there",
	TOKEN_NO:       "no",
	TOKEN_DO:       "do",
	TOKEN_METHOD:   "method",
	TOKEN_ENDPOINT: "endpoint",
	TOKEN_EXCEPT:   "except",
	TOKEN_RATE:     "rate",
	TOKEN_LIMIT:    "limit",
	TOKEN_USES:     "uses",
	TOKEN_SANITIZE: "sanitize",
	TOKEN_VARIABLE: "variable",
	TOKEN_KEY:      "key",
}

// String returns the display name of a token type.
func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TOKEN(%d)", int(t))
}

// Token represents a single lexical token with its position in the source.
type Token struct {
	Type    TokenType
	Literal string // the actual source text of the token
	Line    int    // 1-based line number
	Column  int    // 1-based column number
}

// String returns a human-readable representation of a token.
func (t Token) String() string {
	switch t.Type {
	case TOKEN_EOF:
		return "EOF"
	case TOKEN_NEWLINE:
		return "NEWLINE"
	case TOKEN_INDENT:
		return "INDENT"
	case TOKEN_DEDENT:
		return "DEDENT"
	default:
		if t.Literal != "" {
			return fmt.Sprintf("%s(%q)", t.Type, t.Literal)
		}
		return t.Type.String()
	}
}

// keywords maps lowercase keyword strings to their token types.
// All keyword matching is case-insensitive.
var keywords = map[string]TokenType{
	// Declarations
	"app":            TOKEN_APP,
	"data":           TOKEN_DATA,
	"page":           TOKEN_PAGE,
	"component":      TOKEN_COMPONENT,
	"api":            TOKEN_API,
	"service":        TOKEN_SERVICE,
	"agent":          TOKEN_AGENT,
	"policy":         TOKEN_POLICY,
	"workflow":       TOKEN_WORKFLOW,
	"theme":          TOKEN_THEME,
	"architecture":   TOKEN_ARCHITECTURE,
	"environment":    TOKEN_ENVIRONMENT,
	"integrate":      TOKEN_INTEGRATE,
	"database":       TOKEN_DATABASE,
	"authentication": TOKEN_AUTHENTICATION,
	"build":          TOKEN_BUILD,
	"design":         TOKEN_DESIGN,

	// Types
	"text":     TOKEN_TEXT,
	"number":   TOKEN_NUMBER,
	"decimal":  TOKEN_DECIMAL,
	"boolean":  TOKEN_BOOLEAN,
	"date":     TOKEN_DATE,
	"datetime": TOKEN_DATETIME,
	"email":    TOKEN_EMAIL,
	"url":      TOKEN_URL,
	"file":     TOKEN_FILE,
	"image":    TOKEN_IMAGE,
	"json":     TOKEN_JSON,

	// Actions
	"show":     TOKEN_SHOW,
	"fetch":    TOKEN_FETCH,
	"create":   TOKEN_CREATE,
	"update":   TOKEN_UPDATE,
	"delete":   TOKEN_DELETE,
	"send":     TOKEN_SEND,
	"respond":  TOKEN_RESPOND,
	"navigate": TOKEN_NAVIGATE,
	"check":    TOKEN_CHECK,
	"validate": TOKEN_VALIDATE,
	"filter":   TOKEN_FILTER,
	"sort":     TOKEN_SORT,
	"paginate": TOKEN_PAGINATE,
	"search":   TOKEN_SEARCH,
	"set":      TOKEN_SET,
	"return":   TOKEN_RETURN,
	"publish":  TOKEN_PUBLISH,
	"listen":   TOKEN_LISTEN,
	"notify":   TOKEN_NOTIFY,
	"alert":    TOKEN_ALERT,
	"log":      TOKEN_LOG,
	"track":    TOKEN_TRACK,
	"run":      TOKEN_RUN,
	"deploy":   TOKEN_DEPLOY,
	"keep":     TOKEN_KEEP,
	"backup":   TOKEN_BACKUP,
	"retry":    TOKEN_RETRY,
	"rollback": TOKEN_ROLLBACK,
	"index":    TOKEN_INDEX,
	"enable":   TOKEN_ENABLE,
	"support":  TOKEN_SUPPORT,
	"assign":   TOKEN_ASSIGN,
	"use":      TOKEN_USE,

	// Conditions
	"if":     TOKEN_IF,
	"when":   TOKEN_WHEN,
	"while":  TOKEN_WHILE,
	"unless": TOKEN_UNLESS,
	"until":  TOKEN_UNTIL,
	"after":  TOKEN_AFTER,
	"before": TOKEN_BEFORE,
	"every":  TOKEN_EVERY,

	// Connectors
	"is":    TOKEN_IS,
	"are":   TOKEN_ARE,
	"has":   TOKEN_HAS,
	"with":  TOKEN_WITH,
	"from":  TOKEN_FROM,
	"to":    TOKEN_TO,
	"in":    TOKEN_IN,
	"on":    TOKEN_ON,
	"for":   TOKEN_FOR,
	"by":    TOKEN_BY,
	"as":    TOKEN_AS,
	"and":   TOKEN_AND,
	"or":    TOKEN_OR,
	"not":   TOKEN_NOT,
	"the":   TOKEN_THE,
	"a":     TOKEN_A,
	"an":    TOKEN_AN,
	"which": TOKEN_WHICH,
	"that":  TOKEN_THAT,
	"either": TOKEN_EITHER,
	"of":    TOKEN_OF,
	"its":   TOKEN_ITS,
	"their": TOKEN_THEIR,
	"using": TOKEN_USING,
	"per":   TOKEN_PER,
	"at":    TOKEN_AT,

	// Modifiers
	"requires":  TOKEN_REQUIRES,
	"accepts":   TOKEN_ACCEPTS,
	"only":      TOKEN_ONLY,
	"each":      TOKEN_EACH,
	"all":       TOKEN_ALL,
	"optional":  TOKEN_OPTIONAL,
	"unique":    TOKEN_UNIQUE,
	"encrypted": TOKEN_ENCRYPTED,

	// Relationships
	"belongs":  TOKEN_BELONGS,
	"many":     TOKEN_MANY,
	"through":  TOKEN_THROUGH,
	"defaults": TOKEN_DEFAULTS,

	// Interaction subjects
	"clicking":  TOKEN_CLICKING,
	"typing":    TOKEN_TYPING,
	"hovering":  TOKEN_HOVERING,
	"pressing":  TOKEN_PRESSING,
	"scrolling": TOKEN_SCROLLING,
	"dragging":  TOKEN_DRAGGING,

	// Interaction verbs
	"does":      TOKEN_DOES,
	"navigates": TOKEN_NAVIGATES,
	"opens":     TOKEN_OPENS,
	"triggers":  TOKEN_TRIGGERS,
	"shows":     TOKEN_SHOWS,
	"loads":     TOKEN_LOADS,
	"reorders":  TOKEN_REORDERS,

	// Policy
	"can":    TOKEN_CAN,
	"cannot": TOKEN_CANNOT,
	"must":   TOKEN_MUST,

	// Architecture
	"monolith":      TOKEN_MONOLITH,
	"microservices": TOKEN_MICROSERVICES,
	"serverless":    TOKEN_SERVERLESS,
	"gateway":       TOKEN_GATEWAY,
	"broker":        TOKEN_BROKER,

	// DevOps
	"pipeline":   TOKEN_PIPELINE,
	"monitor":    TOKEN_MONITOR,
	"release":    TOKEN_RELEASE,
	"merge":      TOKEN_MERGE,
	"push":       TOKEN_PUSH,
	"source":     TOKEN_SOURCE,
	"repository": TOKEN_REPOSITORY,
	"branches":   TOKEN_BRANCHES,

	// Other
	"there":    TOKEN_THERE,
	"no":       TOKEN_NO,
	"do":       TOKEN_DO,
	"method":   TOKEN_METHOD,
	"endpoint": TOKEN_ENDPOINT,
	"except":   TOKEN_EXCEPT,
	"rate":     TOKEN_RATE,
	"limit":    TOKEN_LIMIT,
	"uses":     TOKEN_USES,
	"sanitize": TOKEN_SANITIZE,
	"variable": TOKEN_VARIABLE,
	"key":      TOKEN_KEY,
}

// LookupKeyword returns the keyword token type for the given word,
// or TOKEN_IDENTIFIER if the word is not a keyword.
// Matching is case-insensitive.
func LookupKeyword(word string) TokenType {
	if tok, ok := keywords[strings.ToLower(word)]; ok {
		return tok
	}
	return TOKEN_IDENTIFIER
}
