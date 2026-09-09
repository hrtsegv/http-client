package output

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

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

// sniffLen is how much of the body is inspected to tell text from binary.
const sniffLen = 512

// PrintBody writes the response body to the output stream.
//
// When raw is set the body is copied through byte for byte. Otherwise a
// JSON body is pretty-printed, and a body that looks binary is replaced
// with a short notice on the diagnostic stream so it can't scramble the
// terminal.
func PrintBody(resp *http.Response, raw bool) error {
	if raw {
		_, err := io.Copy(out, resp.Body)
		return err
	}

	body := bufio.NewReaderSize(resp.Body, sniffLen)

	prefix, err := body.Peek(sniffLen)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		return err
	}

	if isBinary(prefix, resp.Header.Get("Content-Type")) {
		return printBinaryNotice(resp, body)
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	PrintColoredBody(data)

	return nil
}

// printBinaryNotice reports a body that shouldn't be written to a terminal,
// preferring the declared length over draining the body to measure it.
func printBinaryNotice(resp *http.Response, body io.Reader) error {
	size := resp.ContentLength
	if size < 0 {
		n, err := io.Copy(io.Discard, body)
		if err != nil {
			return err
		}
		size = n
	}

	description := humanBytes(size)
	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		description += ", " + contentType
	}

	fmt.Fprint(errOut, color.Yellow.Sprintf(
		"binary response body (%s) not shown — use -o FILE to save it, or --raw to print it\n",
		description,
	))

	return nil
}

// isBinary reports whether a body should be kept away from the terminal:
// it contains a NUL byte, or its media type isn't textual and the bytes
// aren't valid UTF-8.
func isBinary(prefix []byte, contentType string) bool {
	if len(prefix) == 0 {
		return false
	}

	if bytes.IndexByte(prefix, 0) >= 0 {
		return true
	}

	mediaType := contentType
	if parsed, _, err := mime.ParseMediaType(contentType); err == nil {
		mediaType = parsed
	}
	if isTextualMediaType(mediaType) {
		return false
	}

	return !utf8.Valid(trimPartialRune(prefix))
}

func isTextualMediaType(mediaType string) bool {
	switch {
	case strings.HasPrefix(mediaType, "text/"),
		strings.HasSuffix(mediaType, "/json"), strings.HasSuffix(mediaType, "+json"),
		strings.HasSuffix(mediaType, "/xml"), strings.HasSuffix(mediaType, "+xml"),
		mediaType == "application/javascript",
		mediaType == "application/x-www-form-urlencoded":
		return true
	default:
		return false
	}
}

// trimPartialRune drops a trailing multi-byte rune that the sniff window
// cut in half, so a truncated prefix isn't mistaken for invalid UTF-8.
// Bytes that are simply invalid are left alone — those really do mean the
// body isn't text.
func trimPartialRune(prefix []byte) []byte {
	// Walk back over the final rune's continuation bytes to its start.
	for i := len(prefix) - 1; i >= 0 && i > len(prefix)-utf8.UTFMax; i-- {
		if !utf8.RuneStart(prefix[i]) {
			continue
		}
		if utf8.FullRune(prefix[i:]) {
			return prefix
		}
		return prefix[:i]
	}

	return prefix
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
