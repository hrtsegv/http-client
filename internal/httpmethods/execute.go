package httpmethods

import (
	"bytes"
	"encoding/json"
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
	reqBody, err := utilities.ParseBody(input.Body)
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

	start := time.Now()
	resp, err := httpClient.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return nil, err
	}

	return &Result{Response: resp, Elapsed: elapsed}, nil
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
