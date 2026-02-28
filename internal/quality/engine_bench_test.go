package quality

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/barun-bash/human/internal/ir"
	"github.com/barun-bash/human/internal/parser"
)

func loadApp(b *testing.B, example string) *ir.Application {
	b.Helper()
	path := filepath.Join("..", "..", "examples", example, "app.human")
	source, err := os.ReadFile(path)
	if err != nil {
		b.Fatalf("reading %s: %v", path, err)
	}
	prog, err := parser.Parse(string(source))
	if err != nil {
		b.Fatalf("parsing %s: %v", example, err)
	}
	app, err := ir.Build(prog)
	if err != nil {
		b.Fatalf("IR build %s: %v", example, err)
	}
	return app
}

func BenchmarkQualityRun(b *testing.B) {
	app := loadApp(b, "taskflow")
	b.ResetTimer()
	for b.Loop() {
		dir := b.TempDir()
		if _, err := Run(app, dir); err != nil {
			b.Fatal(err)
		}
	}
}
