package tools

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var SAMLDecodeT = &Tool{
	ID:          "saml-decode",
	Name:        "SAML Decode",
	Category:    "Auth & Tokens",
	Description: "Decode SAMLRequest (base64+deflate) or SAMLResponse (base64). Accepts a full URL or a bare value.",
	Inputs: []InputDef{
		{
			Label:       "URL or SAMLRequest / SAMLResponse value",
			Placeholder: "https://idp.corp.com/SSO?SAMLRequest=jVJdb…  OR  jVJdb6MwEPwr…",
			Multiline:   true,
			Required:    true,
			AcceptsFile: false,
			FlagName:    "input",
			FlagShort:   "i",
		},
		{
			Label:     "Type",
			Options:   []string{"auto-detect", "SAMLRequest", "SAMLResponse"},
			Default:   "auto-detect",
			FlagName:  "type",
			FlagShort: "t",
		},
	},
	Run: func(inputs []string) (string, error) {
		raw := strings.TrimSpace(resolveInput(inputs, 0))
		if raw == "" {
			return "", fmt.Errorf("input is required")
		}
		typeHint := "auto-detect"
		if len(inputs) > 1 && strings.TrimSpace(inputs[1]) != "" {
			typeHint = strings.TrimSpace(inputs[1])
		}
		return decodeSAML(raw, typeHint)
	},
}

func decodeSAML(raw, typeHint string) (string, error) {
	// Step 1 — if a full URL was pasted, extract the SAMLRequest / SAMLResponse param.
	// This also tells us the type when auto-detecting.
	detectedType := ""
	b64value, extractedParam := extractSAMLParam(raw)
	if extractedParam != "" {
		detectedType = extractedParam // "SAMLRequest" or "SAMLResponse"
		raw = b64value
	}

	// Resolve effective type:
	//   explicit flag > detected from URL > fall back to auto
	isRequest := false
	switch {
	case strings.Contains(typeHint, "SAMLRequest"):
		isRequest = true
		detectedType = "SAMLRequest"
	case strings.Contains(typeHint, "SAMLResponse"):
		isRequest = false
		detectedType = "SAMLResponse"
	case detectedType == "SAMLRequest":
		isRequest = true
	case detectedType == "SAMLResponse":
		isRequest = false
	default:
		// auto-detect: try deflate, fall back to raw XML
		detectedType = "auto-detect"
	}

	// Step 2 — percent-decode (%xx only — PathUnescape never touches '+').
	pctDecoded, err := url.PathUnescape(raw)
	if err != nil {
		pctDecoded = raw
	}
	pctDecoded = strings.TrimSpace(pctDecoded)

	// Step 3 — base64 decode (standard alphabet; SAMLRequest/Response both use it).
	decoded, err := base64.StdEncoding.DecodeString(pctDecoded)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(pctDecoded)
		if err != nil {
			return "", fmt.Errorf(
				"base64 decode failed: %w\n\n"+
					"  Tip: paste the raw param value (e.g. jVJdb6Mw…), not the whole URL,\n"+
					"  or paste the entire URL and let the tool extract the parameter.",
				err,
			)
		}
	}

	// Step 4 — inflate / decompress.
	//   SAMLRequest  → always deflated (DEFLATE, RFC 1951 — no zlib wrapper)
	//   SAMLResponse → raw XML (no compression)
	var xmlBytes []byte
	var wasInflated bool

	if isRequest {
		// Explicit SAMLRequest: must be deflated.
		xmlBytes, err = inflate(decoded)
		if err != nil {
			return "", fmt.Errorf("SAMLRequest DEFLATE decompress failed: %w", err)
		}
		wasInflated = true
	} else if detectedType == "SAMLResponse" {
		// Explicit SAMLResponse: skip deflate.
		xmlBytes = decoded
		wasInflated = false
	} else {
		// auto-detect: try deflate first; fall back to raw.
		xmlBytes, wasInflated, err = tryInflate(decoded)
		if err != nil {
			return "", fmt.Errorf("inflate/decompress failed: %w", err)
		}
	}

	// Step 5 — syntax-highlighted pretty-print XML.
	pretty := prettyColorXML(xmlBytes)

	// Determine the label to show in output.
	outputLabel := detectedType
	if outputLabel == "auto-detect" {
		if wasInflated {
			outputLabel = "SAMLRequest (auto-detected)"
		} else {
			outputLabel = "SAMLResponse (auto-detected)"
		}
	}

	var sb strings.Builder
	sb.WriteString(section("SAML Decode — " + outputLabel))
	if wasInflated {
		sb.WriteString("  Encoding : base64  →  DEFLATE decompress  →  XML\n")
		sb.WriteString("  Type     : SAMLRequest\n")
	} else {
		sb.WriteString("  Encoding : base64  →  XML  (no compression)\n")
		sb.WriteString("  Type     : SAMLResponse\n")
	}
	sb.WriteString(section("XML"))
	sb.WriteString(pretty)
	sb.WriteString("\n")
	return sb.String(), nil
}

// extractSAMLParam checks whether input is a full URL and, if so, extracts the
// SAMLRequest or SAMLResponse query parameter value (still percent-encoded).
// Returns (rawParamValue, paramName). If not a URL, returns ("", "").
func extractSAMLParam(input string) (value, paramName string) {
	if !strings.Contains(input, "://") {
		return "", ""
	}
	u, err := url.Parse(input)
	if err != nil {
		return "", ""
	}
	// url.Parse already percent-decoded the query for us — we want the raw
	// value so that our own PathUnescape step runs uniformly.
	raw := u.RawQuery
	for _, pair := range strings.Split(raw, "&") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k, _ := url.PathUnescape(kv[0])
		if k == "SAMLRequest" || k == "SAMLResponse" {
			return kv[1], k // still percent-encoded — decodeSAML will PathUnescape it
		}
	}
	return "", ""
}

// inflate performs raw DEFLATE decompression (RFC 1951, no zlib header).
// Used when the type is explicitly known to be SAMLRequest.
func inflate(data []byte) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(data))
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// tryInflate attempts raw DEFLATE decompression; returns the raw data if it
// looks like XML already (SAMLResponse path). Returns (data, wasInflated, err).
func tryInflate(data []byte) ([]byte, bool, error) {
	r := flate.NewReader(bytes.NewReader(data))
	defer r.Close()
	out, err := io.ReadAll(r)
	if err == nil && len(out) > 0 {
		return out, true, nil
	}
	// Decompression failed — check if the raw bytes are already XML.
	if isXML(data) {
		return data, false, nil
	}
	return nil, false, fmt.Errorf(
		"data is neither valid DEFLATE-compressed data nor raw XML; " +
			"if this is a SAMLResponse, use --type 'SAMLResponse (base64 only)'")
}

func isXML(data []byte) bool {
	s := strings.TrimSpace(string(data))
	return strings.HasPrefix(s, "<") || strings.HasPrefix(s, "<?xml")
}

// prettyColorXML pretty-prints XML with terminal syntax highlighting.
// Falls back to raw bytes on parse errors.
func prettyColorXML(data []byte) string {
	tagS     := lipgloss.NewStyle().Foreground(lipgloss.Color("#00f5ff")).Bold(true)
	nsS      := lipgloss.NewStyle().Foreground(lipgloss.Color("#8892a4"))
	attrS    := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffe600"))
	valS     := lipgloss.NewStyle().Foreground(lipgloss.Color("#39ff14"))
	textS    := lipgloss.NewStyle().Foreground(lipgloss.Color("#e2e8f0"))
	commentS := lipgloss.NewStyle().Foreground(lipgloss.Color("#4a5568")).Italic(true)
	procS    := lipgloss.NewStyle().Foreground(lipgloss.Color("#bf5fff"))
	punctS   := lipgloss.NewStyle().Foreground(lipgloss.Color("#8892a4"))

	eName := func(n xml.Name) string {
		if n.Space == "" {
			return tagS.Render(n.Local)
		}
		return nsS.Render(xmlNSPrefix(n.Space)+":") + tagS.Render(n.Local)
	}
	aName := func(n xml.Name) string {
		switch {
		case n.Space == "":
			return attrS.Render(n.Local)
		case n.Space == "xmlns":
			return attrS.Render("xmlns:" + n.Local)
		default:
			return nsS.Render(xmlNSPrefix(n.Space)+":") + attrS.Render(n.Local)
		}
	}

	var sb strings.Builder
	d := xml.NewDecoder(bytes.NewReader(data))
	depth := 0
	first := true
	hadInlineText := false

	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return string(data)
		}

		switch t := tok.(type) {
		case xml.ProcInst:
			if !first {
				sb.WriteByte('\n')
			}
			sb.WriteString(procS.Render("<?"+t.Target+" "+string(t.Inst)+"?>"))
			first = false
			hadInlineText = false

		case xml.StartElement:
			if !first {
				sb.WriteByte('\n')
				sb.WriteString(strings.Repeat("  ", depth))
			}
			sb.WriteString(punctS.Render("<"))
			sb.WriteString(eName(t.Name))
			for _, a := range t.Attr {
				sb.WriteString(" ")
				sb.WriteString(aName(a.Name))
				sb.WriteString(punctS.Render(`="`))
				sb.WriteString(valS.Render(a.Value))
				sb.WriteString(punctS.Render(`"`))
			}
			sb.WriteString(punctS.Render(">"))
			depth++
			first = false
			hadInlineText = false

		case xml.EndElement:
			depth--
			if !hadInlineText {
				sb.WriteByte('\n')
				sb.WriteString(strings.Repeat("  ", depth))
			}
			sb.WriteString(punctS.Render("</"))
			sb.WriteString(eName(t.Name))
			sb.WriteString(punctS.Render(">"))
			hadInlineText = false

		case xml.CharData:
			s := strings.TrimSpace(string(t))
			if s != "" {
				sb.WriteString(textS.Render(s))
				hadInlineText = true
			}

		case xml.Comment:
			if !first {
				sb.WriteByte('\n')
				sb.WriteString(strings.Repeat("  ", depth))
			}
			sb.WriteString(commentS.Render("<!--" + string(t) + "-->"))
			first = false
			hadInlineText = false
		}
	}
	return sb.String()
}

// xmlNSPrefix maps known XML namespace URIs to their conventional prefix.
// Falls back to the last path/fragment segment for unknown URIs.
func xmlNSPrefix(uri string) string {
	switch uri {
	case "urn:oasis:names:tc:SAML:2.0:protocol":
		return "samlp"
	case "urn:oasis:names:tc:SAML:2.0:assertion":
		return "saml"
	case "urn:oasis:names:tc:SAML:2.0:metadata":
		return "md"
	case "http://www.w3.org/2000/09/xmldsig#":
		return "ds"
	case "http://www.w3.org/2001/XMLSchema-instance":
		return "xsi"
	case "http://www.w3.org/2001/XMLSchema":
		return "xs"
	case "http://www.w3.org/XML/1998/namespace":
		return "xml"
	}
	for _, sep := range []string{"#", "/", ":"} {
		if i := strings.LastIndex(uri, sep); i >= 0 && i < len(uri)-1 {
			return uri[i+1:]
		}
	}
	return uri
}
