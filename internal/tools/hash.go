package tools

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"strings"
)

var HashCalcT = &Tool{
	ID:          "hash",
	Name:        "Hash Calculator",
	Description: "Compute MD5, SHA-1, SHA-256, SHA-512 hashes of any input",
	Category:    "Data",
	Inputs: []InputDef{
		{
			Label:       "Input",
			Placeholder: "Paste text or enter a file path",
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
			return "", fmt.Errorf("input is required")
		}
		return calcHashes([]byte(raw)), nil
	},
}

func calcHashes(data []byte) string {
	md5sum := md5.Sum(data)
	sha1sum := sha1.Sum(data)
	sha256sum := sha256.Sum256(data)
	sha512sum := sha512.Sum512(data)

	var sb strings.Builder
	sb.WriteString(section("Hash Values"))
	sb.WriteString(fmt.Sprintf("  %-10s: %x\n", "MD5", md5sum))
	sb.WriteString(fmt.Sprintf("  %-10s: %x\n", "SHA-1", sha1sum))
	sb.WriteString(fmt.Sprintf("  %-10s: %x\n", "SHA-256", sha256sum))
	sb.WriteString(fmt.Sprintf("  %-10s: %x\n", "SHA-512", sha512sum))

	sb.WriteString(section("Input Info"))
	sb.WriteString(fmt.Sprintf("  %-10s: %d bytes\n", "Size", len(data)))

	return sb.String()
}
