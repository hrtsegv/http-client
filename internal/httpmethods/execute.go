package httpmethods

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aladdin-io/http-client/internal/utilities"
)

// Result holds an HTTP response together with how long the request took.
type Result struct {
	Response *http.Response
	Elapsed  time.Duration
}

func exec(input Input) (*Result, error) {
	reqBody, err := resolveBody(input)
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader = http.NoBody
	if len(reqBody) > 0 {
		bodyReader = bytes.NewReader(reqBody)
	}

	req, err := http.NewRequest(strings.ToUpper(input.HTTPMethod), normalizeURL(input.URL), bodyReader)
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

	if len(reqBody) > 0 && req.Header.Get("Content-Type") == "" && json.Valid(reqBody) {
		req.Header.Set("Content-Type", "application/json")
	}

	client := newHTTPClient(input)

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return nil, err
	}

	return &Result{Response: resp, Elapsed: elapsed}, nil
}

// resolveBody picks the request body from either --body or --data, which
// are mutually exclusive.
func resolveBody(input Input) ([]byte, error) {
	if input.Body != "" && len(input.Data) > 0 {
		return nil, fmt.Errorf("cannot use both --body and --data")
	}

	if len(input.Data) > 0 {
		return utilities.ParseDataFields(input.Data)
	}

	return utilities.ParseBody(input.Body)
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
