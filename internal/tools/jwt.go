package tools

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

var JWTDecodeT = &Tool{
	ID:          "jwt-decode",
	Name:        "JWT Decode",
	Description: "Decode and inspect a JWT token (header, payload, expiry status)",
	Category:    "Auth & Tokens",
	Inputs: []InputDef{
		{
			Label:       "JWT Token",
			Placeholder: "Paste JWT token (eyJ...)",
			Multiline:   false,
			Required:    true,
			AcceptsFile: false,
			FlagName:    "token",
			FlagShort:   "t",
		},
	},
	Run: func(inputs []string) (string, error) {
		raw := strings.TrimSpace(resolveInput(inputs, 0))
		if raw == "" {
			return "", fmt.Errorf("JWT token is required")
		}
		return decodeJWT(raw)
	},
}

func decodeJWT(token string) (string, error) {
	token = strings.TrimSpace(token)
	// Strip "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimPrefix(token, "bearer ")

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT: expected 3 parts (header.payload.signature), got %d", len(parts))
	}

	header, err := decodeJWTPart(parts[0])
	if err != nil {
		return "", fmt.Errorf("decode header: %w", err)
	}

	payload, err := decodeJWTPart(parts[1])
	if err != nil {
		return "", fmt.Errorf("decode payload: %w", err)
	}

	var sb strings.Builder

	sb.WriteString(section("Header"))
	sb.WriteString(prettyJSON(header))

	sb.WriteString(section("Payload"))
	sb.WriteString(prettyJSON(payload))

	// Analyse standard claims
	sb.WriteString(section("Claims Analysis"))

	var claims map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &claims); err == nil {
		writeClaimAnalysis(&sb, claims)
	}

	sb.WriteString(section("Signature"))
	sb.WriteString(fmt.Sprintf("  %s\n", parts[2]))
	sb.WriteString("  (signature is NOT verified — this is decode-only)\n")

	return sb.String(), nil
}

func decodeJWTPart(s string) (string, error) {
	// JWT uses base64url without padding
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		// try raw URL encoding
		b, err = base64.RawURLEncoding.DecodeString(s)
		if err != nil {
			return "", err
		}
	}
	return string(b), nil
}

func prettyJSON(raw string) string {
	var v interface{}
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return "  " + raw + "\n"
	}
	b, err := json.MarshalIndent(v, "  ", "  ")
	if err != nil {
		return "  " + raw + "\n"
	}
	return "  " + string(b) + "\n"
}

func writeClaimAnalysis(sb *strings.Builder, claims map[string]interface{}) {
	now := time.Now()

	if sub, ok := claims["sub"].(string); ok {
		sb.WriteString(kv("Subject (sub)", sub))
	}
	if iss, ok := claims["iss"].(string); ok {
		sb.WriteString(kv("Issuer (iss)", iss))
	}
	if aud, ok := claims["aud"]; ok {
		switch v := aud.(type) {
		case string:
			sb.WriteString(kv("Audience (aud)", v))
		case []interface{}:
			parts := make([]string, 0, len(v))
			for _, a := range v {
				if s, ok := a.(string); ok {
					parts = append(parts, s)
				}
			}
			sb.WriteString(kv("Audience (aud)", strings.Join(parts, ", ")))
		}
	}

	if exp, ok := claims["exp"].(float64); ok {
		t := time.Unix(int64(exp), 0)
		status := "✓ valid"
		if now.After(t) {
			status = fmt.Sprintf("⚠ EXPIRED %s ago", formatDuration(now.Sub(t)))
		} else {
			status = fmt.Sprintf("✓ expires in %s", formatDuration(time.Until(t)))
		}
		sb.WriteString(kv("Expires (exp)", fmt.Sprintf("%s  [%s]", t.UTC().Format(time.RFC3339), status)))
	}

	if nbf, ok := claims["nbf"].(float64); ok {
		t := time.Unix(int64(nbf), 0)
		status := "✓ active"
		if now.Before(t) {
			status = fmt.Sprintf("⚠ not valid for another %s", formatDuration(time.Until(t)))
		}
		sb.WriteString(kv("Not Before (nbf)", fmt.Sprintf("%s  [%s]", t.UTC().Format(time.RFC3339), status)))
	}

	if iat, ok := claims["iat"].(float64); ok {
		t := time.Unix(int64(iat), 0)
		sb.WriteString(kv("Issued At (iat)", fmt.Sprintf("%s  [%s ago]", t.UTC().Format(time.RFC3339), formatDuration(now.Sub(t)))))
	}

	if jti, ok := claims["jti"].(string); ok {
		sb.WriteString(kv("JWT ID (jti)", jti))
	}
}

func formatDuration(d time.Duration) string {
	d = d.Abs()
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dd %dh", int(d.Hours()/24), int(d.Hours())%24)
	}
}
