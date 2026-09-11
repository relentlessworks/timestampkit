# timestampkit

Agentic-first timestamp conversion and date math service. Convert between Unix timestamps and human-readable dates, perform date arithmetic, timezone conversion, and relative time calculations.

**The agent IS the interface.** No UI, no SDK. Plain text API, agent-driven, single Go binary.

## Quick Start

```bash
# Build
make build

# Run (defaults to :7290)
./timestampkit

# Or with custom config
./timestampkit -addr :8080 -secret my-secret
```

## Auth Flow

```bash
# 1. Request OTP
curl -X POST localhost:7290/auth/request -d 'user@example.com'

# 2. Verify OTP (check stderr for code in dev mode)
curl -X POST localhost:7290/auth/verify -d 'user@example.com=123456'
# → token=abc123... email=user@example.com usage=send as Authorization: Bearer <token>

# 3. Use the token
curl -H "Authorization: Bearer abc123..." localhost:7290/now
```

## API Reference

| Method | Path | Body | Description |
|--------|------|------|-------------|
| POST | /auth/request | email | Request OTP |
| POST | /auth/verify | email=code | Verify OTP, get bearer token |
| GET | /now?tz=UTC | — | Current time in all formats |
| POST | /convert?tz=UTC | timestamp | Convert timestamp to all formats |
| POST | /format?tz=UTC | timestamp\|format | Format timestamp with custom layout |
| POST | /add?tz=UTC | timestamp\|duration | Add duration to timestamp |
| POST | /diff | from\|to | Difference between timestamps |
| POST | /relative | timestamp | Human-readable relative time |
| POST | /parse | date string | Parse date string to Unix timestamp |
| GET | /timezones | — | List common IANA timezones |
| GET | /help | — | Operating manual |
| POST | /mcp | JSON-RPC | MCP endpoint for chat clients |

### Timestamp Formats Accepted

- Unix seconds: `1694352000`
- Unix milliseconds: `1694352000000`
- ISO 8601: `2023-09-10T13:20:00Z`
- RFC 2822: `Mon, 10 Sep 2023 13:20:00 UTC`
- Date only: `2023-09-10`
- Date+time: `2023-09-10 13:20:00`
- `now` for current time

### Duration Format

Extends Go's `time.ParseDuration` with days (`d`) and weeks (`w`):
- `2h30m` — 2 hours 30 minutes
- `1d` — 1 day
- `1w` — 1 week
- `3d4h` — 3 days 4 hours
- `-1d` — negative duration (subtract)

### Format Presets

`rfc3339`, `iso8601`, `rfc822`, `rfc1123`, `kitchen`, `stamp`, `date`, `time`, `datetime`

Or use Go time layout: `2006-01-02 15:04:05`

## Response Format

Plain text by default (key=value pairs, one record per line):
```
unix=1694352000 iso8601=2023-09-10T13:20:00Z date=2023-09-10 time=13:20:00 weekday=Sunday
```

JSON on demand via `Accept: application/json` or `?format=json`:
```json
{"unix":1694352000,"iso8601":"2023-09-10T13:20:00Z","date":"2023-09-10"}
```

Errors include hints:
```
error: missing auth token | hint: call POST /auth/request with email to get an OTP
```

## Configuration

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| -addr | TIMESTAMPKIT_ADDR | :7290 | Listen address |
| -secret | TIMESTAMPKIT_SECRET | auto-generated | Token signing secret |

## Build

```bash
make build    # CGO_ENABLED=0, single binary
make test     # go test -race
make vet      # go vet
```

## License

MIT
