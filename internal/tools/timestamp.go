package tools

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var TimestampT = &Tool{
	ID:          "timestamp",
	Name:        "Unix Timestamp Converter",
	Description: "Convert Unix timestamp ↔ human-readable time across common timezones",
	Category:    "Data",
	Inputs: []InputDef{
		{
			Label:       "Timestamp or Date",
			Placeholder: "e.g. 1700000000  or  2024-01-15T10:30:00Z",
			Multiline:   false,
			Required:    true,
			AcceptsFile: false,
			FlagName:    "input",
			FlagShort:   "i",
		},
	},
	Run: func(inputs []string) (string, error) {
		raw := strings.TrimSpace(resolveInput(inputs, 0))
		if raw == "" {
			return "", fmt.Errorf("input is required")
		}
		return convertTimestamp(raw)
	},
}

var tzones = []string{
	"UTC",
	"America/New_York",
	"America/Los_Angeles",
	"Europe/London",
	"Europe/Paris",
	"Asia/Kolkata",
	"Asia/Tokyo",
	"Australia/Sydney",
}

func convertTimestamp(input string) (string, error) {
	var t time.Time

	// Try parsing as integer Unix timestamp (seconds or milliseconds)
	if n, err := strconv.ParseInt(input, 10, 64); err == nil {
		if n > 1e12 { // milliseconds
			t = time.UnixMilli(n)
		} else {
			t = time.Unix(n, 0)
		}
		return formatTime(t, fmt.Sprintf("Unix %d", n))
	}

	// Try parsing as float (e.g. 1700000000.5)
	if f, err := strconv.ParseFloat(input, 64); err == nil {
		sec := int64(f)
		nsec := int64((f - float64(sec)) * 1e9)
		t = time.Unix(sec, nsec)
		return formatTime(t, fmt.Sprintf("Unix %.3f", f))
	}

	// Try a range of common formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"01/02/2006",
		"02 Jan 2006 15:04:05 MST",
		time.RFC1123Z,
		time.RFC1123,
		time.ANSIC,
		time.UnixDate,
	}
	for _, f := range formats {
		if parsed, err := time.Parse(f, input); err == nil {
			t = parsed
			return formatTime(t, fmt.Sprintf("Parsed as \"%s\"", f))
		}
	}

	return "", fmt.Errorf("unable to parse %q — try a Unix timestamp or an ISO-8601 date string", input)
}

func formatTime(t time.Time, source string) (string, error) {
	var sb strings.Builder

	sb.WriteString(section("Input"))
	sb.WriteString(fmt.Sprintf("  Source: %s\n", source))

	sb.WriteString(section("Unix Timestamps"))
	sb.WriteString(kv("Seconds", strconv.FormatInt(t.Unix(), 10)))
	sb.WriteString(kv("Milliseconds", strconv.FormatInt(t.UnixMilli(), 10)))
	sb.WriteString(kv("Microseconds", strconv.FormatInt(t.UnixMicro(), 10)))
	sb.WriteString(kv("Nanoseconds", strconv.FormatInt(t.UnixNano(), 10)))

	sb.WriteString(section("Time Zones"))
	for _, tz := range tzones {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			continue
		}
		local := t.In(loc)
		sb.WriteString(fmt.Sprintf("  %-25s: %s\n", tz, local.Format("2006-01-02 15:04:05 MST")))
	}

	sb.WriteString(section("Relative"))
	diff := time.Since(t)
	if diff > 0 {
		sb.WriteString(fmt.Sprintf("  %s ago\n", formatDuration(diff)))
	} else {
		sb.WriteString(fmt.Sprintf("  in %s\n", formatDuration(-diff)))
	}

	sb.WriteString(section("Formats"))
	sb.WriteString(kv("RFC3339", t.UTC().Format(time.RFC3339)))
	sb.WriteString(kv("RFC1123", t.UTC().Format(time.RFC1123)))
	sb.WriteString(kv("ISO-8601", t.UTC().Format("2006-01-02T15:04:05.000Z")))
	sb.WriteString(kv("HTTP Date", t.UTC().Format(http_date)))

	return sb.String(), nil
}

const http_date = "Mon, 02 Jan 2006 15:04:05 GMT"
