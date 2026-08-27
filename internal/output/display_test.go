package output

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gookit/color"
)

// captureStdout redirects os.Stdout for the duration of fn and returns
// everything written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := color.Disable()
	defer func() {
		color.Enable = orig
	}()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	color.SetOutput(w)
	defer func() {
		os.Stdout = origStdout
		color.SetOutput(origStdout)
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("w.Close() error = %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	return string(out)
}

func TestPrintColoredBody_SortedKeys(t *testing.T) {
	body := []byte(`{"zebra":1,"apple":2,"mango":3}`)

	out := captureStdout(t, func() {
		PrintColoredBody(body)
	})

	iApple := strings.Index(out, "apple")
	iMango := strings.Index(out, "mango")
	iZebra := strings.Index(out, "zebra")

	if iApple >= iMango || iMango >= iZebra {
		t.Errorf("keys not printed in sorted order, got: %s", out)
	}
}

func TestPrintColoredBody_NonJSONIsPassedThrough(t *testing.T) {
	out := captureStdout(t, func() {
		PrintColoredBody([]byte("plain text"))
	})

	if strings.TrimSpace(out) != "plain text" {
		t.Errorf("got %q, want %q", out, "plain text")
	}
}

func TestPrintStatusLine(t *testing.T) {
	resp := &http.Response{Proto: "HTTP/1.1", Status: "200 OK", StatusCode: 200}

	out := captureStdout(t, func() {
		PrintStatusLine(resp, 42*time.Millisecond)
	})

	if !strings.Contains(out, "HTTP/1.1 200 OK") {
		t.Errorf("missing status line, got: %s", out)
	}
	if !strings.Contains(out, "42ms") {
		t.Errorf("missing elapsed time, got: %s", out)
	}
}

func TestWriteToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	if err := WriteToFile(path, []byte("hello")); err != nil {
		t.Fatalf("WriteToFile() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("file content = %q, want %q", got, "hello")
	}

	// Writing again should overwrite, not append.
	if err := WriteToFile(path, []byte("world")); err != nil {
		t.Fatalf("WriteToFile() (overwrite) error = %v", err)
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
