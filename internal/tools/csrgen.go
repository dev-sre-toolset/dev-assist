package tools

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var CSRGenT = &Tool{
	ID:          "csr-gen",
	Name:        "Generate CSR & Private Key",
	Description: "Generate a private key + CSR — equivalent to openssl req -new -newkey",
	Category:    "SSL & Certificates",
	Inputs: []InputDef{
		{
			Label:       "Common Name (CN)",
			Placeholder: "e.g. api.example.com",
			Required:    true,
			FlagName:    "cn",
			FlagShort:   "n",
		},
		{
			Label:       "DNS SANs",
			Placeholder: "app.corp.com, *.corp.com  (comma-separated; CN is always included)",
			Required:    false,
			FlagName:    "san",
			FlagShort:   "s",
		},
		{
			Label:       "Organization (O)",
			Placeholder: "e.g. Example Corp",
			Required:    false,
			FlagName:    "org",
			FlagShort:   "o",
		},
		{
			Label:       "Country / State / City",
			Placeholder: "US / California / San Francisco  (slash-separated, all optional)",
			Required:    false,
			FlagName:    "geo",
			FlagShort:   "g",
		},
		{
			Label:   "Key Algorithm & Size",
			Options: []string{"RSA-2048", "RSA-4096", "ECDSA-P256", "ECDSA-P384"},
			Default: "RSA-2048",
			FlagName:  "algo",
			FlagShort: "a",
		},
	},
	Run: func(inputs []string) (string, error) {
		cn := strings.TrimSpace(resolveInput(inputs, 0))
		if cn == "" {
			return "", fmt.Errorf("Common Name (CN) is required")
		}

		sanStr := strings.TrimSpace(resolveInput(inputs, 1))
		org := strings.TrimSpace(resolveInput(inputs, 2))
		geoStr := strings.TrimSpace(resolveInput(inputs, 3))

		algo := "RSA-2048"
		if len(inputs) > 4 && strings.TrimSpace(inputs[4]) != "" {
			algo = strings.TrimSpace(inputs[4])
		}

		outDir := os.TempDir()
		prefix := sanitizeFilename(cn)

		return generateCSR(cn, sanStr, org, geoStr, algo, outDir, prefix)
	},
}

// ── implementation ────────────────────────────────────────────────────────────

func generateCSR(cn, sanStr, org, geoStr, algo, outDir, prefix string) (string, error) {
	// ── 1. parse SANs ────────────────────────────────────────────────────────
	dnsNames := []string{cn} // CN is always a SAN per RFC 2818
	var ipAddrs []net.IP

	if sanStr != "" {
		for _, raw := range strings.Split(sanStr, ",") {
			san := strings.TrimSpace(raw)
			if san == "" || san == cn {
				continue
			}
			if ip := net.ParseIP(san); ip != nil {
				ipAddrs = append(ipAddrs, ip)
			} else {
				dnsNames = append(dnsNames, san)
			}
		}
	}

	// ── 2. build subject ─────────────────────────────────────────────────────
	country, state, city := parseGeo(geoStr)
	subject := pkix.Name{CommonName: cn}
	if org != "" {
		subject.Organization = []string{org}
	}
	if country != "" {
		subject.Country = []string{country}
	}
	if state != "" {
		subject.Province = []string{state}
	}
	if city != "" {
		subject.Locality = []string{city}
	}

	csrTemplate := &x509.CertificateRequest{
		Subject:     subject,
		DNSNames:    dnsNames,
		IPAddresses: ipAddrs,
	}

	// ── 3. generate key + CSR ────────────────────────────────────────────────
	var privPEM, algoLabel string
	var csrDER []byte

	switch algo {
	case "RSA-2048", "RSA-4096":
		bits := map[string]int{"RSA-2048": 2048, "RSA-4096": 4096}[algo]
		key, err := rsa.GenerateKey(rand.Reader, bits)
		if err != nil {
			return "", fmt.Errorf("generate RSA-%d key: %w", bits, err)
		}
		privPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		}))
		algoLabel = fmt.Sprintf("RSA %d-bit", bits)
		csrDER, err = x509.CreateCertificateRequest(rand.Reader, csrTemplate, key)
		if err != nil {
			return "", fmt.Errorf("create CSR: %w", err)
		}

	case "ECDSA-P256", "ECDSA-P384":
		curve := map[string]elliptic.Curve{
			"ECDSA-P256": elliptic.P256(),
			"ECDSA-P384": elliptic.P384(),
		}[algo]
		key, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			return "", fmt.Errorf("generate ECDSA key: %w", err)
		}
		der, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			return "", fmt.Errorf("marshal EC private key: %w", err)
		}
		privPEM = string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}))
		algoLabel = fmt.Sprintf("ECDSA %s", curve.Params().Name)
		csrDER, err = x509.CreateCertificateRequest(rand.Reader, csrTemplate, key)
		if err != nil {
			return "", fmt.Errorf("create CSR: %w", err)
		}

	default:
		return "", fmt.Errorf("unsupported algorithm %q — choose RSA-2048, RSA-4096, ECDSA-P256, or ECDSA-P384", algo)
	}

	csrPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	}))

	// ── 4. write files ───────────────────────────────────────────────────────
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("create output directory %q: %w", outDir, err)
	}

	keyPath := filepath.Join(outDir, prefix+".key")
	csrPath := filepath.Join(outDir, prefix+".csr")

	if err := os.WriteFile(keyPath, []byte(privPEM), 0o600); err != nil {
		return "", fmt.Errorf("write key file: %w", err)
	}
	if err := os.WriteFile(csrPath, []byte(csrPEM), 0o644); err != nil {
		return "", fmt.Errorf("write CSR file: %w", err)
	}

	// ── 5. verify CSR signature ──────────────────────────────────────────────
	sigStatus := "✓ valid"
	if csr, err := x509.ParseCertificateRequest(csrDER); err == nil {
		if err := csr.CheckSignature(); err != nil {
			sigStatus = fmt.Sprintf("✗ %v", err)
		}
	}

	// ── 6. build result string ───────────────────────────────────────────────
	var sb strings.Builder

	sb.WriteString(section("Files Written"))
	sb.WriteString(kv("Private Key", keyPath+"  (mode 0600)"))
	sb.WriteString(kv("CSR", csrPath))

	sb.WriteString(section("CSR Details"))
	sb.WriteString(kv("Common Name", cn))
	if org != "" {
		sb.WriteString(kv("Organization", org))
	}
	if country != "" {
		sb.WriteString(kv("Country", country))
	}
	if state != "" {
		sb.WriteString(kv("State", state))
	}
	if city != "" {
		sb.WriteString(kv("City", city))
	}
	sb.WriteString(kv("DNS SANs", strings.Join(dnsNames, ", ")))
	if len(ipAddrs) > 0 {
		sb.WriteString(kv("IP SANs", ipList(ipAddrs)))
	}
	sb.WriteString(kv("Algorithm", algoLabel))
	sb.WriteString(kv("Signature", sigStatus))

	sb.WriteString(section("Equivalent OpenSSL Command"))
	sb.WriteString(fmt.Sprintf("  %s\n", opensslEquivalent(cn, org, country, state, city, algo, prefix)))

	sb.WriteString(section("Private Key  ⚠ keep this secret"))
	sb.WriteString(privPEM)

	sb.WriteString(section("CSR  (submit this to your CA)"))
	sb.WriteString(csrPEM)

	return sb.String(), nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func parseGeo(s string) (country, state, city string) {
	parts := strings.SplitN(s, "/", 3)
	if len(parts) >= 1 {
		country = strings.TrimSpace(parts[0])
	}
	if len(parts) >= 2 {
		state = strings.TrimSpace(parts[1])
	}
	if len(parts) >= 3 {
		city = strings.TrimSpace(parts[2])
	}
	return
}

func sanitizeFilename(s string) string {
	r := strings.NewReplacer(
		" ", "-", "*", "wildcard", "/", "-", "\\", "-", ":", "-",
	)
	return strings.ToLower(r.Replace(s))
}

func opensslEquivalent(cn, org, country, state, city, algo, prefix string) string {
	// Build -subj string
	var subjParts []string
	if country != "" {
		subjParts = append(subjParts, "C="+country)
	}
	if state != "" {
		subjParts = append(subjParts, "ST="+state)
	}
	if city != "" {
		subjParts = append(subjParts, "L="+city)
	}
	if org != "" {
		subjParts = append(subjParts, "O="+org)
	}
	subjParts = append(subjParts, "CN="+cn)
	subj := "/" + strings.Join(subjParts, "/")

	switch algo {
	case "RSA-2048", "RSA-4096":
		bits := strings.TrimPrefix(algo, "RSA-")
		return fmt.Sprintf(
			`openssl req -new -newkey rsa:%s -nodes -keyout %s.key -out %s.csr -subj "%s"`,
			bits, prefix, prefix, subj,
		)
	default: // ECDSA
		curve := strings.ToLower(strings.TrimPrefix(algo, "ECDSA-"))
		return fmt.Sprintf(
			`openssl req -new -newkey ec -pkeyopt ec_paramgen_curve:%s -nodes -keyout %s.key -out %s.csr -subj "%s"`,
			curve, prefix, prefix, subj,
		)
	}
}
