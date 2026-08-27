package httpmethods

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExec_MethodHeaderBodyPassthrough(t *testing.T) {
	var gotMethod, gotBody, gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotHeader = r.Header.Get("X-Custom")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	input := Input{
		HTTPMethod: "post",
		URL:        server.URL,
		Body:       `{"key":"value"}`,
		Header:     []string{"X-Custom: hello"},
		Timeout:    5 * time.Second,
	}

	result, err := RunHttpMethod(input)
	if err != nil {
		t.Fatalf("RunHttpMethod() error = %v", err)
	}
	defer func() { _ = result.Response.Body.Close() }()

	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotBody != `{"key":"value"}` {
		t.Errorf("body = %q, want %q", gotBody, `{"key":"value"}`)
	}
	if gotHeader != "hello" {
		t.Errorf("X-Custom header = %q, want hello", gotHeader)
	}
	if result.Response.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", result.Response.StatusCode, http.StatusCreated)
	}
}

func TestExec_DefaultContentTypeAndUserAgent(t *testing.T) {
	var gotContentType, gotUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotUserAgent = r.Header.Get("User-Agent")
	}))
	defer server.Close()

	input := Input{
		HTTPMethod: "POST",
		URL:        server.URL,
		Body:       `{"key":"value"}`,
		Timeout:    5 * time.Second,
	}

	result, err := RunHttpMethod(input)
	if err != nil {
		t.Fatalf("RunHttpMethod() error = %v", err)
	}
	defer func() { _ = result.Response.Body.Close() }()

	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if !strings.HasPrefix(gotUserAgent, "http-client/") {
		t.Errorf("User-Agent = %q, want prefix http-client/", gotUserAgent)
	}
}

func TestExec_NoBodyOmitsBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength != 0 {
			t.Errorf("ContentLength = %d, want 0", r.ContentLength)
		}
	}))
	defer server.Close()

	input := Input{HTTPMethod: "GET", URL: server.URL, Timeout: 5 * time.Second}
	result, err := RunHttpMethod(input)
	if err != nil {
		t.Fatalf("RunHttpMethod() error = %v", err)
	}
	_ = result.Response.Body.Close()
}

func TestExec_Auth(t *testing.T) {
	var gotUser, gotPass string
	var gotOK bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, gotOK = r.BasicAuth()
	}))
	defer server.Close()

	input := Input{HTTPMethod: "GET", URL: server.URL, Auth: "alice:secret", Timeout: 5 * time.Second}
	result, err := RunHttpMethod(input)
	if err != nil {
		t.Fatalf("RunHttpMethod() error = %v", err)
	}
	_ = result.Response.Body.Close()

	if !gotOK || gotUser != "alice" || gotPass != "secret" {
		t.Errorf("BasicAuth() = (%q, %q, %v), want (alice, secret, true)", gotUser, gotPass, gotOK)
	}
}

func TestExec_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	input := Input{HTTPMethod: "GET", URL: server.URL, Timeout: 10 * time.Millisecond}
	_, err := RunHttpMethod(input)
	if err == nil {
		t.Fatal("RunHttpMethod() error = nil, want a timeout error")
	}
}

func TestExec_NoRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()

	input := Input{HTTPMethod: "GET", URL: redirector.URL, NoRedirect: true, Timeout: 5 * time.Second}
	result, err := RunHttpMethod(input)
	if err != nil {
		t.Fatalf("RunHttpMethod() error = %v", err)
	}
	defer func() { _ = result.Response.Body.Close() }()

	if result.Response.StatusCode != http.StatusFound {
		t.Errorf("status = %d, want %d (redirect not followed)", result.Response.StatusCode, http.StatusFound)
	}
}

func TestExec_Insecure(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Without -k, the self-signed cert should be rejected.
	input := Input{HTTPMethod: "GET", URL: server.URL, Timeout: 5 * time.Second}
	if _, err := RunHttpMethod(input); err == nil {
		t.Fatal("RunHttpMethod() error = nil, want a TLS verification error without --insecure")
	}

	input.Insecure = true
	result, err := RunHttpMethod(input)
	if err != nil {
		t.Fatalf("RunHttpMethod() with --insecure error = %v", err)
	}
	defer func() { _ = result.Response.Body.Close() }()

	if result.Response.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", result.Response.StatusCode, http.StatusOK)
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"missing scheme defaults to https", "example.com/path", "https://example.com/path"},
		{"http scheme preserved", "http://example.com", "http://example.com"},
		{"https scheme preserved", "https://example.com", "https://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeURL(tt.in); got != tt.want {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolveBody_MutuallyExclusive(t *testing.T) {
	input := Input{HTTPMethod: "POST", URL: "http://example.com", Body: "raw", Data: []string{"key=value"}}
	if _, err := resolveBody(input); err == nil {
		t.Fatal("resolveBody() error = nil, want error for mutually exclusive --body/--data")
	}
}

func TestNewHTTPClient_Insecure(t *testing.T) {
	client := newHTTPClient(Input{Insecure: true})
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Transport = %T, want *http.Transport", client.Transport)
	}
	if transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify not set on TLS config")
	}
}
