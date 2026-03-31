package tools

import (
	"fmt"
	"strings"
	"time"
)

var DateDiffT = &Tool{
	ID:          "date-diff",
	Name:        "Date / Time Diff",
	Description: "Calculate difference between two dates (days) or two datetimes (h/m/s)",
	Category:    "Data",
	Inputs: []InputDef{
		{
			Label:       "Start date / datetime",
			Placeholder: "e.g. 2024-01-15  or  2024-01-15T09:30:00  or  2024-01-15 09:30:00",
			Multiline:   false,
			Required:    true,
			AcceptsFile: false,
			FlagName:    "from",
			FlagShort:   "f",
		},
		{
			Label:       "End date / datetime",
			Placeholder: "e.g. 2025-03-19  or  2025-03-19T17:45:00  (leave empty for now)",
			Multiline:   false,
			Required:    false,
			AcceptsFile: false,
			FlagName:    "to",
			FlagShort:   "t",
		},
	},
	Run: func(inputs []string) (string, error) {
		fromStr := strings.TrimSpace(resolveInput(inputs, 0))
		if fromStr == "" {
			return "", fmt.Errorf("start date is required")
		}
		toStr := strings.TrimSpace(resolveInput(inputs, 1))
		if toStr == "" {
			toStr = time.Now().Format(time.RFC3339)
		}
		return dateDiff(fromStr, toStr)
	},
}

// dateFormats lists candidates in order from most to least specific.
var dateFormats = []struct {
	layout  string
	dateOnly bool // true when no time component is present
}{
	{"2006-01-02T15:04:05Z07:00", false},
	{"2006-01-02T15:04:05", false},
	{"2006-01-02 15:04:05", false},
	{"2006-01-02 15:04", false},
	{"02/01/2006 15:04:05", false},
	{"01/02/2006 15:04:05", false},
	{"02-Jan-2006 15:04:05", false},
	{"2006-01-02", true},
	{"02/01/2006", true},
	{"01/02/2006", true},
	{"02-Jan-2006", true},
	{"Jan 2, 2006", true},
}

func parseDate(s string) (time.Time, bool, error) {
	for _, f := range dateFormats {
		t, err := time.ParseInLocation(f.layout, s, time.Local)
		if err == nil {
			return t, f.dateOnly, nil
		}
	}
	return time.Time{}, false, fmt.Errorf("unrecognised date/time format: %q\n\nSupported formats:\n  2006-01-02\n  2006-01-02 15:04:05\n  2006-01-02T15:04:05\n  2006-01-02T15:04:05Z07:00\n  02/01/2006  (DD/MM/YYYY)\n  01/02/2006  (MM/DD/YYYY)", s)
}

func dateDiff(fromStr, toStr string) (string, error) {
	from, fromDateOnly, err := parseDate(fromStr)
	if err != nil {
		return "", fmt.Errorf("start: %w", err)
	}
	to, toDateOnly, err := parseDate(toStr)
	if err != nil {
		return "", fmt.Errorf("end: %w", err)
	}

	// Treat as date-only when both inputs are date-only.
	dateOnly := fromDateOnly && toDateOnly

	// Ensure from ≤ to for consistent display (swap + note direction).
	swapped := false
	if to.Before(from) {
		from, to = to, from
		fromStr, toStr = toStr, fromStr
		swapped = true
	}

	dur := to.Sub(from)

	var sb strings.Builder
	sb.WriteString(section("Date / Time Difference"))
	sb.WriteString(kv("From", fromStr))
	sb.WriteString(kv("To", toStr))
	if swapped {
		sb.WriteString(kv("Note", "end is before start — values swapped for display"))
	}

	if dateOnly {
		// Count calendar days using date truncation to avoid DST edge cases.
		fromDate := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
		toDate := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
		days := int(toDate.Sub(fromDate).Hours() / 24)

		sb.WriteString(section("Result"))
		sb.WriteString(kv("Days", fmt.Sprintf("%d", days)))
		sb.WriteString(kv("Weeks", fmt.Sprintf("%d weeks + %d day(s)", days/7, days%7)))

		// Calendar breakdown
		years, months, remDays := calendarDiff(fromDate, toDate)
		sb.WriteString(kv("Calendar", fmt.Sprintf("%d year(s), %d month(s), %d day(s)", years, months, remDays)))
	} else {
		totalSec := int64(dur.Seconds())
		hours := totalSec / 3600
		mins := (totalSec % 3600) / 60
		secs := totalSec % 60

		sb.WriteString(section("Result"))
		sb.WriteString(kv("Total seconds", fmt.Sprintf("%d", totalSec)))
		sb.WriteString(kv("Total minutes", fmt.Sprintf("%d  (%.2f)", int64(dur.Minutes()), dur.Minutes())))
		sb.WriteString(kv("Total hours", fmt.Sprintf("%d  (%.4f)", int64(dur.Hours()), dur.Hours())))
		sb.WriteString(kv("Breakdown", fmt.Sprintf("%d h  %02d m  %02d s", hours, mins, secs)))

		days := int(dur.Hours() / 24)
		remH := int(dur.Hours()) - days*24
		sb.WriteString(kv("With days", fmt.Sprintf("%d day(s)  %d h  %02d m  %02d s", days, remH, mins, secs)))
	}

	return sb.String(), nil
}

// calendarDiff computes the difference in whole years, months, and remaining days.
func calendarDiff(from, to time.Time) (years, months, days int) {
	years = to.Year() - from.Year()
	months = int(to.Month()) - int(from.Month())
	days = to.Day() - from.Day()

	if days < 0 {
		months--
		// Days in the previous month of 'to'
		prev := to.AddDate(0, -1, 0)
		daysInPrev := daysInMonth(prev.Year(), prev.Month())
		days += daysInPrev
	}
	if months < 0 {
		years--
		months += 12
	}
	return
}

func daysInMonth(y int, m time.Month) int {
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
