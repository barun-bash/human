package quality

import (
	"fmt"
	"strings"

	"github.com/barun-bash/human/internal/ir"
)

// checkSecurity scans the IR for security issues.
func checkSecurity(app *ir.Application) []Finding {
	var findings []Finding

	findings = append(findings, checkMissingAuth(app)...)
	findings = append(findings, checkMissingValidation(app)...)
	findings = append(findings, checkHardcodedSecrets(app)...)
	findings = append(findings, checkRateLimiting(app)...)
	findings = append(findings, checkInputSanitization(app)...)
	findings = append(findings, checkCORSConfig(app)...)
	findings = append(findings, checkSecretPatterns(app)...)

	return findings
}

// checkMissingAuth flags API endpoints that modify data but don't require auth.
func checkMissingAuth(app *ir.Application) []Finding {
	var findings []Finding
	for _, ep := range app.APIs {
		if ep.Auth {
			continue
		}
		// Read-only endpoints without auth are fine (public pages).
		// Mutations without auth are a concern.
		method := strings.ToLower(httpMethod(ep.Name))
		if method == "get" {
			continue
		}
		// SignUp and Login are expected to not require auth
		lower := strings.ToLower(ep.Name)
		if lower == "signup" || lower == "login" {
			continue
		}
		findings = append(findings, Finding{
			Severity: "critical",
			Category: "auth",
			Message:  fmt.Sprintf("Endpoint %s modifies data but does not require authentication", ep.Name),
			Target:   ep.Name,
		})
	}
	return findings
}

// checkMissingValidation flags endpoints that accept input but have no validation rules.
func checkMissingValidation(app *ir.Application) []Finding {
	var findings []Finding
	for _, ep := range app.APIs {
		if len(ep.Params) == 0 {
			continue
		}
		if len(ep.Validation) > 0 {
			continue
		}
		findings = append(findings, Finding{
			Severity: "warning",
			Category: "validation",
			Message:  fmt.Sprintf("Endpoint %s accepts input but has no validation rules", ep.Name),
			Target:   ep.Name,
		})
	}
	return findings
}

// checkHardcodedSecrets looks for potential secrets in config values.
func checkHardcodedSecrets(app *ir.Application) []Finding {
	var findings []Finding

	// Check auth config for hardcoded values that look like secrets
	if app.Auth != nil {
		for _, m := range app.Auth.Methods {
			for key, val := range m.Config {
				lower := strings.ToLower(key)
				if strings.Contains(lower, "secret") || strings.Contains(lower, "key") || strings.Contains(lower, "password") {
					if !strings.Contains(val, "env") && !strings.Contains(val, "ENV") && len(val) > 10 {
						findings = append(findings, Finding{
							Severity: "critical",
							Category: "secrets",
							Message:  fmt.Sprintf("Auth config '%s' may contain a hardcoded secret", key),
							Target:   "authentication",
						})
					}
				}
			}
		}
	}

	// Check integration credentials that aren't env var references
	for _, integ := range app.Integrations {
		for desc, envVar := range integ.Credentials {
			if !strings.HasPrefix(envVar, "$") && !isEnvVarName(envVar) {
				findings = append(findings, Finding{
					Severity: "warning",
					Category: "secrets",
					Message:  fmt.Sprintf("Integration %s credential '%s' should reference an environment variable", integ.Service, desc),
					Target:   integ.Service,
				})
			}
		}
	}

	return findings
}

// isEnvVarName checks if a string looks like an environment variable name (ALL_CAPS_WITH_UNDERSCORES).
func isEnvVarName(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r != '_' && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

// checkRateLimiting checks if rate limiting is configured.
func checkRateLimiting(app *ir.Application) []Finding {
	var findings []Finding

	if app.Auth == nil {
		findings = append(findings, Finding{
			Severity: "warning",
			Category: "rate-limiting",
			Message:  "No authentication section found — rate limiting is not configured",
			Target:   "application",
		})
		return findings
	}

	hasRateLimit := false
	for _, rule := range app.Auth.Rules {
		if strings.Contains(strings.ToLower(rule.Text), "rate limit") {
			hasRateLimit = true
			break
		}
	}

	if !hasRateLimit {
		findings = append(findings, Finding{
			Severity: "warning",
			Category: "rate-limiting",
			Message:  "No rate limiting is configured for API endpoints",
			Target:   "authentication",
		})
	}

	return findings
}

// checkInputSanitization flags endpoints with text params that lack validation.
func checkInputSanitization(app *ir.Application) []Finding {
	var findings []Finding
	for _, ep := range app.APIs {
		if len(ep.Params) == 0 {
			continue
		}
		for _, p := range ep.Params {
			if !isTextField(app, p.Name) {
				continue
			}
			// Check if this param has any validation rule
			hasValidation := false
			for _, v := range ep.Validation {
				if strings.EqualFold(v.Field, p.Name) {
					hasValidation = true
					break
				}
			}
			if !hasValidation {
				findings = append(findings, Finding{
					Severity: "warning",
					Category: "sanitization",
					Message:  fmt.Sprintf("Endpoint %s accepts text parameter '%s' without input validation — risk of injection", ep.Name, p.Name),
					Target:   ep.Name,
				})
			}
		}
	}
	return findings
}

// isTextField checks if a parameter name maps to a text-type field in any data model.
func isTextField(app *ir.Application, paramName string) bool {
	lower := strings.ToLower(paramName)
	for _, model := range app.Data {
		for _, field := range model.Fields {
			if strings.EqualFold(field.Name, paramName) || strings.ToLower(field.Name) == lower {
				return field.Type == "text" || field.Type == "email" || field.Type == "url"
			}
		}
	}
	// If we can't find the field in any model, assume text fields based on common names
	textNames := []string{"title", "name", "description", "body", "content", "message", "comment", "note", "bio"}
	for _, tn := range textNames {
		if strings.EqualFold(paramName, tn) {
			return true
		}
	}
	return false
}

// checkCORSConfig checks if CORS is properly configured.
func checkCORSConfig(app *ir.Application) []Finding {
	var findings []Finding

	hasCORSMention := false
	hasCORSEnable := false

	if app.Auth != nil {
		for _, rule := range app.Auth.Rules {
			lower := strings.ToLower(rule.Text)
			if strings.Contains(lower, "cors") {
				hasCORSMention = true
				if strings.Contains(lower, "enable cors") {
					hasCORSEnable = true
				}
			}
		}
	}

	if hasCORSMention && !hasCORSEnable {
		findings = append(findings, Finding{
			Severity: "warning",
			Category: "cors",
			Message:  "CORS is mentioned in auth rules but 'enable cors' is not explicitly configured",
			Target:   "authentication",
		})
	} else if !hasCORSMention && len(app.APIs) > 0 {
		findings = append(findings, Finding{
			Severity: "info",
			Category: "cors",
			Message:  "No CORS configuration found — consider adding CORS rules if the API is accessed from a browser",
			Target:   "application",
		})
	}

	return findings
}

// checkSecretPatterns scans environment configs for values that look like secrets.
func checkSecretPatterns(app *ir.Application) []Finding {
	var findings []Finding

	for _, env := range app.Environments {
		for key, val := range env.Config {
			if looksLikeSecret(val) {
				findings = append(findings, Finding{
					Severity: "critical",
					Category: "secrets",
					Message:  fmt.Sprintf("Environment '%s' config '%s' contains a value that looks like a hardcoded secret", env.Name, key),
					Target:   env.Name,
				})
			}
		}
	}

	return findings
}

// looksLikeSecret checks if a value matches known secret prefixes or patterns.
func looksLikeSecret(val string) bool {
	// Known secret prefixes
	prefixes := []string{"sk_live_", "sk_test_", "AKIA", "ghp_", "gho_", "xoxb-", "xoxp-", "pk_live_", "pk_test_", "rk_live_"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(val, prefix) {
			return true
		}
	}

	// Long alphanumeric strings (32+ chars) that aren't env var names or URLs
	if len(val) >= 32 && isAlphanumeric(val) && !isEnvVarName(val) {
		return true
	}

	return false
}

// isAlphanumeric checks if a string consists only of alphanumeric characters and common delimiters.
func isAlphanumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// renderSecurityReport produces a security-report.md.
func renderSecurityReport(app *ir.Application, findings []Finding) string {
	var b strings.Builder

	b.WriteString("# Security Report\n\n")
	b.WriteString("Generated by Human compiler quality engine.\n\n")

	criticals := 0
	warnings := 0
	infos := 0
	for _, f := range findings {
		switch f.Severity {
		case "critical":
			criticals++
		case "warning":
			warnings++
		case "info":
			infos++
		}
	}

	fmt.Fprintf(&b, "**Summary:** %d critical, %d warnings, %d info\n\n", criticals, warnings, infos)

	if len(findings) == 0 {
		b.WriteString("No security issues found.\n")
		return b.String()
	}

	b.WriteString("## Findings\n\n")
	b.WriteString("| Severity | Category | Target | Message |\n")
	b.WriteString("|----------|----------|--------|---------|\n")
	for _, f := range findings {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", f.Severity, f.Category, f.Target, f.Message)
	}
	b.WriteString("\n")

	return b.String()
}
