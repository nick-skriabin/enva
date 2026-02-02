// Package shell provides shell export formatting utilities.
package shell

import (
	"fmt"
	"strings"

	"github.com/nick-skriabin/enva/internal/env"
)

// ParsedVar holds parsed value and description.
type ParsedVar struct {
	Value       string
	Description string
}

// FormatExport formats a single variable as a POSIX-sh export line.
// Uses robust single-quote quoting: single quotes around value,
// with embedded single quotes escaped as '\"
func FormatExport(key, value string) string {
	escaped := escapeSingleQuote(value)
	return fmt.Sprintf("export %s='%s'", key, escaped)
}

// FormatExportWithDesc formats an export line with optional description as comment.
func FormatExportWithDesc(key, value, description string) string {
	escaped := escapeSingleQuote(value)
	if description != "" {
		return fmt.Sprintf("export %s='%s' # %s", key, escaped, description)
	}
	return fmt.Sprintf("export %s='%s'", key, escaped)
}

// FormatKeyValue formats a variable as KEY=value (for display).
func FormatKeyValue(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

// FormatExportLines formats all resolved vars as export lines.
func FormatExportLines(ctx *env.ResolveContext) string {
	vars := ctx.GetSortedVars()
	var lines []string
	for _, v := range vars {
		lines = append(lines, FormatExport(v.Key, v.Value))
	}
	return strings.Join(lines, "\n")
}

// FormatKeyValueLines formats all resolved vars as KEY=value lines.
func FormatKeyValueLines(ctx *env.ResolveContext) string {
	vars := ctx.GetSortedVars()
	var lines []string
	for _, v := range vars {
		lines = append(lines, FormatKeyValue(v.Key, v.Value))
	}
	return strings.Join(lines, "\n")
}

// escapeSingleQuote escapes a value for single-quoted shell strings.
// Embedded single quotes become: '\”
// (end quote, escaped single quote, start quote)
func escapeSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", `'\''`)
}

// ParseKeyValue parses a KEY=value line (without description).
// Returns key, value, ok.
func ParseKeyValue(line string) (string, string, bool) {
	key, parsed, ok := ParseKeyValueWithDesc(line)
	return key, parsed.Value, ok
}

// ParseKeyValueWithDesc parses a KEY=value line with optional trailing # description.
// Supports:
// - KEY=value
// - KEY=value # description
// - export KEY=value
// - KEY='value' or KEY="value" (strips surrounding quotes)
// Returns key, ParsedVar{value, description}, ok.
func ParseKeyValueWithDesc(line string) (string, ParsedVar, bool) {
	line = strings.TrimSpace(line)

	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return "", ParsedVar{}, false
	}

	// Strip "export " prefix
	if strings.HasPrefix(line, "export ") {
		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)
	}

	// Find the first '='
	idx := strings.Index(line, "=")
	if idx == -1 {
		return "", ParsedVar{}, false
	}

	key := strings.TrimSpace(line[:idx])
	rest := line[idx+1:]

	// Validate key
	if !IsValidKey(key) {
		return "", ParsedVar{}, false
	}

	// Parse value and description
	value, description := parseValueAndDescription(rest)

	return key, ParsedVar{Value: value, Description: description}, true
}

// parseValueAndDescription extracts value and trailing # description.
// Handles quoted values correctly (# inside quotes is not a comment).
func parseValueAndDescription(s string) (value, description string) {
	if s == "" {
		return "", ""
	}

	// Check if value is quoted (after trimming leading space only for quote check)
	trimmed := strings.TrimSpace(s)
	if len(trimmed) >= 2 && (trimmed[0] == '\'' || trimmed[0] == '"') {
		quote := trimmed[0]
		// Find closing quote
		endQuote := -1
		for i := 1; i < len(trimmed); i++ {
			if trimmed[i] == quote {
				endQuote = i
				break
			}
		}
		if endQuote > 0 {
			value = trimmed[1:endQuote]
			rest := strings.TrimSpace(trimmed[endQuote+1:])
			if strings.HasPrefix(rest, "#") {
				description = strings.TrimSpace(rest[1:])
			}
			return value, description
		}
	}

	// Unquoted value - find # for description (space before # required)
	if idx := strings.Index(s, " #"); idx >= 0 {
		value = s[:idx]
		description = strings.TrimSpace(s[idx+2:])
	} else {
		value = s
	}

	return value, description
}

// stripQuotes removes surrounding single or double quotes.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// IsValidKey checks if a key matches [A-Za-z_][A-Za-z0-9_]*
func IsValidKey(key string) bool {
	if len(key) == 0 {
		return false
	}

	first := key[0]
	if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || first == '_') {
		return false
	}

	for i := 1; i < len(key); i++ {
		c := key[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}

	return true
}

// ParseEnvFile parses multiple KEY=value lines (without descriptions).
// Returns a map of key->value and a list of invalid lines.
// Last value wins for duplicate keys.
func ParseEnvFile(content string) (map[string]string, []string) {
	result := make(map[string]string)
	var invalid []string

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := ParseKeyValue(line)
		if ok {
			result[key] = value
		} else {
			invalid = append(invalid, line)
		}
	}

	return result, invalid
}

// ParseEnvFileWithDesc parses multiple KEY=value lines with descriptions.
// Returns a map of key->ParsedVar and a list of invalid lines.
// Last value wins for duplicate keys.
func ParseEnvFileWithDesc(content string) (map[string]ParsedVar, []string) {
	result := make(map[string]ParsedVar)
	var invalid []string

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, parsed, ok := ParseKeyValueWithDesc(line)
		if ok {
			result[key] = parsed
		} else {
			invalid = append(invalid, line)
		}
	}

	return result, invalid
}
