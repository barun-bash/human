package llm_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoLLMImportInCore verifies that core compiler packages never import
// the LLM connector. This enforces the CLAUDE.md rule:
// "Do not add any AI/LLM dependency to the core compiler."
func TestNoLLMImportInCore(t *testing.T) {
	corePackages := []string{
		"internal/parser",
		"internal/lexer",
		"internal/ir",
		"internal/codegen",
		"internal/quality",
		"internal/analyzer",
		"internal/errors",
		"internal/cli",
	}

	// Walk up from the test file to find the repo root.
	root := findRepoRoot(t)

	for _, pkg := range corePackages {
		pkgDir := filepath.Join(root, pkg)
		if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(pkgDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			if strings.Contains(string(content), `"github.com/barun-bash/human/internal/llm`) {
				relPath, _ := filepath.Rel(root, path)
				t.Errorf("IMPORT ISOLATION VIOLATION: %s imports internal/llm — core packages must not depend on the LLM connector", relPath)
			}

			return nil
		})
		if err != nil {
			t.Errorf("walking %s: %v", pkg, err)
		}
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("could not get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (no go.mod found)")
		}
		dir = parent
	}
}
