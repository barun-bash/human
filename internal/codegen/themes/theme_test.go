package themes

import (
	"strings"
	"testing"

	"github.com/barun-bash/human/internal/ir"
)

// ── Normalize ──

func TestNormalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Material UI", "material"},
		{"MUI", "material"},
		{"material", "material"},
		{"shadcn/ui", "shadcn"},
		{"Shadcn", "shadcn"},
		{"Ant Design", "ant"},
		{"antd", "ant"},
		{"Chakra UI", "chakra"},
		{"chakra", "chakra"},
		{"Bootstrap", "bootstrap"},
		{"Tailwind CSS", "tailwind"},
		{"tailwindcss", "tailwind"},
		{"Tailwind", "tailwind"},
		{"Untitled UI", "untitled"},
		{"untitled", "untitled"},
		{"unknown-system", "unknown-system"},
	}

	for _, tt := range tests {
		got := Normalize(tt.input)
		if got != tt.want {
			t.Errorf("Normalize(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ── Registry ──

func TestRegistry(t *testing.T) {
	// All 7 systems should resolve
	for _, id := range AllIDs() {
		ds := Registry(id)
		if ds == nil {
			t.Errorf("Registry(%q) returned nil", id)
			continue
		}
		if ds.ID != id {
			t.Errorf("Registry(%q).ID = %q", id, ds.ID)
		}
		if ds.Name == "" {
			t.Errorf("Registry(%q).Name is empty", id)
		}
	}

	// Unknown returns nil
	if Registry("unknown") != nil {
		t.Error("Registry(\"unknown\") should return nil")
	}
}

func TestAllIDs(t *testing.T) {
	ids := AllIDs()
	if len(ids) != 7 {
		t.Errorf("AllIDs: expected 7, got %d", len(ids))
	}
	expected := map[string]bool{
		"material": true, "shadcn": true, "ant": true,
		"chakra": true, "bootstrap": true, "tailwind": true, "untitled": true,
	}
	for _, id := range ids {
		if !expected[id] {
			t.Errorf("unexpected ID: %q", id)
		}
	}
}

// ── Dependencies ──

func TestDependencies(t *testing.T) {
	tests := []struct {
		system    string
		framework string
		wantDep   string // a key that should exist in deps
		wantDev   string // a key that should exist in devDeps
	}{
		{"material", "react", "@mui/material", ""},
		{"material", "vue", "vuetify", ""},
		{"shadcn", "react", "class-variance-authority", "tailwindcss"},
		{"chakra", "react", "@chakra-ui/react", ""},
		{"bootstrap", "react", "react-bootstrap", ""},
		{"tailwind", "react", "", "tailwindcss"},
		{"ant", "react", "antd", ""},
		{"ant", "angular", "ng-zorro-antd", ""},
	}

	for _, tt := range tests {
		deps, devDeps := Dependencies(tt.system, tt.framework)
		if tt.wantDep != "" {
			if _, ok := deps[tt.wantDep]; !ok {
				t.Errorf("Dependencies(%q, %q): missing dep %q", tt.system, tt.framework, tt.wantDep)
			}
		}
		if tt.wantDev != "" {
			if _, ok := devDeps[tt.wantDev]; !ok {
				t.Errorf("Dependencies(%q, %q): missing devDep %q", tt.system, tt.framework, tt.wantDev)
			}
		}
	}
}

func TestDependenciesFallback(t *testing.T) {
	// Chakra + Vue should fallback to tailwind
	_, devDeps := Dependencies("chakra", "vue")
	if _, ok := devDeps["tailwindcss"]; !ok {
		t.Error("chakra+vue fallback: should include tailwindcss")
	}

	// Unknown system should fallback to tailwind
	_, devDeps = Dependencies("unknown", "react")
	if _, ok := devDeps["tailwindcss"]; !ok {
		t.Error("unknown system: should fallback to tailwindcss")
	}
}

// ── DefaultTokens ──

func TestDefaultTokens(t *testing.T) {
	for _, id := range AllIDs() {
		tokens := DefaultTokens(id)
		for _, key := range []string{"primary", "secondary", "background", "text"} {
			if _, ok := tokens[key]; !ok {
				t.Errorf("DefaultTokens(%q): missing %q", id, key)
			}
		}
	}

	// Unknown system uses neutral defaults
	tokens := DefaultTokens("unknown")
	if tokens["primary"] == "" {
		t.Error("DefaultTokens(\"unknown\"): should return neutral defaults")
	}
}

// ── MergeTokens ──

func TestMergeTokens(t *testing.T) {
	theme := &ir.Theme{
		Colors: map[string]string{
			"primary": "#ff0000",
		},
		Fonts: map[string]string{
			"body": "Inter",
		},
		Spacing:      "compact",
		BorderRadius: "rounded",
	}

	tokens := MergeTokens("material", theme)

	// User color should override default
	if tokens["--color-primary"] != "#ff0000" {
		t.Errorf("primary: got %q, want #ff0000", tokens["--color-primary"])
	}

	// Default color should fill gap
	if tokens["--color-secondary"] == "" {
		t.Error("secondary should be filled from defaults")
	}

	// Font token
	if tokens["--font-body"] != "Inter" {
		t.Errorf("font-body: got %q", tokens["--font-body"])
	}

	// Spacing tokens
	if tokens["--spacing-sm"] != "4px" {
		t.Errorf("spacing-sm: got %q, want 4px (compact)", tokens["--spacing-sm"])
	}

	// Border radius
	if tokens["--radius"] != "12px" {
		t.Errorf("radius: got %q, want 12px (rounded)", tokens["--radius"])
	}
}

func TestMergeTokensDefaults(t *testing.T) {
	// nil theme should still produce color, spacing, and radius defaults
	tokens := MergeTokens("tailwind", nil)
	if tokens["--color-primary"] == "" {
		t.Error("should have default primary color")
	}
	if tokens["--spacing-sm"] != "8px" {
		t.Errorf("nil theme: --spacing-sm got %q, want 8px", tokens["--spacing-sm"])
	}
	if tokens["--spacing-md"] != "16px" {
		t.Errorf("nil theme: --spacing-md got %q, want 16px", tokens["--spacing-md"])
	}
	if tokens["--spacing-lg"] != "24px" {
		t.Errorf("nil theme: --spacing-lg got %q, want 24px", tokens["--spacing-lg"])
	}
	if tokens["--radius"] != "6px" {
		t.Errorf("nil theme: --radius got %q, want 6px", tokens["--radius"])
	}
}

// ── GenerateReactTheme ──

func TestGenerateReactThemeMaterial(t *testing.T) {
	theme := &ir.Theme{
		DesignSystem: "material",
		Colors:       map[string]string{"primary": "#1976d2"},
		Fonts:        map[string]string{"body": "Roboto"},
	}

	files := GenerateReactTheme(theme)

	// Should have theme.ts
	themeTs, ok := files["src/theme.ts"]
	if !ok {
		t.Fatal("missing src/theme.ts")
	}
	if !strings.Contains(themeTs, "createTheme") {
		t.Error("theme.ts should contain createTheme")
	}
	if !strings.Contains(themeTs, "#1976d2") {
		t.Error("theme.ts should contain primary color")
	}

	// Should have global.css
	if _, ok := files["src/styles/global.css"]; !ok {
		t.Error("missing src/styles/global.css")
	}

	// Should NOT have tailwind.config.js
	if _, ok := files["tailwind.config.js"]; ok {
		t.Error("material should not generate tailwind.config.js")
	}
}

func TestGenerateReactThemeShadcn(t *testing.T) {
	theme := &ir.Theme{
		DesignSystem: "shadcn",
		Colors:       map[string]string{},
	}

	files := GenerateReactTheme(theme)

	// Should have tailwind.config.js
	if _, ok := files["tailwind.config.js"]; !ok {
		t.Error("shadcn should generate tailwind.config.js")
	}

	// Should have utils
	if _, ok := files["src/lib/utils.ts"]; !ok {
		t.Error("shadcn should generate src/lib/utils.ts")
	}
}

func TestGenerateReactThemeChakra(t *testing.T) {
	theme := &ir.Theme{
		DesignSystem: "chakra",
		Colors:       map[string]string{},
	}

	files := GenerateReactTheme(theme)

	themeTs, ok := files["src/theme.ts"]
	if !ok {
		t.Fatal("missing src/theme.ts")
	}
	if !strings.Contains(themeTs, "extendTheme") {
		t.Error("theme.ts should contain extendTheme")
	}
}

// ── GenerateVueTheme ──

func TestGenerateVueThemeMaterial(t *testing.T) {
	theme := &ir.Theme{
		DesignSystem: "material",
		Colors:       map[string]string{"primary": "#1976d2"},
	}

	files := GenerateVueTheme(theme)

	plugin, ok := files["src/plugins/vuetify.ts"]
	if !ok {
		t.Fatal("missing src/plugins/vuetify.ts")
	}
	if !strings.Contains(plugin, "createVuetify") {
		t.Error("vuetify plugin should contain createVuetify")
	}
}

// ── GenerateCSSVariables ──

func TestGenerateCSSVariables(t *testing.T) {
	tokens := map[string]string{
		"--color-primary": "#3b82f6",
		"--color-text":    "#111827",
		"--radius":        "6px",
	}
	theme := &ir.Theme{
		Fonts: map[string]string{"body": "Inter", "headings": "Poppins"},
	}

	output := GenerateCSSVariables(tokens, theme)

	if !strings.Contains(output, ":root") {
		t.Error("should contain :root")
	}
	if !strings.Contains(output, "--color-primary") {
		t.Error("should contain --color-primary")
	}
	if !strings.Contains(output, "font-family: 'Inter'") {
		t.Error("should contain Inter font-family")
	}
	if !strings.Contains(output, "font-family: 'Poppins'") {
		t.Error("should contain Poppins headings font-family")
	}
	if !strings.Contains(output, "box-sizing: border-box") {
		t.Error("should contain CSS reset")
	}
}

func TestGenerateCSSVariablesDarkMode(t *testing.T) {
	tokens := map[string]string{"--color-primary": "#3b82f6"}
	theme := &ir.Theme{DarkMode: true}

	output := GenerateCSSVariables(tokens, theme)

	if !strings.Contains(output, "@media (prefers-color-scheme: dark)") {
		t.Error("should contain dark mode media query")
	}
	if !strings.Contains(output, ".dark {") {
		t.Error("should contain .dark class for toggling")
	}
}

func TestGenerateCSSVariablesFontImport(t *testing.T) {
	tokens := map[string]string{}
	theme := &ir.Theme{
		Fonts: map[string]string{"body": "Inter"},
	}

	output := GenerateCSSVariables(tokens, theme)

	if !strings.Contains(output, "fonts.googleapis.com") {
		t.Error("should import Google Fonts")
	}
	if !strings.Contains(output, "family=Inter") {
		t.Error("should import Inter font")
	}
}

// ── GenerateTailwindConfig ──

func TestGenerateTailwindConfig(t *testing.T) {
	theme := &ir.Theme{
		DarkMode: true,
		Fonts:    map[string]string{"body": "Inter"},
	}
	tokens := map[string]string{
		"--color-primary":   "#3b82f6",
		"--color-secondary": "#8b5cf6",
		"--radius":          "6px",
		"--spacing-sm":      "8px",
		"--spacing-md":      "16px",
		"--spacing-lg":      "24px",
		"--font-body":       "Inter",
	}

	output := GenerateTailwindConfig(theme, tokens, "react")

	if !strings.Contains(output, "theme.extend.colors") || !strings.Contains(output, "colors: {") {
		// Check for "colors:" inside theme.extend
		if !strings.Contains(output, "colors: {") {
			t.Error("should contain theme.extend.colors")
		}
	}
	if !strings.Contains(output, "'primary'") {
		t.Error("should contain primary color key")
	}
	if !strings.Contains(output, "darkMode: 'class'") {
		t.Error("should contain darkMode class config")
	}
	if !strings.Contains(output, "borderRadius") {
		t.Error("should contain borderRadius")
	}
	if !strings.Contains(output, "fontFamily") {
		t.Error("should contain fontFamily")
	}
}

// ── HasFrameworkSupport ──

func TestHasFrameworkSupport(t *testing.T) {
	tests := []struct {
		system    string
		framework string
		want      bool
	}{
		{"material", "react", true},
		{"material", "vue", true},
		{"chakra", "react", true},
		{"chakra", "vue", false},
		{"chakra", "angular", false},
		{"tailwind", "react", true},
		{"tailwind", "vue", true},
		{"unknown", "react", false},
	}

	for _, tt := range tests {
		got := HasFrameworkSupport(tt.system, tt.framework)
		if got != tt.want {
			t.Errorf("HasFrameworkSupport(%q, %q): got %v, want %v", tt.system, tt.framework, got, tt.want)
		}
	}
}

// ── NeedsTailwind ──

func TestNeedsTailwind(t *testing.T) {
	tests := []struct {
		system string
		want   bool
	}{
		{"tailwind", true},
		{"shadcn", true},
		{"untitled", true},
		{"material", false},
		{"ant", false},
		{"chakra", false},
		{"bootstrap", false},
	}

	for _, tt := range tests {
		got := NeedsTailwind(tt.system)
		if got != tt.want {
			t.Errorf("NeedsTailwind(%q): got %v, want %v", tt.system, got, tt.want)
		}
	}
}

// ── GenerateReactTheme Bootstrap ──

func TestGenerateReactThemeBootstrap(t *testing.T) {
	theme := &ir.Theme{
		DesignSystem: "bootstrap",
		Colors:       map[string]string{"primary": "#0d6efd"},
	}

	files := GenerateReactTheme(theme)

	scss, ok := files["src/styles/custom.scss"]
	if !ok {
		t.Fatal("bootstrap should generate src/styles/custom.scss")
	}
	if !strings.Contains(scss, "$primary: #0d6efd") {
		t.Error("custom.scss should contain $primary")
	}
	if !strings.Contains(scss, "@import 'bootstrap/scss/bootstrap'") {
		t.Error("custom.scss should import bootstrap scss")
	}

	// Bootstrap should NOT have tailwind.config.js
	if _, ok := files["tailwind.config.js"]; ok {
		t.Error("bootstrap should not generate tailwind.config.js")
	}
}

// ── Bootstrap sass devDep ──

func TestBootstrapHasSassDevDep(t *testing.T) {
	_, devDeps := Dependencies("bootstrap", "react")
	if _, ok := devDeps["sass"]; !ok {
		t.Error("bootstrap+react should include sass devDep")
	}
}

// ── GenerateAngularTheme ──

func TestGenerateAngularThemeMaterial(t *testing.T) {
	theme := &ir.Theme{
		DesignSystem: "material",
		Colors:       map[string]string{"primary": "#1976d2"},
	}

	files := GenerateAngularTheme(theme)

	if _, ok := files["src/styles.css"]; !ok {
		t.Error("missing src/styles.css")
	}

	themeTs, ok := files["src/app/theme.ts"]
	if !ok {
		t.Fatal("missing src/app/theme.ts")
	}
	if !strings.Contains(themeTs, "primary") {
		t.Error("theme.ts should contain primary")
	}
}

func TestGenerateAngularThemeTailwind(t *testing.T) {
	theme := &ir.Theme{
		DesignSystem: "tailwind",
		Colors:       map[string]string{},
	}

	files := GenerateAngularTheme(theme)

	if _, ok := files["tailwind.config.js"]; !ok {
		t.Error("tailwind angular should generate tailwind.config.js")
	}
}

// ── GenerateSvelteTheme ──

func TestGenerateSvelteThemeTailwind(t *testing.T) {
	theme := &ir.Theme{
		DesignSystem: "tailwind",
		Colors:       map[string]string{},
	}

	files := GenerateSvelteTheme(theme)

	if _, ok := files["src/app.css"]; !ok {
		t.Error("missing src/app.css")
	}
	if _, ok := files["src/lib/theme.ts"]; !ok {
		t.Error("missing src/lib/theme.ts")
	}
	if _, ok := files["tailwind.config.js"]; !ok {
		t.Error("tailwind svelte should generate tailwind.config.js")
	}
}

func TestGenerateSvelteThemeDefault(t *testing.T) {
	theme := &ir.Theme{
		DesignSystem: "",
		Colors:       map[string]string{},
	}

	files := GenerateSvelteTheme(theme)

	// Default should get tailwind config
	if _, ok := files["tailwind.config.js"]; !ok {
		t.Error("default svelte theme should generate tailwind.config.js")
	}
}

// ── CSS @import ordering ──

func TestCSSVariablesFontImportBeforeRoot(t *testing.T) {
	tokens := map[string]string{"--color-primary": "#3b82f6"}
	theme := &ir.Theme{
		Fonts: map[string]string{"body": "Inter"},
	}

	output := GenerateCSSVariables(tokens, theme)

	importIdx := strings.Index(output, "@import url(")
	rootIdx := strings.Index(output, ":root")

	if importIdx == -1 {
		t.Fatal("should contain @import")
	}
	if rootIdx == -1 {
		t.Fatal("should contain :root")
	}
	if importIdx >= rootIdx {
		t.Error("@import should appear before :root for CSS spec compliance")
	}
}

// ── CSS var to JS ──

func TestCssVarToJS(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"--color-primary", "colorPrimary"},
		{"--spacing-sm", "spacingSm"},
		{"--font-body", "fontBody"},
		{"--radius", "radius"},
	}

	for _, tt := range tests {
		got := cssVarToJS(tt.input)
		if got != tt.want {
			t.Errorf("cssVarToJS(%q): got %q, want %q", tt.input, got, tt.want)
		}
	}
}
