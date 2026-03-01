package figma

import (
	"testing"
)

func TestShouldExtractAsset(t *testing.T) {
	tests := []struct {
		name   string
		node   *FigmaNode
		expect bool
	}{
		{
			name:   "image fill",
			node:   &FigmaNode{Name: "photo-frame", Fills: []Paint{{Type: "IMAGE"}}},
			expect: true,
		},
		{
			name:   "logo by name",
			node:   &FigmaNode{Name: "Company Logo", Fills: []Paint{{Type: "SOLID"}}},
			expect: true,
		},
		{
			name:   "icon by name",
			node:   &FigmaNode{Name: "search-icon", Fills: []Paint{{Type: "SOLID"}}},
			expect: true,
		},
		{
			name:   "avatar by name",
			node:   &FigmaNode{Name: "user-avatar", Fills: []Paint{{Type: "SOLID"}}},
			expect: true,
		},
		{
			name:   "standalone vector",
			node:   &FigmaNode{Name: "arrow", Type: "VECTOR", Children: nil},
			expect: true,
		},
		{
			name:   "vector with children",
			node:   &FigmaNode{Name: "group", Type: "VECTOR", Children: []*FigmaNode{{Name: "child"}}},
			expect: false,
		},
		{
			name:   "plain frame",
			node:   &FigmaNode{Name: "container", Type: "FRAME", Fills: []Paint{{Type: "SOLID"}}},
			expect: false,
		},
		{
			name:   "text node",
			node:   &FigmaNode{Name: "heading", Type: "TEXT"},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldExtractAsset(tt.node)
			if got != tt.expect {
				t.Errorf("ShouldExtractAsset() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestClassifyAssetType(t *testing.T) {
	tests := []struct {
		node     *FigmaNode
		expected string
	}{
		{&FigmaNode{Name: "Company Logo"}, "logo"},
		{&FigmaNode{Name: "search-icon"}, "icon"},
		{&FigmaNode{Name: "arrow", Type: "VECTOR"}, "icon"},
		{&FigmaNode{Name: "hero-image"}, "image"},
		{&FigmaNode{Name: "photo"}, "image"},
	}

	for _, tt := range tests {
		got := classifyAssetType(tt.node)
		if got != tt.expected {
			t.Errorf("classifyAssetType(%q) = %q, want %q", tt.node.Name, got, tt.expected)
		}
	}
}

func TestSplitByFormat(t *testing.T) {
	candidates := []*assetCandidate{
		{node: &FigmaNode{ID: "1", Name: "logo"}, assetType: "logo"},
		{node: &FigmaNode{ID: "2", Name: "icon"}, assetType: "icon"},
		{node: &FigmaNode{ID: "3", Name: "photo", Fills: []Paint{{Type: "IMAGE"}}}, assetType: "image"},
		{node: &FigmaNode{ID: "4", Name: "raster-logo", Fills: []Paint{{Type: "IMAGE"}}}, assetType: "logo"},
	}

	svgIDs, pngIDs := splitByFormat(candidates)
	if len(svgIDs) != 2 {
		t.Errorf("expected 2 SVG IDs, got %d: %v", len(svgIDs), svgIDs)
	}
	if len(pngIDs) != 2 {
		t.Errorf("expected 2 PNG IDs, got %d: %v", len(pngIDs), pngIDs)
	}
}

func TestResolveAssetDir(t *testing.T) {
	tests := []struct {
		frontend string
		contains string
	}{
		{"React", "src/assets"},
		{"Vue", "src/assets"},
		{"Angular", "src/assets"},
		{"Svelte", "static/assets"},
		{"Unknown", "public/assets"},
	}

	for _, tt := range tests {
		got := resolveAssetDir("/out", tt.frontend)
		if got == "" {
			t.Errorf("resolveAssetDir(%q) returned empty", tt.frontend)
		}
	}
}

func TestCleanFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Company Logo", "company-logo"},
		{"icon/search", "icon-search"},
		{"test  file", "test-file"},
		{"A*B?C", "abc"},
		{"-leading-", "leading"},
	}

	for _, tt := range tests {
		got := cleanFilename(tt.input)
		if got != tt.expected {
			t.Errorf("cleanFilename(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestWalkNodes(t *testing.T) {
	tree := []*FigmaNode{
		{
			Name: "root",
			Children: []*FigmaNode{
				{Name: "child1"},
				{Name: "child2", Children: []*FigmaNode{
					{Name: "grandchild"},
				}},
			},
		},
	}

	var names []string
	walkNodes(tree, func(n *FigmaNode) {
		names = append(names, n.Name)
	})

	if len(names) != 4 {
		t.Fatalf("expected 4 nodes, got %d: %v", len(names), names)
	}
	expected := []string{"root", "child1", "child2", "grandchild"}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("names[%d] = %q, want %q", i, names[i], name)
		}
	}
}
