package shell

import (
	"testing"
)

func TestIsValidKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		{"API_KEY", true},
		{"_PRIVATE", true},
		{"lowercase", true},
		{"MixedCase123", true},
		{"A", true},
		{"_", true},
		{"_123", true},

		{"", false},
		{"123KEY", false},
		{"KEY-NAME", false},
		{"KEY.NAME", false},
		{"KEY NAME", false},
		{"KEY=VALUE", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := IsValidKey(tt.key)
			if got != tt.valid {
				t.Errorf("IsValidKey(%q) = %v, want %v", tt.key, got, tt.valid)
			}
		})
	}
}

func TestParseKeyValue(t *testing.T) {
	tests := []struct {
		input    string
		wantKey  string
		wantVal  string
		wantOk   bool
	}{
		// Basic cases
		{"KEY=value", "KEY", "value", true},
		{"API_KEY=secret123", "API_KEY", "secret123", true},
		{"EMPTY=", "EMPTY", "", true},

		// With export prefix
		{"export KEY=value", "KEY", "value", true},
		{"export  KEY=value", "KEY", "value", true},

		// With quotes
		{"KEY='value'", "KEY", "value", true},
		{"KEY=\"value\"", "KEY", "value", true},
		{"KEY='value with spaces'", "KEY", "value with spaces", true},

		// With whitespace (line is trimmed, but value after = is preserved)
		{"  KEY=value  ", "KEY", "value", true},
		{"KEY= value", "KEY", " value", true},

		// Value with equals sign
		{"KEY=val=ue", "KEY", "val=ue", true},
		{"URL=http://example.com?a=1&b=2", "URL", "http://example.com?a=1&b=2", true},

		// Invalid cases
		{"", "", "", false},
		{"# comment", "", "", false},
		{"NOEQUALS", "", "", false},
		{"123=value", "", "", false},
		{"KEY-NAME=value", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			key, val, ok := ParseKeyValue(tt.input)
			if ok != tt.wantOk {
				t.Errorf("ParseKeyValue(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
				return
			}
			if ok {
				if key != tt.wantKey {
					t.Errorf("ParseKeyValue(%q) key = %q, want %q", tt.input, key, tt.wantKey)
				}
				if val != tt.wantVal {
					t.Errorf("ParseKeyValue(%q) val = %q, want %q", tt.input, val, tt.wantVal)
				}
			}
		})
	}
}

func TestFormatExport(t *testing.T) {
	tests := []struct {
		key      string
		value    string
		expected string
	}{
		{"KEY", "value", "export KEY='value'"},
		{"KEY", "", "export KEY=''"},
		{"KEY", "hello world", "export KEY='hello world'"},
		{"KEY", "it's a test", "export KEY='it'\\''s a test'"},
		{"KEY", "multi'quote'test", "export KEY='multi'\\''quote'\\''test'"},
		{"KEY", "special$chars", "export KEY='special$chars'"},
		{"KEY", "back\\slash", "export KEY='back\\slash'"},
	}

	for _, tt := range tests {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			got := FormatExport(tt.key, tt.value)
			if got != tt.expected {
				t.Errorf("FormatExport(%q, %q) = %q, want %q", tt.key, tt.value, got, tt.expected)
			}
		})
	}
}

func TestFormatKeyValue(t *testing.T) {
	got := FormatKeyValue("API_KEY", "secret")
	want := "API_KEY=secret"
	if got != want {
		t.Errorf("FormatKeyValue = %q, want %q", got, want)
	}
}

func TestParseEnvFile(t *testing.T) {
	content := `
# This is a comment
API_KEY=secret123
DATABASE_URL=postgres://localhost/db

export DEBUG=true
QUOTED='single quoted'

# Another comment
EMPTY=
`
	vars, invalid := ParseEnvFile(content)

	if len(invalid) != 0 {
		t.Errorf("ParseEnvFile returned invalid lines: %v", invalid)
	}

	expected := map[string]string{
		"API_KEY":      "secret123",
		"DATABASE_URL": "postgres://localhost/db",
		"DEBUG":        "true",
		"QUOTED":       "single quoted",
		"EMPTY":        "",
	}

	if len(vars) != len(expected) {
		t.Errorf("ParseEnvFile returned %d vars, want %d", len(vars), len(expected))
	}

	for k, want := range expected {
		got, ok := vars[k]
		if !ok {
			t.Errorf("ParseEnvFile missing key %q", k)
			continue
		}
		if got != want {
			t.Errorf("ParseEnvFile[%q] = %q, want %q", k, got, want)
		}
	}
}

func TestParseEnvFileInvalid(t *testing.T) {
	content := `
VALID=value
123INVALID=value
ALSO-INVALID=value
VALID2=value2
`
	vars, invalid := ParseEnvFile(content)

	if len(vars) != 2 {
		t.Errorf("ParseEnvFile returned %d vars, want 2", len(vars))
	}

	if len(invalid) != 2 {
		t.Errorf("ParseEnvFile returned %d invalid, want 2", len(invalid))
	}
}

func TestParseEnvFileDuplicates(t *testing.T) {
	content := `
KEY=first
KEY=second
KEY=third
`
	vars, _ := ParseEnvFile(content)

	if vars["KEY"] != "third" {
		t.Errorf("ParseEnvFile duplicate handling: got %q, want 'third'", vars["KEY"])
	}
}
