package timeutil

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TimeInfo holds a comprehensive breakdown of a time value.
type TimeInfo struct {
	Unix        int64  `json:"unix"`
	UnixMilli   int64  `json:"unix_milli"`
	ISO8601     string `json:"iso8601"`
	RFC3339     string `json:"rfc3339"`
	RFC822      string `json:"rfc822"`
	RFC1123     string `json:"rfc1123"`
	Date        string `json:"date"`
	Time        string `json:"time"`
	Weekday     string `json:"weekday"`
	DayOfYear   int    `json:"day_of_year"`
	WeekNumber  int    `json:"week_number"`
	Timezone    string `json:"timezone"`
	Offset      string `json:"offset"`
}

// DiffInfo holds the difference between two timestamps.
type DiffInfo struct {
	Seconds float64 `json:"seconds"`
	Minutes float64 `json:"minutes"`
	Hours   float64 `json:"hours"`
	Days    float64 `json:"days"`
	Weeks   float64 `json:"weeks"`
}

// Info returns a comprehensive breakdown of a time.Time.
func Info(t time.Time) TimeInfo {
	_, offset := t.Zone()
	offsetStr := fmt.Sprintf("%+03d%02d", offset/3600, (offset%3600)/60)
	yearDay := t.YearDay()
	_, week := t.ISOWeek()

	return TimeInfo{
		Unix:        t.Unix(),
		UnixMilli:   t.UnixMilli(),
		ISO8601:     t.Format("2006-01-02T15:04:05Z07:00"),
		RFC3339:     t.Format(time.RFC3339),
		RFC822:      t.Format(time.RFC822),
		RFC1123:     t.Format(time.RFC1123),
		Date:        t.Format("2006-01-02"),
		Time:        t.Format("15:04:05"),
		Weekday:     t.Weekday().String(),
		DayOfYear:   yearDay,
		WeekNumber:  week,
		Timezone:    t.Location().String(),
		Offset:      offsetStr,
	}
}

// ParseTimestamp parses a timestamp from various formats.
// Accepts: Unix seconds, Unix milliseconds, ISO 8601, RFC 2822,
// common date formats, or "now".
func ParseTimestamp(input string) (time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return time.Time{}, fmt.Errorf("empty input")
	}

	if strings.EqualFold(input, "now") {
		return time.Now(), nil
	}

	// Try Unix timestamp (seconds or milliseconds)
	if isNumeric(input) {
		n, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid number: %s", input)
		}
		// Heuristic: if the number is very large, treat as milliseconds
		if len(input) >= 13 {
			return time.UnixMilli(n).UTC(), nil
		}
		return time.Unix(n, 0).UTC(), nil
	}

	// Try various date formats
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		time.RFC1123,
		time.RFC1123Z,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		"01/02/2006 15:04:05",
		"01/02/2006 15:04",
		"01/02/2006",
		"02 Jan 2006 15:04:05",
		"02 Jan 2006 15:04",
		"02 Jan 2006",
		"Jan 2, 2006 3:04:05 PM",
		"Jan 2, 2006 3:04 PM",
		"Jan 2, 2006",
		"2 Jan 2006",
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, input)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse: %s", input)
}

// ParseTimestampWithTZ parses a timestamp and optionally applies a timezone.
func ParseTimestampWithTZ(input, tz string) (time.Time, error) {
	t, err := ParseTimestamp(input)
	if err != nil {
		return time.Time{}, err
	}

	if tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return time.Time{}, fmt.Errorf("unknown timezone: %s", tz)
		}
		// If the parsed time has no zone info (was parsed as UTC),
		// reinterpret it in the target timezone
		if t.Location() == time.UTC && !strings.Contains(input, "Z") && !strings.Contains(input, "+") && !strings.Contains(input, "T") {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
		} else {
			t = t.In(loc)
		}
	}

	return t, nil
}

// ResolveFormat maps format preset names to Go time layouts.
func ResolveFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "rfc3339":
		return time.RFC3339
	case "iso8601":
		return "2006-01-02T15:04:05Z07:00"
	case "rfc822":
		return time.RFC822
	case "rfc1123":
		return time.RFC1123
	case "kitchen":
		return time.Kitchen
	case "stamp":
		return time.Stamp
	case "date":
		return "2006-01-02"
	case "time":
		return "15:04:05"
	case "datetime":
		return "2006-01-02 15:04:05"
	default:
		return format
	}
}

var durationRe = regexp.MustCompile(`^(-?)(\d+w)?(\d+d)?(\d+h)?(\d+m)?(\d+s)?(\d+ms)?(\d+us)?(\d+ns)?$`)

// ParseDuration extends time.ParseDuration with support for days (d) and weeks (w).
// Examples: "2h30m", "1d", "1w", "3d4h", "-1d"
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// If no 'd' or 'w' units, use standard time.ParseDuration
	if !strings.Contains(s, "d") && !strings.Contains(s, "w") {
		d, err := time.ParseDuration(s)
		if err != nil {
			return 0, fmt.Errorf("invalid duration: %s", s)
		}
		return d, nil
	}

	matches := durationRe.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	negative := matches[1] == "-"
	var total time.Duration

	if matches[2] != "" {
		n, _ := strconv.Atoi(strings.TrimSuffix(matches[2], "w"))
		total += time.Duration(n) * 7 * 24 * time.Hour
	}
	if matches[3] != "" {
		n, _ := strconv.Atoi(strings.TrimSuffix(matches[3], "d"))
		total += time.Duration(n) * 24 * time.Hour
	}
	if matches[4] != "" {
		n, _ := strconv.Atoi(strings.TrimSuffix(matches[4], "h"))
		total += time.Duration(n) * time.Hour
	}
	if matches[5] != "" {
		n, _ := strconv.Atoi(strings.TrimSuffix(matches[5], "m"))
		total += time.Duration(n) * time.Minute
	}
	if matches[6] != "" {
		n, _ := strconv.Atoi(strings.TrimSuffix(matches[6], "s"))
		total += time.Duration(n) * time.Second
	}
	if matches[7] != "" {
		n, _ := strconv.Atoi(strings.TrimSuffix(matches[7], "ms"))
		total += time.Duration(n) * time.Millisecond
	}
	if matches[8] != "" {
		n, _ := strconv.Atoi(strings.TrimSuffix(matches[8], "us"))
		total += time.Duration(n) * time.Microsecond
	}
	if matches[9] != "" {
		n, _ := strconv.Atoi(strings.TrimSuffix(matches[9], "ns"))
		total += time.Duration(n) * time.Nanosecond
	}

	if total == 0 && s != "0" && s != "-0" {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	if negative {
		total = -total
	}

	return total, nil
}

// Diff computes the difference between two timestamps.
func Diff(from, to time.Time) DiffInfo {
	d := to.Sub(from)
	absD := d
	if absD < 0 {
		absD = -absD
	}
	seconds := d.Seconds()
	return DiffInfo{
		Seconds: seconds,
		Minutes: seconds / 60,
		Hours:   seconds / 3600,
		Days:    seconds / 86400,
		Weeks:   seconds / (86400 * 7),
	}
}

// HumanDiff returns a human-readable duration string.
func HumanDiff(d time.Duration) string {
	abs := d
	if abs < 0 {
		abs = -abs
	}

	switch {
	case abs < time.Minute:
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	case abs < time.Hour:
		return fmt.Sprintf("%.1f minutes", d.Minutes())
	case abs < 24*time.Hour:
		return fmt.Sprintf("%.1f hours", d.Hours())
	case abs < 7*24*time.Hour:
		return fmt.Sprintf("%.1f days", d.Hours()/24)
	case abs < 30*24*time.Hour:
		return fmt.Sprintf("%.1f weeks", d.Hours()/(24*7))
	case abs < 365*24*time.Hour:
		return fmt.Sprintf("%.1f months", d.Hours()/(24*30))
	default:
		return fmt.Sprintf("%.1f years", d.Hours()/(24*365))
	}
}

// RelativeTime returns a human-readable relative time string like
// "2 hours ago", "in 3 days", "just now".
func RelativeTime(t time.Time) string {
	d := time.Since(t)
	return relativeDuration(d)
}

func relativeDuration(d time.Duration) string {
	abs := d
	future := false
	if d < 0 {
		abs = -d
		future = true
	}

	var unit string
	var value float64

	switch {
	case abs < 10*time.Second:
		return "just now"
	case abs < time.Minute:
		value = abs.Seconds()
		unit = "second"
	case abs < time.Hour:
		value = abs.Minutes()
		unit = "minute"
	case abs < 24*time.Hour:
		value = abs.Hours()
		unit = "hour"
	case abs < 7*24*time.Hour:
		value = abs.Hours() / 24
		unit = "day"
	case abs < 30*24*time.Hour:
		value = abs.Hours() / (24 * 7)
		unit = "week"
	case abs < 365*24*time.Hour:
		value = abs.Hours() / (24 * 30)
		unit = "month"
	default:
		value = abs.Hours() / (24 * 365)
		unit = "year"
	}

	rounded := int(math.Round(value))
	if rounded == 0 {
		return "just now"
	}

	if unit != "second" && rounded == 1 {
		// keep singular
	} else {
		unit += "s"
	}

	if future {
		return fmt.Sprintf("in %d %s", rounded, unit)
	}
	return fmt.Sprintf("%d %s ago", rounded, unit)
}

// CommonTimezones returns a list of commonly used IANA timezone names.
func CommonTimezones() []string {
	return []string{
		"UTC",
		"America/New_York",
		"America/Chicago",
		"America/Denver",
		"America/Los_Angeles",
		"America/Anchorage",
		"America/Toronto",
		"America/Vancouver",
		"America/Mexico_City",
		"America/Sao_Paulo",
		"America/Buenos_Aires",
		"Europe/London",
		"Europe/Dublin",
		"Europe/Paris",
		"Europe/Berlin",
		"Europe/Madrid",
		"Europe/Rome",
		"Europe/Amsterdam",
		"Europe/Stockholm",
		"Europe/Oslo",
		"Europe/Helsinki",
		"Europe/Warsand",
		"Europe/Moscow",
		"Europe/Istanbul",
		"Africa/Cairo",
		"Africa/Johannesburg",
		"Africa/Lagos",
		"Asia/Dubai",
		"Asia/Tehran",
		"Asia/Karachi",
		"Asia/Kolkata",
		"Asia/Dhaka",
		"Asia/Bangkok",
		"Asia/Singapore",
		"Asia/Hong_Kong",
		"Asia/Shanghai",
		"Asia/Tokyo",
		"Asia/Seoul",
		"Australia/Sydney",
		"Australia/Melbourne",
		"Australia/Perth",
		"Pacific/Auckland",
		"Pacific/Honolulu",
	}
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
