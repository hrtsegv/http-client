package httpmethods

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func exec(input Input) (*Result, error) {
	body, err := resolveBody(input)
	if err != nil {
		return nil, err
	}
	reqBody := body.data

	var bodyReader io.Reader = http.NoBody
	if len(reqBody) > 0 {
		bodyReader = bytes.NewReader(reqBody)
	}

	requestURL, err := buildURL(input)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(strings.ToUpper(input.HTTPMethod), requestURL, bodyReader)
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

	client := newHTTPClient(input)

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return nil, err
	}

	if input.Verbose {
		output.PrintVerboseResponseHeaders(resp)
	}

	return &Result{Response: resp, Elapsed: elapsed}, nil
}

// requestBody is the body to send plus the Content-Type it implies, which
// is applied only when the caller didn't set one with -H.
type requestBody struct {
	data        []byte
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
	if len(given) > 1 {
		return requestBody{}, fmt.Errorf("use only one of %s", strings.Join(given, ", "))
	}

	switch {
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
