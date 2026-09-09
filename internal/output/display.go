package output

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gookit/color"
)

// Everything printed goes through these writers rather than os.Stdout and
// os.Stderr directly: stdout is buffered, since a large JSON body would
// otherwise cost a write syscall per key and per value, and tests can
// capture output by swapping the writers instead of the process globals.
var (
	stdout           = bufio.NewWriter(os.Stdout)
	out    io.Writer = stdout
	errOut io.Writer = os.Stderr
)

// SetOutput redirects the response output writer and the diagnostic
// writer. A nil writer leaves that stream unchanged.
func SetOutput(response, diagnostic io.Writer) {
	if response != nil {
		out = response
	}
	if diagnostic != nil {
		errOut = diagnostic
	}
}

// Flush writes out anything still buffered for stdout. Callers that print
// must defer it.
func Flush() error {
	return stdout.Flush()
}

// PrintStatusLine prints the response status and how long the request took,
// colored green/yellow/red for 2xx/3xx/4xx+ responses. It goes to the
// diagnostic stream so that stdout carries only the response body and
// stays pipeable into jq and friends.
func PrintStatusLine(resp *http.Response, elapsed time.Duration) {
	status := color.New(statusColor(resp.StatusCode)).Sprintf("%s %s", resp.Proto, resp.Status)
	fmt.Fprintf(errOut, "%s  (%s)\n", status, elapsed.Round(time.Millisecond))
}

func statusColor(statusCode int) color.Color {
	switch {
	case statusCode >= 400:
		return color.FgRed
	case statusCode >= 300:
		return color.FgYellow
	default:
		return color.FgGreen
	}
}

// PrintVerboseRequest prints the outgoing request line, headers, and body
// (if any) as diagnostics.
func PrintVerboseRequest(req *http.Request, body []byte) {
	fmt.Fprintf(errOut, "> %s %s\n", req.Method, req.URL.String())
	for key, values := range req.Header {
		for _, value := range values {
			fmt.Fprintf(errOut, "> %s: %s\n", key, value)
		}
	}
	if len(body) > 0 {
		fmt.Fprintf(errOut, ">\n%s\n", body)
	}
}

// PrintVerboseResponseHeaders prints the response headers as diagnostics.
func PrintVerboseResponseHeaders(resp *http.Response) {
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Fprintf(errOut, "< %s: %s\n", key, value)
		}
	}
}

// PrintColoredHeaders prints response headers as the command's output.
func PrintColoredHeaders(headers map[string][]string) {
	for key, values := range headers {
		for _, value := range values {
			fmt.Fprintf(out, "%s%s\n", color.Bold.Sprintf("%s: ", key), color.Cyan.Sprint(value))
		}
	}
}

// PrintColoredBody pretty-prints a JSON body, or prints it unchanged when
// it isn't JSON.
func PrintColoredBody(body []byte) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	// Without UseNumber every number decodes as a float64, which prints
	// large integers such as IDs as 1.2345678901234567e+19.
	decoder.UseNumber()

	var data interface{}
	// Anything trailing the first value means this isn't a JSON document,
	// so fall through and print it verbatim rather than truncating it.
	if err := decoder.Decode(&data); err != nil || decoder.More() {
		fmt.Fprintln(out, string(body))
		return
	}

	printColoredJSON(data, 0)
	fmt.Fprintln(out)
}

func printColoredJSON(data interface{}, indent int) {
	keyColor := color.New(color.FgLightGreen)
	valueColor := color.New(color.FgLightCyan)
	indentStr := strings.Repeat("  ", indent)

	switch v := data.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := v[key]
			fmt.Fprint(out, indentStr)
			fmt.Fprint(out, keyColor.Sprintf("%s: ", key))
			if m, ok := value.(map[string]interface{}); ok {
				fmt.Fprintln(out)
				printColoredJSON(m, indent+1)
			} else if s, ok := value.([]interface{}); ok {
				fmt.Fprintln(out)
				printColoredJSON(s, indent+1)
			} else {
				fmt.Fprintln(out, valueColor.Sprint(value))
			}
		}
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				printColoredJSON(m, indent)
				fmt.Fprintln(out)
			} else {
				fmt.Fprint(out, indentStr)
				fmt.Fprintln(out, valueColor.Sprint(item))
			}
		}
	default:
		fmt.Fprintln(out, valueColor.Sprint(v))
	}
}

// StreamToFile writes everything from r to fileName, replacing any
// existing contents, and reports how many bytes were written. The copy is
// chunked, so downloading a large file never requires holding it in
// memory.
func StreamToFile(fileName string, r io.Reader) (int64, error) {
	if fileName == "" {
		return 0, errors.New("missing file name")
	}

	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return 0, err
	}

	written, err := io.Copy(file, r)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return written, err
	}

	fmt.Fprint(errOut, color.Green.Sprintf("Saved %s to %s\n", humanBytes(written), fileName))

	return written, nil
}

// humanBytes formats a byte count for the save notice.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}

	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
