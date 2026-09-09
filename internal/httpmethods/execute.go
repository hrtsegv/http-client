package httpmethods

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hrtsegv/http-client/internal/output"
	"github.com/hrtsegv/http-client/internal/utilities"
)

// Result holds an HTTP response together with how long the request took.
type Result struct {
	Response *http.Response
	Elapsed  time.Duration
}

func exec(ctx context.Context, input Input) (*Result, error) {
	body, err := resolveBody(input)
	if err != nil {
		return nil, err
	}
	reqBody := body.data

	var bodyReader io.Reader = http.NoBody
	switch {
	case body.reader != nil:
		bodyReader = body.reader
	case len(reqBody) > 0:
		bodyReader = bytes.NewReader(reqBody)
	}

	requestURL, err := buildURL(input)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(input.HTTPMethod), requestURL, bodyReader)
	if err != nil {
		return nil, err
	}

	headers, err := utilities.ParseHeaders(input.Header)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	if body.contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", body.contentType)
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "http-client/"+Version)
	}

	if input.Auth != "" && req.Header.Get("Authorization") == "" {
		user, pass, ok := strings.Cut(input.Auth, ":")
		if !ok {
			return nil, fmt.Errorf("invalid --auth value: expected user:pass")
		}
		req.SetBasicAuth(user, pass)
	}

	if input.Verbose {
		output.PrintVerboseRequest(req, reqBody)
	}

	client, err := newHTTPClient(input)
	if err != nil {
		return nil, err
	}

	result, err := send(ctx, client, req, input)
	if err != nil {
		return nil, err
	}

	if input.Verbose {
		output.PrintVerboseResponseHeaders(result.Response)
	}

	return result, nil
}

// send performs the request, retrying transient failures when --retry asks
// for it, and reports how long the successful attempt took.
func send(ctx context.Context, client *http.Client, req *http.Request, input Input) (*Result, error) {
	attempts := input.Retry + 1

	// A streamed body (a multipart upload) is consumed by the first
	// attempt and http.NewRequest can't rewind it, so it can't be retried.
	// http.NoBody is non-nil but carries nothing, so it replays fine.
	if attempts > 1 && req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
		output.PrintNotice("request body is streamed and cannot be replayed, so --retry is ignored")
		attempts = 1
	}

	for attempt := 1; ; attempt++ {
		start := time.Now()
		resp, err := client.Do(req)
		elapsed := time.Since(start)

		done := attempt >= attempts || (err == nil && !isTransient(resp.StatusCode))
		if done {
			if err != nil {
				return nil, err
			}
			return &Result{Response: resp, Elapsed: elapsed}, nil
		}

		delay := retryDelay(input.RetryDelay, attempt, resp)
		reason := describeAttempt(resp, err)
		output.PrintNotice("%s, retrying in %s (attempt %d of %d)", reason, delay.Round(time.Millisecond), attempt+1, attempts)

		if resp != nil {
			// Drain before discarding so the connection can be reused.
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}

		if err := waitFor(ctx, delay); err != nil {
			return nil, err
		}

		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req.Body = body
		}
	}
}

// isTransient reports whether a status is worth retrying: rate limiting or
// a server-side failure.
func isTransient(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError
}

func describeAttempt(resp *http.Response, err error) string {
	if err != nil {
		return err.Error()
	}
	return "got " + resp.Status
}

// retryDelay prefers the server's Retry-After and otherwise backs off
// exponentially from the base delay, capping the growth so a large --retry
// can't schedule an absurd wait.
func retryDelay(base time.Duration, attempt int, resp *http.Response) time.Duration {
	if resp != nil {
		if after, ok := retryAfter(resp.Header.Get("Retry-After")); ok {
			return after
		}
	}

	return base << min(attempt-1, 6)
}

// retryAfter parses a Retry-After header, which is either a number of
// seconds or an HTTP date.
func retryAfter(value string) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0, false
		}
		return time.Duration(seconds) * time.Second, true
	}

	if date, err := http.ParseTime(value); err == nil {
		if delay := time.Until(date); delay > 0 {
			return delay, true
		}
		// A date already in the past means retry now.
		return 0, true
	}

	return 0, false
}

// waitFor sleeps for delay, or returns early if the context is canceled.
func waitFor(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// requestBody is the body to send plus the Content-Type it implies, which
// is applied only when the caller didn't set one with -H. A streamed body
// sets reader instead of data and is never held in memory.
type requestBody struct {
	data        []byte
	reader      io.Reader
	contentType string
}

// resolveBody picks the request body from whichever body flag was given.
// They are mutually exclusive, since each implies a different encoding.
func resolveBody(input Input) (requestBody, error) {
	given := make([]string, 0, 3)
	if input.Body != "" {
		given = append(given, "--body")
	}
	if len(input.Data) > 0 {
		given = append(given, "--data")
	}
	if len(input.Form) > 0 {
		given = append(given, "--form")
	}
	if len(input.Multipart) > 0 {
		given = append(given, "--multipart")
	}
	if len(given) > 1 {
		return requestBody{}, fmt.Errorf("use only one of %s", strings.Join(given, ", "))
	}

	switch {
	case len(input.Multipart) > 0:
		reader, contentType, err := utilities.MultipartBody(input.Multipart)
		if err != nil {
			return requestBody{}, err
		}
		return requestBody{reader: reader, contentType: contentType}, nil

	case len(input.Data) > 0:
		data, err := utilities.ParseDataFields(input.Data)
		if err != nil {
			return requestBody{}, err
		}
		return requestBody{data: data, contentType: "application/json"}, nil

	case len(input.Form) > 0:
		data, err := utilities.ParseFormFields(input.Form)
		if err != nil {
			return requestBody{}, err
		}
		return requestBody{data: data, contentType: "application/x-www-form-urlencoded"}, nil

	default:
		data, err := utilities.ParseBody(input.Body)
		if err != nil {
			return requestBody{}, err
		}
		// A raw body only implies JSON when it actually parses as JSON.
		contentType := ""
		if len(data) > 0 && json.Valid(data) {
			contentType = "application/json"
		}
		return requestBody{data: data, contentType: contentType}, nil
	}
}

// buildURL normalizes the URL and merges any --query parameters into it,
// encoding them so values with spaces or '&' survive the trip. Repeating a
// key adds another value rather than replacing it.
func buildURL(input Input) (string, error) {
	parsed, err := url.Parse(normalizeURL(input.URL))
	if err != nil {
		return "", err
	}

	if len(input.Query) == 0 {
		return parsed.String(), nil
	}

	pairs, err := utilities.ParseKeyValuePairs(input.Query, "--query")
	if err != nil {
		return "", err
	}

	values := parsed.Query()
	for _, pair := range pairs {
		values.Add(pair.Key, pair.Value)
	}
	parsed.RawQuery = values.Encode()

	return parsed.String(), nil
}

// normalizeURL defaults to https:// when the given URL has no scheme,
// since http.NewRequest otherwise fails with a confusing "unsupported
// protocol scheme" error.
func normalizeURL(rawURL string) string {
	if !strings.Contains(rawURL, "://") {
		return "https://" + rawURL
	}
	return rawURL
}
