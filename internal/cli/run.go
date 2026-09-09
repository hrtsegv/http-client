// Package cli wires the parsed command-line input to an HTTP request and
// prints the result, returning a process exit code instead of terminating
// the program directly so it stays testable.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/gookit/color"

	"github.com/hrtsegv/http-client/internal/httpmethods"
	"github.com/hrtsegv/http-client/internal/output"
)

// exitCanceled is the conventional shell exit code for a command stopped
// by Ctrl+C (128 + SIGINT).
const exitCanceled = 130

// Run executes the HTTP request described by input, prints the response,
// and returns the process exit code.
func Run(input httpmethods.Input) int {
	// Formatting is for humans: when stdout is redirected or piped the body
	// is passed through byte for byte, so `... | jq` sees real JSON.
	isTerminal := color.IsTerminal(os.Stdout.Fd())
	raw := input.Raw || !isTerminal

	if input.NoColor || !isTerminal {
		color.Disable()
	}
	defer func() {
		if err := output.Flush(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
	}()

	// Ctrl+C cancels the request in flight rather than leaving the process
	// blocked until the timeout expires.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	result, err := httpmethods.RunHttpMethod(ctx, input)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, "canceled")
			return exitCanceled
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	defer func() { _ = result.Response.Body.Close() }()

	output.PrintStatusLine(result.Response, result.Elapsed)

	if strings.EqualFold(input.HTTPMethod, "head") {
		output.PrintColoredHeaders(result.Response.Header)
		return exitCode(input, result.Response.StatusCode)
	}

	if input.Include {
		output.PrintResponseHead(result.Response)
	}

	// With -o the body is streamed straight to disk and not printed, so a
	// large download costs a fixed-size buffer rather than its full size
	// in memory.
	if input.Output != "" {
		if _, err := output.StreamToFile(input.Output, result.Response.Body); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		return exitCode(input, result.Response.StatusCode)
	}

	if err := output.PrintBody(result.Response, raw); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
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
