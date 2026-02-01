package template

import (
	"testing"
)

func TestHasTemplateSyntax(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty string", "", false},
		{"plain string", "hello world", false},
		{"single braces", "{hello}", false},
		{"simple template", "{{ .Foo }}", true},
		{"template with function", "{{ env \"HOME\" }}", true},
		{"mixed content", "prefix-{{ .Value }}-suffix", true},
		{"multiple templates", "{{ .A }} and {{ .B }}", true},
		{"nested braces - invalid", "{{{{", false}, // No content between {{ }}
		{"double open only", "{{ {{", false},       // Malformed
		{"empty template", "{{}}", false},          // No content
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasTemplateSyntax(tt.input)
			if got != tt.want {
				t.Errorf("HasTemplateSyntax(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestProcessString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		data    interface{}
		want    string
		wantErr bool
	}{
		{
			name:  "no template - passthrough",
			input: "plain string",
			data:  nil,
			want:  "plain string",
		},
		{
			name:  "simple variable",
			input: "Hello {{ .Name }}",
			data:  map[string]string{"Name": "World"},
			want:  "Hello World",
		},
		{
			name:  "sprig function - upper",
			input: "{{ upper .Name }}",
			data:  map[string]string{"Name": "test"},
			want:  "TEST",
		},
		{
			name:  "sprig function - default",
			input: "{{ .Missing | default \"fallback\" }}",
			data:  map[string]string{},
			want:  "fallback",
		},
		{
			name:    "invalid template function",
			input:   "{{ unknownFunc .Name }}",
			data:    map[string]string{"Name": "test"},
			wantErr: true,
		},
		{
			name:  "env map access",
			input: "{{ .Env.TEST_VAR }}",
			data:  &TemplateData{Env: map[string]string{"TEST_VAR": "test_value"}},
			want:  "test_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProcessString(tt.input, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ProcessString() = %q, want %q", got, tt.want)
			}
		})
	}
}
