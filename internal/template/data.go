package template

import (
	"os"
	"strings"
)

// TemplateData provides the data context for template execution.
type TemplateData struct {
	Env map[string]string // Environment variables
}

// NewTemplateData creates a new TemplateData with the current environment.
func NewTemplateData() *TemplateData {
	return &TemplateData{
		Env: environToMap(),
	}
}

// environToMap converts os.Environ() to a map.
func environToMap() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		if idx := strings.Index(e, "="); idx != -1 {
			env[e[:idx]] = e[idx+1:]
		}
	}
	return env
}
