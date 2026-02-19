package ir

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ToJSON serializes the IR Application to formatted JSON.
func ToJSON(app *Application) ([]byte, error) {
	return json.MarshalIndent(app, "", "  ")
}

// FromJSON deserializes an IR Application from JSON.
func FromJSON(data []byte) (*Application, error) {
	app := &Application{}
	if err := json.Unmarshal(data, app); err != nil {
		return nil, fmt.Errorf("ir: invalid JSON: %w", err)
	}
	return app, nil
}

// ToYAML serializes the IR Application to YAML format.
// Uses a zero-dependency approach: JSON round-trip then YAML formatting.
func ToYAML(app *Application) (string, error) {
	jsonBytes, err := json.Marshal(app)
	if err != nil {
		return "", fmt.Errorf("ir: JSON marshal failed: %w", err)
	}

	var data interface{}
	dec := json.NewDecoder(bytes.NewReader(jsonBytes))
	dec.UseNumber()
	if err := dec.Decode(&data); err != nil {
		return "", fmt.Errorf("ir: JSON decode failed: %w", err)
	}

	var buf strings.Builder
	writeYAML(&buf, data, 0)
	return buf.String(), nil
}

// writeYAML recursively formats a generic value as YAML.
func writeYAML(buf *strings.Builder, v interface{}, indent int) {
	switch val := v.(type) {
	case nil:
		buf.WriteString("null")

	case bool:
		if val {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}

	case json.Number:
		buf.WriteString(val.String())

	case string:
		writeYAMLString(buf, val)

	case []interface{}:
		writeYAMLArray(buf, val, indent)

	case map[string]interface{}:
		writeYAMLMap(buf, val, indent)
	}
}

// writeYAMLString writes a YAML string, quoting when necessary.
func writeYAMLString(buf *strings.Builder, s string) {
	if s == "" {
		buf.WriteString(`""`)
		return
	}
	if needsYAMLQuoting(s) {
		buf.WriteString(fmt.Sprintf("%q", s))
		return
	}
	buf.WriteString(s)
}

// needsYAMLQuoting returns true if a string needs quotes in YAML.
func needsYAMLQuoting(s string) bool {
	if s == "" || s == "true" || s == "false" || s == "null" || s == "~" {
		return true
	}
	// Quote if starts with special characters
	if s[0] == '#' || s[0] == '&' || s[0] == '*' || s[0] == '!' ||
		s[0] == '|' || s[0] == '>' || s[0] == '\'' || s[0] == '"' ||
		s[0] == '%' || s[0] == '@' || s[0] == '{' || s[0] == '[' {
		return true
	}
	// Quote if contains : followed by space, or has newlines
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' || s[i] == '\r' {
			return true
		}
		if s[i] == ':' && i+1 < len(s) && s[i+1] == ' ' {
			return true
		}
	}
	// Quote if it looks like a number
	if looksLikeNumber(s) {
		return true
	}
	return false
}

// looksLikeNumber returns true if the string could be parsed as a YAML number.
func looksLikeNumber(s string) bool {
	if len(s) == 0 {
		return false
	}
	start := 0
	if s[0] == '-' || s[0] == '+' {
		start = 1
	}
	if start >= len(s) {
		return false
	}
	hasDigit := false
	hasDot := false
	for i := start; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			hasDigit = true
		} else if s[i] == '.' && !hasDot {
			hasDot = true
		} else {
			return false
		}
	}
	return hasDigit
}

// writeYAMLArray writes a YAML array.
func writeYAMLArray(buf *strings.Builder, arr []interface{}, indent int) {
	if len(arr) == 0 {
		buf.WriteString("[]")
		return
	}

	prefix := strings.Repeat("  ", indent)

	for i, item := range arr {
		if i > 0 {
			// Items after the first start on a new line
		}
		buf.WriteString("\n")
		buf.WriteString(prefix)
		buf.WriteString("- ")

		switch val := item.(type) {
		case map[string]interface{}:
			// Inline first key, rest indented under the dash
			writeYAMLMapInline(buf, val, indent+1)
		default:
			writeYAML(buf, item, indent+1)
		}
	}
}

// writeYAMLMap writes a YAML map with sorted keys.
func writeYAMLMap(buf *strings.Builder, m map[string]interface{}, indent int) {
	if len(m) == 0 {
		buf.WriteString("{}")
		return
	}

	keys := sortedKeys(m)
	prefix := strings.Repeat("  ", indent)

	for i, key := range keys {
		val := m[key]

		if i > 0 {
			buf.WriteString("\n")
			buf.WriteString(prefix)
		}

		buf.WriteString(key)
		buf.WriteString(":")

		switch v := val.(type) {
		case map[string]interface{}:
			if len(v) == 0 {
				buf.WriteString(" {}")
			} else {
				buf.WriteString("\n")
				buf.WriteString(strings.Repeat("  ", indent+1))
				writeYAMLMap(buf, v, indent+1)
			}

		case []interface{}:
			writeYAMLArray(buf, v, indent+1)

		default:
			buf.WriteString(" ")
			writeYAML(buf, val, indent+1)
		}
	}
}

// writeYAMLMapInline writes a map where the first key-value is on the
// same line as the array dash, and subsequent keys are indented.
func writeYAMLMapInline(buf *strings.Builder, m map[string]interface{}, indent int) {
	if len(m) == 0 {
		buf.WriteString("{}")
		return
	}

	keys := sortedKeys(m)
	prefix := strings.Repeat("  ", indent)

	for i, key := range keys {
		val := m[key]

		if i > 0 {
			buf.WriteString("\n")
			buf.WriteString(prefix)
		}

		buf.WriteString(key)
		buf.WriteString(":")

		switch v := val.(type) {
		case map[string]interface{}:
			if len(v) == 0 {
				buf.WriteString(" {}")
			} else {
				buf.WriteString("\n")
				buf.WriteString(strings.Repeat("  ", indent+1))
				writeYAMLMap(buf, v, indent+1)
			}

		case []interface{}:
			writeYAMLArray(buf, v, indent+1)

		default:
			buf.WriteString(" ")
			writeYAML(buf, val, indent+1)
		}
	}
}

// ── Key ordering ──

// topLevelKeyOrder defines the preferred ordering for Application-level keys.
var topLevelKeyOrder = map[string]int{
	"name": 0, "platform": 1, "config": 2,
	"data": 3, "pages": 4, "components": 5,
	"apis": 6, "policies": 7, "workflows": 8,
	"theme": 9, "auth": 10, "database": 11,
	"integrations": 12, "environments": 13,
	"error_handlers": 14, "pipelines": 15,
}

// commonKeyOrder defines ordering for commonly used keys in nested objects.
var commonKeyOrder = map[string]int{
	"name": 0, "type": 1, "kind": 2,
	"service": 3, "trigger": 4, "condition": 5,
	"engine": 6, "entity": 7, "field": 8,
	"rule": 9, "value": 10, "text": 11,
	"target": 12, "through": 13,
	"required": 14, "unique": 15, "encrypted": 16,
	"auth": 17, "params": 18, "validation": 19,
	"steps": 20, "content": 21, "props": 22,
	"fields": 23, "relations": 24,
	"permissions": 25, "restrictions": 26,
	"methods": 27, "rules": 28,
	"indexes": 29, "credentials": 30, "purpose": 31,
	"config": 32, "options": 33,
	"colors": 34, "fonts": 35,
	"enum_values": 36, "default": 37, "message": 38,
	"frontend": 39, "backend": 40, "database": 41, "deploy": 42,
	"provider": 43,
}

// sortedKeys returns map keys sorted by a priority ordering.
func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		pi, oki := topLevelKeyOrder[keys[i]]
		pj, okj := topLevelKeyOrder[keys[j]]
		if !oki {
			pi2, ok2 := commonKeyOrder[keys[i]]
			if ok2 {
				pi = pi2 + 100
				oki = true
			}
		}
		if !okj {
			pj2, ok2 := commonKeyOrder[keys[j]]
			if ok2 {
				pj = pj2 + 100
				okj = true
			}
		}
		if oki && okj {
			return pi < pj
		}
		if oki {
			return true
		}
		if okj {
			return false
		}
		return keys[i] < keys[j]
	})
	return keys
}
