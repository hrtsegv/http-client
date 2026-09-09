package output

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gookit/color"
)

// capture collects what fn writes to the output and diagnostic writers,
// with color disabled so assertions see plain text.
func capture(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	origEnable := color.Disable()
	origOut, origErr := out, errOut
	defer func() {
		color.Enable = origEnable
		out, errOut = origOut, origErr
	}()

	var outBuf, errBuf bytes.Buffer
	SetOutput(&outBuf, &errBuf)

	fn()

	return outBuf.String(), errBuf.String()
}

func TestPrintColoredBody_SortedKeys(t *testing.T) {
	body := []byte(`{"zebra":1,"apple":2,"mango":3}`)

	out, _ := capture(t, func() {
		PrintColoredBody(body)
	})

	iApple := strings.Index(out, "apple")
	iMango := strings.Index(out, "mango")
	iZebra := strings.Index(out, "zebra")

	if iApple >= iMango || iMango >= iZebra {
		t.Errorf("keys not printed in sorted order, got: %s", out)
	}
}

func TestPrintColoredBody_PreservesLargeNumbers(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"int64 id", `{"id":12345678901234567890}`, "12345678901234567890"},
		{"high precision", `{"v":0.1234567890123456789}`, "0.1234567890123456789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _ := capture(t, func() {
				PrintColoredBody([]byte(tt.body))
			})

			if !strings.Contains(out, tt.want) {
				t.Errorf("got %q, want it to contain %q", strings.TrimSpace(out), tt.want)
			}
		})
	}
}

func TestPrintColoredBody_TrailingContentPrintsVerbatim(t *testing.T) {
	body := `{"a":1} not json anymore`

	out, _ := capture(t, func() {
		PrintColoredBody([]byte(body))
	})

	if strings.TrimSpace(out) != body {
		t.Errorf("got %q, want the body verbatim %q", strings.TrimSpace(out), body)
	}
}

func TestPrintColoredBody_NonJSONIsPassedThrough(t *testing.T) {
	out, _ := capture(t, func() {
		PrintColoredBody([]byte("plain text"))
	})

	if strings.TrimSpace(out) != "plain text" {
		t.Errorf("got %q, want %q", out, "plain text")
	}
}

func TestPrintStatusLine(t *testing.T) {
	resp := &http.Response{Proto: "HTTP/1.1", Status: "200 OK", StatusCode: 200}

	stdout, stderr := capture(t, func() {
		PrintStatusLine(resp, 42*time.Millisecond)
	})

	if !strings.Contains(stderr, "HTTP/1.1 200 OK") {
		t.Errorf("missing status line, got: %s", stderr)
	}
	if !strings.Contains(stderr, "42ms") {
		t.Errorf("missing elapsed time, got: %s", stderr)
	}
	// stdout must stay clean so the body can be piped into other tools.
	if stdout != "" {
		t.Errorf("status line leaked to stdout: %q", stdout)
	}
}

func TestStreamToFile_NoticeGoesToStderr(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")

	var writeErr error
	stdout, stderr := capture(t, func() { _, writeErr = StreamToFile(path, strings.NewReader("hello")) })
	if writeErr != nil {
		t.Fatalf("StreamToFile() error = %v", writeErr)
	}

	if !strings.Contains(stderr, "Saved 5 B to") {
		t.Errorf("missing save notice on stderr, got: %s", stderr)
	}
	if stdout != "" {
		t.Errorf("save notice leaked to stdout: %q", stdout)
	}
}

func TestPrintColoredHeaders_SortedNames(t *testing.T) {
	headers := map[string][]string{
		"Zebra":        {"z"},
		"Content-Type": {"application/json"},
		"Age":          {"1"},
	}

	out, _ := capture(t, func() {
		PrintColoredHeaders(headers)
	})

	iAge := strings.Index(out, "Age")
	iContentType := strings.Index(out, "Content-Type")
	iZebra := strings.Index(out, "Zebra")

	if iAge >= iContentType || iContentType >= iZebra {
		t.Errorf("headers not printed in sorted order, got: %s", out)
	}
}

// response builds a minimal *http.Response around a body for PrintBody.
func response(contentType, body string) *http.Response {
	header := http.Header{}
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}

	return &http.Response{
		Header:        header,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}

func TestPrintBody(t *testing.T) {
	png := "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR binary payload"

	tests := []struct {
		name          string
		contentType   string
		body          string
		raw           bool
		wantStdout    string
		wantStderrHas string
	}{
		{
			name:        "raw passes JSON through untouched",
			contentType: "application/json",
			body:        `{"b":2,"a":1}`,
			raw:         true,
			wantStdout:  `{"b":2,"a":1}`,
		},
		{
			name:        "raw prints binary without complaint",
			contentType: "image/png",
			body:        png,
			raw:         true,
			wantStdout:  png,
		},
		{
			name:          "binary body is replaced with a notice",
			contentType:   "image/png",
			body:          png,
			wantStdout:    "",
			wantStderrHas: "binary response body",
		},
		{
			name:        "utf-8 text is printed even with an unknown type",
			contentType: "application/vnd.acme.thing",
			body:        "héllo wörld",
			wantStdout:  "héllo wörld",
		},
		{
			name:        "declared text type is trusted",
			contentType: "text/csv",
			body:        "a,b\n1,2",
			wantStdout:  "a,b\n1,2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			stdout, stderr := capture(t, func() {
				err = PrintBody(response(tt.contentType, tt.body), tt.raw)
			})
			if err != nil {
				t.Fatalf("PrintBody() error = %v", err)
			}

			if strings.TrimRight(stdout, "\n") != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout, tt.wantStdout)
			}
			if tt.wantStderrHas != "" && !strings.Contains(stderr, tt.wantStderrHas) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, tt.wantStderrHas)
			}
		})
	}
}

func TestPrintBody_PrettyPrintsJSON(t *testing.T) {
	stdout, _ := capture(t, func() {
		if err := PrintBody(response("application/json", `{"b":2,"a":1}`), false); err != nil {
			t.Fatalf("PrintBody() error = %v", err)
		}
	})

	if !strings.Contains(stdout, "a: 1") || !strings.Contains(stdout, "b: 2") {
		t.Errorf("body not pretty-printed, got: %q", stdout)
	}
}

func TestIsBinary(t *testing.T) {
	tests := []struct {
		name        string
		prefix      string
		contentType string
		want        bool
	}{
		{"empty body", "", "", false},
		{"nul byte", "abc\x00def", "text/plain", true},
		{"plain text", "hello", "text/plain", false},
		{"json without a type", `{"a":1}`, "", false},
		{"invalid utf-8, unknown type", "\xff\xfe\xfa\xfb", "", true},
		{"gzip type", "\x1f\x8b\x08 compressed", "application/gzip", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBinary([]byte(tt.prefix), tt.contentType); got != tt.want {
				t.Errorf("isBinary(%q, %q) = %v, want %v", tt.prefix, tt.contentType, got, tt.want)
			}
		})
	}
}

func TestTrimPartialRune(t *testing.T) {
	// A 512-byte sniff window can cut a multi-byte rune in half; the
	// truncated tail must not make valid UTF-8 look binary.
	full := []byte("hello é")
	truncated := full[:len(full)-1]

	if utf8.Valid(truncated) {
		t.Fatal("test setup: expected the truncated slice to be invalid UTF-8")
	}
	if !utf8.Valid(trimPartialRune(truncated)) {
		t.Errorf("trimPartialRune(%q) still invalid UTF-8", truncated)
	}

	// Genuinely invalid bytes must survive the trim so they still read as
	// binary.
	invalid := []byte("\xff\xfe\xfa\xfb")
	if utf8.Valid(trimPartialRune(invalid)) {
		t.Errorf("trimPartialRune(%q) trimmed away invalid bytes", invalid)
	}
}

func TestStreamToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	var (
		written  int64
		writeErr error
	)
	capture(t, func() { written, writeErr = StreamToFile(path, strings.NewReader("hello")) })
	if writeErr != nil {
		t.Fatalf("StreamToFile() error = %v", writeErr)
	}
	if written != 5 {
		t.Errorf("written = %d, want 5", written)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("file content = %q, want %q", got, "hello")
	}

	// Writing again should overwrite, not append.
	capture(t, func() { _, writeErr = StreamToFile(path, strings.NewReader("hi")) })
	if writeErr != nil {
		t.Fatalf("StreamToFile() (overwrite) error = %v", writeErr)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "hi" {
		t.Errorf("file content after overwrite = %q, want %q", got, "hi")
	}
}

func TestStreamToFile_EmptyFileName(t *testing.T) {
	if _, err := StreamToFile("", strings.NewReader("hello")); err == nil {
		t.Fatal("StreamToFile() error = nil, want error for empty file name")
	}
}

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{512, "512 B"},
		{1024, "1.0 KB"},
		{10 * 1024 * 1024, "10.0 MB"},
	}

	for _, tt := range tests {
		if got := humanBytes(tt.in); got != tt.want {
			t.Errorf("humanBytes(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
