package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/relentlessworks/timestampkit/internal/timeutil"
)

// MCPRequest is a JSON-RPC 2.0 request.
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse is a JSON-RPC 2.0 response.
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError is a JSON-RPC 2.0 error.
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPTool represents a tool available via MCP.
type MCPTool struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// handleMCP implements a minimal MCP JSON-RPC 2.0 endpoint.
func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeMCPError(w, nil, -32700, "parse error")
		return
	}

	var req MCPRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeMCPError(w, nil, -32700, "parse error")
		return
	}

	if req.JSONRPC == "" {
		req.JSONRPC = "2.0"
	}

	switch req.Method {
	case "initialize":
		writeMCPResult(w, req.ID, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]interface{}{
				"name":    "timestampkit",
				"version": "0.1.0",
			},
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
		})

	case "tools/list":
		writeMCPResult(w, req.ID, map[string]interface{}{
			"tools": mcpTools(),
		})

	case "tools/call":
		s.handleMCPToolCall(w, r, req)

	case "ping":
		writeMCPResult(w, req.ID, map[string]interface{}{})

	default:
		writeMCPError(w, req.ID, -32601, "method not found: "+req.Method)
	}
}

func mcpTools() []MCPTool {
	return []MCPTool{
		{
			Name:        "now",
			Description: "Get current time in multiple formats (unix, iso8601, rfc3339, date, time, weekday, timezone, offset)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"tz": map[string]interface{}{
						"type":        "string",
						"description": "IANA timezone name (e.g. America/New_York). Defaults to UTC.",
					},
				},
			},
		},
		{
			Name:        "convert",
			Description: "Convert a timestamp to all formats (unix, iso8601, rfc3339, rfc822, rfc1123, date, time, weekday, day_of_year, week_number, timezone, offset)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"timestamp": map[string]interface{}{
						"type":        "string",
						"description": "Unix timestamp, ISO 8601 date, or 'now'",
					},
					"tz": map[string]interface{}{
						"type":        "string",
						"description": "IANA timezone name. Defaults to UTC.",
					},
				},
				"required": []string{"timestamp"},
			},
		},
		{
			Name:        "format",
			Description: "Format a timestamp with a custom format or preset (rfc3339, iso8601, rfc822, rfc1123, kitchen, stamp, date, time, datetime)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"timestamp": map[string]interface{}{
						"type":        "string",
						"description": "Unix timestamp, ISO 8601 date, or 'now'",
					},
					"format": map[string]interface{}{
						"type":        "string",
						"description": "Format preset or Go time layout (e.g. 2006-01-02 15:04:05)",
					},
					"tz": map[string]interface{}{
						"type":        "string",
						"description": "IANA timezone name. Defaults to UTC.",
					},
				},
				"required": []string{"timestamp", "format"},
			},
		},
		{
			Name:        "add",
			Description: "Add a duration to a timestamp. Supports days (d) and weeks (w) beyond Go stdlib.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"timestamp": map[string]interface{}{
						"type":        "string",
						"description": "Unix timestamp, ISO 8601 date, or 'now'",
					},
					"duration": map[string]interface{}{
						"type":        "string",
						"description": "Duration like 2h30m, 1d, 1w, 3d4h, -1d",
					},
					"tz": map[string]interface{}{
						"type":        "string",
						"description": "IANA timezone name. Defaults to UTC.",
					},
				},
				"required": []string{"timestamp", "duration"},
			},
		},
		{
			Name:        "diff",
			Description: "Compute the difference between two timestamps (seconds, minutes, hours, days, weeks, human)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"from": map[string]interface{}{
						"type":        "string",
						"description": "From timestamp (Unix, ISO 8601, or 'now')",
					},
					"to": map[string]interface{}{
						"type":        "string",
						"description": "To timestamp (Unix, ISO 8601, or 'now')",
					},
				},
				"required": []string{"from", "to"},
			},
		},
		{
			Name:        "relative",
			Description: "Get human-readable relative time (e.g. '2 hours ago', 'in 3 days')",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"timestamp": map[string]interface{}{
						"type":        "string",
						"description": "Unix timestamp, ISO 8601 date, or 'now'",
					},
				},
				"required": []string{"timestamp"},
			},
		},
		{
			Name:        "parse",
			Description: "Parse any date string to Unix timestamp and ISO 8601",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"input": map[string]interface{}{
						"type":        "string",
						"description": "Date string to parse (e.g. '2023-09-10 12:00:00')",
					},
				},
				"required": []string{"input"},
			},
		},
		{
			Name:        "timezones",
			Description: "List common IANA timezone names with current time and offset",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

func (s *Server) handleMCPToolCall(w http.ResponseWriter, r *http.Request, req MCPRequest) {
	var params struct {
		Name      string            `json:"name"`
		Arguments map[string]string `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeMCPError(w, req.ID, -32602, "invalid params")
		return
	}

	var result map[string]interface{}

	switch params.Name {
	case "now":
		tz := params.Arguments["tz"]
		now := time.Now()
		if tz != "" {
			loc, err := time.LoadLocation(tz)
			if err != nil {
				writeMCPError(w, req.ID, -32602, "unknown timezone: "+tz)
				return
			}
			now = now.In(loc)
		}
		info := timeutil.Info(now)
		result = structToMap(info)

	case "convert":
		ts := params.Arguments["timestamp"]
		tz := params.Arguments["tz"]
		t, err := timeutil.ParseTimestampWithTZ(ts, tz)
		if err != nil {
			writeMCPError(w, req.ID, -32602, "could not parse timestamp: "+ts)
			return
		}
		info := timeutil.Info(t)
		result = structToMap(info)

	case "format":
		ts := params.Arguments["timestamp"]
		formatStr := params.Arguments["format"]
		tz := params.Arguments["tz"]
		t, err := timeutil.ParseTimestampWithTZ(ts, tz)
		if err != nil {
			writeMCPError(w, req.ID, -32602, "could not parse timestamp: "+ts)
			return
		}
		layout := timeutil.ResolveFormat(formatStr)
		result = map[string]interface{}{
			"input":    ts,
			"format":   formatStr,
			"result":   t.Format(layout),
			"timezone": t.Location().String(),
		}

	case "add":
		ts := params.Arguments["timestamp"]
		durStr := params.Arguments["duration"]
		tz := params.Arguments["tz"]
		t, err := timeutil.ParseTimestampWithTZ(ts, tz)
		if err != nil {
			writeMCPError(w, req.ID, -32602, "could not parse timestamp: "+ts)
			return
		}
		d, err := timeutil.ParseDuration(durStr)
		if err != nil {
			writeMCPError(w, req.ID, -32602, "could not parse duration: "+durStr)
			return
		}
		resultTime := t.Add(d)
		info := timeutil.Info(resultTime)
		result = map[string]interface{}{
			"input":    ts,
			"duration": durStr,
			"unix":     info.Unix,
			"iso8601":  info.ISO8601,
			"rfc3339":  info.RFC3339,
			"date":     info.Date,
			"time":     info.Time,
			"timezone": info.Timezone,
		}

	case "diff":
		fromStr := params.Arguments["from"]
		toStr := params.Arguments["to"]
		from, err := timeutil.ParseTimestamp(fromStr)
		if err != nil {
			writeMCPError(w, req.ID, -32602, "could not parse 'from': "+fromStr)
			return
		}
		to, err := timeutil.ParseTimestamp(toStr)
		if err != nil {
			writeMCPError(w, req.ID, -32602, "could not parse 'to': "+toStr)
			return
		}
		diff := timeutil.Diff(from, to)
		human := timeutil.HumanDiff(to.Sub(from))
		result = map[string]interface{}{
			"from":    fromStr,
			"to":      toStr,
			"seconds": diff.Seconds,
			"minutes": diff.Minutes,
			"hours":   diff.Hours,
			"days":    diff.Days,
			"weeks":   diff.Weeks,
			"human":   human,
		}

	case "relative":
		ts := params.Arguments["timestamp"]
		t, err := timeutil.ParseTimestamp(ts)
		if err != nil {
			writeMCPError(w, req.ID, -32602, "could not parse timestamp: "+ts)
			return
		}
		rel := timeutil.RelativeTime(t)
		result = map[string]interface{}{
			"input":    ts,
			"relative": rel,
			"unix":     t.Unix(),
			"iso8601":  t.Format("2006-01-02T15:04:05Z07:00"),
		}

	case "parse":
		input := params.Arguments["input"]
		t, err := timeutil.ParseTimestamp(input)
		if err != nil {
			writeMCPError(w, req.ID, -32602, "could not parse: "+input)
			return
		}
		info := timeutil.Info(t)
		result = map[string]interface{}{
			"input":      input,
			"unix":       info.Unix,
			"unix_milli": info.UnixMilli,
			"iso8601":    info.ISO8601,
			"rfc3339":    info.RFC3339,
			"date":       info.Date,
			"time":       info.Time,
			"timezone":   info.Timezone,
		}

	case "timezones":
		tzs := timeutil.CommonTimezones()
		var entries []map[string]interface{}
		for _, tz := range tzs {
			loc, err := time.LoadLocation(tz)
			if err != nil {
				continue
			}
			now := time.Now().In(loc)
			entries = append(entries, map[string]interface{}{
				"timezone": tz,
				"current":  now.Format("15:04:05"),
				"offset":   now.Format("-0700"),
			})
		}
		result = map[string]interface{}{
			"timezones": entries,
		}

	default:
		writeMCPError(w, req.ID, -32601, "unknown tool: "+params.Name)
		return
	}

	// Build MCP content response
	var textBuilder strings.Builder
	for k, v := range result {
		textBuilder.WriteString(k)
		textBuilder.WriteString("=")
		textBuilder.WriteString(strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(toJSON(v), "\"", ""), ",", " ")))
		textBuilder.WriteString(" ")
	}

	writeMCPResult(w, req.ID, map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": strings.TrimSpace(textBuilder.String()),
			},
		},
	})
}

func writeMCPResult(w http.ResponseWriter, id interface{}, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func writeMCPError(w http.ResponseWriter, id interface{}, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: msg,
		},
	})
}

func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func structToMap(info timeutil.TimeInfo) map[string]interface{} {
	return map[string]interface{}{
		"unix":         info.Unix,
		"unix_milli":   info.UnixMilli,
		"iso8601":      info.ISO8601,
		"rfc3339":      info.RFC3339,
		"rfc822":       info.RFC822,
		"rfc1123":      info.RFC1123,
		"date":         info.Date,
		"time":         info.Time,
		"weekday":      info.Weekday,
		"day_of_year":  info.DayOfYear,
		"week_number":  info.WeekNumber,
		"timezone":     info.Timezone,
		"offset":       info.Offset,
	}
}
