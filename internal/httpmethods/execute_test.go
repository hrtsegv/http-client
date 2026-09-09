package httpmethods

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
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

	result, err := RunHttpMethod(t.Context(), input)
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

	result, err := RunHttpMethod(t.Context(), input)
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
	result, err := RunHttpMethod(t.Context(), input)
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
	result, err := RunHttpMethod(t.Context(), input)
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
	_, err := RunHttpMethod(t.Context(), input)
	if err == nil {
		t.Fatal("RunHttpMethod() error = nil, want a timeout error")
	}
}

func TestExec_ContextCancellationAbortsRequest(t *testing.T) {
	released := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-released
	}))
	defer server.Close()
	defer close(released)

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// A long timeout would keep this blocked; cancellation must win.
	input := Input{HTTPMethod: "GET", URL: server.URL, Timeout: time.Minute}

	_, err := RunHttpMethod(ctx, input)
	if err == nil {
		t.Fatal("RunHttpMethod() error = nil, want a cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want it to wrap context.Canceled", err)
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
	result, err := RunHttpMethod(t.Context(), input)
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
	if _, err := RunHttpMethod(t.Context(), input); err == nil {
		t.Fatal("RunHttpMethod() error = nil, want a TLS verification error without --insecure")
	}

	input.Insecure = true
	result, err := RunHttpMethod(t.Context(), input)
	if err != nil {
		t.Fatalf("RunHttpMethod() with --insecure error = %v", err)
	}
	defer func() { _ = result.Response.Body.Close() }()

	if result.Response.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", result.Response.StatusCode, http.StatusOK)
	}
}

func TestExec_QueryParameters(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
	}))
	defer server.Close()

	input := Input{
		HTTPMethod: "GET",
		URL:        server.URL + "/search?existing=kept",
		Query:      []string{"q=hello world", "tag=a", "tag=b"},
		Timeout:    5 * time.Second,
	}

	result, err := RunHttpMethod(t.Context(), input)
	if err != nil {
		t.Fatalf("RunHttpMethod() error = %v", err)
	}
	_ = result.Response.Body.Close()

	if got := gotQuery.Get("q"); got != "hello world" {
		t.Errorf("q = %q, want %q", got, "hello world")
	}
	if got := gotQuery["tag"]; len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("tag = %v, want [a b]", got)
	}
	if got := gotQuery.Get("existing"); got != "kept" {
		t.Errorf("existing = %q, want kept (query already in the URL must survive)", got)
	}
}

func TestBuildURL_InvalidQueryField(t *testing.T) {
	_, err := buildURL(Input{URL: "http://example.com", Query: []string{"missing-equals"}})
	if err == nil {
		t.Fatal("buildURL() error = nil, want error for a malformed --query value")
	}
}

func TestSend_RetriesTransientFailures(t *testing.T) {
	tests := []struct {
		name         string
		failures     int
		failStatus   int
		retry        int
		wantAttempts int
		wantStatus   int
	}{
		{"recovers after two 503s", 2, http.StatusServiceUnavailable, 3, 3, http.StatusOK},
		{"retries rate limiting", 1, http.StatusTooManyRequests, 2, 2, http.StatusOK},
		{"no retry flag means one attempt", 1, http.StatusServiceUnavailable, 0, 1, http.StatusServiceUnavailable},
		{"gives up and returns the last response", 5, http.StatusBadGateway, 2, 3, http.StatusBadGateway},
		{"does not retry a 404", 1, http.StatusNotFound, 3, 1, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attempts int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts++
				if attempts <= tt.failures {
					w.WriteHeader(tt.failStatus)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			input := Input{
				HTTPMethod: "GET",
				URL:        server.URL,
				Retry:      tt.retry,
				RetryDelay: time.Millisecond,
				Timeout:    5 * time.Second,
			}

			result, err := RunHttpMethod(t.Context(), input)
			if err != nil {
				t.Fatalf("RunHttpMethod() error = %v", err)
			}
			_ = result.Response.Body.Close()

			if attempts != tt.wantAttempts {
				t.Errorf("attempts = %d, want %d", attempts, tt.wantAttempts)
			}
			if result.Response.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", result.Response.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestSend_RetriesReplayTheBody(t *testing.T) {
	var bodies []string
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		received, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(received))
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	}))
	defer server.Close()

	input := Input{
		HTTPMethod: "POST",
		URL:        server.URL,
		Body:       `{"key":"value"}`,
		Retry:      1,
		RetryDelay: time.Millisecond,
		Timeout:    5 * time.Second,
	}

	result, err := RunHttpMethod(t.Context(), input)
	if err != nil {
		t.Fatalf("RunHttpMethod() error = %v", err)
	}
	_ = result.Response.Body.Close()

	if len(bodies) != 2 {
		t.Fatalf("attempts = %d, want 2", len(bodies))
	}
	if bodies[0] != bodies[1] {
		t.Errorf("retry sent a different body: %q then %q", bodies[0], bodies[1])
	}
}

func TestSend_StreamedBodyIsNotRetried(t *testing.T) {
	path := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(path, []byte("payload"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	input := Input{
		HTTPMethod: "POST",
		URL:        server.URL,
		Multipart:  []string{"upload=@" + path},
		Retry:      3,
		RetryDelay: time.Millisecond,
		Timeout:    5 * time.Second,
	}

	result, err := RunHttpMethod(t.Context(), input)
	if err != nil {
		t.Fatalf("RunHttpMethod() error = %v", err)
	}
	_ = result.Response.Body.Close()

	if attempts != 1 {
		t.Errorf("attempts = %d, want 1: a streamed body cannot be replayed", attempts)
	}
}

func TestRetryAfter(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  time.Duration
		ok    bool
	}{
		{"seconds", "5", 5 * time.Second, true},
		{"padded seconds", " 2 ", 2 * time.Second, true},
		{"negative", "-1", 0, false},
		{"empty", "", 0, false},
		{"garbage", "soon", 0, false},
		{"past date", "Mon, 02 Jan 2006 15:04:05 GMT", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := retryAfter(tt.value)
			if ok != tt.ok {
				t.Fatalf("retryAfter(%q) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("retryAfter(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestRetryDelay(t *testing.T) {
	base := 100 * time.Millisecond

	if got := retryDelay(base, 1, nil); got != base {
		t.Errorf("first retry delay = %v, want %v", got, base)
	}
	if got := retryDelay(base, 3, nil); got != 4*base {
		t.Errorf("third retry delay = %v, want %v", got, 4*base)
	}
	// A huge attempt count must not shift the delay into nonsense.
	if got := retryDelay(base, 40, nil); got != 64*base {
		t.Errorf("capped delay = %v, want %v", got, 64*base)
	}

	resp := &http.Response{Header: http.Header{"Retry-After": {"3"}}}
	if got := retryDelay(base, 5, resp); got != 3*time.Second {
		t.Errorf("delay with Retry-After = %v, want 3s", got)
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
	tests := []struct {
		name  string
		input Input
	}{
		{"body and data", Input{Body: "raw", Data: []string{"key=value"}}},
		{"body and form", Input{Body: "raw", Form: []string{"key=value"}}},
		{"data and form", Input{Data: []string{"a=1"}, Form: []string{"b=2"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := resolveBody(tt.input); err == nil {
				t.Error("resolveBody() error = nil, want error for conflicting body flags")
			}
		})
	}
}

func TestExec_FormBody(t *testing.T) {
	var (
		gotContentType string
		gotForm        url.Values
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm() error = %v", err)
		}
		gotForm = r.PostForm
	}))
	defer server.Close()

	input := Input{
		HTTPMethod: "POST",
		URL:        server.URL,
		Form:       []string{"name=ada lovelace", "lang=go", "lang=rust"},
		Timeout:    5 * time.Second,
	}

	result, err := RunHttpMethod(t.Context(), input)
	if err != nil {
		t.Fatalf("RunHttpMethod() error = %v", err)
	}
	_ = result.Response.Body.Close()

	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", gotContentType)
	}
	if got := gotForm.Get("name"); got != "ada lovelace" {
		t.Errorf("name = %q, want %q", got, "ada lovelace")
	}
	if got := gotForm["lang"]; len(got) != 2 {
		t.Errorf("lang = %v, want two values", got)
	}
}

func TestExec_MultipartUpload(t *testing.T) {
	const fileContent = "col1,col2\n1,2\n"

	dir := t.TempDir()
	path := filepath.Join(dir, "data.csv")
	if err := os.WriteFile(path, []byte(fileContent), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var (
		gotField    string
		gotFileName string
		gotFileBody string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("ParseMultipartForm() error = %v", err)
			return
		}
		gotField = r.FormValue("description")

		file, header, err := r.FormFile("upload")
		if err != nil {
			t.Errorf("FormFile() error = %v", err)
			return
		}
		defer func() { _ = file.Close() }()

		gotFileName = header.Filename
		content, err := io.ReadAll(file)
		if err != nil {
			t.Errorf("reading uploaded file: %v", err)
			return
		}
		gotFileBody = string(content)
	}))
	defer server.Close()

	input := Input{
		HTTPMethod: "POST",
		URL:        server.URL,
		Multipart:  []string{"description=monthly export", "upload=@" + path},
		Timeout:    5 * time.Second,
	}

	result, err := RunHttpMethod(t.Context(), input)
	if err != nil {
		t.Fatalf("RunHttpMethod() error = %v", err)
	}
	_ = result.Response.Body.Close()

	if gotField != "monthly export" {
		t.Errorf("description = %q, want %q", gotField, "monthly export")
	}
	if gotFileName != "data.csv" {
		t.Errorf("uploaded file name = %q, want data.csv", gotFileName)
	}
	if gotFileBody != fileContent {
		t.Errorf("uploaded content = %q, want %q", gotFileBody, fileContent)
	}
}

func TestExec_MultipartMissingFileFailsBeforeSending(t *testing.T) {
	var reached bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	}))
	defer server.Close()

	input := Input{
		HTTPMethod: "POST",
		URL:        server.URL,
		Multipart:  []string{"upload=@" + filepath.Join(t.TempDir(), "does-not-exist")},
		Timeout:    5 * time.Second,
	}

	if _, err := RunHttpMethod(t.Context(), input); err == nil {
		t.Fatal("RunHttpMethod() error = nil, want error for a missing upload file")
	}
	if reached {
		t.Error("request was sent despite the missing file")
	}
}

func TestExec_PlainTextBodyGetsNoJSONContentType(t *testing.T) {
	var gotContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
	}))
	defer server.Close()

	input := Input{HTTPMethod: "POST", URL: server.URL, Body: "just text", Timeout: 5 * time.Second}

	result, err := RunHttpMethod(t.Context(), input)
	if err != nil {
		t.Fatalf("RunHttpMethod() error = %v", err)
	}
	_ = result.Response.Body.Close()

	if gotContentType != "" {
		t.Errorf("Content-Type = %q, want it unset for a non-JSON body", gotContentType)
	}
}

func TestNewHTTPClient_Insecure(t *testing.T) {
	client, err := newHTTPClient(Input{Insecure: true})
	if err != nil {
		t.Fatalf("newHTTPClient() error = %v", err)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Transport = %T, want *http.Transport", client.Transport)
	}
	if transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify not set on TLS config")
	}
	// Customizing TLS must not cost the environment proxy or HTTP/2.
	if transport.Proxy == nil {
		t.Error("environment proxy support dropped from the custom transport")
	}
	if !transport.ForceAttemptHTTP2 {
		t.Error("HTTP/2 disabled by the custom transport")
	}
}

func TestNewHTTPClient_Proxy(t *testing.T) {
	if _, err := newHTTPClient(Input{Proxy: "not-a-proxy"}); err == nil {
		t.Error("newHTTPClient() error = nil, want error for a proxy without a scheme or host")
	}

	client, err := newHTTPClient(Input{Proxy: "http://127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("newHTTPClient() error = %v", err)
	}

	transport := client.Transport.(*http.Transport)
	proxyURL, err := transport.Proxy(httptest.NewRequest(http.MethodGet, "http://example.com", nil))
	if err != nil {
		t.Fatalf("Proxy() error = %v", err)
	}
	if proxyURL == nil || proxyURL.Host != "127.0.0.1:8080" {
		t.Errorf("proxy = %v, want http://127.0.0.1:8080", proxyURL)
	}
}

func TestExec_ProxyReceivesRequest(t *testing.T) {
	var proxiedHost string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxiedHost = r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer proxy.Close()

	input := Input{
		HTTPMethod: "GET",
		URL:        "http://example.com/resource",
		Proxy:      proxy.URL,
		Timeout:    5 * time.Second,
	}

	result, err := RunHttpMethod(t.Context(), input)
	if err != nil {
		t.Fatalf("RunHttpMethod() error = %v", err)
	}
	_ = result.Response.Body.Close()

	if proxiedHost != "example.com" {
		t.Errorf("proxied Host = %q, want example.com", proxiedHost)
	}
}
