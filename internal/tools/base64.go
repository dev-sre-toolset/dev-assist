package tools

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"
)

var Base64T = &Tool{
	ID:          "base64",
	Name:        "Base64 Encode / Decode",
	Description: "Encode or decode base64 (standard & URL-safe variants)",
	Category:    "Auth & Tokens",
	Inputs: []InputDef{
		{
			Label:       "Input",
			Placeholder: "Paste raw text or base64 value",
			Multiline:   true,
			Required:    true,
			AcceptsFile: true,
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
		raw := resolveInput(inputs, 0)
		if raw == "" {
			return "", fmt.Errorf("input is required")
		}
		mode := "auto"
		if len(inputs) > 1 && inputs[1] != "" {
			mode = inputs[1]
		}
		return runBase64(raw, mode)
	},
}

func runBase64(input, mode string) (string, error) {
	var sb strings.Builder

	switch mode {
	case "encode":
		showEncoded(&sb, input)
	case "decode":
		if err := showDecoded(&sb, input); err != nil {
			return "", err
		}
	default: // auto — show encode AND attempt decode
		showEncoded(&sb, input)
		// Try decode; only show if it succeeds and differs from input
		decoded, std, safe := tryDecode(input)
		if decoded != "" && decoded != input {
			sb.WriteString(section("Decoded"))
			if std {
				sb.WriteString("  Variant : Standard base64\n")
			} else if safe {
				sb.WriteString("  Variant : URL-safe base64\n")
			}
			if utf8.ValidString(decoded) {
				sb.WriteString(fmt.Sprintf("  Result  : %s\n", decoded))
			} else {
				sb.WriteString(fmt.Sprintf("  Result  : (binary, %d bytes)\n", len(decoded)))
			}
		}
	}

	return sb.String(), nil
}

func showEncoded(sb *strings.Builder, input string) {
	std := base64.StdEncoding.EncodeToString([]byte(input))
	safe := base64.URLEncoding.EncodeToString([]byte(input))
	stdNoPad := base64.RawStdEncoding.EncodeToString([]byte(input))

	sb.WriteString(section("Encoded"))
	sb.WriteString(fmt.Sprintf("  Standard       : %s\n", std))
	if safe != std {
		sb.WriteString(fmt.Sprintf("  URL-safe       : %s\n", safe))
	}
	if stdNoPad != std {
		sb.WriteString(fmt.Sprintf("  No-padding     : %s\n", stdNoPad))
	}
}

func showDecoded(sb *strings.Builder, input string) error {
	decoded, std, safe := tryDecode(strings.TrimSpace(input))
	if decoded == "" {
		return fmt.Errorf("not valid base64: tried standard and URL-safe variants")
	}
	variant := "Standard"
	if safe && !std {
		variant = "URL-safe"
	}
	sb.WriteString(section("Decoded"))
	sb.WriteString(fmt.Sprintf("  Variant: %s base64\n", variant))
	if utf8.ValidString(decoded) {
		sb.WriteString(fmt.Sprintf("  Result : %s\n", decoded))
	} else {
		sb.WriteString(fmt.Sprintf("  Result : (binary data, %d bytes)\n", len(decoded)))
	}
	return nil
}

// tryDecode attempts both standard and URL-safe decoding.
// Returns (decoded string, wasStd, wasSafe).
func tryDecode(s string) (string, bool, bool) {
	s = strings.TrimSpace(s)
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return string(b), true, false
	}
	if b, err := base64.URLEncoding.DecodeString(s); err == nil {
		return string(b), false, true
	}
	if b, err := base64.RawStdEncoding.DecodeString(s); err == nil {
		return string(b), true, false
	}
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return string(b), false, true
	}
	return "", false, false
}
