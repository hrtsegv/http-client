// Package cli wires the parsed command-line input to an HTTP request and
// prints the result, returning a process exit code instead of terminating
// the program directly so it stays testable.
package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aladdin-io/http-client/internal/httpmethods"
	"github.com/aladdin-io/http-client/internal/output"
)

// Run executes the HTTP request described by input, prints the response,
// and returns the process exit code.
func Run(input httpmethods.Input) int {
	result, err := httpmethods.RunHttpMethod(input)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	defer result.Response.Body.Close()

	output.PrintStatusLine(result.Response, result.Elapsed)

	if strings.EqualFold(input.HTTPMethod, "head") {
		output.PrintColoredHeaders(result.Response.Header)
		return exitCode(input, result.Response.StatusCode)
	}

	body, err := io.ReadAll(result.Response.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	output.PrintColoredBody(body)

	if input.Output != "" {
		if err := output.WriteToFile(input.Output, body); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
	}

	return exitCode(input, result.Response.StatusCode)
}

// exitCode returns 1 when --fail is set and the response status is an
// error (>= 400), and 0 otherwise.
func exitCode(input httpmethods.Input, statusCode int) int {
	if input.Fail && statusCode >= 400 {
		return 1
	}
	return 0
}
