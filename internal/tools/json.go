package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

var JSONYAMLT = &Tool{
	ID:          "json-yaml",
	Name:        "JSON / YAML Prettify & Convert",
	Description: "Pretty-print, validate, and convert between JSON and YAML",
	Category:    "Data",
	Inputs: []InputDef{
		{
			Label:       "Input (JSON or YAML)",
			Placeholder: "Paste JSON or YAML, or enter a file path",
			Multiline:   true,
			Required:    true,
			AcceptsFile: true,
			FlagName:    "input",
			FlagShort:   "i",
		},
		{
			Label:     "Convert to",
			Options:   []string{"auto", "json", "yaml"},
			Default:   "auto",
			FlagName:  "to",
			FlagShort: "t",
		},
	},
	Run: func(inputs []string) (string, error) {
		raw := resolveInput(inputs, 0)
		if raw == "" {
			return "", fmt.Errorf("input is required")
		}
		target := "auto"
		if len(inputs) > 1 && inputs[1] != "" {
			target = strings.ToLower(inputs[1])
		}
		return processJSONYAML(raw, target)
	},
}

func processJSONYAML(input, target string) (string, error) {
	input = strings.TrimSpace(input)

	// Detect format
	isJSON := strings.HasPrefix(input, "{") || strings.HasPrefix(input, "[")

	var parsed interface{}
	var sourceFormat string

	if isJSON {
		sourceFormat = "JSON"
		if err := json.Unmarshal([]byte(input), &parsed); err != nil {
			return "", fmt.Errorf("invalid JSON: %w", err)
		}
	} else {
		sourceFormat = "YAML"
		if err := yaml.Unmarshal([]byte(input), &parsed); err != nil {
			return "", fmt.Errorf("invalid YAML: %w", err)
		}
	}

	var sb strings.Builder
	sb.WriteString(section(fmt.Sprintf("Input: %s  ✓ Valid", sourceFormat)))

	// Determine output format
	if target == "auto" {
		target = sourceFormat // pretty-print same format
	}

	switch strings.ToUpper(target) {
	case "JSON":
		sb.WriteString(section("Pretty JSON"))
		out, err := json.MarshalIndent(parsed, "", "  ")
		if err != nil {
			return "", fmt.Errorf("JSON marshal: %w", err)
		}
		sb.WriteString(string(out))
		sb.WriteString("\n")

	case "YAML":
		sb.WriteString(section("Pretty YAML"))
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(parsed); err != nil {
			return "", fmt.Errorf("YAML marshal: %w", err)
		}
		sb.WriteString(buf.String())
	}

	// Stats
	sb.WriteString(section("Stats"))
	switch v := parsed.(type) {
	case map[string]interface{}:
		sb.WriteString(fmt.Sprintf("  Type : Object  (%d top-level keys)\n", len(v)))
	case []interface{}:
		sb.WriteString(fmt.Sprintf("  Type : Array   (%d elements)\n", len(v)))
	default:
		sb.WriteString(fmt.Sprintf("  Type : %T\n", parsed))
	}
	sb.WriteString(fmt.Sprintf("  Size : %d bytes\n", len(input)))

	return sb.String(), nil
}
