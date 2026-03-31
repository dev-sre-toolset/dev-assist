package tools

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
)

var PEMParseT = &Tool{
	ID:          "pem-parse",
	Name:        "PEM Parser",
	Description: "Detect PEM type and decode cert, private key, CSR, or public key",
	Category:    "SSL & Certificates",
	Inputs: []InputDef{
		{
			Label:       "PEM Data",
			Placeholder: "Paste PEM block or enter a file path",
			Multiline:   true,
			Required:    true,
			AcceptsFile: true,
			FlagName:    "input",
			FlagShort:   "i",
		},
	},
	Run: func(inputs []string) (string, error) {
		raw := resolveInput(inputs, 0)
		if raw == "" {
			return "", fmt.Errorf("PEM input is required")
		}
		return parsePEM(raw)
	},
}

func parsePEM(input string) (string, error) {
	var sb strings.Builder
	data := []byte(input)
	blockIdx := 0

	for {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}
		blockIdx++
		sb.WriteString(section(fmt.Sprintf("Block %d — %s", blockIdx, block.Type)))

		switch block.Type {
		case "CERTIFICATE":
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				sb.WriteString(fmt.Sprintf("  parse error: %v\n", err))
			} else {
				sb.WriteString(kv("Subject", cert.Subject.CommonName))
				sb.WriteString(kv("Issuer", cert.Issuer.CommonName))
				sb.WriteString(kv("Not Before", cert.NotBefore.UTC().Format("2006-01-02")))
				sb.WriteString(kv("Not After", cert.NotAfter.UTC().Format("2006-01-02")))
				sb.WriteString(kv("SANs", strings.Join(cert.DNSNames, ", ")))
				if len(cert.IPAddresses) > 0 {
					sb.WriteString(kv("IP SANs", ipList(cert.IPAddresses)))
				}
				sb.WriteString(kv("Is CA", fmt.Sprintf("%v", cert.IsCA)))
				sb.WriteString(kv("Key Usage", keyUsageStr(cert.KeyUsage)))
			}

		case "CERTIFICATE REQUEST":
			csr, err := x509.ParseCertificateRequest(block.Bytes)
			if err != nil {
				sb.WriteString(fmt.Sprintf("  parse error: %v\n", err))
			} else {
				sb.WriteString(kv("Subject", csr.Subject.CommonName))
				if len(csr.Subject.Organization) > 0 {
					sb.WriteString(kv("Organization", strings.Join(csr.Subject.Organization, ", ")))
				}
				if len(csr.Subject.Country) > 0 {
					sb.WriteString(kv("Country", strings.Join(csr.Subject.Country, ", ")))
				}
				sb.WriteString(kv("DNS SANs", strings.Join(csr.DNSNames, ", ")))
				if len(csr.IPAddresses) > 0 {
					sb.WriteString(kv("IP SANs", ipList(csr.IPAddresses)))
				}
				switch k := csr.PublicKey.(type) {
				case *rsa.PublicKey:
					sb.WriteString(kv("Key Type", fmt.Sprintf("RSA %d-bit", k.N.BitLen())))
				case *ecdsa.PublicKey:
					sb.WriteString(kv("Key Type", fmt.Sprintf("ECDSA %s", k.Curve.Params().Name)))
				}
				if err := csr.CheckSignature(); err != nil {
					sb.WriteString(kv("Signature", fmt.Sprintf("✗ invalid: %v", err)))
				} else {
					sb.WriteString(kv("Signature", "✓ valid"))
				}
			}

		case "RSA PRIVATE KEY":
			key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				sb.WriteString(fmt.Sprintf("  parse error: %v\n", err))
			} else {
				sb.WriteString(kv("Type", "RSA Private Key"))
				sb.WriteString(kv("Key Size", fmt.Sprintf("%d bits", key.N.BitLen())))
				sb.WriteString(kv("Public Exp", fmt.Sprintf("%d", key.PublicKey.E)))
				sb.WriteString("  ⚠ Do NOT paste private keys into untrusted tools\n")
			}

		case "EC PRIVATE KEY":
			key, err := x509.ParseECPrivateKey(block.Bytes)
			if err != nil {
				sb.WriteString(fmt.Sprintf("  parse error: %v\n", err))
			} else {
				sb.WriteString(kv("Type", "EC Private Key"))
				sb.WriteString(kv("Curve", key.Curve.Params().Name))
				sb.WriteString("  ⚠ Do NOT paste private keys into untrusted tools\n")
			}

		case "PRIVATE KEY": // PKCS#8
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				sb.WriteString(fmt.Sprintf("  parse error: %v\n", err))
			} else {
				switch k := key.(type) {
				case *rsa.PrivateKey:
					sb.WriteString(kv("Type", "PKCS#8 RSA Private Key"))
					sb.WriteString(kv("Key Size", fmt.Sprintf("%d bits", k.N.BitLen())))
				case *ecdsa.PrivateKey:
					sb.WriteString(kv("Type", "PKCS#8 EC Private Key"))
					sb.WriteString(kv("Curve", k.Curve.Params().Name))
				default:
					sb.WriteString(kv("Type", "PKCS#8 Private Key"))
				}
				sb.WriteString("  ⚠ Do NOT paste private keys into untrusted tools\n")
			}

		case "PUBLIC KEY":
			key, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				sb.WriteString(fmt.Sprintf("  parse error: %v\n", err))
			} else {
				switch k := key.(type) {
				case *rsa.PublicKey:
					sb.WriteString(kv("Type", fmt.Sprintf("RSA Public Key (%d-bit)", k.N.BitLen())))
				case *ecdsa.PublicKey:
					sb.WriteString(kv("Type", fmt.Sprintf("EC Public Key (%s)", k.Curve.Params().Name)))
				default:
					sb.WriteString(kv("Type", fmt.Sprintf("%T", key)))
				}
			}

		default:
			sb.WriteString(fmt.Sprintf("  Type    : %s\n", block.Type))
			sb.WriteString(fmt.Sprintf("  Size    : %d bytes\n", len(block.Bytes)))
			sb.WriteString("  (no further decoding available for this block type)\n")
		}

		data = rest
	}

	if blockIdx == 0 {
		return "", fmt.Errorf("no PEM blocks found in input")
	}

	sb.WriteString(section("Summary"))
	sb.WriteString(fmt.Sprintf("  %d PEM block(s) found\n", blockIdx))

	return sb.String(), nil
}
