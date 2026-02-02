// Package shell provides shell export formatting utilities.
package shell

import (
	"fmt"
	"strings"

	"github.com/nick-skriabin/enva/internal/env"
)

// FormatExport formats a single variable as a POSIX-sh export line.
// Uses robust single-quote quoting: single quotes around value,
// with embedded single quotes escaped as '\”
func FormatExport(key, value string) string {
	escaped := escapeSingleQuote(value)
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

// ParseKeyValue parses a KEY=value line.
// Supports:
// - KEY=value
// - export KEY=value (strips "export ")
// - KEY='value' or KEY="value" (strips surrounding quotes)
// Returns key, value, ok.
func ParseKeyValue(line string) (string, string, bool) {
	line = strings.TrimSpace(line)

	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}

	// Strip "export " prefix
	if strings.HasPrefix(line, "export ") {
		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)
	}

	// Find the first '='
	idx := strings.Index(line, "=")
	if idx == -1 {
		return "", "", false
	}

	key := strings.TrimSpace(line[:idx])
	value := line[idx+1:]

	// Validate key
	if !IsValidKey(key) {
		return "", "", false
	}

	// Strip surrounding quotes from value
	value = stripQuotes(value)

	return key, value, true
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

// ParseEnvFile parses multiple KEY=value lines.
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
