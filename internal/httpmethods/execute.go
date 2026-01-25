package httpmethods

import (
	"bytes"
	"net/http"

	"github.com/aladdin-io/http-client/internal/utilities"
)

func exec(input Input) (*http.Response, error) {
	reqBody, err := utilities.ParseBody(input.Body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(input.HTTPMethod, input.URL, bytes.NewReader(reqBody))
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

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, err
}
