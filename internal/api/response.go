package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// wantsJSON checks if the client wants JSON responses.
func wantsJSON(r *http.Request) bool {
	if r.URL.Query().Get("format") == "json" {
		return true
	}
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

// writeRecord writes a plain text record or JSON depending on the client preference.
func writeRecord(w http.ResponseWriter, r *http.Request, fields map[string]interface{}) {
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fields)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	var sb strings.Builder
	for k, v := range fields {
		sb.WriteString(fmt.Sprintf("%s=%v ", k, v))
	}
	out := strings.TrimSpace(sb.String())
	fmt.Fprintln(w, out)
}

// writeList writes a list of records, one per line (plain text) or as a JSON array.
func writeList(w http.ResponseWriter, r *http.Request, records []map[string]interface{}) {
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(records)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	for _, rec := range records {
		var sb strings.Builder
		for k, v := range rec {
			sb.WriteString(fmt.Sprintf("%s=%v ", k, v))
		}
		fmt.Fprintln(w, strings.TrimSpace(sb.String()))
	}
}

// writeError writes an error response with a hint.
func writeError(w http.ResponseWriter, r *http.Request, status int, msg, hint string) {
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{
			"error": msg,
			"hint":  hint,
		})
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, "error: %s | hint: %s\n", msg, hint)
}

// writeText writes a plain text response.
func writeText(w http.ResponseWriter, r *http.Request, text string) {
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"result": text})
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, text)
}
