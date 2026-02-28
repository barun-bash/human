package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/lexer"
	"github.com/barun-bash/human/internal/parser"
)

func loadSource(b *testing.B, example string) string {
	b.Helper()
	path := filepath.Join("..", "..", "examples", example, "app.human")
	source, err := os.ReadFile(path)
	if err != nil {
		b.Fatalf("reading %s: %v", path, err)
	}
	return string(source)
}

func BenchmarkLexTaskflow(b *testing.B) {
	source := loadSource(b, "taskflow")
	b.ResetTimer()
	for b.Loop() {
		lex := lexer.New(source)
		if _, err := lex.Tokenize(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseTaskflow(b *testing.B) {
	source := loadSource(b, "taskflow")
	lex := lexer.New(source)
	tokens, err := lex.Tokenize()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		if _, err := parser.ParseTokens(tokens); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIRBuild(b *testing.B) {
	source := loadSource(b, "taskflow")
	prog, err := parser.Parse(source)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		if _, err := ir.Build(prog); err != nil {
			b.Fatal(err)
		}
	}
}
