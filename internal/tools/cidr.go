package tools

import (
	"fmt"
	"math/big"
	"net"
	"strings"
)

var CIDRCalcT = &Tool{
	ID:          "cidr",
	Name:        "CIDR Calculator",
	Description: "Subnet math: network, broadcast, host range, mask, and IP membership check",
	Category:    "Network",
	Inputs: []InputDef{
		{
			Label:       "CIDR Notation",
			Placeholder: "e.g. 10.0.0.0/24",
			Multiline:   false,
			Required:    true,
			AcceptsFile: false,
			FlagName:    "cidr",
			FlagShort:   "c",
		},
		{
			Label:       "IP to Check (optional)",
			Placeholder: "e.g. 10.0.0.42 — leave empty to skip",
			Multiline:   false,
			Required:    false,
			AcceptsFile: false,
			FlagName:    "ip",
			FlagShort:   "i",
		},
	},
	Run: func(inputs []string) (string, error) {
		cidrStr := strings.TrimSpace(resolveInput(inputs, 0))
		if cidrStr == "" {
			return "", fmt.Errorf("CIDR input is required")
		}
		ipToCheck := strings.TrimSpace(resolveInput(inputs, 1))
		return calcCIDR(cidrStr, ipToCheck)
	},
}

func calcCIDR(cidrStr, ipToCheck string) (string, error) {
	ip, ipNet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return "", fmt.Errorf("invalid CIDR %q: %w", cidrStr, err)
	}

	ones, bits := ipNet.Mask.Size()
	isIPv6 := bits == 128

	networkIP := ipNet.IP
	broadcastIP := broadcastAddr(ipNet)
	firstHost := nextIP(networkIP)
	lastHost := prevIP(broadcastIP)
	hostCount := hostCount(ones, bits)

	var sb strings.Builder
	sb.WriteString(section("CIDR Info"))
	sb.WriteString(kv("Input IP", ip.String()))
	sb.WriteString(kv("Network", ipNet.String()))
	sb.WriteString(kv("Network IP", networkIP.String()))

	if !isIPv6 {
		sb.WriteString(kv("Subnet Mask", net.IP(ipNet.Mask).String()))
		sb.WriteString(kv("Wildcard Mask", wildcardMask(ipNet.Mask)))
		sb.WriteString(kv("Broadcast", broadcastIP.String()))
		sb.WriteString(kv("First Host", firstHost.String()))
		sb.WriteString(kv("Last Host", lastHost.String()))
	}

	sb.WriteString(kv("Prefix Length", fmt.Sprintf("/%d", ones)))
	sb.WriteString(kv("Total Addresses", fmt.Sprintf("%s", hostCount.String())))
	if !isIPv6 {
		usable := new(big.Int).Sub(hostCount, big.NewInt(2))
		if usable.Sign() < 0 {
			usable = big.NewInt(0)
		}
		sb.WriteString(kv("Usable Hosts", usable.String()))
	}
	sb.WriteString(kv("IP Version", fmt.Sprintf("IPv%d", bits/32*4)))

	// Binary representation for IPv4
	if !isIPv6 {
		sb.WriteString(section("Binary"))
		sb.WriteString(fmt.Sprintf("  Network : %s\n", ipToBinary(networkIP.To4())))
		sb.WriteString(fmt.Sprintf("  Mask    : %s\n", maskToBinary(ipNet.Mask)))
	}

	if ipToCheck != "" {
		checkIP := net.ParseIP(ipToCheck)
		if checkIP == nil {
			sb.WriteString(section("Membership Check"))
			sb.WriteString(fmt.Sprintf("  ✗ %q is not a valid IP address\n", ipToCheck))
		} else if ipNet.Contains(checkIP) {
			sb.WriteString(section("Membership Check"))
			sb.WriteString(fmt.Sprintf("  ✓ %s IS in %s\n", checkIP, cidrStr))
		} else {
			sb.WriteString(section("Membership Check"))
			sb.WriteString(fmt.Sprintf("  ✗ %s is NOT in %s\n", checkIP, cidrStr))
		}
	}

	return sb.String(), nil
}

func broadcastAddr(n *net.IPNet) net.IP {
	ip := n.IP.To4()
	if ip == nil {
		ip = n.IP.To16()
	}
	result := make(net.IP, len(ip))
	for i := range ip {
		result[i] = ip[i] | ^n.Mask[i]
	}
	return result
}

func nextIP(ip net.IP) net.IP {
	ip = cloneIP(ip)
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
	return ip
}

func prevIP(ip net.IP) net.IP {
	ip = cloneIP(ip)
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]--
		if ip[i] != 0xFF {
			break
		}
	}
	return ip
}

func cloneIP(ip net.IP) net.IP {
	c := make(net.IP, len(ip))
	copy(c, ip)
	return c
}

func hostCount(ones, bits int) *big.Int {
	exp := big.NewInt(2)
	exp.Exp(exp, big.NewInt(int64(bits-ones)), nil)
	return exp
}

func wildcardMask(mask net.IPMask) string {
	wc := make(net.IPMask, len(mask))
	for i, b := range mask {
		wc[i] = ^b
	}
	return net.IP(wc).String()
}

func ipToBinary(ip net.IP) string {
	if len(ip) != 4 {
		return "(n/a)"
	}
	parts := make([]string, 4)
	for i, b := range ip {
		parts[i] = fmt.Sprintf("%08b", b)
	}
	return strings.Join(parts, ".")
}

func maskToBinary(mask net.IPMask) string {
	if len(mask) != 4 {
		return "(n/a)"
	}
	parts := make([]string, 4)
	for i, b := range mask {
		parts[i] = fmt.Sprintf("%08b", b)
	}
	return strings.Join(parts, ".")
}
