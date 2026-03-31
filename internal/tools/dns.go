package tools

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var DNSLookupT = &Tool{
	ID:          "dns",
	Name:        "DNS Lookup",
	Description: "Query A, AAAA, MX, TXT, CNAME, NS, PTR records for a hostname",
	Category:    "Network",
	Inputs: []InputDef{
		{
			Label:       "Hostname / IP",
			Placeholder: "e.g. example.com or 8.8.8.8",
			Multiline:   false,
			Required:    true,
			AcceptsFile: false,
			FlagName:    "host",
			FlagShort:   "H",
		},
		{
			Label:     "Record Type",
			Options:   []string{"ALL", "A", "AAAA", "MX", "TXT", "CNAME", "NS", "PTR"},
			Default:   "ALL",
			FlagName:  "type",
			FlagShort: "t",
		},
	},
	Run: func(inputs []string) (string, error) {
		host := strings.TrimSpace(resolveInput(inputs, 0))
		if host == "" {
			return "", fmt.Errorf("hostname is required")
		}
		recType := "ALL"
		if len(inputs) > 1 && inputs[1] != "" {
			recType = strings.ToUpper(inputs[1])
		}
		return dnsLookup(host, recType)
	},
}

func dnsLookup(host, recType string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resolver := net.DefaultResolver
	var sb strings.Builder

	sb.WriteString(section(fmt.Sprintf("DNS Lookup: %s", host)))

	lookup := func(rtype string) {
		switch rtype {
		case "A":
			addrs, err := resolver.LookupHost(ctx, host)
			sb.WriteString(section("A / AAAA Records"))
			if err != nil {
				sb.WriteString(fmt.Sprintf("  error: %v\n", err))
				return
			}
			v4, v6 := []string{}, []string{}
			for _, a := range addrs {
				ip := net.ParseIP(a)
				if ip == nil {
					continue
				}
				if ip.To4() != nil {
					v4 = append(v4, a)
				} else {
					v6 = append(v6, a)
				}
			}
			for _, a := range v4 {
				sb.WriteString(fmt.Sprintf("  A    %s\n", a))
			}
			for _, a := range v6 {
				sb.WriteString(fmt.Sprintf("  AAAA %s\n", a))
			}

		case "AAAA":
			addrs, err := resolver.LookupIPAddr(ctx, host)
			sb.WriteString(section("AAAA Records"))
			if err != nil {
				sb.WriteString(fmt.Sprintf("  error: %v\n", err))
				return
			}
			found := false
			for _, a := range addrs {
				if a.IP.To4() == nil {
					sb.WriteString(fmt.Sprintf("  %s\n", a.IP.String()))
					found = true
				}
			}
			if !found {
				sb.WriteString("  (none)\n")
			}

		case "MX":
			records, err := resolver.LookupMX(ctx, host)
			sb.WriteString(section("MX Records"))
			if err != nil {
				if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
					sb.WriteString("  (none)\n")
				} else {
					sb.WriteString(fmt.Sprintf("  error: %v\n", err))
				}
				return
			}
			sort.Slice(records, func(i, j int) bool {
				return records[i].Pref < records[j].Pref
			})
			if len(records) == 0 {
				sb.WriteString("  (none)\n")
			}
			for _, mx := range records {
				sb.WriteString(fmt.Sprintf("  %-5d  %s\n", mx.Pref, mx.Host))
			}

		case "TXT":
			records, err := resolver.LookupTXT(ctx, host)
			sb.WriteString(section("TXT Records"))
			if err != nil {
				if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
					sb.WriteString("  (none)\n")
				} else {
					sb.WriteString(fmt.Sprintf("  error: %v\n", err))
				}
				return
			}
			if len(records) == 0 {
				sb.WriteString("  (none)\n")
			}
			for _, txt := range records {
				sb.WriteString(fmt.Sprintf("  %s\n", txt))
			}

		case "CNAME":
			cname, err := resolver.LookupCNAME(ctx, host)
			sb.WriteString(section("CNAME Record"))
			if err != nil {
				if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
					sb.WriteString("  (none)\n")
				} else {
					sb.WriteString(fmt.Sprintf("  error: %v\n", err))
				}
				return
			}
			sb.WriteString(fmt.Sprintf("  %s → %s\n", host, cname))

		case "NS":
			records, err := resolver.LookupNS(ctx, host)
			sb.WriteString(section("NS Records"))
			if err != nil {
				if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
					sb.WriteString("  (none)\n")
				} else {
					sb.WriteString(fmt.Sprintf("  error: %v\n", err))
				}
				return
			}
			if len(records) == 0 {
				sb.WriteString("  (none)\n")
			}
			for _, ns := range records {
				sb.WriteString(fmt.Sprintf("  %s\n", ns.Host))
			}

		case "PTR":
			names, err := resolver.LookupAddr(ctx, host)
			sb.WriteString(section("PTR Records (Reverse DNS)"))
			if err != nil {
				if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
					sb.WriteString("  (none)\n")
				} else {
					sb.WriteString(fmt.Sprintf("  error: %v\n", err))
				}
				return
			}
			if len(names) == 0 {
				sb.WriteString("  (none)\n")
			}
			for _, n := range names {
				sb.WriteString(fmt.Sprintf("  %s → %s\n", host, n))
			}
		}
	}

	if recType == "ALL" {
		for _, t := range []string{"A", "MX", "TXT", "CNAME", "NS"} {
			lookup(t)
		}
		// PTR only makes sense for IPs
		if net.ParseIP(host) != nil {
			lookup("PTR")
		}
	} else {
		lookup(recType)
	}

	return sb.String(), nil
}

// ── WHOIS ────────────────────────────────────────────────────────────────────

var WhoisT = &Tool{
	ID:          "whois",
	Name:        "WHOIS Lookup",
	Description: "Query WHOIS for a domain/IP — auto-detects structured formats and renders as a hierarchy tree",
	Category:    "Network",
	Inputs: []InputDef{
		{
			Label:       "Domain / IP",
			Placeholder: "e.g. example.com  or  8.8.8.8",
			Multiline:   false,
			Required:    true,
			AcceptsFile: false,
			FlagName:    "host",
			FlagShort:   "H",
		},
		{
			Label:       "WHOIS Server",
			Placeholder: "whois.iana.org  (default: whois.iana.org)",
			Multiline:   false,
			Required:    false,
			AcceptsFile: false,
			FlagName:    "server",
			FlagShort:   "s",
		},
	},
	Run: func(inputs []string) (string, error) {
		host := strings.TrimSpace(resolveInput(inputs, 0))
		if host == "" {
			return "", fmt.Errorf("domain or IP is required")
		}
		server := ""
		if len(inputs) > 1 {
			server = strings.TrimSpace(inputs[1])
		}
		if server == "" {
			server = "whois.iana.org"
		}
		return whoisLookup(host, server)
	},
}

func whoisLookup(query, server string) (string, error) {
	addr := server
	if !strings.Contains(addr, ":") {
		addr += ":43"
	}

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("connect to %s: %w", server, err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(15 * time.Second))

	if _, err := fmt.Fprintf(conn, "%s\r\n", query); err != nil {
		return "", fmt.Errorf("send query: %w", err)
	}

	var lines []string
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	return renderWhoisResponse(lines, query, server), nil
}

// renderWhoisResponse detects structured whois formats (sections ending with "matches:" or
// "DNS results:") and renders each section as a coloured subnet hierarchy tree.
// Falls back to plain text for unrecognised formats.
func renderWhoisResponse(lines []string, query, server string) string {
	var sb strings.Builder
	sb.WriteString(section(fmt.Sprintf("WHOIS: %s  (via %s)", query, server)))

	// Detect structured format: any line that ends with "matches:" or equals "DNS results:"
	structuredFormat := false
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasSuffix(t, "matches:") || t == "DNS results:" {
			structuredFormat = true
			break
		}
	}

	if structuredFormat {
		renderStructuredWhois(&sb, lines)
	} else {
		for _, l := range lines {
			sb.WriteString("  " + l + "\n")
		}
	}
	return sb.String()
}

// looksLikeCIDR returns true if s looks like an IPv4/IPv6 address or CIDR block.
// Used to filter out metadata lines in structured WHOIS output that don't start with a subnet.
func looksLikeCIDR(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Accept CIDR notation (e.g. "10.0.0.0/8" or "2001:db8::/32")
	_, _, err := net.ParseCIDR(s)
	if err == nil {
		return true
	}
	// Accept bare IP address
	return net.ParseIP(s) != nil
}

// whoisEntry holds one structured line from an Apple WHOIS section.
type whoisEntry struct {
	depth int
	cidr  string
	desc  string
}

// renderStructuredWhois parses and renders structured whois output where sections are
// delimited by non-indented lines ending with ":". Each section is rendered as an
// indented tree where deeper subnets appear as children of their supernets.
// The most-specific (leaf) entry in each section is highlighted in green.
func renderStructuredWhois(sb *strings.Builder, lines []string) {
	cidrS    := lipgloss.NewStyle().Foreground(lipgloss.Color("#00f5ff")).Bold(true)
	leafS    := lipgloss.NewStyle().Foreground(lipgloss.Color("#39ff14")).Bold(true)
	descS    := lipgloss.NewStyle().Foreground(lipgloss.Color("#e2e8f0"))
	treeS    := lipgloss.NewStyle().Foreground(lipgloss.Color("#4a5568"))
	ipS      := lipgloss.NewStyle().Foreground(lipgloss.Color("#39ff14")).Bold(true)
	hostS    := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffe600"))

	// treePrefix builds the visual connector for a given depth.
	// depth 1  →  "  "              (no connector, flush left)
	// depth 2  →  "  └── "
	// depth 3  →  "      └── "
	// depth N  →  "  " + (N-2)*"    " + "└── "
	treePrefix := func(depth int) string {
		if depth <= 1 {
			return treeS.Render("  ")
		}
		indent := strings.Repeat("    ", depth-2)
		return treeS.Render("  "+indent+"└── ")
	}

	flushSection := func(title string, entries []whoisEntry) {
		if len(entries) == 0 {
			return
		}
		// Align CIDR column within this section.
		maxW := 0
		for _, e := range entries {
			if len(e.cidr) > maxW {
				maxW = len(e.cidr)
			}
		}
		sb.WriteByte('\n')
		for i, e := range entries {
			isLeaf := i == len(entries)-1
			padded := fmt.Sprintf("%-*s", maxW, e.cidr)
			var cidrStr string
			if isLeaf {
				cidrStr = leafS.Render(padded)
			} else {
				cidrStr = cidrS.Render(padded)
			}
			sb.WriteString(treePrefix(e.depth) + cidrStr + "  " + descS.Render(e.desc) + "\n")
		}
	}

	var (
		curTitle   string
		curEntries []whoisEntry
		isDNS      bool
	)

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t\r")
		if trimmed == "" {
			continue
		}

		// Section header: non-indented line ending with ":"
		if !strings.HasPrefix(trimmed, " ") && strings.HasSuffix(trimmed, ":") {
			flushSection(curTitle, curEntries)
			curEntries = nil
			curTitle = strings.TrimSuffix(trimmed, ":")
			isDNS = curTitle == "DNS results"
			sb.WriteString(section(curTitle))
			continue
		}

		// Indented content line — skip non-indented, non-header lines (noise)
		if !strings.HasPrefix(line, " ") {
			continue
		}

		depth := len(line) - len(strings.TrimLeft(line, " "))
		rest := strings.TrimSpace(line)
		if rest == "" {
			continue
		}

		if isDNS {
			// "DNS results:" lines: "<IP>  <hostname>"
			fields := strings.Fields(rest)
			switch len(fields) {
			case 1:
				sb.WriteString("  " + ipS.Render(fields[0]) + "\n")
			default:
				sb.WriteString("  " + ipS.Render(fields[0]) + "  →  " + hostS.Render(fields[1]) + "\n")
			}
			continue
		}

		// Subnet entry: first token = CIDR/IP, remainder = description
		idx := strings.IndexAny(rest, " \t")
		var cidr, desc string
		if idx < 0 {
			cidr = rest
		} else {
			cidr = rest[:idx]
			desc = strings.TrimSpace(rest[idx:])
		}
		// Skip non-CIDR metadata/commentary lines that some WHOIS servers
		// inject (e.g. "network details follow"). They're not subnet entries.
		if !looksLikeCIDR(cidr) {
			continue
		}
		curEntries = append(curEntries, whoisEntry{depth: depth, cidr: cidr, desc: desc})
	}
	flushSection(curTitle, curEntries)
}
