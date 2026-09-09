package output

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestWriteToFile_NoticeGoesToStderr(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")

	var writeErr error
	stdout, stderr := capture(t, func() { writeErr = WriteToFile(path, []byte("hello")) })
	if writeErr != nil {
		t.Fatalf("WriteToFile() error = %v", writeErr)
	}

	if !strings.Contains(stderr, "Saved to") {
		t.Errorf("missing save notice on stderr, got: %s", stderr)
	}
	if stdout != "" {
		t.Errorf("save notice leaked to stdout: %q", stdout)
	}
}

func TestWriteToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	var writeErr error
	capture(t, func() { writeErr = WriteToFile(path, []byte("hello")) })
	if writeErr != nil {
		t.Fatalf("WriteToFile() error = %v", writeErr)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("file content = %q, want %q", got, "hello")
	}

	// Writing again should overwrite, not append.
	capture(t, func() { writeErr = WriteToFile(path, []byte("world")) })
	if writeErr != nil {
		t.Fatalf("WriteToFile() (overwrite) error = %v", writeErr)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "world" {
		t.Errorf("file content after overwrite = %q, want %q", got, "world")
	}
}

func TestWriteToFile_EmptyFileName(t *testing.T) {
	if err := WriteToFile("", []byte("hello")); err == nil {
		t.Fatal("WriteToFile() error = nil, want error for empty file name")
	}
}
