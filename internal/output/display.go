package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/gookit/color"
)

// PrintStatusLine prints the response status and how long the request took,
// colored green/yellow/red for 2xx/3xx/4xx+ responses.
func PrintStatusLine(resp *http.Response, elapsed time.Duration) {
	color.New(statusColor(resp.StatusCode)).Printf("%s %s", resp.Proto, resp.Status)
	fmt.Printf("  (%s)\n", elapsed.Round(time.Millisecond))
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
// (if any) to stderr.
func PrintVerboseRequest(req *http.Request, body []byte) {
	fmt.Fprintf(os.Stderr, "> %s %s\n", req.Method, req.URL.String())
	for key, values := range req.Header {
		for _, value := range values {
			fmt.Fprintf(os.Stderr, "> %s: %s\n", key, value)
		}
	}
	if len(body) > 0 {
		fmt.Fprintf(os.Stderr, ">\n%s\n", body)
	}
}

// PrintVerboseResponseHeaders prints the response headers to stderr.
func PrintVerboseResponseHeaders(resp *http.Response) {
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Fprintf(os.Stderr, "< %s: %s\n", key, value)
		}
	}
}

func PrintColoredHeaders(headers map[string][]string) {
	for key, values := range headers {
		for _, value := range values {
			color.Bold.Printf("%s: ", key)
			color.Cyan.Printf("%s\n", value)
		}
	}
}

func PrintColoredBody(body []byte) {
	var data interface{}
	err := json.Unmarshal(body, &data)
	if err == nil {
		printColoredJSON(data, 0)
		fmt.Println()
		return
	}

	// If parsing fails, print the raw body as plain text
	fmt.Println(string(body))
}

func printColoredJSON(data interface{}, indent int) {
	keyColor := color.New(color.FgLightGreen)
	valueColor := color.New(color.FgLightCyan)
	indentStr := ""
	for i := 0; i < indent; i++ {
		indentStr += "  "
	}

	switch v := data.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := v[key]
			fmt.Print(indentStr)
			keyColor.Printf("%s: ", key)
			if m, ok := value.(map[string]interface{}); ok {
				fmt.Println()
				printColoredJSON(m, indent+1)
			} else if s, ok := value.([]interface{}); ok {
				fmt.Println()
				printColoredJSON(s, indent+1)
			} else {
				valueColor.Println(value)
			}
		}
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				printColoredJSON(m, indent)
				fmt.Println()
			} else {
				fmt.Print(indentStr)
				valueColor.Println(item)
			}
		}
	default:
		valueColor.Println(v)
	}
}

func WriteToFile(fileName string, data []byte) error {
	if fileName == "" {
		return errors.New("missing file name")
	}

	if err := os.WriteFile(fileName, data, 0o644); err != nil {
		return err
	}

	color.Greenp("Saved to ", fileName, "\n")

	return nil
}
