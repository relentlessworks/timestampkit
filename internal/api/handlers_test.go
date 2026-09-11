package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/relentlessworks/timestampkit/internal/auth"
)

func TestHandleHelp(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/help")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "timestampkit") {
		t.Error("expected help text to contain 'timestampkit'")
	}
	if !strings.Contains(string(body), "AUTH:") {
		t.Error("expected help text to contain 'AUTH:'")
	}
}

func TestHandleHelp_AgentMD(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/.well-known/agent.md")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthFlow(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	// Request OTP
	resp, err := http.Post(ts.URL+"/auth/request", "text/plain", strings.NewReader("user@example.com"))
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Verify with wrong code
	resp2, err := http.Post(ts.URL+"/auth/verify", "text/plain", strings.NewReader("user@example.com=000000"))
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp2.StatusCode)
	}
}

func TestAuthRequest_InvalidMethod(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/auth/request")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}
}

func TestAuthRequest_EmptyBody(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/auth/request", "text/plain", strings.NewReader(""))
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestAuthVerify_InvalidFormat(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/auth/verify", "text/plain", strings.NewReader("no-equals-sign"))
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestProtectedEndpoint_NoToken(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/now")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "hint:") {
		t.Error("expected error to contain hint")
	}
}

func TestProtectedEndpoint_InvalidToken(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/now", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestNow_WithToken(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	// Get a valid token
	a.RequestOTP("test@example.com")
	// We need to get the OTP - let's use the auth service directly
	// Since OTP is random, we'll use a workaround: generate token directly
	// by requesting OTP and then using the stored value
	// Actually, let's just use the auth's internal methods
	token := getTestToken(t, ts, a)

	resp, err := httpGetWithAuth(ts.URL+"/now", token)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "unix=") {
		t.Errorf("expected response to contain 'unix=', got: %s", string(body))
	}
}

func TestNow_JSON(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	token := getTestToken(t, ts, a)

	req, _ := http.NewRequest("GET", ts.URL+"/now?format=json", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		t.Errorf("expected JSON content type, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestConvert_WithToken(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	token := getTestToken(t, ts, a)

	resp, err := httpPostWithAuth(ts.URL+"/convert", "1694352000", token)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "unix=1694352000") {
		t.Errorf("expected response to contain 'unix=1694352000', got: %s", string(body))
	}
	if !strings.Contains(string(body), "iso8601=2023-09-10T13:20:00Z") {
		t.Errorf("expected response to contain iso8601=2023-09-10T13:20:00Z, got: %s", string(body))
	}
}

func TestConvert_Now(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	token := getTestToken(t, ts, a)

	resp, err := httpPostWithAuth(ts.URL+"/convert", "now", token)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestConvert_InvalidTimestamp(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	token := getTestToken(t, ts, a)

	resp, err := httpPostWithAuth(ts.URL+"/convert", "not-a-timestamp", token)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestFormat_WithToken(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	token := getTestToken(t, ts, a)

	resp, err := httpPostWithAuth(ts.URL+"/format", "1694352000|rfc3339", token)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "result=2023-09-10T13:20:00Z") {
		t.Errorf("expected response to contain 'result=2023-09-10T13:20:00Z', got: %s", string(body))
	}
}

func TestAdd_WithToken(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	token := getTestToken(t, ts, a)

	resp, err := httpPostWithAuth(ts.URL+"/add", "1694352000|1d", token)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "unix=1694438400") {
		t.Errorf("expected response to contain 'unix=1694438400', got: %s", string(body))
	}
}

func TestAdd_NegativeDuration(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	token := getTestToken(t, ts, a)

	resp, err := httpPostWithAuth(ts.URL+"/add", "1694352000|-1d", token)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "unix=1694265600") {
		t.Errorf("expected response to contain 'unix=1694265600', got: %s", string(body))
	}
}

func TestDiff_WithToken(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	token := getTestToken(t, ts, a)

	resp, err := httpPostWithAuth(ts.URL+"/diff", "1694352000|1694438400", token)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "seconds=86400") {
		t.Errorf("expected response to contain 'seconds=86400', got: %s", string(body))
	}
	if !strings.Contains(string(body), "days=1") {
		t.Errorf("expected response to contain 'days=1', got: %s", string(body))
	}
}

func TestRelative_WithToken(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	token := getTestToken(t, ts, a)

	resp, err := httpPostWithAuth(ts.URL+"/relative", "1694352000", token)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "relative=") {
		t.Errorf("expected response to contain 'relative=', got: %s", string(body))
	}
}

func TestParse_WithToken(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	token := getTestToken(t, ts, a)

	resp, err := httpPostWithAuth(ts.URL+"/parse", "2023-09-10 12:00:00", token)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "unix=1694347200") {
		t.Errorf("expected response to contain 'unix=1694347200', got: %s", string(body))
	}
}

func TestTimezones_WithToken(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	token := getTestToken(t, ts, a)

	resp, err := httpGetWithAuth(ts.URL+"/timezones", token)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "timezone=UTC") {
		t.Errorf("expected response to contain 'timezone=UTC', got: %s", string(body))
	}
}

func TestMCP_Initialize(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"initialize"}`
	resp, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["jsonrpc"] != "2.0" {
		t.Error("expected jsonrpc=2.0")
	}
	serverInfo, ok := result["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be an object")
	}
	info := serverInfo["serverInfo"].(map[string]interface{})
	if info["name"] != "timestampkit" {
		t.Errorf("expected name=timestampkit, got %v", info["name"])
	}
}

func TestMCP_ToolsList(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	reqBody := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	resp, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	res, ok := result["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be an object")
	}
	tools, ok := res["tools"].([]interface{})
	if !ok {
		t.Fatal("expected tools to be an array")
	}
	if len(tools) == 0 {
		t.Error("expected non-empty tools list")
	}
}

func TestMCP_Ping(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	reqBody := `{"jsonrpc":"2.0","id":3,"method":"ping"}`
	resp, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestMCP_UnknownMethod(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	reqBody := `{"jsonrpc":"2.0","id":4,"method":"unknown/method"}`
	resp, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["error"] == nil {
		t.Error("expected error for unknown method")
	}
}

func TestMCP_InvalidMethod(t *testing.T) {
	a := auth.New("test-secret")
	server := NewServer(a)
	ts := httptest.NewServer(server.Routes())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/mcp")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}
}

// Helper functions

func getTestToken(t *testing.T, ts *httptest.Server, a *auth.Auth) string {
	t.Helper()
	// RequestOTP returns the generated code, so we can use it directly
	code := a.RequestOTP("test@example.com")
	resp, err := http.Post(ts.URL+"/auth/verify", "text/plain", strings.NewReader("test@example.com="+code))
	if err != nil {
		t.Fatalf("failed to verify OTP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on verify, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	// Parse token from response: "token=xxx email=xxx usage=xxx"
	parts := strings.Fields(string(body))
	for _, p := range parts {
		if strings.HasPrefix(p, "token=") {
			return strings.TrimPrefix(p, "token=")
		}
	}

	t.Fatalf("could not extract token from response: %s", string(body))
	return ""
}

func httpGetWithAuth(url, token string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return http.DefaultClient.Do(req)
}

func httpPostWithAuth(url, body, token string) (*http.Response, error) {
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "text/plain")
	return http.DefaultClient.Do(req)
}
