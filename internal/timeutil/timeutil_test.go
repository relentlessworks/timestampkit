package timeutil

import (
	"testing"
	"time"
)

func TestInfo(t *testing.T) {
	tm := time.Date(2023, 9, 10, 13, 20, 0, 0, time.UTC)
	info := Info(tm)

	if info.Unix != 1694352000 {
		t.Errorf("expected unix=1694352000, got %d", info.Unix)
	}
	if info.UnixMilli != 1694352000000 {
		t.Errorf("expected unix_milli=1694352000000, got %d", info.UnixMilli)
	}
	if info.ISO8601 != "2023-09-10T13:20:00Z" {
		t.Errorf("expected iso8601=2023-09-10T13:20:00Z, got %s", info.ISO8601)
	}
	if info.RFC3339 != "2023-09-10T13:20:00Z" {
		t.Errorf("expected rfc3339=2023-09-10T13:20:00Z, got %s", info.RFC3339)
	}
	if info.Date != "2023-09-10" {
		t.Errorf("expected date=2023-09-10, got %s", info.Date)
	}
	if info.Time != "13:20:00" {
		t.Errorf("expected time=13:20:00, got %s", info.Time)
	}
	if info.Weekday != "Sunday" {
		t.Errorf("expected weekday=Sunday, got %s", info.Weekday)
	}
	if info.DayOfYear != 253 {
		t.Errorf("expected day_of_year=253, got %d", info.DayOfYear)
	}
	if info.Timezone != "UTC" {
		t.Errorf("expected timezone=UTC, got %s", info.Timezone)
	}
	if info.Offset != "+0000" {
		t.Errorf("expected offset=+0000, got %s", info.Offset)
	}
}

func TestParseTimestamp_Now(t *testing.T) {
	before := time.Now()
	tm, err := ParseTimestamp("now")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := time.Now()
	if tm.Before(before) || tm.After(after.Add(time.Second)) {
		t.Errorf("expected time between %v and %v, got %v", before, after.Add(time.Second), tm)
	}
}

func TestParseTimestamp_UnixSeconds(t *testing.T) {
	tm, err := ParseTimestamp("1694352000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tm.Unix() != 1694352000 {
		t.Errorf("expected unix=1694352000, got %d", tm.Unix())
	}
}

func TestParseTimestamp_UnixMilli(t *testing.T) {
	tm, err := ParseTimestamp("1694352000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tm.UnixMilli() != 1694352000000 {
		t.Errorf("expected unix_milli=1694352000000, got %d", tm.UnixMilli())
	}
}

func TestParseTimestamp_ISO8601(t *testing.T) {
	tm, err := ParseTimestamp("2023-09-10T13:20:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tm.Year() != 2023 || tm.Month() != 9 || tm.Day() != 10 {
		t.Errorf("expected 2023-09-10, got %v", tm)
	}
	if tm.Hour() != 13 || tm.Minute() != 20 {
		t.Errorf("expected 13:20, got %v", tm)
	}
}

func TestParseTimestamp_DateOnly(t *testing.T) {
	tm, err := ParseTimestamp("2023-09-10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tm.Year() != 2023 || tm.Month() != 9 || tm.Day() != 10 {
		t.Errorf("expected 2023-09-10, got %v", tm)
	}
}

func TestParseTimestamp_DateTime(t *testing.T) {
	tm, err := ParseTimestamp("2023-09-10 12:00:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tm.Hour() != 12 || tm.Minute() != 0 || tm.Second() != 0 {
		t.Errorf("expected 12:00:00, got %v", tm)
	}
}

func TestParseTimestamp_RFC1123(t *testing.T) {
	tm, err := ParseTimestamp("Sun, 10 Sep 2023 13:20:00 UTC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tm.Unix() != 1694352000 {
		t.Errorf("expected unix=1694352000, got %d", tm.Unix())
	}
}

func TestParseTimestamp_Invalid(t *testing.T) {
	_, err := ParseTimestamp("not-a-date")
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestParseTimestamp_Empty(t *testing.T) {
	_, err := ParseTimestamp("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseTimestampWithTZ(t *testing.T) {
	tm, err := ParseTimestampWithTZ("2023-09-10 12:00:00", "America/New_York")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	loc, _ := time.LoadLocation("America/New_York")
	if tm.Location().String() != loc.String() {
		t.Errorf("expected timezone=America/New_York, got %s", tm.Location().String())
	}
}

func TestParseTimestampWithTZ_InvalidTZ(t *testing.T) {
	_, err := ParseTimestampWithTZ("2023-09-10", "Invalid/Zone")
	if err == nil {
		t.Error("expected error for invalid timezone")
	}
}

func TestParseTimestampWithTZ_NoTZ(t *testing.T) {
	tm, err := ParseTimestampWithTZ("1694352000", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tm.Unix() != 1694352000 {
		t.Errorf("expected unix=1694352000, got %d", tm.Unix())
	}
}

func TestResolveFormat_Presets(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"rfc3339", time.RFC3339},
		{"iso8601", "2006-01-02T15:04:05Z07:00"},
		{"rfc822", time.RFC822},
		{"rfc1123", time.RFC1123},
		{"kitchen", time.Kitchen},
		{"stamp", time.Stamp},
		{"date", "2006-01-02"},
		{"time", "15:04:05"},
		{"datetime", "2006-01-02 15:04:05"},
		{"2006-01-02 15:04:05", "2006-01-02 15:04:05"},
	}

	for _, tt := range tests {
		got := ResolveFormat(tt.input)
		if got != tt.expected {
			t.Errorf("ResolveFormat(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestParseDuration_Standard(t *testing.T) {
	d, err := ParseDuration("2h30m")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := 2*time.Hour + 30*time.Minute
	if d != expected {
		t.Errorf("expected %v, got %v", expected, d)
	}
}

func TestParseDuration_Days(t *testing.T) {
	d, err := ParseDuration("1d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 24*time.Hour {
		t.Errorf("expected %v, got %v", 24*time.Hour, d)
	}
}

func TestParseDuration_Weeks(t *testing.T) {
	d, err := ParseDuration("1w")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 7*24*time.Hour {
		t.Errorf("expected %v, got %v", 7*24*time.Hour, d)
	}
}

func TestParseDuration_Combined(t *testing.T) {
	d, err := ParseDuration("3d4h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := 3*24*time.Hour + 4*time.Hour
	if d != expected {
		t.Errorf("expected %v, got %v", expected, d)
	}
}

func TestParseDuration_Negative(t *testing.T) {
	d, err := ParseDuration("-1d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != -24*time.Hour {
		t.Errorf("expected %v, got %v", -24*time.Hour, d)
	}
}

func TestParseDuration_Complex(t *testing.T) {
	d, err := ParseDuration("1w2d3h4m5s")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := 7*24*time.Hour + 2*24*time.Hour + 3*time.Hour + 4*time.Minute + 5*time.Second
	if d != expected {
		t.Errorf("expected %v, got %v", expected, d)
	}
}

func TestParseDuration_Empty(t *testing.T) {
	_, err := ParseDuration("")
	if err == nil {
		t.Error("expected error for empty duration")
	}
}

func TestParseDuration_Invalid(t *testing.T) {
	_, err := ParseDuration("abc")
	if err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestDiff(t *testing.T) {
	from := time.Unix(1694352000, 0)
	to := time.Unix(1694438400, 0) // 1 day later
	diff := Diff(from, to)

	if diff.Seconds != 86400 {
		t.Errorf("expected seconds=86400, got %f", diff.Seconds)
	}
	if diff.Minutes != 1440 {
		t.Errorf("expected minutes=1440, got %f", diff.Minutes)
	}
	if diff.Hours != 24 {
		t.Errorf("expected hours=24, got %f", diff.Hours)
	}
	if diff.Days != 1 {
		t.Errorf("expected days=1, got %f", diff.Days)
	}
}

func TestDiff_Negative(t *testing.T) {
	from := time.Unix(1694438400, 0)
	to := time.Unix(1694352000, 0) // 1 day earlier
	diff := Diff(from, to)

	if diff.Seconds != -86400 {
		t.Errorf("expected seconds=-86400, got %f", diff.Seconds)
	}
}

func TestHumanDiff(t *testing.T) {
	tests := []struct {
		d        time.Duration
		contains string
	}{
		{30 * time.Second, "seconds"},
		{5 * time.Minute, "minutes"},
		{3 * time.Hour, "hours"},
		{2 * 24 * time.Hour, "days"},
		{2 * 7 * 24 * time.Hour, "weeks"},
	}

	for _, tt := range tests {
		result := HumanDiff(tt.d)
		if !contains(result, tt.contains) {
			t.Errorf("HumanDiff(%v) = %q, expected to contain %q", tt.d, result, tt.contains)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestRelativeTime_Past(t *testing.T) {
	tm := time.Now().Add(-2 * time.Hour)
	rel := RelativeTime(tm)
	if rel == "" {
		t.Error("expected non-empty relative time")
	}
}

func TestRelativeTime_Future(t *testing.T) {
	tm := time.Now().Add(3 * 24 * time.Hour)
	rel := RelativeTime(tm)
	if rel == "" {
		t.Error("expected non-empty relative time")
	}
}

func TestRelativeTime_JustNow(t *testing.T) {
	tm := time.Now()
	rel := RelativeTime(tm)
	if rel != "just now" {
		t.Errorf("expected 'just now', got %q", rel)
	}
}

func TestCommonTimezones(t *testing.T) {
	tzs := CommonTimezones()
	if len(tzs) == 0 {
		t.Error("expected non-empty timezone list")
	}
	found := false
	for _, tz := range tzs {
		if tz == "UTC" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected UTC in timezone list")
	}
}

func TestIsNumeric(t *testing.T) {
	if !isNumeric("12345") {
		t.Error("expected true for '12345'")
	}
	if isNumeric("abc") {
		t.Error("expected false for 'abc'")
	}
	if isNumeric("") {
		t.Error("expected false for empty string")
	}
	if isNumeric("12.34") {
		t.Error("expected false for '12.34'")
	}
}
