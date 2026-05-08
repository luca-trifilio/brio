package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luca-trifilio/brio/internal/config"
)

// Parse extracts a key→value map from hook output.
//
//   - type=stdout: parses stdout bytes as dotenv (KEY=VALUE lines)
//   - type=file:   reads output.path from disk, then dispatches on output.format
func Parse(output config.Output, stdout []byte) (map[string]string, error) {
	switch output.Type {
	case "stdout":
		return parseDotenv(stdout)
	case "file":
		path, err := expandPath(output.Path)
		if err != nil {
			return nil, fmt.Errorf("hooks: expand output path: %w", err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("hooks: read output file %q: %w", path, err)
		}
		return parseFormat(output.Format, data)
	default:
		return nil, fmt.Errorf("hooks: unknown output type %q", output.Type)
	}
}

// parseFormat dispatches to the correct parser based on the declared format.
func parseFormat(format string, data []byte) (map[string]string, error) {
	switch format {
	case "", "dotenv":
		return parseDotenv(data)
	case "json":
		return parseJSON(data)
	case "yaml":
		return parseYAML(data)
	case "bruno-env":
		return parseBrunoEnv(data)
	default:
		return nil, fmt.Errorf("hooks: unknown output format %q", format)
	}
}

// parseDotenv reads KEY=VALUE lines.
// Blank lines and lines starting with '#' are ignored.
// Surrounding quotes on the value are stripped.
func parseDotenv(data []byte) (map[string]string, error) {
	out := make(map[string]string)
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = stripQuotes(strings.TrimSpace(v))
	}
	return out, nil
}

// parseJSON reads a flat {"KEY": "value"} JSON object.
func parseJSON(data []byte) (map[string]string, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("hooks: json parse: %w", err)
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out, nil
}

// parseYAML reads a flat "KEY: value" YAML mapping (one pair per line).
// Blank lines and lines starting with '#' are ignored.
func parseYAML(data []byte) (map[string]string, error) {
	out := make(map[string]string)
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = stripQuotes(strings.TrimSpace(v))
	}
	return out, nil
}

// brunoEnvFile mirrors the Bruno environment YAML schema.
//
//	variables:
//	  - name: KEY
//	    value: VALUE
type brunoEnvFile struct {
	Variables []brunoVar
}
type brunoVar struct {
	Name  string
	Value string
}

// parseBrunoEnv reads a Bruno environment YAML file.
// Uses a minimal hand-rolled parser to avoid pulling in a YAML library.
func parseBrunoEnv(data []byte) (map[string]string, error) {
	out := make(map[string]string)

	var currentName string
	inVariables := false

	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)

		if line == "variables:" {
			inVariables = true
			continue
		}
		if !inVariables {
			continue
		}
		// new list item resets the pair
		if strings.HasPrefix(line, "- ") {
			currentName = ""
			line = strings.TrimPrefix(line, "- ")
		}

		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key := strings.TrimSpace(k)
		val := stripQuotes(strings.TrimSpace(v))

		switch key {
		case "name":
			currentName = val
		case "value":
			if currentName != "" {
				out[currentName] = val
			}
		}
	}
	return out, nil
}

// expandPath expands a leading ~ to the user home directory.
func expandPath(p string) (string, error) {
	if !strings.HasPrefix(p, "~") {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, p[1:]), nil
}

// stripQuotes removes a matching pair of surrounding single or double quotes.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
