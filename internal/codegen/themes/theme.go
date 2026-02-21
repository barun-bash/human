package themes

import "strings"

// DesignSystem holds metadata for a design system.
type DesignSystem struct {
	ID         string
	Name       string
	Frameworks map[string]FrameworkSupport // key: "react", "vue", "angular", "svelte"
}

// FrameworkSupport holds framework-specific package info.
type FrameworkSupport struct {
	Packages    map[string]string // npm package → version
	DevPackages map[string]string // dev dependencies
	Imports     []string          // import lines for generated code
	Provider    string            // wrapper component (e.g. "<ThemeProvider>")
}

// registry holds all supported design systems.
var registry = map[string]*DesignSystem{
	"material": {
		ID:   "material",
		Name: "Material UI",
		Frameworks: map[string]FrameworkSupport{
			"react": {
				Packages:    map[string]string{"@mui/material": "^6.0.0", "@emotion/react": "^11.13.0", "@emotion/styled": "^11.13.0"},
				Imports:     []string{"import { ThemeProvider, createTheme } from '@mui/material/styles';", "import CssBaseline from '@mui/material/CssBaseline';"},
				Provider:    "ThemeProvider",
			},
			"vue": {
				Packages: map[string]string{"vuetify": "^3.7.0"},
				Imports:  []string{"import { createVuetify } from 'vuetify';", "import 'vuetify/styles';"},
				Provider: "v-app",
			},
			"angular": {
				Packages: map[string]string{"@angular/material": "^17.0.0", "@angular/cdk": "^17.0.0"},
				Imports:  []string{"import { MatButtonModule } from '@angular/material/button';"},
			},
			"svelte": {
				// smelte is unmaintained; use Tailwind with Material palette instead
				DevPackages: map[string]string{"tailwindcss": "^3.4.0", "autoprefixer": "^10.4.0", "postcss": "^8.4.0"},
			},
		},
	},
	"shadcn": {
		ID:   "shadcn",
		Name: "Shadcn/ui",
		Frameworks: map[string]FrameworkSupport{
			"react": {
				Packages:    map[string]string{"class-variance-authority": "^0.7.0", "clsx": "^2.1.0", "tailwind-merge": "^2.5.0", "@radix-ui/react-slot": "^1.1.0"},
				DevPackages: map[string]string{"tailwindcss": "^3.4.0", "autoprefixer": "^10.4.0", "postcss": "^8.4.0"},
			},
			"vue": {
				Packages:    map[string]string{"class-variance-authority": "^0.7.0", "clsx": "^2.1.0", "tailwind-merge": "^2.5.0", "radix-vue": "^1.9.0"},
				DevPackages: map[string]string{"tailwindcss": "^3.4.0", "autoprefixer": "^10.4.0", "postcss": "^8.4.0"},
			},
			"svelte": {
				Packages:    map[string]string{"bits-ui": "^0.21.0", "clsx": "^2.1.0", "tailwind-merge": "^2.5.0"},
				DevPackages: map[string]string{"tailwindcss": "^3.4.0", "autoprefixer": "^10.4.0", "postcss": "^8.4.0"},
			},
		},
	},
	"ant": {
		ID:   "ant",
		Name: "Ant Design",
		Frameworks: map[string]FrameworkSupport{
			"react": {
				Packages: map[string]string{"antd": "^5.22.0"},
				Imports:  []string{"import { ConfigProvider } from 'antd';"},
				Provider: "ConfigProvider",
			},
			"vue": {
				Packages: map[string]string{"ant-design-vue": "^4.2.0"},
				Imports:  []string{"import Antd from 'ant-design-vue';", "import 'ant-design-vue/dist/reset.css';"},
			},
			"angular": {
				Packages: map[string]string{"ng-zorro-antd": "^18.0.0"},
			},
		},
	},
	"chakra": {
		ID:   "chakra",
		Name: "Chakra UI",
		Frameworks: map[string]FrameworkSupport{
			"react": {
				Packages: map[string]string{"@chakra-ui/react": "^2.10.0", "@emotion/react": "^11.13.0", "@emotion/styled": "^11.13.0", "framer-motion": "^11.11.0"},
				Imports:  []string{"import { ChakraProvider, extendTheme } from '@chakra-ui/react';"},
				Provider: "ChakraProvider",
			},
		},
	},
	"bootstrap": {
		ID:   "bootstrap",
		Name: "Bootstrap",
		Frameworks: map[string]FrameworkSupport{
			"react": {
				Packages:    map[string]string{"react-bootstrap": "^2.10.0", "bootstrap": "^5.3.0"},
				DevPackages: map[string]string{"sass": "^1.80.0"},
				Imports:     []string{"import 'bootstrap/dist/css/bootstrap.min.css';"},
			},
			"vue": {
				Packages: map[string]string{"bootstrap-vue-next": "^0.25.0", "bootstrap": "^5.3.0"},
				Imports:  []string{"import 'bootstrap/dist/css/bootstrap.min.css';"},
			},
			"angular": {
				Packages: map[string]string{"ngx-bootstrap": "^18.0.0", "bootstrap": "^5.3.0"},
			},
			"svelte": {
				Packages: map[string]string{"sveltestrap": "^6.2.0", "bootstrap": "^5.3.0"},
				Imports:  []string{"import 'bootstrap/dist/css/bootstrap.min.css';"},
			},
		},
	},
	"tailwind": {
		ID:   "tailwind",
		Name: "Tailwind CSS",
		Frameworks: map[string]FrameworkSupport{
			"react": {
				DevPackages: map[string]string{"tailwindcss": "^3.4.0", "autoprefixer": "^10.4.0", "postcss": "^8.4.0"},
			},
			"vue": {
				DevPackages: map[string]string{"tailwindcss": "^3.4.0", "autoprefixer": "^10.4.0", "postcss": "^8.4.0"},
			},
			"angular": {
				DevPackages: map[string]string{"tailwindcss": "^3.4.0", "autoprefixer": "^10.4.0", "postcss": "^8.4.0"},
			},
			"svelte": {
				DevPackages: map[string]string{"tailwindcss": "^3.4.0", "autoprefixer": "^10.4.0", "postcss": "^8.4.0"},
			},
		},
	},
	"untitled": {
		ID:   "untitled",
		Name: "Untitled UI",
		Frameworks: map[string]FrameworkSupport{
			"react": {
				DevPackages: map[string]string{"tailwindcss": "^3.4.0", "autoprefixer": "^10.4.0", "postcss": "^8.4.0"},
			},
			"vue": {
				DevPackages: map[string]string{"tailwindcss": "^3.4.0", "autoprefixer": "^10.4.0", "postcss": "^8.4.0"},
			},
			"angular": {
				DevPackages: map[string]string{"tailwindcss": "^3.4.0", "autoprefixer": "^10.4.0", "postcss": "^8.4.0"},
			},
			"svelte": {
				DevPackages: map[string]string{"tailwindcss": "^3.4.0", "autoprefixer": "^10.4.0", "postcss": "^8.4.0"},
			},
		},
	},
}

// Registry returns the design system by ID, or nil if unknown.
func Registry(id string) *DesignSystem {
	return registry[strings.ToLower(id)]
}

// AllIDs returns all supported design system IDs.
func AllIDs() []string {
	return []string{"material", "shadcn", "ant", "chakra", "bootstrap", "tailwind", "untitled"}
}

// Normalize maps user-facing names to registry IDs.
func Normalize(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	lower = strings.TrimSuffix(lower, " css")
	lower = strings.TrimSuffix(lower, " ui")

	aliases := map[string]string{
		"material":     "material",
		"mui":          "material",
		"material ui":  "material",
		"shadcn":       "shadcn",
		"shadcn/ui":    "shadcn",
		"ant":          "ant",
		"ant design":   "ant",
		"antd":         "ant",
		"chakra":       "chakra",
		"chakra ui":    "chakra",
		"bootstrap":    "bootstrap",
		"tailwind":     "tailwind",
		"tailwindcss":  "tailwind",
		"tailwind css": "tailwind",
		"untitled":     "untitled",
		"untitled ui":  "untitled",
	}

	if id, ok := aliases[lower]; ok {
		return id
	}
	return lower
}

// HasFrameworkSupport returns true if the design system supports the given framework.
func HasFrameworkSupport(systemID, framework string) bool {
	ds := Registry(systemID)
	if ds == nil {
		return false
	}
	_, ok := ds.Frameworks[strings.ToLower(framework)]
	return ok
}

// NeedsTailwind returns true if the design system uses Tailwind CSS.
func NeedsTailwind(systemID string) bool {
	switch systemID {
	case "tailwind", "shadcn", "untitled":
		return true
	default:
		return false
	}
}
