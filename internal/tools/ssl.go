package tools

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"strings"
	"time"
)

// SSLDecodeT decodes a PEM certificate and prints its fields.
var SSLDecodeT = &Tool{
	ID:          "ssl-decode",
	Name:        "SSL Certificate Decode",
	Description: "Parse a PEM cert and display all fields",
	Category:    "SSL & Certificates",
	Inputs: []InputDef{
		{
			Label:       "Certificate (PEM or bare base64)",
			Placeholder: "Paste full PEM (with -----BEGIN-----) or just the bare base64 DER, or enter a file path",
			Multiline:   true,
			Required:    true,
			AcceptsFile: true,
			FlagName:    "cert",
			FlagShort:   "c",
		},
	},
	Run: func(inputs []string) (string, error) {
		raw := resolveInput(inputs, 0)
		if raw == "" {
			return "", fmt.Errorf("certificate input is required")
		}
		return decodeCert(raw)
	},
}

// SSLVerifyT verifies a cert against a CA bundle (or system roots).
var SSLVerifyT = &Tool{
	ID:          "ssl-verify",
	Name:        "SSL Certificate Verify",
	Description: "Verify cert chain against a CA bundle or system roots",
	Category:    "SSL & Certificates",
	Inputs: []InputDef{
		{
			Label:       "Certificate (PEM or bare base64)",
			Placeholder: "Paste full PEM (with -----BEGIN-----) or just the bare base64 DER, or enter a file path",
			Multiline:   true,
			Required:    true,
			AcceptsFile: true,
			FlagName:    "cert",
			FlagShort:   "c",
		},
		{
			Label:       "CA Bundle (PEM) — optional",
			Placeholder: "Leave empty to use system CA roots",
			Multiline:   true,
			Required:    false,
			AcceptsFile: true,
			FlagName:    "ca",
		},
	},
	Run: func(inputs []string) (string, error) {
		certPEM := resolveInput(inputs, 0)
		if certPEM == "" {
			return "", fmt.Errorf("certificate input is required")
		}
		caPEM := resolveInput(inputs, 1)
		return verifyCert(certPEM, caPEM)
	},
}

// ── implementations ──────────────────────────────────────────────────────────

// normalizeCertPEM ensures the input has PEM headers.
// If the input is bare base64-encoded DER (no "-----BEGIN" found), the
// CERTIFICATE headers are added automatically.
func normalizeCertPEM(input string) string {
	s := strings.TrimSpace(input)
	if strings.Contains(s, "-----BEGIN") {
		return s
	}
	// Strip all whitespace from the bare base64 blob.
	stripped := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r':
			return -1
		}
		return r
	}, s)
	// Re-wrap into 64-character lines (standard PEM line length).
	var lines []string
	for len(stripped) > 0 {
		n := 64
		if n > len(stripped) {
			n = len(stripped)
		}
		lines = append(lines, stripped[:n])
		stripped = stripped[n:]
	}
	return "-----BEGIN CERTIFICATE-----\n" +
		strings.Join(lines, "\n") +
		"\n-----END CERTIFICATE-----"
}

func decodeCert(pemData string) (string, error) {
	pemData = normalizeCertPEM(pemData)
	block, rest := pem.Decode([]byte(pemData))
	if block == nil {
		return "", fmt.Errorf("no PEM block found — is this a valid PEM certificate?")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse certificate: %w", err)
	}

	var sb strings.Builder

	sb.WriteString(section("Subject"))
	sb.WriteString(kv("Common Name", cert.Subject.CommonName))
	if len(cert.Subject.Organization) > 0 {
		sb.WriteString(kv("Organization", strings.Join(cert.Subject.Organization, ", ")))
	}
	if len(cert.Subject.OrganizationalUnit) > 0 {
		sb.WriteString(kv("Org Unit", strings.Join(cert.Subject.OrganizationalUnit, ", ")))
	}
	if len(cert.Subject.Country) > 0 {
		sb.WriteString(kv("Country", strings.Join(cert.Subject.Country, ", ")))
	}

	sb.WriteString(section("Issuer"))
	sb.WriteString(kv("Common Name", cert.Issuer.CommonName))
	if len(cert.Issuer.Organization) > 0 {
		sb.WriteString(kv("Organization", strings.Join(cert.Issuer.Organization, ", ")))
	}

	now := time.Now()
	status := ""
	switch {
	case now.After(cert.NotAfter):
		status = "  ⚠ EXPIRED"
	case now.Before(cert.NotBefore):
		status = "  ⚠ NOT YET VALID"
	default:
		days := int(time.Until(cert.NotAfter).Hours() / 24)
		status = fmt.Sprintf("  ✓ %d days remaining", days)
	}

	sb.WriteString(section("Validity"))
	sb.WriteString(kv("Not Before", cert.NotBefore.UTC().Format(time.RFC3339)))
	sb.WriteString(kv("Not After", cert.NotAfter.UTC().Format(time.RFC3339)+status))

	sb.WriteString(section("Subject Alternative Names"))
	sanCount := 0
	for _, d := range cert.DNSNames {
		sb.WriteString(fmt.Sprintf("  DNS  : %s\n", d))
		sanCount++
	}
	for _, ip := range cert.IPAddresses {
		sb.WriteString(fmt.Sprintf("  IP   : %s\n", ip))
		sanCount++
	}
	for _, u := range cert.URIs {
		sb.WriteString(fmt.Sprintf("  URI  : %s\n", u))
		sanCount++
	}
	for _, e := range cert.EmailAddresses {
		sb.WriteString(fmt.Sprintf("  Email: %s\n", e))
		sanCount++
	}
	if sanCount == 0 {
		sb.WriteString("  (none)\n")
	}

	sb.WriteString(section("Public Key"))
	switch k := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		sb.WriteString(kv("Type", fmt.Sprintf("RSA %d-bit", k.N.BitLen())))
	case *ecdsa.PublicKey:
		sb.WriteString(kv("Type", fmt.Sprintf("ECDSA %s", k.Curve.Params().Name)))
	default:
		sb.WriteString(kv("Type", fmt.Sprintf("%T", cert.PublicKey)))
	}

	sha1fp := sha1.Sum(block.Bytes)
	sha256fp := sha256.Sum256(block.Bytes)
	sb.WriteString(section("Fingerprints"))
	sb.WriteString(kv("SHA-1  ", colonHex(sha1fp[:])))
	sb.WriteString(kv("SHA-256", colonHex(sha256fp[:])))

	sb.WriteString(section("Miscellaneous"))
	sb.WriteString(kv("Serial", cert.SerialNumber.String()))
	sb.WriteString(kv("Key Usage", keyUsageStr(cert.KeyUsage)))
	if len(cert.ExtKeyUsage) > 0 {
		sb.WriteString(kv("Ext Key Usage", extKeyUsageStr(cert.ExtKeyUsage)))
	}

	chainCount := 0
	for {
		var b *pem.Block
		b, rest = pem.Decode(rest)
		if b == nil {
			break
		}
		chainCount++
	}
	if chainCount > 0 {
		sb.WriteString(kv("Chain certs", fmt.Sprintf("%d additional certificate(s) in input", chainCount)))
	}

	return sb.String(), nil
}

func verifyCert(certPEM, caPEM string) (string, error) {
	certPEM = normalizeCertPEM(certPEM)
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return "", fmt.Errorf("no PEM block found in certificate input")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse certificate: %w", err)
	}

	var roots *x509.CertPool
	if caPEM != "" {
		roots = x509.NewCertPool()
		if !roots.AppendCertsFromPEM([]byte(caPEM)) {
			return "", fmt.Errorf("no valid CA certificates found in CA bundle")
		}
	}

	intermediates := x509.NewCertPool()
	rest := []byte(certPEM)
	_, rest = pem.Decode(rest) // skip leaf
	for {
		var b *pem.Block
		b, rest = pem.Decode(rest)
		if b == nil {
			break
		}
		if c, e := x509.ParseCertificate(b.Bytes); e == nil {
			intermediates.AddCert(c)
		}
	}

	chains, err := cert.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		CurrentTime:   time.Now(),
	})
	if err != nil {
		return fmt.Sprintf("✗ VERIFICATION FAILED\n\n  Error  : %v\n  Subject: %s\n  Issuer : %s\n",
			err, cert.Subject.CommonName, cert.Issuer.CommonName), nil
	}

	var sb strings.Builder
	sb.WriteString("✓ CERTIFICATE VERIFIED SUCCESSFULLY\n")
	sb.WriteString(kv("\nSubject", cert.Subject.CommonName))
	sb.WriteString(kv("Issuer", cert.Issuer.CommonName))
	sb.WriteString(kv("Expires", cert.NotAfter.UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("\n  Verified %d chain(s):\n", len(chains)))
	for i, chain := range chains {
		sb.WriteString(fmt.Sprintf("\n  Chain %d:\n", i+1))
		for j, c := range chain {
			indent := strings.Repeat("  ", j+1)
			sb.WriteString(fmt.Sprintf("%s└ %s\n", indent, c.Subject.CommonName))
		}
	}
	if caPEM == "" {
		sb.WriteString("\n  (verified against system CA roots)\n")
	}
	return sb.String(), nil
}

// ── shared formatting helpers (used by other tool files too) ─────────────────

func section(title string) string {
	line := strings.Repeat("─", imax(0, 42-len(title)))
	return fmt.Sprintf("\n── %s %s\n", title, line)
}

func kv(key, value string) string {
	return fmt.Sprintf("  %-14s: %s\n", key, value)
}

func colonHex(b []byte) string {
	parts := make([]string, len(b))
	for i, v := range b {
		parts[i] = fmt.Sprintf("%02X", v)
	}
	return strings.Join(parts, ":")
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func keyUsageStr(ku x509.KeyUsage) string {
	m := map[x509.KeyUsage]string{
		x509.KeyUsageDigitalSignature:  "Digital Signature",
		x509.KeyUsageContentCommitment: "Non Repudiation",
		x509.KeyUsageKeyEncipherment:   "Key Encipherment",
		x509.KeyUsageDataEncipherment:  "Data Encipherment",
		x509.KeyUsageKeyAgreement:      "Key Agreement",
		x509.KeyUsageCertSign:          "Cert Sign",
		x509.KeyUsageCRLSign:           "CRL Sign",
	}
	var parts []string
	for bit, name := range m {
		if ku&bit != 0 {
			parts = append(parts, name)
		}
	}
	if len(parts) == 0 {
		return "(none)"
	}
	return strings.Join(parts, ", ")
}

func extKeyUsageStr(ekus []x509.ExtKeyUsage) string {
	m := map[x509.ExtKeyUsage]string{
		x509.ExtKeyUsageServerAuth:      "TLS Server Auth",
		x509.ExtKeyUsageClientAuth:      "TLS Client Auth",
		x509.ExtKeyUsageCodeSigning:     "Code Signing",
		x509.ExtKeyUsageEmailProtection: "Email Protection",
		x509.ExtKeyUsageTimeStamping:    "Time Stamping",
		x509.ExtKeyUsageOCSPSigning:     "OCSP Signing",
	}
	var parts []string
	for _, eku := range ekus {
		if n, ok := m[eku]; ok {
			parts = append(parts, n)
		}
	}
	return strings.Join(parts, ", ")
}

// resolveInput returns inputs[i]; if the value looks like an existing file path,
// reads and returns its contents instead.
func resolveInput(inputs []string, i int) string {
	if i >= len(inputs) {
		return ""
	}
	v := strings.TrimSpace(inputs[i])
	if v == "" {
		return ""
	}
	if !strings.Contains(v, "\n") {
		if data, err := readFile(v); err == nil {
			return data
		}
	}
	return v
}

// ipList is used in pem.go.
func ipList(ips []net.IP) string {
	parts := make([]string, len(ips))
	for i, ip := range ips {
		parts[i] = ip.String()
	}
	return strings.Join(parts, ", ")
}
