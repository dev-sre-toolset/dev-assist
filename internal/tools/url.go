package tools

import (
	"fmt"
	"net/url"
	"strings"
)

var URLCodecT = &Tool{
	ID:          "url-codec",
	Name:        "URL Encode / Decode",
	Description: "Percent-encode or decode a URL / query string value",
	Category:    "Auth & Tokens",
	Inputs: []InputDef{
		{
			Label:       "Input",
			Placeholder: "Paste URL or query string value",
			Multiline:   true,
			Required:    true,
			AcceptsFile: false,
			FlagName:    "input",
			FlagShort:   "i",
		},
		{
			Label:     "Mode",
			Options:   []string{"auto", "encode", "decode"},
			Default:   "auto",
			FlagName:  "mode",
			FlagShort: "m",
		},
	},
	Run: func(inputs []string) (string, error) {
		raw := strings.TrimSpace(resolveInput(inputs, 0))
		if raw == "" {
			return "", fmt.Errorf("input is required")
		}
		mode := "auto"
		if len(inputs) > 1 && inputs[1] != "" {
			mode = inputs[1]
		}
		return runURLCodec(raw, mode)
	},
}

func runURLCodec(input, mode string) (string, error) {
	var sb strings.Builder

	switch mode {
	case "encode":
		sb.WriteString(section("URL Encoded"))
		sb.WriteString(fmt.Sprintf("  Query  : %s\n", url.QueryEscape(input)))
		sb.WriteString(fmt.Sprintf("  Path   : %s\n", url.PathEscape(input)))

	case "decode":
		qd, err := url.QueryUnescape(input)
		if err != nil {
			return "", fmt.Errorf("URL decode failed: %w", err)
		}
		sb.WriteString(section("URL Decoded"))
		sb.WriteString(fmt.Sprintf("  Result: %s\n", qd))

	default: // auto — show both
		sb.WriteString(section("URL Encoded"))
		sb.WriteString(fmt.Sprintf("  Query: %s\n", url.QueryEscape(input)))
		sb.WriteString(fmt.Sprintf("  Path : %s\n", url.PathEscape(input)))

		if decoded, err := url.QueryUnescape(input); err == nil && decoded != input {
			sb.WriteString(section("URL Decoded"))
			sb.WriteString(fmt.Sprintf("  Result: %s\n", decoded))
		}

		// Parse as full URL if it looks like one
		if strings.Contains(input, "://") || strings.HasPrefix(input, "/") {
			if u, err := url.Parse(input); err == nil {
				sb.WriteString(section("Parsed URL"))
				if u.Scheme != "" {
					sb.WriteString(kv("Scheme", u.Scheme))
				}
				if u.Host != "" {
					sb.WriteString(kv("Host", u.Host))
				}
				if u.Path != "" {
					sb.WriteString(kv("Path", u.Path))
				}
				if u.RawQuery != "" {
					sb.WriteString(kv("Query", u.RawQuery))
					sb.WriteString(section("Query Parameters"))
					for k, vals := range u.Query() {
						sb.WriteString(fmt.Sprintf("  %-20s = %s\n", k, strings.Join(vals, ", ")))
					}
				}
				if u.Fragment != "" {
					sb.WriteString(kv("Fragment", u.Fragment))
				}
			}
		}
	}

	return sb.String(), nil
}
