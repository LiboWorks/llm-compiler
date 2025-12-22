package runtime_test

import (
	"testing"

	"github.com/LiboWorks/llm-compiler/internal/runtime"
)

func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple variable",
			input:    "Hello {{name}}",
			vars:     map[string]string{"name": "World"},
			expected: "Hello World",
		},
		{
			name:     "multiple variables",
			input:    "{{greeting}}, {{name}}!",
			vars:     map[string]string{"greeting": "Hello", "name": "Alice"},
			expected: "Hello, Alice!",
		},
		{
			name:     "no variables",
			input:    "Plain text",
			vars:     map[string]string{},
			expected: "Plain text",
		},
		{
			name:     "missing variable",
			input:    "Hello {{name}}",
			vars:     map[string]string{},
			expected: "Hello ",
		},
		{
			name:     "variable with spaces",
			input:    "Hello {{ name }}",
			vars:     map[string]string{"name": "World"},
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := runtime.RenderTemplate(tt.input, tt.vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("RenderTemplate() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestEvalCondition(t *testing.T) {
	ctx := runtime.NewRuntimeContext()
	ctx.Set("flag", "yes")
	ctx.Set("count", "5")
	ctx.Set("empty", "")

	tests := []struct {
		name      string
		condition string
		expected  bool
	}{
		{
			name:      "equals true",
			condition: "{{flag}} == 'yes'",
			expected:  true,
		},
		{
			name:      "equals false",
			condition: "{{flag}} == 'no'",
			expected:  false,
		},
		{
			name:      "with double quotes",
			condition: `{{flag}} == "yes"`,
			expected:  true,
		},
		{
			name:      "numeric string",
			condition: "{{count}} == '5'",
			expected:  true,
		},
		{
			name:      "empty value",
			condition: "{{empty}} == ''",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runtime.EvalCondition(ctx, tt.condition)
			if result != tt.expected {
				t.Errorf("EvalCondition(%q) = %v, want %v", tt.condition, result, tt.expected)
			}
		})
	}
}

func TestSanitizeForShell(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "with newlines",
			input:    "hello\nworld",
			expected: "hello world",
		},
		{
			name:     "with tabs",
			input:    "hello\tworld",
			expected: "hello world",
		},
		{
			name:     "multiple whitespace",
			input:    "hello   world",
			expected: "hello world",
		},
		{
			name:     "with quotes",
			input:    `say "hello"`,
			expected: `say \"hello\"`,
		},
		{
			name:     "leading/trailing whitespace",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runtime.SanitizeForShell(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeForShell(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRuntimeContext(t *testing.T) {
	ctx := runtime.NewRuntimeContext()

	// Test Set and Get
	ctx.Set("key1", "value1")
	if got := ctx.Get("key1"); got != "value1" {
		t.Errorf("Get(key1) = %q, want %q", got, "value1")
	}

	// Test missing key
	if got := ctx.Get("missing"); got != "" {
		t.Errorf("Get(missing) = %q, want empty string", got)
	}

	// Test overwrite
	ctx.Set("key1", "value2")
	if got := ctx.Get("key1"); got != "value2" {
		t.Errorf("Get(key1) after overwrite = %q, want %q", got, "value2")
	}
}
