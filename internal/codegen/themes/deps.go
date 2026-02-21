package themes

import "strings"

// Dependencies returns (deps, devDeps) for a given design system + framework.
// If the design system doesn't support the framework, falls back to tailwind.
func Dependencies(systemID, framework string) (map[string]string, map[string]string) {
	framework = strings.ToLower(framework)
	ds := Registry(systemID)
	if ds == nil {
		// Unknown system — return tailwind as fallback
		return nil, map[string]string{
			"tailwindcss":  "^3.4.0",
			"autoprefixer": "^10.4.0",
			"postcss":      "^8.4.0",
		}
	}

	fs, ok := ds.Frameworks[framework]
	if !ok {
		// Design system doesn't support this framework — fallback to tailwind
		// but use the design system's color palette via CSS variables
		return nil, map[string]string{
			"tailwindcss":  "^3.4.0",
			"autoprefixer": "^10.4.0",
			"postcss":      "^8.4.0",
		}
	}

	deps := make(map[string]string)
	devDeps := make(map[string]string)

	for k, v := range fs.Packages {
		deps[k] = v
	}
	for k, v := range fs.DevPackages {
		devDeps[k] = v
	}

	return deps, devDeps
}
