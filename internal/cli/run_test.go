package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hrtsegv/http-client/internal/httpmethods"
	"github.com/hrtsegv/http-client/internal/output"
)

// run executes Run against a test server with output captured, returning
// the exit code plus whatever was written to each stream.
func run(t *testing.T, input httpmethods.Input) (code int, stdout, stderr string) {
	t.Helper()

	var outBuf, errBuf bytes.Buffer
	output.SetOutput(&outBuf, &errBuf)
	defer output.SetOutput(os.Stdout, os.Stderr)

	code = Run(input)

	return code, outBuf.String(), errBuf.String()
}

func TestRun_PrintsBodyToStdoutAndStatusToStderr(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hello":"world"}`))
	}))
	defer server.Close()

	code, stdout, stderr := run(t, httpmethods.Input{
		HTTPMethod: "GET",
		URL:        server.URL,
		Timeout:    5 * time.Second,
	})

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "hello") {
		t.Errorf("body missing from stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "200 OK") {
		t.Errorf("status line missing from stderr: %q", stderr)
	}
	if strings.Contains(stdout, "200 OK") {
		t.Errorf("status line leaked into stdout: %q", stdout)
	}
}

func TestRun_OutputFileStreamsBodyAndSkipsPrinting(t *testing.T) {
	const payload = "the quick brown fox"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "body.txt")

	code, stdout, stderr := run(t, httpmethods.Input{
		HTTPMethod: "GET",
		URL:        server.URL,
		Output:     path,
		Timeout:    5 * time.Second,
	})

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != payload {
		t.Errorf("file content = %q, want %q", got, payload)
	}

	if stdout != "" {
		t.Errorf("body printed as well as saved: %q", stdout)
	}
	if !strings.Contains(stderr, "Saved") {
		t.Errorf("missing save notice on stderr: %q", stderr)
	}
}

func TestRun_IncludePrintsHeadersBeforeBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Marker", "here")
		_, _ = w.Write([]byte("the body"))
	}))
	defer server.Close()

	_, stdout, _ := run(t, httpmethods.Input{
		HTTPMethod: "GET",
		URL:        server.URL,
		Include:    true,
		Timeout:    5 * time.Second,
	})

	iStatus := strings.Index(stdout, "200 OK")
	iHeader := strings.Index(stdout, "X-Marker")
	iBody := strings.Index(stdout, "the body")

	if iStatus < 0 || iHeader < 0 || iBody < 0 {
		t.Fatalf("missing status, header, or body in output: %q", stdout)
	}
	if iStatus >= iHeader || iHeader >= iBody {
		t.Errorf("status/headers/body out of order: %q", stdout)
	}
}

func TestRun_FailFlagExitCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	input := httpmethods.Input{HTTPMethod: "GET", URL: server.URL, Timeout: 5 * time.Second}

	if code, _, _ := run(t, input); code != 0 {
		t.Errorf("exit code without --fail = %d, want 0", code)
	}

	input.Fail = true
	if code, _, _ := run(t, input); code != 1 {
		t.Errorf("exit code with --fail = %d, want 1", code)
	}
}

func TestRun_RequestErrorExitsNonZero(t *testing.T) {
	code, _, _ := run(t, httpmethods.Input{
		HTTPMethod: "GET",
		URL:        "http://127.0.0.1:1", // nothing listening
		Timeout:    2 * time.Second,
	})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}
