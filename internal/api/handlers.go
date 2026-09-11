package api

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/relentlessworks/timestampkit/internal/auth"
	"github.com/relentlessworks/timestampkit/internal/timeutil"
)

// Server holds dependencies for the API.
type Server struct {
	auth *auth.Auth
}

// NewServer creates a new API server.
func NewServer(a *auth.Auth) *Server {
	return &Server{auth: a}
}

// Routes returns the HTTP mux with all routes registered.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/help", s.handleHelp)
	mux.HandleFunc("/.well-known/agent.md", s.handleHelp)
	mux.HandleFunc("/auth/request", s.handleAuthRequest)
	mux.HandleFunc("/auth/verify", s.handleAuthVerify)
	mux.HandleFunc("/mcp", s.handleMCP)

	// Authenticated routes
	mux.HandleFunc("/now", requireAuth(s.auth, s.handleNow))
	mux.HandleFunc("/convert", requireAuth(s.auth, s.handleConvert))
	mux.HandleFunc("/format", requireAuth(s.auth, s.handleFormat))
	mux.HandleFunc("/add", requireAuth(s.auth, s.handleAdd))
	mux.HandleFunc("/diff", requireAuth(s.auth, s.handleDiff))
	mux.HandleFunc("/relative", requireAuth(s.auth, s.handleRelative))
	mux.HandleFunc("/parse", requireAuth(s.auth, s.handleParse))
	mux.HandleFunc("/timezones", requireAuth(s.auth, s.handleTimezones))

	return mux
}

func (s *Server) handleAuthRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not read request body", "send email as plain text body")
		return
	}
	email := strings.TrimSpace(string(body))
	if email == "" {
		writeError(w, r, http.StatusBadRequest, "email is required", "send your email as the request body, e.g. curl -X POST localhost:7290/auth/request -d 'user@example.com'")
		return
	}
	s.auth.RequestOTP(email)
	writeText(w, r, "OTP sent to "+email+" (check stderr in dev mode)")
}

func (s *Server) handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not read request body", "send 'email=code' as plain text body")
		return
	}
	parts := strings.SplitN(strings.TrimSpace(string(body)), "=", 2)
	if len(parts) != 2 {
		writeError(w, r, http.StatusBadRequest, "invalid format", "send 'email=code' as the request body, e.g. curl -X POST localhost:7290/auth/verify -d 'user@example.com=123456'")
		return
	}
	email := strings.TrimSpace(parts[0])
	code := strings.TrimSpace(parts[1])
	if email == "" || code == "" {
		writeError(w, r, http.StatusBadRequest, "email and code are required", "send 'email=code' as the request body")
		return
	}
	token, err := s.auth.VerifyOTP(email, code)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "invalid OTP code", "request a new OTP via POST /auth/request with your email")
		return
	}
	writeRecord(w, r, map[string]interface{}{
		"token":  token,
		"email":  email,
		"usage":  "send as Authorization: Bearer <token>",
	})
}

func (s *Server) handleNow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use GET")
		return
	}
	tz := r.URL.Query().Get("tz")
	now := time.Now()
	if tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "unknown timezone: "+tz, "use GET /timezones to list valid timezone names, or omit tz to use UTC")
			return
		}
		now = now.In(loc)
	}
	info := timeutil.Info(now)
	writeRecord(w, r, map[string]interface{}{
		"unix":       info.Unix,
		"unix_milli": info.UnixMilli,
		"iso8601":    info.ISO8601,
		"rfc3339":    info.RFC3339,
		"rfc822":     info.RFC822,
		"date":       info.Date,
		"time":       info.Time,
		"weekday":    info.Weekday,
		"timezone":   info.Timezone,
		"offset":     info.Offset,
	})
}

func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not read request body", "send timestamp as plain text body")
		return
	}
	input := strings.TrimSpace(string(body))
	if input == "" {
		writeError(w, r, http.StatusBadRequest, "timestamp is required", "send a Unix timestamp, ISO 8601 date, or 'now' as the request body")
		return
	}
	tz := r.URL.Query().Get("tz")
	t, err := timeutil.ParseTimestampWithTZ(input, tz)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not parse timestamp: "+input, "use a Unix timestamp (e.g. 1694352000), ISO 8601 (e.g. 2023-09-10T13:20:00Z), or 'now'")
		return
	}
	info := timeutil.Info(t)
	writeRecord(w, r, map[string]interface{}{
		"unix":        info.Unix,
		"unix_milli":  info.UnixMilli,
		"iso8601":     info.ISO8601,
		"rfc3339":     info.RFC3339,
		"rfc822":      info.RFC822,
		"rfc1123":     info.RFC1123,
		"date":        info.Date,
		"time":        info.Time,
		"weekday":     info.Weekday,
		"day_of_year": info.DayOfYear,
		"week_number": info.WeekNumber,
		"timezone":    info.Timezone,
		"offset":      info.Offset,
	})
}

func (s *Server) handleFormat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not read request body", "send 'timestamp|format' as plain text body")
		return
	}
	input := strings.TrimSpace(string(body))
	if input == "" {
		writeError(w, r, http.StatusBadRequest, "timestamp and format are required", "send 'timestamp|format' as the request body, e.g. '1694352000|rfc3339' or 'now|2006-01-02 15:04:05'")
		return
	}
	parts := strings.SplitN(input, "|", 2)
	if len(parts) != 2 {
		writeError(w, r, http.StatusBadRequest, "invalid format", "send 'timestamp|format' as the request body, e.g. '1694352000|rfc3339' or 'now|2006-01-02 15:04:05'")
		return
	}
	tsStr := strings.TrimSpace(parts[0])
	formatStr := strings.TrimSpace(parts[1])
	if tsStr == "" || formatStr == "" {
		writeError(w, r, http.StatusBadRequest, "timestamp and format are required", "send 'timestamp|format' as the request body")
		return
	}
	tz := r.URL.Query().Get("tz")
	t, err := timeutil.ParseTimestampWithTZ(tsStr, tz)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not parse timestamp: "+tsStr, "use a Unix timestamp, ISO 8601 date, or 'now'")
		return
	}
	layout := timeutil.ResolveFormat(formatStr)
	formatted := t.Format(layout)
	writeRecord(w, r, map[string]interface{}{
		"input":     tsStr,
		"format":    formatStr,
		"result":    formatted,
		"timezone":  t.Location().String(),
	})
}

func (s *Server) handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not read request body", "send 'timestamp|duration' as plain text body")
		return
	}
	input := strings.TrimSpace(string(body))
	if input == "" {
		writeError(w, r, http.StatusBadRequest, "timestamp and duration are required", "send 'timestamp|duration' as the request body, e.g. 'now|2h30m' or '1694352000|1d'")
		return
	}
	parts := strings.SplitN(input, "|", 2)
	if len(parts) != 2 {
		writeError(w, r, http.StatusBadRequest, "invalid format", "send 'timestamp|duration' as the request body, e.g. 'now|2h30m' or '1694352000|1d'")
		return
	}
	tsStr := strings.TrimSpace(parts[0])
	durStr := strings.TrimSpace(parts[1])
	if tsStr == "" || durStr == "" {
		writeError(w, r, http.StatusBadRequest, "timestamp and duration are required", "send 'timestamp|duration' as the request body")
		return
	}
	tz := r.URL.Query().Get("tz")
	t, err := timeutil.ParseTimestampWithTZ(tsStr, tz)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not parse timestamp: "+tsStr, "use a Unix timestamp, ISO 8601 date, or 'now'")
		return
	}
	d, err := timeutil.ParseDuration(durStr)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not parse duration: "+durStr, "use Go duration format (e.g. 2h30m, 1d, 1w, 3d4h) with optional negative sign")
		return
	}
	result := t.Add(d)
	info := timeutil.Info(result)
	writeRecord(w, r, map[string]interface{}{
		"input":     tsStr,
		"duration":  durStr,
		"unix":      info.Unix,
		"iso8601":   info.ISO8601,
		"rfc3339":   info.RFC3339,
		"date":      info.Date,
		"time":      info.Time,
		"timezone":  info.Timezone,
	})
}

func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not read request body", "send 'from|to' as plain text body")
		return
	}
	input := strings.TrimSpace(string(body))
	if input == "" {
		writeError(w, r, http.StatusBadRequest, "from and to timestamps are required", "send 'from|to' as the request body, e.g. '1694352000|1694438400' or '2023-09-10|now'")
		return
	}
	parts := strings.SplitN(input, "|", 2)
	if len(parts) != 2 {
		writeError(w, r, http.StatusBadRequest, "invalid format", "send 'from|to' as the request body, e.g. '1694352000|1694438400' or '2023-09-10|now'")
		return
	}
	fromStr := strings.TrimSpace(parts[0])
	toStr := strings.TrimSpace(parts[1])
	if fromStr == "" || toStr == "" {
		writeError(w, r, http.StatusBadRequest, "from and to timestamps are required", "send 'from|to' as the request body")
		return
	}
	from, err := timeutil.ParseTimestamp(fromStr)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not parse 'from' timestamp: "+fromStr, "use a Unix timestamp, ISO 8601 date, or 'now'")
		return
	}
	to, err := timeutil.ParseTimestamp(toStr)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not parse 'to' timestamp: "+toStr, "use a Unix timestamp, ISO 8601 date, or 'now'")
		return
	}
	diff := timeutil.Diff(from, to)
	human := timeutil.HumanDiff(to.Sub(from))
	writeRecord(w, r, map[string]interface{}{
		"from":    fromStr,
		"to":      toStr,
		"seconds": diff.Seconds,
		"minutes": diff.Minutes,
		"hours":   diff.Hours,
		"days":    diff.Days,
		"weeks":   diff.Weeks,
		"human":   human,
	})
}

func (s *Server) handleRelative(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not read request body", "send timestamp as plain text body")
		return
	}
	input := strings.TrimSpace(string(body))
	if input == "" {
		writeError(w, r, http.StatusBadRequest, "timestamp is required", "send a Unix timestamp, ISO 8601 date, or 'now' as the request body")
		return
	}
	t, err := timeutil.ParseTimestamp(input)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not parse timestamp: "+input, "use a Unix timestamp, ISO 8601 date, or 'now'")
		return
	}
	rel := timeutil.RelativeTime(t)
	writeRecord(w, r, map[string]interface{}{
		"input":    input,
		"relative": rel,
		"unix":     t.Unix(),
		"iso8601":  t.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (s *Server) handleParse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not read request body", "send date string as plain text body")
		return
	}
	input := strings.TrimSpace(string(body))
	if input == "" {
		writeError(w, r, http.StatusBadRequest, "date string is required", "send a date string as the request body, e.g. '2023-09-10 13:20:00' or 'Mon, 10 Sep 2023 13:20:00 UTC'")
		return
	}
	t, err := timeutil.ParseTimestamp(input)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "could not parse date string: "+input, "try formats like '2023-09-10', '2023-09-10 13:20:00', '2023-09-10T13:20:00Z', or 'Mon, 10 Sep 2023 13:20:00 UTC'")
		return
	}
	info := timeutil.Info(t)
	writeRecord(w, r, map[string]interface{}{
		"input":      input,
		"unix":       info.Unix,
		"unix_milli": info.UnixMilli,
		"iso8601":    info.ISO8601,
		"rfc3339":    info.RFC3339,
		"date":       info.Date,
		"time":       info.Time,
		"timezone":   info.Timezone,
	})
}

func (s *Server) handleTimezones(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use GET")
		return
	}
	tzs := timeutil.CommonTimezones()
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		var records []map[string]interface{}
		for _, tz := range tzs {
			records = append(records, map[string]interface{}{"timezone": tz})
		}
		writeList(w, r, records)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	for _, tz := range tzs {
		// Show current time in each timezone
		loc, err := time.LoadLocation(tz)
		if err != nil {
			continue
		}
		now := time.Now().In(loc)
		w.Write([]byte("timezone=" + tz + " current=" + now.Format("15:04:05") + " offset=" + now.Format("-0700") + "\n"))
	}
}

func (s *Server) handleHelp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(helpText))
}

const helpText = `timestampkit — Agentic-first timestamp conversion and date math service

AUTH:
  1. Request OTP:   POST /auth/request   body: user@example.com
  2. Verify OTP:    POST /auth/verify     body: user@example.com=123456
     → Returns: token=xxx email=xxx usage=send as Authorization: Bearer <token>

ENDPOINTS (all require Authorization: Bearer <token>):

  GET  /now?tz=UTC
     → Current time in unix, iso8601, rfc3339, date, time, weekday, timezone, offset
     → tz is optional (IANA timezone name, e.g. America/New_York)

  POST /convert?tz=UTC    body: <timestamp>
     → Full breakdown: unix, unix_milli, iso8601, rfc3339, rfc822, rfc1123, date, time, weekday, day_of_year, week_number, timezone, offset
     → Accepts: Unix seconds, Unix millis, ISO 8601, RFC 2822, common date formats, or "now"

  POST /format?tz=UTC     body: <timestamp>|<format>
     → Custom format output: result=formatted_string
     → Format presets: rfc3339, iso8601, rfc822, rfc1123, kitchen, stamp, date, time, datetime
     → Or use Go time layout: 2006-01-02 15:04:05

  POST /add?tz=UTC        body: <timestamp>|<duration>
     → Add duration to timestamp: unix, iso8601, date, time, timezone
     → Duration: 2h30m, 1d, 1w, 3d4h, -1d (supports days and weeks beyond Go stdlib)

  POST /diff             body: <from>|<to>
     → Difference between timestamps: seconds, minutes, hours, days, weeks, human
     → Both from and to accept any timestamp format including "now"

  POST /relative         body: <timestamp>
     → Relative time: relative="2 hours ago" or "in 3 days"

  POST /parse            body: <date string>
     → Parse any date string to Unix timestamp and ISO 8601

  GET  /timezones
     → List common IANA timezone names with current time and offset

  GET  /help
     → This help text

  POST /mcp
     → MCP JSON-RPC 2.0 endpoint for chat client integration

RESPONSE FORMAT:
  Plain text by default (key=value pairs, one record per line)
  JSON via Accept: application/json header or ?format=json query param
  Errors: error: message | hint: what to do next

EXAMPLES:
  curl -H "Authorization: Bearer TOKEN" localhost:7290/now
  curl -H "Authorization: Bearer TOKEN" -d '1694352000' localhost:7290/convert
  curl -H "Authorization: Bearer TOKEN" -d 'now|1d' localhost:7290/add
  curl -H "Authorization: Bearer TOKEN" -d '2023-09-10|now' localhost:7290/diff
  curl -H "Authorization: Bearer TOKEN" -d '1694352000' localhost:7290/relative
`
