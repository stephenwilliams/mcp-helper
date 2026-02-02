package template

import (
	"fmt"

	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// ProcessServer processes all template fields in a server configuration.
// It returns a new Server with processed values, leaving the original unchanged.
//
// Templated fields: Command, Args, URL, EnvVar.Default
// Non-templated fields: Description, Transport, EnvVar.Description, EnvVar.Required
//
// Rationale for non-templated fields:
// - Description fields are user-facing documentation and should be static
// - Transport is a fixed enum value
// - Required is a boolean flag
func ProcessServer(server *config.Server, data *TemplateData) (*config.Server, error) {
	if server == nil {
		return nil, nil
	}

	processed := &config.Server{
		Description: server.Description, // Not templated (documentation)
		Transport:   server.Transport,   // Not templated (enum)
		Env:         make(map[string]config.EnvVar, len(server.Env)),
	}

	// Process Command
	cmd, err := ProcessString(server.Command, data)
	if err != nil {
		return nil, fmt.Errorf("processing command %q: %w", server.Command, err)
	}
	processed.Command = cmd

	// Process Args
	processed.Args = make([]string, len(server.Args))
	for i, arg := range server.Args {
		processedArg, err := ProcessString(arg, data)
		if err != nil {
			return nil, fmt.Errorf("processing arg[%d] %q: %w", i, arg, err)
		}
		processed.Args[i] = processedArg
	}

	// Process URL
	url, err := ProcessString(server.URL, data)
	if err != nil {
		return nil, fmt.Errorf("processing URL %q: %w", server.URL, err)
	}
	processed.URL = url

	// Process Headers (only values are templated, names are literal)
	if server.Headers != nil {
		processed.Headers = make(map[string]string, len(server.Headers))
		for name, value := range server.Headers {
			processedValue, err := ProcessString(value, data)
			if err != nil {
				return nil, fmt.Errorf("processing header %q value %q: %w", name, value, err)
			}
			processed.Headers[name] = processedValue
		}
	}

	// Process EnvVar defaults (CRITICAL: must happen before CollectEnvVars)
	for name, envVar := range server.Env {
		processedDefault, err := ProcessString(envVar.Default, data)
		if err != nil {
			return nil, fmt.Errorf("processing env %q default %q: %w", name, envVar.Default, err)
		}
		processed.Env[name] = config.EnvVar{
			Required:    envVar.Required,
			Description: envVar.Description, // Not templated (documentation)
			Default:     processedDefault,
		}
	}

	return processed, nil
}
