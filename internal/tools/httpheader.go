package tools

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

var HTTPHeadersT = &Tool{
	ID:          "http-headers",
	Name:        "HTTP Header Inspector",
	Description: "Fetch a URL and display response headers, status, and security analysis",
	Category:    "Network",
	Inputs: []InputDef{
		{
			Label:       "URL",
			Placeholder: "https://example.com",
			Multiline:   false,
			Required:    true,
			AcceptsFile: false,
			FlagName:    "url",
			FlagShort:   "u",
		},
	},
	Run: func(inputs []string) (string, error) {
		rawURL := strings.TrimSpace(resolveInput(inputs, 0))
		if rawURL == "" {
			return "", fmt.Errorf("URL is required")
		}
		if !strings.Contains(rawURL, "://") {
			rawURL = "https://" + rawURL
		}
		return inspectHTTPHeaders(rawURL)
	},
}

func inspectHTTPHeaders(rawURL string) (string, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // capture final response, not follow
		},
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "decode-me/1.0 (SRE utility)")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var sb strings.Builder

	sb.WriteString(section("Response"))
	sb.WriteString(kv("URL", rawURL))
	sb.WriteString(kv("Status", resp.Status))
	sb.WriteString(kv("Protocol", resp.Proto))

	// Group headers
	allHeaders := make(map[string][]string)
	for k, v := range resp.Header {
		allHeaders[k] = v
	}

	// Security headers
	securityHeaders := []string{
		"Strict-Transport-Security",
		"Content-Security-Policy",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"Referrer-Policy",
		"Permissions-Policy",
		"X-XSS-Protection",
		"Cross-Origin-Opener-Policy",
		"Cross-Origin-Embedder-Policy",
		"Cross-Origin-Resource-Policy",
	}

	sb.WriteString(section("Security Headers"))
	for _, h := range securityHeaders {
		if v, ok := resp.Header[h]; ok {
			sb.WriteString(fmt.Sprintf("  ✓ %-42s %s\n", h+":", strings.Join(v, "; ")))
			delete(allHeaders, h)
		} else {
			sb.WriteString(fmt.Sprintf("  ✗ %s (missing)\n", h))
		}
	}

	// Cache headers
	cacheHeaders := []string{
		"Cache-Control", "Expires", "Etag", "Last-Modified",
		"Vary", "Age", "Pragma",
	}
	sb.WriteString(section("Cache Headers"))
	for _, h := range cacheHeaders {
		if v, ok := resp.Header[h]; ok {
			sb.WriteString(fmt.Sprintf("  %-20s: %s\n", h, strings.Join(v, ", ")))
			delete(allHeaders, h)
		}
	}

	// Content headers
	contentHeaders := []string{
		"Content-Type", "Content-Encoding", "Content-Length", "Content-Language",
		"Transfer-Encoding",
	}
	sb.WriteString(section("Content Headers"))
	for _, h := range contentHeaders {
		if v, ok := resp.Header[h]; ok {
			sb.WriteString(fmt.Sprintf("  %-20s: %s\n", h, strings.Join(v, ", ")))
			delete(allHeaders, h)
		}
	}

	// Remaining headers
	if len(allHeaders) > 0 {
		sb.WriteString(section("Other Headers"))
		keys := make([]string, 0, len(allHeaders))
		for k := range allHeaders {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("  %-30s: %s\n", k, strings.Join(allHeaders[k], ", ")))
		}
	}

	// Redirect info
	if loc := resp.Header.Get("Location"); loc != "" {
		sb.WriteString(section("Redirect"))
		sb.WriteString(fmt.Sprintf("  → %s\n", loc))
	}

	return sb.String(), nil
}
